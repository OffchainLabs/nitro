// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbnode

import (
	"context"
	"sync"
	"time"

	"github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/execution"
	"github.com/offchainlabs/nitro/util/stopwaiter"
	"github.com/offchainlabs/nitro/validator"
)

type BlockRecordingsPruner struct {
	stopwaiter.StopWaiter
	recorder      execution.ExecutionRecorder
	validated     func() arbutil.MessageIndex
	config        BlockRecordingsPrunerConfigFetcher
	pruningLock   sync.Mutex
	lastPruneDone time.Time
}

type BlockRecordingsPrunerConfig struct {
	Enable           bool          `koanf:"enable"`
	MinPruneInterval time.Duration `koanf:"min-prune-interval" reload:"hot"`
}

type BlockRecordingsPrunerConfigFetcher func() *BlockRecordingsPrunerConfig

var DefaultBlockRecordingsPrunerConfig = BlockRecordingsPrunerConfig{
	Enable:           true,
	MinPruneInterval: time.Minute,
}

var TestBlockRecordingsPrunerConfig = BlockRecordingsPrunerConfig{
	Enable:           true,
	MinPruneInterval: 100 * time.Millisecond,
}

func BlockRecordingsPrunerConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Bool(prefix+".enable", DefaultBlockRecordingsPrunerConfig.Enable, "enable pruning of chain-tip block recordings below the latest confirmed message")
	f.Duration(prefix+".min-prune-interval", DefaultBlockRecordingsPrunerConfig.MinPruneInterval, "minimum time between runs of the block recordings pruner")
}

func NewBlockRecordingsPruner(recorder execution.ExecutionRecorder, validated func() arbutil.MessageIndex, config BlockRecordingsPrunerConfigFetcher) *BlockRecordingsPruner {
	return &BlockRecordingsPruner{
		recorder:  recorder,
		validated: validated,
		config:    config,
	}
}

func (p *BlockRecordingsPruner) Start(ctxIn context.Context) {
	p.StopWaiter.Start(ctxIn, p)
}

func (p *BlockRecordingsPruner) UpdateLatestConfirmed(count arbutil.MessageIndex, _ validator.GoGlobalState) {
	locked := p.pruningLock.TryLock()
	if !locked {
		return
	}

	if time.Since(p.lastPruneDone) < p.config().MinPruneInterval {
		p.pruningLock.Unlock()
		return
	}
	if p.validated != nil {
		count = min(count, p.validated())
	}
	err := p.LaunchThreadSafe(func(ctx context.Context) {
		defer p.pruningLock.Unlock()
		_, err := p.recorder.PruneBlockRecordings(count).Await(ctx)
		if err != nil && ctx.Err() == nil {
			log.Error("error while pruning block recordings", "err", err)
			return
		}
		p.lastPruneDone = time.Now()
	})
	if err != nil {
		log.Info("failed launching block recordings prune thread", "err", err)
		p.pruningLock.Unlock()
	}
}
