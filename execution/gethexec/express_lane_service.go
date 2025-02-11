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
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/solgen/go/express_lane_auctiongen"
	"github.com/offchainlabs/nitro/timeboost"
	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

var (
	auctionResolutionLatency = metrics.NewRegisteredHistogram("arb/sequencer/timeboost/auctionresolution", nil, metrics.NewBoundedHistogramSample())
)

type transactionPublisher interface {
	PublishTimeboostedTransaction(context.Context, *types.Transaction, *arbitrum_types.ConditionalOptions, chan error)
}

type msgAndResult struct {
	msg        *timeboost.ExpressLaneSubmission
	resultChan chan error
}

type expressLaneRoundInfo struct {
	sequence                     uint64
	msgAndResultBySequenceNumber map[uint64]*msgAndResult
}

type expressLaneService struct {
	stopwaiter.StopWaiter
	transactionPublisher transactionPublisher
	seqConfig            SequencerConfigFetcher
	auctionContractAddr  common.Address
	apiBackend           *arbitrum.APIBackend
	roundTimingInfo      timeboost.RoundTimingInfo
	earlySubmissionGrace time.Duration
	chainConfig          *params.ChainConfig
	auctionContract      *express_lane_auctiongen.ExpressLaneAuction
	redisCoordinator     *timeboost.RedisCoordinator
	roundControl         containers.SyncMap[uint64, common.Address] // thread safe

	roundInfoMutex sync.Mutex
	roundInfo      *containers.LruCache[uint64, *expressLaneRoundInfo]
}

func newExpressLaneService(
	transactionPublisher transactionPublisher,
	seqConfig SequencerConfigFetcher,
	apiBackend *arbitrum.APIBackend,
	filterSystem *filters.FilterSystem,
	auctionContractAddr common.Address,
	bc *core.BlockChain,
	earlySubmissionGrace time.Duration,
) (*expressLaneService, error) {
	chainConfig := bc.Config()

	var contractBackend bind.ContractBackend = &contractAdapter{filters.NewFilterAPI(filterSystem), nil, apiBackend}

	auctionContract, err := express_lane_auctiongen.NewExpressLaneAuction(auctionContractAddr, contractBackend)
	if err != nil {
		return nil, err
	}

	retries := 0

pending:
	rawRoundTimingInfo, err := auctionContract.RoundTimingInfo(&bind.CallOpts{})
	if err != nil {
		const maxRetries = 5
		if errors.Is(err, bind.ErrNoCode) && retries < maxRetries {
			wait := time.Millisecond * 250 * (1 << retries)
			log.Info("ExpressLaneAuction contract not ready, will retry afer wait", "err", err, "auctionContractAddr", auctionContractAddr, "wait", wait, "maxRetries", maxRetries)
			retries++
			time.Sleep(wait)
			goto pending
		}
		return nil, err
	}
	roundTimingInfo, err := timeboost.NewRoundTimingInfo(rawRoundTimingInfo)
	if err != nil {
		return nil, err
	}

	var redisCoordinator *timeboost.RedisCoordinator
	if seqConfig().Dangerous.Timeboost.RedisUrl != "" {
		redisCoordinator, err = timeboost.NewRedisCoordinator(seqConfig().Dangerous.Timeboost.RedisUrl, roundTimingInfo.Round)
		if err != nil {
			return nil, fmt.Errorf("error initializing expressLaneService redis: %w", err)
		}
	}

	return &expressLaneService{
		transactionPublisher: transactionPublisher,
		seqConfig:            seqConfig,
		auctionContract:      auctionContract,
		apiBackend:           apiBackend,
		chainConfig:          chainConfig,
		roundTimingInfo:      *roundTimingInfo,
		earlySubmissionGrace: earlySubmissionGrace,
		auctionContractAddr:  auctionContractAddr,
		redisCoordinator:     redisCoordinator,
		roundInfo:            containers.NewLruCache[uint64, *expressLaneRoundInfo](8),
	}, nil
}

