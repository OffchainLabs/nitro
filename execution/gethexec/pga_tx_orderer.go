// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package gethexec

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/execution/gethexec/pga"
	"github.com/offchainlabs/nitro/util/arbmath"
)

var (
	// PGA Round Metrics
	pgaRoundsCompletedCounter          = metrics.NewRegisteredCounter("arb/sequencer/pga/rounds/completed", nil)
	pgaRoundsTxExhaustedCounter        = metrics.NewRegisteredCounter("arb/sequencer/pga/rounds/txexhausted", nil)
	pgaRoundsBlockFilledCounter        = metrics.NewRegisteredCounter("arb/sequencer/pga/rounds/blockfilled", nil)
	pgaRoundsDeadlineReachedCounter    = metrics.NewRegisteredCounter("arb/sequencer/pga/rounds/deadlinereached", nil)
	pgaRoundExecutionDurationHistogram = metrics.NewRegisteredHistogram("arb/sequencer/pga/rounds/executionduration", nil, metrics.NewBoundedHistogramSample())
	// Transaction Metrics
	pgaTimeToInclusionHistogram       = metrics.NewRegisteredHistogram("arb/sequencer/pga/tx/timetoinclusion", nil, metrics.NewBoundedHistogramSample())
	pgaPayingTimeToInclusionHistogram = metrics.NewRegisteredHistogram("arb/sequencer/pga/tx/payingtimetoinclusion", nil, metrics.NewBoundedHistogramSample())
	pgaDwellTimeHistogram             = metrics.NewRegisteredHistogram("arb/sequencer/pga/tx/dwelltime", nil, metrics.NewBoundedHistogramSample())
	pgaPayingDwellTimeHistogram       = metrics.NewRegisteredHistogram("arb/sequencer/pga/tx/payingdwelltime", nil, metrics.NewBoundedHistogramSample())
	pgaTxTipHistogram                 = metrics.NewRegisteredHistogram("arb/sequencer/pga/tx/tips", nil, metrics.NewBoundedHistogramSample())
	pgaTxBoostHistogram               = metrics.NewRegisteredHistogram("arb/sequencer/pga/tx/boost", nil, metrics.NewBoundedHistogramSample())
	pgaTxPriorityHistogram            = metrics.NewRegisteredHistogram("arb/sequencer/pga/tx/priority", nil, metrics.NewBoundedHistogramSample())
	pgaTxStarvationSLAHistogram       = metrics.NewRegisteredHistogram("arb/sequencer/pga/tx/starvationsla", nil, metrics.NewBoundedHistogramSample())
)

// pgaTxOrderer is the priority-gas-auction TxOrderer.
type pgaTxOrderer struct {
	txOrdererConfig
	seq txOrdererSequencer
	ctx context.Context // bounds the round-boundary waits

	mempool  *pga.Mempool[txQueueItem]
	schedule *pga.Schedule

	roundsPerBlock uint
}

var _ txOrderer = (*pgaTxOrderer)(nil)

func NewPGATxOrderer(ctx context.Context, seq txOrdererSequencer, config txOrdererConfig, roundsPerBlock uint) *pgaTxOrderer {
	return &pgaTxOrderer{
		txOrdererConfig: config,
		seq:             seq,
		ctx:             ctx,
		roundsPerBlock:  roundsPerBlock,
	}
}

func (p *pgaTxOrderer) CurrentRound() uint64 {
	return p.schedule.Round()
}

func (p *pgaTxOrderer) NextQueueItem(statedb *state.StateDB, remainingBlockSize int, blockGasLeft uint64) (txQueueItem, ordererStatus) {
	if blockGasLeft < params.TxGas {
		p.endRound(pgaRoundsBlockFilledCounter)
		return txQueueItem{}, blockGasLimitReached
	}

	for {
		if queueEmpty := p.mempool.PriorityQueueLen() == 0; queueEmpty || p.schedule.RoundIsOver() {
			reason := pgaRoundsDeadlineReachedCounter
			if queueEmpty {
				reason = pgaRoundsTxExhaustedCounter
			}
			p.endRound(reason)
			if p.schedule.IsLastRound() {
				limitReason := blockTimeLimitReached
				if queueEmpty {
					limitReason = exhaustedQueue
				}
				return txQueueItem{}, limitReason
			}
			err := p.schedule.WaitAndAdvanceRound(p.ctx)
			if err != nil {
				log.Warn("PGA round wait interrupted; ending the block early", "err", err)
				return txQueueItem{}, blockInterrupted
			}
			mempoolCapacity := max(p.maxBlockTxCandidates-p.mempool.PriorityQueueLen(), 0)
			p.promote(p.seq.drainValidatedTxs(statedb, p.baseFee, mempoolCapacity))
		}
		item, ok := p.mempool.ValidateAndPeek()
		if !ok {
			continue
		}

		// If the next tx is too big to fit in the remaining block space, we leave it in the mempool and stop
		// sequencing. The sequencer will finalize the block and start a new one, with a fresh mempool and schedule.
		if item.txSize > remainingBlockSize {
			p.endRound(pgaRoundsBlockFilledCounter)
			return txQueueItem{}, blockSizeLimitReached
		}

		popped, ok := p.mempool.Pop()
		if !ok {
			continue
		}
		return popped, fetchedTx
	}
}

