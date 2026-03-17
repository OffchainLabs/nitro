// Copyright 2024-2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package timeboost

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/util/ctxhelper"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

type expressLaneRoundInfo struct {
	sequence uint64

	// The per-round sequence number reordering queue
	msgBySequenceNumber map[uint64]*ExpressLaneSubmission
}

type ExpressLaneService struct {
	stopwaiter.StopWaiter
	transactionPublisher TransactionPublisher
	seqConfig            ExpressLaneServiceConfigFetcher
	roundTimingInfo      RoundTimingInfo
	redisCoordinator     *RedisCoordinator

	roundInfoMutex sync.Mutex
	roundInfo      *containers.LruCache[uint64, *expressLaneRoundInfo]

	tracker *ExpressLaneTracker
}

func NewExpressLaneService(
	transactionPublisher TransactionPublisher,
	seqConfig ExpressLaneServiceConfigFetcher,
	roundTimingInfo *RoundTimingInfo,
	expressLaneTracker *ExpressLaneTracker,
) (*ExpressLaneService, error) {
	var err error
	var redisCoordinator *RedisCoordinator
	if seqConfig().RedisUrl != "" {
		redisCoordinator, err = NewRedisCoordinator(seqConfig().RedisUrl, roundTimingInfo, seqConfig().RedisUpdateEventsChannelSize)
		if err != nil {
			return nil, fmt.Errorf("error initializing ExpressLaneService redis: %w", err)
		}
	}

	return &ExpressLaneService{
		transactionPublisher: transactionPublisher,
		seqConfig:            seqConfig,
		roundTimingInfo:      *roundTimingInfo,
		redisCoordinator:     redisCoordinator,
		roundInfo:            containers.NewLruCache[uint64, *expressLaneRoundInfo](8),
		tracker:              expressLaneTracker,
	}, nil
}

func (es *ExpressLaneService) Start(ctxIn context.Context) {
	es.StopWaiter.Start(ctxIn, es)

	if es.redisCoordinator != nil {
		es.redisCoordinator.Start(es.GetContext())
	}
}

func (es *ExpressLaneService) StopAndWait() {
	if es.redisCoordinator != nil {
		es.redisCoordinator.StopAndWait()
	}
	// tracker is started by ExecutionNode, not by expressLaneService,
	// but stopped here because no one else does it.
	if es.tracker != nil {
		es.tracker.StopAndWait()
	}
	es.StopWaiter.StopAndWait()
}

// DontCareSequence is a special sequence number that indicates a transaction should bypass the
// normal sequence ordering requirements and be processed immediately
const DontCareSequence = math.MaxUint64