func (es *expressLaneService) Start(ctxIn context.Context) {
	es.StopWaiter.Start(ctxIn, es)

	if es.redisCoordinator != nil {
		es.redisCoordinator.Start(ctxIn)
	}

	es.LaunchThread(func(ctx context.Context) {
		// Log every new express lane auction round.
		log.Info("Watching for new express lane rounds")

		// Wait until the next round starts
		waitTime := es.roundTimingInfo.TimeTilNextRound()
		select {
		case <-ctx.Done():
			return
		case <-time.After(waitTime):
		}

		// First tick happened, now set up regular ticks
		ticker := time.NewTicker(es.roundTimingInfo.Round)
		defer ticker.Stop()
		for {
			var t time.Time
			select {
			case <-ctx.Done():
				return
			case t = <-ticker.C:
			}

			round := es.roundTimingInfo.RoundNumber()
			// TODO (BUG?) is there a race here where messages for a new round can come
			// in before this tick has been processed?
			log.Info(
				"New express lane auction round",
				"round", round,
				"timestamp", t,
			)

			// Cleanup previous round controller data
			es.roundControl.Delete(round - 1)
		}
	})

	es.LaunchThread(func(ctx context.Context) {
		// Monitor for auction resolutions from the auction manager smart contract
		// and set the express lane controller for the upcoming round accordingly.
		log.Info("Monitoring express lane auction contract")

		var fromBlock uint64
		maxBlockSpeed := es.seqConfig().MaxBlockSpeed
		latestBlock, err := es.apiBackend.HeaderByNumber(ctx, rpc.LatestBlockNumber)
		if err != nil {
			log.Error("ExpressLaneService could not get the latest header", "err", err)
		} else {
			maxBlocksPerRound := es.roundTimingInfo.Round / maxBlockSpeed
			fromBlock = latestBlock.Number.Uint64()
			// #nosec G115
			if fromBlock > uint64(maxBlocksPerRound) {
				// #nosec G115
				fromBlock -= uint64(maxBlocksPerRound)
			}
		}

		ticker := time.NewTicker(maxBlockSpeed)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				newMaxBlockSpeed := es.seqConfig().MaxBlockSpeed
				if newMaxBlockSpeed != maxBlockSpeed {
					maxBlockSpeed = newMaxBlockSpeed
					ticker.Reset(maxBlockSpeed)
				}
			}

			latestBlock, err := es.apiBackend.HeaderByNumber(ctx, rpc.LatestBlockNumber)
			if err != nil {
				log.Error("ExpressLaneService could not get the latest header", "err", err)
				continue
			}
			toBlock := latestBlock.Number.Uint64()
			if fromBlock > toBlock {
				continue
			}
			filterOpts := &bind.FilterOpts{
				Context: ctx,
				Start:   fromBlock,
				End:     &toBlock,
			}

			it, err := es.auctionContract.FilterAuctionResolved(filterOpts, nil, nil, nil)
			if err != nil {
				log.Error("Could not filter auction resolutions event", "error", err)
				continue
			}
			for it.Next() {
				timeSinceAuctionClose := es.roundTimingInfo.AuctionClosing - es.roundTimingInfo.TimeTilNextRound()
				auctionResolutionLatency.Update(timeSinceAuctionClose.Nanoseconds())
				log.Info(
					"AuctionResolved: New express lane controller assigned",
					"round", it.Event.Round,
					"controller", it.Event.FirstPriceExpressLaneController,
					"timeSinceAuctionClose", timeSinceAuctionClose,
				)
				es.roundControl.Store(it.Event.Round, it.Event.FirstPriceExpressLaneController)
			}

			// setExpressLaneIterator, err := es.auctionContract.FilterSetExpressLaneController(filterOpts, nil, nil, nil)
			// if err != nil {
			// 	log.Error("Could not filter express lane controller transfer event", "error", err)
			// 	continue
			// }
			// for setExpressLaneIterator.Next() {
			// 	if (setExpressLaneIterator.Event.PreviousExpressLaneController == common.Address{}) {
			// 		// The ExpressLaneAuction contract emits both AuctionResolved and SetExpressLaneController
			// 		// events when an auction is resolved. They contain redundant information so
			// 		// the SetExpressLaneController event can be skipped if it's related to a new round, as
			// 		// indicated by an empty PreviousExpressLaneController field (a new round has no
			// 		// previous controller).
			// 		// It is more explicit and thus clearer to use the AuctionResovled event only for the
			// 		// new round setup logic and SetExpressLaneController event only for transfers, rather
			// 		// than trying to overload everything onto SetExpressLaneController.
			// 		continue
			// 	}
			// 	currentRound := es.roundTimingInfo.RoundNumber()
			// 	round := setExpressLaneIterator.Event.Round
			// 	if round < currentRound {
			// 		log.Info("SetExpressLaneController event's round is lower than current round, not transferring control", "eventRound", round, "currentRound", currentRound)
			// 		continue
			// 	}
			// 	roundController, ok := es.roundControl.Load(round)
			// 	if !ok {
			// 		log.Warn("Could not find round info for ExpressLaneConroller transfer event", "round", round)
			// 		continue
			// 	}
			// 	if roundController != setExpressLaneIterator.Event.PreviousExpressLaneController {
			// 		log.Warn("Previous ExpressLaneController in SetExpressLaneController event does not match Sequencer previous controller, continuing with transfer to new controller anyway",
			// 			"round", round,
			// 			"sequencerRoundController", roundController,
			// 			"previous", setExpressLaneIterator.Event.PreviousExpressLaneController,
			// 			"new", setExpressLaneIterator.Event.NewExpressLaneController)
			// 	}
			// 	if roundController == setExpressLaneIterator.Event.NewExpressLaneController {
			// 		log.Warn("SetExpressLaneController: Previous and New ExpressLaneControllers are the same, not transferring control.",
			// 			"round", round,
			// 			"previous", roundController,
			// 			"new", setExpressLaneIterator.Event.NewExpressLaneController)
			// 		continue
			// 	}
			// 	es.roundControl.Store(round, setExpressLaneIterator.Event.NewExpressLaneController)
			// 	if round == currentRound {
			// 		es.roundInfoMutex.Lock()
			// 		if es.roundInfo.Contains(round) {
			// 			es.roundInfo.Add(round, &expressLaneRoundInfo{
			// 				0,
			// 				make(map[uint64]*msgAndResult),
			// 			})
			// 		}
			// 		es.roundInfoMutex.Unlock()
			// 	}
			// }
			fromBlock = toBlock + 1
		}
	})
}

