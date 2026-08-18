// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package gethexec

import (
	"context"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/execution/gethexec/pga"
)

// pgaTxOrderer is the priority-gas-auction TxOrderer.
type pgaTxOrderer struct {
	seq txOrdererSequencer
	ctx context.Context // bounds the round-boundary waits

	mempool  *pga.Mempool[txQueueItem]
	schedule *pga.Schedule

	baseFee        *big.Int
	roundsPerBlock uint
	roundLength    time.Duration
	stats          ordererStats
}

var _ txOrderer = (*pgaTxOrderer)(nil)

func NewPGATxOrderer(ctx context.Context, seq txOrdererSequencer, roundsPerBlock uint, roundLength time.Duration, baseFee *big.Int) *pgaTxOrderer {
	return &pgaTxOrderer{
		seq:            seq,
		ctx:            ctx,
		roundsPerBlock: roundsPerBlock,
		roundLength:    roundLength,
		baseFee:        baseFee,
	}
}

func (p *pgaTxOrderer) CurrentRound() uint64 {
	return p.schedule.Round()
}

func (p *pgaTxOrderer) NextQueueItem(statedb *state.StateDB, remainingBlockSize int, blockGasLeft uint64) (txQueueItem, bool) {
	if blockGasLeft < params.TxGas {
		p.stats.blockGasLimitReached = true
		p.mempool.ApplyRoundBoost()
		return txQueueItem{}, false
	}

	for {
		if p.mempool.PriorityQueueLen() == 0 || p.schedule.RoundIsOver() {
			p.mempool.ApplyRoundBoost()
			if p.schedule.IsLastRound() {
				p.stats.exhaustedQueue = p.mempool.PriorityQueueLen() == 0
				return txQueueItem{}, false
			}
			err := p.schedule.WaitAndAdvanceRound(p.ctx)
			if err != nil {
				log.Warn("PGA round wait interrupted; ending the block early", "err", err)
				return txQueueItem{}, false
			}
			p.mempool.PushBatch(p.seq.drainValidatedTxs(statedb, p.baseFee))
		}
		item, ok := p.mempool.Pop()
		if !ok {
			continue
		}

		// If the next tx is too big to fit in the remaining block space, we add it back to the mempool and stop sequencing.
		// The sequencer will finalize the block and start a new one, which will have a fresh mempool and schedule.
		if item.txSize > remainingBlockSize {
			p.stats.blockSizeLimitReached = true
			p.mempool.Push(item)
			p.mempool.ApplyRoundBoost()
			return txQueueItem{}, false
		}

		return item, true
	}
}

func (p *pgaTxOrderer) StartBlock(statedb *state.StateDB) (hasWork bool) {
	p.schedule = pga.NewSchedule(uint64(p.roundsPerBlock), p.roundLength)
	p.mempool = pga.NewMempool[txQueueItem](p.roundsPerBlock, p.baseFee)

	p.mempool.PushBatch(p.seq.drainValidatedTxs(statedb, p.baseFee))

	return p.mempool.PriorityQueueLen() > 0
}

func (p *pgaTxOrderer) TakeRemaining() []txQueueItem {
	// The deferred block cleanup can run before StartBlock arms the mempool.
	if p.mempool == nil {
		return nil
	}
	return p.mempool.TakeRemaining()
}

func (p *pgaTxOrderer) OrdererStats() ordererStats {
	return p.stats
}

func (p *pgaTxOrderer) OnTxInclusion(queueItem txQueueItem) {
	p.mempool.RecordIncludedTx(queueItem.GetPriority())
}

// BlockInterval spans the block's elapsed rounds; a no-work block never leaves round 1, so
// empty attempts retry on the round cadence.
func (p *pgaTxOrderer) BlockInterval() time.Duration {
	return p.schedule.ElapsedInterval()
}

// OnNonceGapResolved pushes the revived tx straight into the current round's auction: waiting
// for the next round-boundary drain would delay it, and in the last round no drain is coming.
func (p *pgaTxOrderer) OnNonceGapResolved(queueItem txQueueItem) {
	p.mempool.Push(queueItem)
}
