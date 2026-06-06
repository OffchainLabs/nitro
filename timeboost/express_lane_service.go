// Copyright 2024-2026, Offchain Labs, Inc.
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
	"github.com/ethereum/go-ethereum/metrics"

	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/util/ctxhelper"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

var (
	// total time inside SequenceExpressLaneSubmission
	expressLaneSequencingDurationHistogram = metrics.NewRegisteredHistogram("arb/sequencer/timeboost/expresslane/sequencingduration", nil, metrics.NewBoundedHistogramSample())
	// wait to acquire roundInfoMutex
	expressLaneMutexWaitHistogram = metrics.NewRegisteredHistogram("arb/sequencer/timeboost/expresslane/mutexwait", nil, metrics.NewBoundedHistogramSample())
	// time a submission sat in the reordering queue before publishing
	expressLaneReorderResidencyHistogram = metrics.NewRegisteredHistogram("arb/sequencer/timeboost/expresslane/reorderresidency", nil, metrics.NewBoundedHistogramSample())
	// pending (buffered, not yet publishable) submissions
	expressLaneReorderBufferSizeGauge = metrics.NewRegisteredGauge("arb/sequencer/timeboost/expresslane/reorderbuffersize", nil)
	// submissions arriving with a future sequence number
	expressLaneFutureSeqCounter = metrics.NewRegisteredCounter("arb/sequencer/timeboost/expresslane/futureseqsubmissions", nil)
)

type expressLaneRoundInfo struct {
	sequence uint64

	// The per-round sequence number reordering queue
	msgBySequenceNumber map[uint64]*ExpressLaneSubmission

	// arrival time per not-yet-published submission; deleted on publish, so len = pending backlog
	arrivalTimeBySequenceNumber map[uint64]time.Time
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
		es.StartAndTrackChild(es.redisCoordinator)
	}
	// tracker is started and tracked by ExecutionNode, not by ExpressLaneService.
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

	lockRequestedAt := time.Now()
	es.roundInfoMutex.Lock()
	defer es.roundInfoMutex.Unlock()
	expressLaneMutexWaitHistogram.Update(time.Since(lockRequestedAt).Microseconds())

	sequencingStart := time.Now()
	defer func() {
		expressLaneSequencingDurationHistogram.Update(time.Since(sequencingStart).Microseconds())
	}()

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
			sequence:                    0,
			msgBySequenceNumber:         make(map[uint64]*ExpressLaneSubmission),
			arrivalTimeBySequenceNumber: make(map[uint64]time.Time),
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
		expressLaneFutureSeqCounter.Inc(1)
		log.Info("Received express lane submission with future sequence number; buffering until predecessors arrive",
			"round", msg.Round,
			"SequenceNumber", msg.SequenceNumber,
			"expectedSequenceNumber", roundInfo.sequence,
			"gap", msg.SequenceNumber-roundInfo.sequence,
			"txHash", msg.Transaction.Hash(),
		)
	}

	// Put into the sequence number map.
	roundInfo.msgBySequenceNumber[msg.SequenceNumber] = msg
	roundInfo.arrivalTimeBySequenceNumber[msg.SequenceNumber] = time.Now()

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
		if arrival, ok := roundInfo.arrivalTimeBySequenceNumber[roundInfo.sequence]; ok {
			residency := time.Since(arrival)
			expressLaneReorderResidencyHistogram.Update(residency.Microseconds())
			if residency > time.Second {
				log.Info("Express lane tx waited in reordering queue before being published",
					"round", msg.Round,
					"SequenceNumber", roundInfo.sequence,
					"residency", residency,
					"txHash", nextMsg.Transaction.Hash(),
				)
			}
			delete(roundInfo.arrivalTimeBySequenceNumber, roundInfo.sequence)
		}
		// Increase the global round sequence number.
		roundInfo.sequence += 1
	}
	expressLaneReorderBufferSizeGauge.Update(int64(len(roundInfo.arrivalTimeBySequenceNumber)))
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
		roundInfo = &expressLaneRoundInfo{
			sequence:                    0,
			msgBySequenceNumber:         make(map[uint64]*ExpressLaneSubmission),
			arrivalTimeBySequenceNumber: make(map[uint64]time.Time),
		}
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

func (es *ExpressLaneService) GetRoundTimingInfo() RoundTimingInfo {
	return es.roundTimingInfo
}

func (es *ExpressLaneService) AuctionContractAddr() common.Address {
	return es.tracker.AuctionContractAddr()
}

func (es *ExpressLaneService) ValidateExpressLaneTx(msg *ExpressLaneSubmission) error {
	return es.tracker.ValidateExpressLaneTx(msg)
}