func (es *expressLaneService) StopAndWait() {
	es.StopWaiter.StopAndWait()
	if es.redisCoordinator != nil {
		es.redisCoordinator.StopAndWait()
	}
}

func (es *expressLaneService) currentRoundHasController() bool {
	controller, ok := es.roundControl.Load(es.roundTimingInfo.RoundNumber())
	if !ok {
		return false
	}
	return controller != (common.Address{})
}

// sequenceExpressLaneSubmission with the roundInfo lock held, validates sequence number and sender address fields of the message
// adds the message to the transaction queue and waits for the response
func (es *expressLaneService) sequenceExpressLaneSubmission(
	ctx context.Context,
	msg *timeboost.ExpressLaneSubmission,
) error {
	unlockByDefer := true
	es.roundInfoMutex.Lock()
	defer func() {
		if unlockByDefer {
			es.roundInfoMutex.Unlock()
		}
	}()

	// Below code block isn't a repetition, it prevents stale messages to be accepted during control transfer within or after the round ends!
	controller, ok := es.roundControl.Load(msg.Round)
	if !ok {
		return timeboost.ErrNoOnchainController
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
			make(map[uint64]*msgAndResult),
		})
	}
	roundInfo, _ := es.roundInfo.Get(msg.Round)

	prev, exists := roundInfo.msgAndResultBySequenceNumber[msg.SequenceNumber]

	// Check if the submission nonce is too low.
	if msg.SequenceNumber < roundInfo.sequence {
		if exists && bytes.Equal(prev.msg.Signature, msg.Signature) {
			return nil
		}
		return timeboost.ErrSequenceNumberTooLow
	}

	// Check if a duplicate submission exists already, and reject if so.
	if exists {
		if bytes.Equal(prev.msg.Signature, msg.Signature) {
			return nil
		}
		return timeboost.ErrDuplicateSequenceNumber
	}

	seqConfig := es.seqConfig()

	// Log an informational warning if the message's sequence number is in the future.
	if msg.SequenceNumber > roundInfo.sequence {
		if seqConfig.Dangerous.Timeboost.MaxQueuedTxCount != 0 &&
			// Pending msgs count=(total msgs present in the map)-(number of processed messages=roundInfo.Sequence)
			// #nosec G115
			len(roundInfo.msgAndResultBySequenceNumber)-int(roundInfo.sequence) >= seqConfig.Dangerous.Timeboost.MaxQueuedTxCount {
			return fmt.Errorf("reached limit for queuing of future sequence number transactions, please try again with the correct sequence number. Limit: %d, Current sequence number: %d", seqConfig.Dangerous.Timeboost.MaxQueuedTxCount, roundInfo.sequence)
		}
		if msg.SequenceNumber > roundInfo.sequence+seqConfig.Dangerous.Timeboost.MaxFutureSequenceDistance {
			return fmt.Errorf("message sequence number has reached max allowed limit. SequenceNumber: %d, Limit: %d", msg.SequenceNumber, roundInfo.sequence+seqConfig.Dangerous.Timeboost.MaxFutureSequenceDistance)
		}
		log.Info("Received express lane submission with future sequence number", "SequenceNumber", msg.SequenceNumber)
	}

	// Put into the sequence number map.
	resultChan := make(chan error, 1)
	roundInfo.msgAndResultBySequenceNumber[msg.SequenceNumber] = &msgAndResult{msg, resultChan}

	if es.redisCoordinator != nil {
		es.LaunchThread(func(context.Context) {
			// Persist accepted expressLane txs to redis
			if err := es.redisCoordinator.AddAcceptedTx(msg); err != nil {
				log.Error("Error adding accepted ExpressLaneSubmission to redis. Loss of msg possible if sequencer switch happens", "seqNum", msg.SequenceNumber, "txHash", msg.Transaction.Hash(), "err", err)
			}
		})
	}

	now := time.Now()
	queueTimeout := seqConfig.QueueTimeout
	for es.roundTimingInfo.RoundNumber() == msg.Round { // This check ensures that the controller for this round is not allowed to send transactions from msgAndResultBySequenceNumber map once the next round starts
		// Get the next message in the sequence.
		nextMsgAndResult, exists := roundInfo.msgAndResultBySequenceNumber[roundInfo.sequence]
		if !exists {
			break
		}
		// Queued txs cannot use this message's context as it would lead to context canceled error once the result for this message is available and returned
		// Hence using es.GetContext() allows unblocking of queued up txs even if current tx's context has errored out
		var queueCtx context.Context
		var cancel context.CancelFunc
		queueCtx, _ = ctxWithTimeout(es.GetContext(), queueTimeout)
		if nextMsgAndResult.msg.SequenceNumber == msg.SequenceNumber {
			queueCtx, cancel = ctxWithTimeout(ctx, queueTimeout)
			defer cancel()
		}
		es.transactionPublisher.PublishTimeboostedTransaction(queueCtx, nextMsgAndResult.msg.Transaction, nextMsgAndResult.msg.Options, nextMsgAndResult.resultChan)
		// Increase the global round sequence number.
		roundInfo.sequence += 1
	}

	seqCount := roundInfo.sequence
	es.roundInfo.Add(msg.Round, roundInfo)
	unlockByDefer = false
	es.roundInfoMutex.Unlock() // Release lock so that other timeboost txs can be processed

	abortCtx, cancel := ctxWithTimeout(ctx, queueTimeout*2) // We use the same timeout value that sequencer imposes
	defer cancel()
	select {
	case err = <-resultChan:
	case <-abortCtx.Done():
		if ctx.Err() == nil {
			log.Warn("Transaction sequencing hit abort deadline", "err", abortCtx.Err(), "submittedAt", now, "TxProcessingTimeout", queueTimeout*2, "txHash", msg.Transaction.Hash())
		}
		err = fmt.Errorf("Transaction sequencing hit timeout, result for the submitted transaction is not yet available: %w", abortCtx.Err())
	}

	if es.redisCoordinator != nil {
		es.LaunchThread(func(context.Context) {
			// We update the sequence count in redis only after receiving a result for sequencing this message, instead of updating while holding roundInfoMutex,
			// because this prevents any loss of transactions when the prev chosen sequencer updates the count but some how fails to forward txs to the current chosen.
			// If the prev chosen ends up forwarding the tx, it is ok as the duplicate txs will be discarded
			if redisErr := es.redisCoordinator.UpdateSequenceCount(msg.Round, seqCount); redisErr != nil {
				log.Error("Error updating round's sequence count in redis", "err", redisErr) // this shouldn't be a problem if future msgs succeed in updating the count
			}
		})
	}

	if err != nil {
		// If the tx fails we return an error with all the necessary info for the controller
		return fmt.Errorf("%w: Sequence number: %d (consumed), Transaction hash: %v, Error: %w", timeboost.ErrAcceptedTxFailed, msg.SequenceNumber, msg.Transaction.Hash(), err)
	}
	return nil
}

