//go:build benchsequencer

package gethexec

import (
	"context"

	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/util/stopwaiter"
	"github.com/spf13/pflag"
)

func BenchSequencerConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Bool(prefix+".enable", BenchSequencerConfigDefault.Enable, "enables transaction indexer")
}

func (c *BenchSequencerConfig) Validate() error {
	if c.Enable {
		log.Warn("DANGER! BenchSequencer enabled")
	}
	return nil
}

func NewBenchSequencer(sequencer *Sequencer) (TransactionPublisher, interface{}) {
	benchSequencer := &BenchSequencer{
		Sequencer: sequencer,
		semaphore: make(chan struct{}, 1),
	}
	return benchSequencer, NewBenchSequencerAPI(benchSequencer)
}

type BenchSequencer struct {
	*Sequencer
	semaphore chan struct{}
}

func (s *BenchSequencer) Start(ctx context.Context) error {
	// override Sequencer.Start to not start the inner sequencer
	s.StopWaiter.Start(ctx, s)
	s.semaphore <- struct{}{}
	return nil
}

func (s *BenchSequencer) TxQueueLength(includeRetryTxQueue bool) int {
	if includeRetryTxQueue {
		return len(s.Sequencer.txQueue) + s.Sequencer.txRetryQueue.Len()
	}
	return len(s.Sequencer.txQueue)
}

func (s *BenchSequencer) TxRetryQueueLength() int {
	return s.Sequencer.txRetryQueue.Len()
}

func (s *BenchSequencer) CreateBlock() containers.PromiseInterface[bool] {
	return stopwaiter.LaunchPromiseThread[bool](s, func(ctx context.Context) (bool, error) {
		select {
		// createBlock can't be run in parallel
		case <-s.semaphore:
			defer func() {
				// release semaphore, also in case of panic
				s.semaphore <- struct{}{}
			}()
			return s.createBlock(ctx), nil
		case <-ctx.Done():
			return false, ctx.Err()
		}
	})
}

type BenchSequencerAPI struct {
	benchSequencer *BenchSequencer
}

func (a *BenchSequencerAPI) TxQueueLength(includeRetryTxQueue bool) int {
	return a.benchSequencer.TxQueueLength(includeRetryTxQueue)
}

func (a *BenchSequencerAPI) TxRetryQueueLength() int {
	return a.benchSequencer.TxRetryQueueLength()
}

func (a *BenchSequencerAPI) CreateBlock(ctx context.Context) (bool, error) {
	return a.benchSequencer.CreateBlock().Await(ctx)
}

func NewBenchSequencerAPI(benchSequencer *BenchSequencer) *BenchSequencerAPI {
	return &BenchSequencerAPI{benchSequencer: benchSequencer}
}