// SequenceExpressLaneSubmission with the roundInfo lock held, validates sequence number and sender address fields of the message
// adds the message to the sequencer transaction queue
func (es *ExpressLaneService) SequenceExpressLaneSubmission(msg *ExpressLaneSubmission) error {
	if msg.SequenceNumber == DontCareSequence {
		// Don't store DontCareSequence txs with the redisCoordinator. The redisCoordinator is
		// meant for restoring messages in the reordering queue if the sequencer fails over,
		// but for messages with DontCareSequence we skip the reordernig queue.

		if es.roundTimingInfo.RoundNumber() != msg.Round {
			return errors.Wrapf(ErrBadRoundNumber, "express lane tx round %d does not match current round %d", msg.Round, es.roundTimingInfo.RoundNumber())
		}

		// Process immediately without affecting sequence ordering
		timeout := min(es.roundTimingInfo.TimeTilNextRound(), es.seqConfig().QueueTimeout)
		queueCtx, _ := ctxhelper.WithTimeoutOrCancel(es.GetContext(), timeout)
		return es.transactionPublisher.PublishTimeboostedTransaction(queueCtx, msg.Transaction, msg.Options)
	}

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
		return ErrNotExpressLaneController
	}

	// If expressLaneRoundInfo for current round doesn't exist yet, we'll add it to the cache
	if !es.roundInfo.Contains(msg.Round) {
		es.roundInfo.Add(msg.Round, &expressLaneRoundInfo{
			0,
			make(map[uint64]*ExpressLaneSubmission),
		})
	}
	roundInfo, _ := es.roundInfo.Get(msg.Round)

	prev, exists := roundInfo.msgBySequenceNumber[msg.SequenceNumber]

	// Check if the submission nonce is too low.
	if msg.SequenceNumber < roundInfo.sequence {
		if exists && bytes.Equal(prev.Signature, msg.Signature) {
			return nil
		}
		return ErrSequenceNumberTooLow
	}

	// Check if a duplicate submission exists already, and reject if so.
	if exists {
		if bytes.Equal(prev.Signature, msg.Signature) {
			return nil
		}
		return ErrDuplicateSequenceNumber
	}

	seqConfig := es.seqConfig()
	// Log an informational warning if the message's sequence number is in the future.
	if msg.SequenceNumber > roundInfo.sequence {
		if msg.SequenceNumber > roundInfo.sequence+seqConfig.MaxFutureSequenceDistance {
			return fmt.Errorf("message sequence number has reached max allowed limit. SequenceNumber: %d, ExpectedSequenceNumber: %d, Limit: %d", msg.SequenceNumber, roundInfo.sequence, roundInfo.sequence+seqConfig.MaxFutureSequenceDistance)
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
		queueCtx, _ := ctxhelper.WithTimeoutOrCancel(es.GetContext(), timeout)
		if err := es.transactionPublisher.PublishTimeboostedTransaction(queueCtx, nextMsg.Transaction, nextMsg.Options); err != nil {
			logLevel := log.Error
			// If tx sequencing was attempted right around the edge of a round then an error due to context timing out is expected, so we log a warning in such a case
			if errors.Is(err, queueCtx.Err()) && timeout < time.Second {
				logLevel = log.Warn
			}
			logLevel("Error queuing expressLane transaction", "seqNum", nextMsg.SequenceNumber, "txHash", nextMsg.Transaction.Hash(), "err", err)
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

func (es *ExpressLaneService) SyncFromRedis() {
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
		roundInfo = &expressLaneRoundInfo{0, make(map[uint64]*ExpressLaneSubmission)}
	}
	if redisSeqCount > roundInfo.sequence {
		roundInfo.sequence = redisSeqCount
	}
	es.roundInfo.Add(currentRound, roundInfo)
	sequenceCount := roundInfo.sequence
	es.roundInfoMutex.Unlock()

	pendingMsgs := es.redisCoordinator.GetAcceptedTxs(currentRound, sequenceCount, sequenceCount+es.seqConfig().MaxFutureSequenceDistance)
	log.Info("Attempting to sequence pending expressLane transactions from redis", "count", len(pendingMsgs))
	for _, msg := range pendingMsgs {
		if err := es.SequenceExpressLaneSubmission(msg); err != nil {
			log.Error("Untracked expressLaneSubmission returned an error while sequencing", "round", msg.Round, "seqNum", msg.SequenceNumber, "txHash", msg.Transaction.Hash(), "err", err)
		}
	}
}

func (es *ExpressLaneService) CurrentRoundHasController() bool {
	controller, err := es.tracker.RoundController(es.roundTimingInfo.RoundNumber())
	if err != nil {
		return false
	}
	return controller != (common.Address{})
}

func (es *ExpressLaneService) GetRoundTimingInfo() *RoundTimingInfo {
	return &es.roundTimingInfo
}

func (es *ExpressLaneService) AuctionContractAddr() common.Address {
	return es.tracker.AuctionContractAddr()
}

func (es *ExpressLaneService) ValidateExpressLaneTx(msg *ExpressLaneSubmission) error {
	return es.tracker.ValidateExpressLaneTx(msg)
}