// validateExpressLaneTx checks for the correctness of all fields of msg
func (es *expressLaneService) validateExpressLaneTx(msg *timeboost.ExpressLaneSubmission) error {
	if msg == nil || msg.Transaction == nil || msg.Signature == nil {
		return timeboost.ErrMalformedData
	}
	if msg.ChainId.Cmp(es.chainConfig.ChainID) != 0 {
		return errors.Wrapf(timeboost.ErrWrongChainId, "express lane tx chain ID %d does not match current chain ID %d", msg.ChainId, es.chainConfig.ChainID)
	}
	if msg.AuctionContractAddress != es.auctionContractAddr {
		return errors.Wrapf(timeboost.ErrWrongAuctionContract, "msg auction contract address %s does not match sequencer auction contract address %s", msg.AuctionContractAddress, es.auctionContractAddr)
	}

	currentRound := es.roundTimingInfo.RoundNumber()
	if msg.Round != currentRound {
		timeTilNextRound := es.roundTimingInfo.TimeTilNextRound()
		// We allow txs to come in for the next round if it is close enough to that round,
		// but we sleep until the round starts.
		if msg.Round == currentRound+1 && timeTilNextRound <= es.earlySubmissionGrace {
			time.Sleep(timeTilNextRound)
		} else {
			return errors.Wrapf(timeboost.ErrBadRoundNumber, "express lane tx round %d does not match current round %d", msg.Round, currentRound)
		}
	}

	controller, ok := es.roundControl.Load(msg.Round)
	if !ok {
		return timeboost.ErrNoOnchainController
	}
	// Extract sender address and cache it to be later used by sequenceExpressLaneSubmission
	sender, err := msg.Sender()
	if err != nil {
		return err
	}
	if sender != controller {
		return timeboost.ErrNotExpressLaneController
	}
	return nil
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
		roundInfo = &expressLaneRoundInfo{0, make(map[uint64]*msgAndResult)}
	}
	if redisSeqCount > roundInfo.sequence {
		roundInfo.sequence = redisSeqCount
	}
	es.roundInfo.Add(currentRound, roundInfo)
	sequenceCount := roundInfo.sequence
	es.roundInfoMutex.Unlock()

	pendingMsgs := es.redisCoordinator.GetAcceptedTxs(currentRound, sequenceCount, sequenceCount+es.seqConfig().Dangerous.Timeboost.MaxFutureSequenceDistance)
	log.Info("Attempting to sequence pending expressLane transactions from redis", "count", len(pendingMsgs))
	for _, msg := range pendingMsgs {
		es.LaunchThread(func(ctx context.Context) {
			if err := es.sequenceExpressLaneSubmission(ctx, msg); err != nil {
				log.Error("Untracked expressLaneSubmission returned an error", "round", msg.Round, "seqNum", msg.SequenceNumber, "txHash", msg.Transaction.Hash(), "err", err)
			}
		})
	}
}
