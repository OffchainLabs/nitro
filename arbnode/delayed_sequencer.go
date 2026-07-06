// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbnode

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbnode/mel"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/execution"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

type DelayedMessageFetcher interface {
	GetDelayedCount() (uint64, error)
	FinalizedDelayedMessageAtPosition(
		ctx context.Context, finalizedBlock uint64, lastDelayedAccumulator common.Hash, requestedPosition uint64,
	) (*arbostypes.L1IncomingMessage, common.Hash, uint64, error)
}

type DelayedSequencer struct {
	stopwaiter.StopWaiter
	l1Reader                 *headerreader.HeaderReader
	bridge                   *DelayedBridge
	delayedMessageFetcher    DelayedMessageFetcher
	exec                     execution.ExecutionSequencer
	coordinator              *SeqCoordinator
	waitingForFinalizedBlock atomic.Pointer[uint64] // short-circuit: skip work until finalized parent chain block advances past this value
	config                   DelayedSequencerConfigFetcher
	mutex                    sync.Mutex
}

type DelayedSequencerConfig struct {
	Enable              bool          `koanf:"enable" reload:"hot"`
	FinalizeDistance    int64         `koanf:"finalize-distance" reload:"hot"`
	RequireFullFinality bool          `koanf:"require-full-finality" reload:"hot"`
	UseMergeFinality    bool          `koanf:"use-merge-finality" reload:"hot"`
	RescanInterval      time.Duration `koanf:"rescan-interval" reload:"hot"`
}

type DelayedSequencerConfigFetcher func() *DelayedSequencerConfig

func DelayedSequencerConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Bool(prefix+".enable", DefaultDelayedSequencerConfig.Enable, "enable delayed sequencer")
	f.Int64(prefix+".finalize-distance", DefaultDelayedSequencerConfig.FinalizeDistance, "how many blocks in the past L1 block is considered final (ignored when using Merge finality)")
	f.Bool(prefix+".require-full-finality", DefaultDelayedSequencerConfig.RequireFullFinality, "whether to wait for full finality before sequencing delayed messages")
	f.Bool(prefix+".use-merge-finality", DefaultDelayedSequencerConfig.UseMergeFinality, "whether to use The Merge's notion of finality before sequencing delayed messages")
	f.Duration(prefix+".rescan-interval", DefaultDelayedSequencerConfig.RescanInterval, "frequency to rescan for new delayed messages (the parent chain reader's poll-interval config is more important than this)")
}

var DefaultDelayedSequencerConfig = DelayedSequencerConfig{
	Enable:              false,
	FinalizeDistance:    20,
	RequireFullFinality: false,
	UseMergeFinality:    true,
	RescanInterval:      time.Second,
}

var TestDelayedSequencerConfig = DelayedSequencerConfig{
	Enable:              true,
	FinalizeDistance:    20,
	RequireFullFinality: false,
	UseMergeFinality:    false,
	RescanInterval:      time.Millisecond * 100,
}

func NewDelayedSequencer(l1Reader *headerreader.HeaderReader, delayedMessageFetcher DelayedMessageFetcher, delayedBridge *DelayedBridge, exec execution.ExecutionSequencer, coordinator *SeqCoordinator, config DelayedSequencerConfigFetcher) (*DelayedSequencer, error) {
	d := &DelayedSequencer{
		l1Reader:              l1Reader,
		bridge:                delayedBridge,
		delayedMessageFetcher: delayedMessageFetcher,
		coordinator:           coordinator,
		exec:                  exec,
		config:                config,
	}
	if coordinator != nil {
		coordinator.SetDelayedSequencer(d)
	}
	return d, nil
}

func (d *DelayedSequencer) getDelayedMessagesRead() (uint64, error) {
	return d.exec.NextDelayedMessageNumber()
}

func (d *DelayedSequencer) tryToEnqueue(ctx context.Context, lastBlockHeader *types.Header) error {
	if d.coordinator != nil && !d.coordinator.CurrentlyChosen() {
		return nil
	}

	return d.enqueueWithoutLockout(ctx, lastBlockHeader)
}

