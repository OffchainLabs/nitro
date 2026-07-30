package gethexec

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/execution/gethexec/pga"
)

// PGATxOrderer is the priority-gas-auction TxOrderer. See the PGA design doc.
type PGATxOrderer struct {
	seq           txOrdererSequencer
	configFetcher SequencerConfigFetcher
	ctx           context.Context // bounds the round-boundary waits

	mempool  *pga.Mempool[txQueueItem]
	schedule *pga.Schedule

	baseFee *big.Int

	// lastYieldedPriority is the priority of the tx most recently yielded by NextQueueItem;
	// OnTxInclusion reports it back to the mempool.
	lastYieldedPriority uint64
}

var _ txOrderer = (*PGATxOrderer)(nil)

func NewPGATxOrderer(ctx context.Context, seq txOrdererSequencer, configFetcher SequencerConfigFetcher, baseFee *big.Int) *PGATxOrderer {
	return &PGATxOrderer{
		seq:           seq,
		configFetcher: configFetcher,
		ctx:           ctx,
		baseFee:       baseFee,
	}
}

func (p *PGATxOrderer) NextQueueItem() (txQueueItem, bool) {
	if p.mempool.PriorityQueueLen() == 0 || p.schedule.RoundIsOver() {
		p.mempool.ApplyRoundBoost()
		if p.schedule.IsLastRound() {
			return txQueueItem{}, false
		}
		err := p.schedule.WaitAndAdvanceRound(p.ctx)
		if err != nil {
			log.Warn("PGA round wait interrupted; returning no txs for this block", "err", err)
			return txQueueItem{}, false
		}
		for _, item := range p.seq.drainValidatedTxs() {
			p.mempool.PushPrioritized(item, item.pgaBoost)
		}
	}

	item, ok := p.mempool.Pop()
	if !ok {
		return txQueueItem{}, false
	}

	tx := item.Tx()
	tx.pgaBoost = item.Boost()
	p.lastYieldedPriority = item.Priority()

	return tx, true
}

func (p *PGATxOrderer) StartBlock() (hasWork bool) {
	config := p.configFetcher()
	p.schedule = pga.NewSchedule(config.ExperimentalPGA.RoundsPerBlock, config.PGARoundLength())
	p.mempool = pga.NewMempool[txQueueItem](config.ExperimentalPGA.RoundsPerBlock, p.baseFee)

	for _, item := range p.seq.drainValidatedTxs() {
		p.mempool.PushPrioritized(item, item.pgaBoost)
	}

	return p.mempool.PriorityQueueLen() > 0
}

func (p *PGATxOrderer) TakeRemaining() []txQueueItem {
	if p.mempool == nil {
		return nil
	}

	list := make([]txQueueItem, 0, p.mempool.PriorityQueueLen())
	for {
		item, ok := p.mempool.Pop()
		if !ok {
			break
		}
		queueItem := item.Tx()
		queueItem.pgaBoost = item.Boost()
		list = append(list, queueItem)
	}
	return list
}

func (p *PGATxOrderer) OnTxInclusion() {
	p.mempool.RecordIncludedTx(p.lastYieldedPriority)
}
