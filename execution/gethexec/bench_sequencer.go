// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

//go:build benchmarking-sequencer

package gethexec

import (
	"context"

	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/util/stopwaiter"
	"github.com/spf13/pflag"
)

func BenchmarkingSequencerConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Bool(prefix+".enable", BenchmarkingSequencerConfigDefault.Enable, "enable benchmarking sequencer RPC (manual block creation; requires benchmarking-sequencer build tag)")
}

func (c *BenchmarkingSequencerConfig) Validate() error {
	if c.Enable {
		log.Warn("DANGER! benchmarking sequencer enabled (manual block creation); do not use in production")
	}
	return nil
}

func NewBenchmarkingSequencer(sequencer *Sequencer) (TransactionPublisher, interface{}) {
	benchmarkingSequencer := &BenchmarkingSequencer{
		Sequencer: sequencer,
		semaphore: make(chan struct{}, 1),
	}
	return benchmarkingSequencer, NewBenchmarkingSequencerAPI(benchmarkingSequencer)
}

type BenchmarkingSequencer struct {
	*Sequencer
	semaphore chan struct{}
}

func (s *BenchmarkingSequencer) Start(ctx context.Context) error {
	// override Sequencer.Start to not start the inner sequencer
	s.StopWaiter.Start(ctx, s)
	s.semaphore <- struct{}{}
	return nil
}

func (s *BenchmarkingSequencer) TxQueueLength(includeRetryTxQueue bool) int {
	if includeRetryTxQueue {
		return len(s.Sequencer.txQueue) + s.Sequencer.txRetryQueue.Len()
	}
	return len(s.Sequencer.txQueue)
}

func (s *BenchmarkingSequencer) TxRetryQueueLength() int {
	return s.Sequencer.txRetryQueue.Len()
}

func (s *BenchmarkingSequencer) CreateBlock() containers.PromiseInterface[bool] {
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

type BenchmarkingSequencerAPI struct {
	benchmarkingSequencer *BenchmarkingSequencer
}

func (a *BenchmarkingSequencerAPI) TxQueueLength(includeRetryTxQueue bool) int {
	return a.benchmarkingSequencer.TxQueueLength(includeRetryTxQueue)
}

func (a *BenchmarkingSequencerAPI) TxRetryQueueLength() int {
	return a.benchmarkingSequencer.TxRetryQueueLength()
}

func (a *BenchmarkingSequencerAPI) CreateBlock(ctx context.Context) (bool, error) {
	return a.benchmarkingSequencer.CreateBlock().Await(ctx)
}

func NewBenchmarkingSequencerAPI(benchmarkingSequencer *BenchmarkingSequencer) *BenchmarkingSequencerAPI {
	return &BenchmarkingSequencerAPI{benchmarkingSequencer: benchmarkingSequencer}
}