func (d *DelayedSequencer) enqueueWithoutLockout(ctx context.Context, lastBlockHeader *types.Header) error {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	config := d.config()
	if !config.Enable {
		return nil
	}

	var finalized uint64
	var finalizedHash common.Hash
	if config.UseMergeFinality && headerreader.HeaderIndicatesFinalitySupport(lastBlockHeader) {
		var header *types.Header
		var err error
		if config.RequireFullFinality {
			header, err = d.l1Reader.LatestFinalizedBlockHeader(ctx)
		} else {
			header, err = d.l1Reader.LatestSafeBlockHeader(ctx)
		}
		if err != nil {
			return err
		}
		finalized = header.Number.Uint64()
		finalizedHash = header.Hash()
	} else {
		currentNum := lastBlockHeader.Number.Int64()
		if currentNum < config.FinalizeDistance {
			return nil
		}
		// #nosec G115
		finalized = uint64(currentNum - config.FinalizeDistance)
	}

	if w := d.waitingForFinalizedBlock.Load(); w != nil && *w > finalized {
		return nil
	}

	// Reset what block we're waiting for if we've caught up
	d.waitingForFinalizedBlock.Store(nil)

	dbDelayedCount, err := d.delayedMessageFetcher.GetDelayedCount()
	if err != nil {
		return err
	}

	startPos, err := d.getDelayedMessagesRead()
	if err != nil {
		return err
	}

	// Retrieve all finalized delayed messages
	pos := startPos
	var lastDelayedAcc common.Hash
	var messages []*arbostypes.L1IncomingMessage
	for pos < dbDelayedCount {
		msg, acc, parentChainBlockNumber, err := d.delayedMessageFetcher.FinalizedDelayedMessageAtPosition(ctx, finalized, lastDelayedAcc, pos)
		if errors.Is(err, mel.ErrDelayedMessageNotYetFinalized) {
			d.waitingForFinalizedBlock.Store(&parentChainBlockNumber)
			break
		} else if err != nil {
			return err
		}
		lastDelayedAcc = acc
		messages = append(messages, msg)
		pos++
	}

	// Sequence the delayed messages, if any
	if len(messages) > 0 {
		if err := d.checkAccumulatorReorg(
			ctx, lastDelayedAcc, pos, finalizedHash, finalized,
		); err != nil {
			return err
		}
		d.exec.EnqueueDelayedMessages(messages, startPos)
		log.Info("Delayed messages enqueued", "msgnum", len(messages), "startpos", startPos)
	}

	return nil
}

// Dangerous: bypasses lockout check!
func (d *DelayedSequencer) ForceEnqueue(ctx context.Context) error {
	lastBlockHeader, err := d.l1Reader.LastHeader(ctx)
	if err != nil {
		return err
	}
	return d.enqueueWithoutLockout(ctx, lastBlockHeader)
}

func (d *DelayedSequencer) run(ctx context.Context) {
	headerChan, cancel := d.l1Reader.Subscribe(false)
	defer cancel()

	latestHeader, err := d.l1Reader.LastHeader(ctx)
	if err != nil {
		log.Warn("delayed sequencer: failed to get latest header", "err", err)
		latestHeader = nil
	}
	config := d.config()
	rescanTimer := time.NewTimer(config.RescanInterval)
	for {
		if !rescanTimer.Stop() {
			select {
			case <-rescanTimer.C:
			default:
			}
		}
		if latestHeader != nil {
			rescanTimer.Reset(d.config().RescanInterval)
		}
		var ok bool
		select {
		case latestHeader, ok = <-headerChan:
			if !ok {
				log.Debug("delayed sequencer: header channel close")
				return
			}
		case <-rescanTimer.C:
			if latestHeader == nil {
				continue
			}
		case <-ctx.Done():
			log.Debug("delayed sequencer: context done", "err", ctx.Err())
			return
		}
		if err := d.tryToEnqueue(ctx, latestHeader); err != nil {
			if errors.Is(err, execution.ExecutionEngineBlockCreationStopped) {
				log.Info("stopping block creation in delayed sequencer because execution engine has stopped")
				return
			}
			log.Error("Delayed sequencer error", "err", err)
		}
	}
}

func (d *DelayedSequencer) Start(ctxIn context.Context) {
	d.StopWaiter.Start(ctxIn, d)
	d.LaunchThread(d.run)
}

func (d *DelayedSequencer) WaitingForFinalizedBlock(t *testing.T) (uint64, bool) {
	if w := d.waitingForFinalizedBlock.Load(); w != nil {
		return *w, true
	}
	return 0, false
}

func (d *DelayedSequencer) checkAccumulatorReorg(
	ctx context.Context,
	lastDelayedAcc common.Hash,
	pos uint64,
	finalizedHash common.Hash,
	finalized uint64,
) error {
	delayedBridgeAcc, err := d.bridge.GetAccumulator(ctx, pos-1, new(big.Int).SetUint64(finalized), finalizedHash)
	if err != nil {
		return err
	}
	if delayedBridgeAcc != lastDelayedAcc {
		// Probably a reorg that hasn't been picked up by the inbox reader
		return fmt.Errorf("inbox reader at delayed message %v db accumulator %v doesn't match delayed bridge accumulator %v at L1 block %v", pos-1, lastDelayedAcc, delayedBridgeAcc, finalized)
	}
	return nil
}
