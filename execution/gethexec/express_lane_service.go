// Copyright 2024-2025, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package gethexec

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/arbitrum"
	"github.com/ethereum/go-ethereum/arbitrum_types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/filters"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"

	"github.com/offchainlabs/nitro/solgen/go/express_lane_auctiongen"
	"github.com/offchainlabs/nitro/timeboost"
	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

var (
	auctionResolutionLatency = metrics.NewRegisteredGauge("arb/sequencer/timeboost/auctionresolution", nil)
)

type transactionPublisher interface {
	PublishTimeboostedTransaction(context.Context, *types.Transaction, *arbitrum_types.ConditionalOptions) error
}

type expressLaneRoundInfo struct {
	sequence            uint64
	msgBySequenceNumber map[uint64]*timeboost.ExpressLaneSubmission
}

type expressLaneService struct {
	stopwaiter.StopWaiter
	transactionPublisher transactionPublisher
	seqConfig            SequencerConfigFetcher
	roundTimingInfo      timeboost.RoundTimingInfo
	redisCoordinator     *timeboost.RedisCoordinator

	roundInfoMutex sync.Mutex
	roundInfo      *containers.LruCache[uint64, *expressLaneRoundInfo]

	tracker *ExpressLaneTracker
}

func NewExpressLaneAuctionFromInternalAPI(
	apiBackend *arbitrum.APIBackend,
	filterSystem *filters.FilterSystem,
	auctionContractAddr common.Address,
) (*express_lane_auctiongen.ExpressLaneAuction, error) {
	var contractBackend bind.ContractBackend = &contractAdapter{filters.NewFilterAPI(filterSystem), nil, apiBackend}

	auctionContract, err := express_lane_auctiongen.NewExpressLaneAuction(auctionContractAddr, contractBackend)
	if err != nil {
		return nil, err
	}

	return auctionContract, nil
}

func GetRoundTimingInfo(
	auctionContract *express_lane_auctiongen.ExpressLaneAuction,
) (*timeboost.RoundTimingInfo, error) {
	retries := 0

pending:
	rawRoundTimingInfo, err := auctionContract.RoundTimingInfo(&bind.CallOpts{})
	if err != nil {
		const maxRetries = 5
		if errors.Is(err, bind.ErrNoCode) && retries < maxRetries {
			wait := time.Millisecond * 250 * (1 << retries)
			log.Info("ExpressLaneAuction contract not ready, will retry afer wait", "err", err, "wait", wait, "maxRetries", maxRetries)
			retries++
			time.Sleep(wait)
			goto pending
		}
		return nil, err
	}
	return timeboost.NewRoundTimingInfo(rawRoundTimingInfo)
}

func newExpressLaneService(
	transactionPublisher transactionPublisher,
	seqConfig SequencerConfigFetcher,
	roundTimingInfo *timeboost.RoundTimingInfo,
	bc *core.BlockChain,
	expressLaneTracker *ExpressLaneTracker,
) (*expressLaneService, error) {
	var err error
	var redisCoordinator *timeboost.RedisCoordinator
	if seqConfig().Timeboost.RedisUrl != "" {
		redisCoordinator, err = timeboost.NewRedisCoordinator(seqConfig().Timeboost.RedisUrl, roundTimingInfo, seqConfig().Timeboost.RedisUpdateEventsChannelSize)
		if err != nil {
			return nil, fmt.Errorf("error initializing expressLaneService redis: %w", err)
		}
	}

	return &expressLaneService{
		transactionPublisher: transactionPublisher,
		seqConfig:            seqConfig,
		roundTimingInfo:      *roundTimingInfo,
		redisCoordinator:     redisCoordinator,
		roundInfo:            containers.NewLruCache[uint64, *expressLaneRoundInfo](8),
		tracker:              expressLaneTracker,
	}, nil
}

func (es *expressLaneService) Start(ctxIn context.Context) {
	es.StopWaiter.Start(ctxIn, es)

	if es.redisCoordinator != nil {
		es.redisCoordinator.Start(ctxIn)
	}
}

func (es *expressLaneService) StopAndWait() {
	es.StopWaiter.StopAndWait()
	if es.redisCoordinator != nil {
		es.redisCoordinator.StopAndWait()
	}
	if es.tracker != nil {
		es.tracker.StopAndWait()
	}
}