// endRound counts a round ending for the given reason and records its execute-phase duration.
func (p *pgaTxOrderer) endRound(reason *metrics.Counter) {
	pgaRoundsCompletedCounter.Inc(1)
	reason.Inc(1)
	pgaRoundExecutionDurationHistogram.Update(p.schedule.RoundElapsed().Microseconds())
	p.mempool.ApplyRoundBoost()
}

// promote pushes the drained txs into the priority queue and records the dwell-time metrics.
func (p *pgaTxOrderer) promote(items []txQueueItem) {
	for _, item := range p.mempool.PushBatch(items) {
		recordPromotedTx(item)
	}
}

// recordPromotedTx records the dwell time on the tx's first promotion into the priority queue.
func recordPromotedTx(item txQueueItem) {
	if !item.MarkPromoted() {
		return
	}
	dwell := time.Since(item.firstAppearance).Microseconds()
	pgaDwellTimeHistogram.Update(dwell)
	if item.GetTip() > 0 {
		pgaPayingDwellTimeHistogram.Update(dwell)
	}
}

// recordIncludedPGATx records the tx-level PGA metrics for a tx included in the block being built.
func recordIncludedPGATx(item txQueueItem) {
	inclusion := time.Since(item.firstAppearance).Microseconds()
	pgaTimeToInclusionHistogram.Update(inclusion)
	if item.GetTip() > 0 {
		pgaPayingTimeToInclusionHistogram.Update(inclusion)
	}
	pgaTxTipHistogram.Update(arbmath.SaturatingCast[int64](item.GetTip()))
	pgaTxBoostHistogram.Update(arbmath.SaturatingCast[int64](item.GetBoost()))
	pgaTxPriorityHistogram.Update(arbmath.SaturatingCast[int64](item.GetPriority()))
	pgaTxStarvationSLAHistogram.Update(arbmath.SaturatingCast[int64](item.GetRoundsWaited()))
}

func (p *pgaTxOrderer) StartBlock(statedb *state.StateDB) (hasWork bool) {
	roundLength := pgaRoundLength(p.maxBlockSpeed, p.roundsPerBlock)
	p.schedule = pga.NewSchedule(uint64(p.roundsPerBlock), roundLength)
	p.mempool = pga.NewMempool[txQueueItem](p.roundsPerBlock, p.baseFee)

	p.promote(p.seq.drainValidatedTxs(statedb, p.baseFee, p.maxBlockTxCandidates))

	return p.mempool.PriorityQueueLen() > 0
}

func (p *pgaTxOrderer) TakeRemaining() []txQueueItem {
	// The deferred block cleanup can run before StartBlock arms the mempool.
	if p.mempool == nil {
		return nil
	}
	return p.mempool.TakeRemaining()
}

func (p *pgaTxOrderer) OnTxInclusion(queueItem txQueueItem) {
	p.mempool.RecordIncludedTx(queueItem.GetPriority())
	recordIncludedPGATx(queueItem)
}

// BlockInterval spans the block's elapsed rounds; a no-work block never leaves round 1, so
// empty attempts retry on the round cadence.
func (p *pgaTxOrderer) BlockInterval() time.Duration {
	return p.schedule.ElapsedInterval()
}

// OnNonceGapResolved pushes the revived tx straight into the current round's auction: waiting
// for the next round-boundary drain would delay it, and in the last round no drain is coming.
func (p *pgaTxOrderer) OnNonceGapResolved(queueItem txQueueItem) {
	if p.mempool.Push(queueItem) {
		recordPromotedTx(queueItem)
	}
}
