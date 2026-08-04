// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package gethexec

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/execution/gethexec/pga"
)

// pgaTxOrderer is the priority-gas-auction TxOrderer.
type pgaTxOrderer struct {
	seq           txOrdererSequencer
	configFetcher SequencerConfigFetcher
	ctx           context.Context // bounds the round-boundary waits

	mempool  *pga.Mempool[txQueueItem]
	schedule *pga.Schedule

	baseFee *big.Int
}

var _ txOrderer = (*pgaTxOrderer)(nil)

func NewPGATxOrderer(ctx context.Context, seq txOrdererSequencer, configFetcher SequencerConfigFetcher, baseFee *big.Int) *pgaTxOrderer {
	return &pgaTxOrderer{
		seq:           seq,
		configFetcher: configFetcher,
		ctx:           ctx,
		baseFee:       baseFee,
	}
}

func (p *pgaTxOrderer) NextQueueItem() (txQueueItem, bool) {
	for {
		if p.mempool.PriorityQueueLen() == 0 || p.schedule.RoundIsOver() {
			p.mempool.ApplyRoundBoost()
			if p.schedule.IsLastRound() {
				return txQueueItem{}, false
			}
			err := p.schedule.WaitAndAdvanceRound(p.ctx)
			if err != nil {
				log.Warn("PGA round wait interrupted; ending the block early", "err", err)
				return txQueueItem{}, false
			}
			p.mempool.PushBatch(p.seq.drainValidatedTxs())
		}

		entry, ok := p.mempool.Pop()
		if !ok {
			continue
		}
		item := entry.Tx()
		item.pgaPriority = entry.Priority()
		item.pgaBoost = entry.Boost()
		return item, true
	}
}

func (p *pgaTxOrderer) StartBlock() (hasWork bool) {
	config := p.configFetcher()
	p.schedule = pga.NewSchedule(config.ExperimentalPGA.RoundsPerBlock, config.PGARoundLength())
	p.mempool = pga.NewMempool[txQueueItem](config.ExperimentalPGA.RoundsPerBlock, p.baseFee)

	p.mempool.PushBatch(p.seq.drainValidatedTxs())

	return p.mempool.PriorityQueueLen() > 0
}

func (p *pgaTxOrderer) TakeRemaining() []txQueueItem {
	// The deferred block cleanup can run before StartBlock arms the mempool.
	if p.mempool == nil {
		return nil
	}
	items := make([]txQueueItem, 0, p.mempool.PriorityQueueLen())
	for {
		entry, ok := p.mempool.Pop()
		if !ok {
			return items
		}
		item := entry.Tx()
		item.pgaBoost = entry.Boost()
		items = append(items, item)
	}
}

func (p *pgaTxOrderer) OnTxInclusion(queueItem txQueueItem) {
	p.mempool.RecordIncludedTx(queueItem.pgaPriority)
}

// OnNonceGapResolved pushes the revived tx into the current block's auction: its nonce is
// valid against the in-progress state, and the round-boundary drain would wrongly re-park it
// by prechecking against the last committed state.
func (p *pgaTxOrderer) OnNonceGapResolved(queueItem txQueueItem) {
	p.mempool.Push(queueItem)
}