// sequenceExpressLaneSubmission with the roundInfo lock held, validates sequence number and sender address fields of the message
// adds the message to the sequencer transaction queue
func (es *expressLaneService) sequenceExpressLaneSubmission(msg *timeboost.ExpressLaneSubmission) error {
	es.roundInfoMutex.Lock()
	defer es.roundInfoMutex.Unlock()

	// Below code block isn't a repetition, it prevents stale messages to be accepted during control transfer within or after the round ends!
	controller, err := es.tracker.RoundController(msg.Round)
	if err != nil {
		return err
	}
	sender, err := msg.Sender() // Doesn't recompute sender address
	if err != nil {
		return err
	}
	if sender != controller {
		return timeboost.ErrNotExpressLaneController
	}

	// If expressLaneRoundInfo for current round doesn't exist yet, we'll add it to the cache
	if !es.roundInfo.Contains(msg.Round) {
		es.roundInfo.Add(msg.Round, &expressLaneRoundInfo{
			0,
			make(map[uint64]*timeboost.ExpressLaneSubmission),
		})
	}
	roundInfo, _ := es.roundInfo.Get(msg.Round)

	prev, exists := roundInfo.msgBySequenceNumber[msg.SequenceNumber]

	// Check if the submission nonce is too low.
	if msg.SequenceNumber < roundInfo.sequence {
		if exists && bytes.Equal(prev.Signature, msg.Signature) {
			return nil
		}
		return timeboost.ErrSequenceNumberTooLow
	}

	// Check if a duplicate submission exists already, and reject if so.
	if exists {
		if bytes.Equal(prev.Signature, msg.Signature) {
			return nil
		}
		return timeboost.ErrDuplicateSequenceNumber
	}

	seqConfig := es.seqConfig()

	// Log an informational warning if the message's sequence number is in the future.
	if msg.SequenceNumber > roundInfo.sequence {
		if msg.SequenceNumber > roundInfo.sequence+seqConfig.Timeboost.MaxFutureSequenceDistance {
			return fmt.Errorf("message sequence number has reached max allowed limit. SequenceNumber: %d, ExpectedSequenceNumber: %d, Limit: %d", msg.SequenceNumber, roundInfo.sequence, roundInfo.sequence+seqConfig.Timeboost.MaxFutureSequenceDistance)
		}
		log.Info("Received express lane submission with future sequence number", "SequenceNumber", msg.SequenceNumber)
	}

	// Put into the sequence number map.
	roundInfo.msgBySequenceNumber[msg.SequenceNumber] = msg

	if es.redisCoordinator != nil {
		// Persist accepted expressLane txs to redis
		if err := es.redisCoordinator.AddAcceptedTx(msg); err != nil {
			log.Error("Error adding accepted ExpressLaneSubmission to redis. Loss of msg possible if sequencer switch happens", "seqNum", msg.SequenceNumber, "txHash", msg.Transaction.Hash(), "err", err)
		}
	}

	var retErr error
	queueTimeout := seqConfig.QueueTimeout
	for es.roundTimingInfo.RoundNumber() == msg.Round { // This check ensures that the controller for this round is not allowed to send transactions from msgBySequenceNumber map once the next round starts
		// Get the next message in the sequence.
		nextMsg, exists := roundInfo.msgBySequenceNumber[roundInfo.sequence]
		if !exists {
			break
		}
		// Txs (current or buffered) cannot use this function's context as it would lead to context canceled error later on, once the tx is queued and this function returns, hence we
		// use es.GetContext(). Txs sequenced this round shouldn't be processed by sequencer into next round, to enforce this, queueCtx has a timeout = min(TimeTilNextRound, queueTimeout)
		timeout := min(es.roundTimingInfo.TimeTilNextRound(), queueTimeout)
		queueCtx, _ := ctxWithTimeout(es.GetContext(), timeout)
		if err := es.transactionPublisher.PublishTimeboostedTransaction(queueCtx, nextMsg.Transaction, nextMsg.Options); err != nil {
			log.Error("Error queuing expressLane transaction", "seqNum", nextMsg.SequenceNumber, "txHash", nextMsg.Transaction.Hash(), "err", err)
			if nextMsg.SequenceNumber == msg.SequenceNumber {
				retErr = err
			}
		}
		// Increase the global round sequence number.
		roundInfo.sequence += 1
	}
	es.roundInfo.Add(msg.Round, roundInfo)

	if es.redisCoordinator != nil {
		// We update the sequence count in redis after we were able to queue the txs up until roundInfo.sequence
		if redisErr := es.redisCoordinator.UpdateSequenceCount(msg.Round, roundInfo.sequence); redisErr != nil {
			log.Error("Error updating round's sequence count in redis", "err", redisErr) // this shouldn't be a problem if future msgs succeed in updating the count
		}
	}

	return retErr
}

func (es *expressLaneService) syncFromRedis() {
	if es.redisCoordinator == nil {
		return
	}

	currentRound := es.roundTimingInfo.RoundNumber()
	redisSeqCount, err := es.redisCoordinator.GetSequenceCount(currentRound)
	if err != nil {
		log.Error("error fetching current round's global sequence count from redis", "err", err)
	}

	es.roundInfoMutex.Lock()
	roundInfo, exists := es.roundInfo.Get(currentRound)
	if !exists {
		// If expressLaneRoundInfo for current round doesn't exist yet, we'll add it to the cache
		roundInfo = &expressLaneRoundInfo{0, make(map[uint64]*timeboost.ExpressLaneSubmission)}
	}
	if redisSeqCount > roundInfo.sequence {
		roundInfo.sequence = redisSeqCount
	}
	es.roundInfo.Add(currentRound, roundInfo)
	sequenceCount := roundInfo.sequence
	es.roundInfoMutex.Unlock()

	pendingMsgs := es.redisCoordinator.GetAcceptedTxs(currentRound, sequenceCount, sequenceCount+es.seqConfig().Timeboost.MaxFutureSequenceDistance)
	log.Info("Attempting to sequence pending expressLane transactions from redis", "count", len(pendingMsgs))
	for _, msg := range pendingMsgs {
		if err := es.sequenceExpressLaneSubmission(msg); err != nil {
			log.Error("Untracked expressLaneSubmission returned an error while sequencing", "round", msg.Round, "seqNum", msg.SequenceNumber, "txHash", msg.Transaction.Hash(), "err", err)
		}
	}
}

func (es *expressLaneService) currentRoundHasController() bool {
	controller, err := es.tracker.RoundController(es.roundTimingInfo.RoundNumber())
	if err != nil {
		return false
	}
	return controller != (common.Address{})
}

func (es *expressLaneService) AuctionContractAddr() common.Address {
	return es.tracker.AuctionContractAddr()
}

func (es *expressLaneService) ValidateExpressLaneTx(msg *timeboost.ExpressLaneSubmission) error {
	return es.tracker.ValidateExpressLaneTx(msg)
}
