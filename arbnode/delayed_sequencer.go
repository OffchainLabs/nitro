// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/nitro/blob/master/LICENSE

package arbnode

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	flag "github.com/spf13/pflag"

	"github.com/offchainlabs/nitro/arbnode/execution"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

type DelayedSequencer struct {
	stopwaiter.StopWaiter
	l1Reader                 *headerreader.HeaderReader
	bridge                   *DelayedBridge
	inbox                    *InboxTracker
	exec                     *execution.ExecutionEngine
	coordinator              *SeqCoordinator
	waitingForFinalizedBlock uint64
	mutex                    sync.Mutex
	config                   DelayedSequencerConfigFetcher
}

type DelayedSequencerConfig struct {
	Enable              bool  `koanf:"enable" reload:"hot"`
	FinalizeDistance    int64 `koanf:"finalize-distance" reload:"hot"`
	RequireFullFinality bool  `koanf:"require-full-finality" reload:"hot"`
	UseMergeFinality    bool  `koanf:"use-merge-finality" reload:"hot"`
}

type DelayedSequencerConfigFetcher func() *DelayedSequencerConfig

func DelayedSequencerConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".enable", DefaultSeqCoordinatorConfig.Enable, "enable sequence coordinator")
	f.Int64(prefix+".finalize-distance", DefaultDelayedSequencerConfig.FinalizeDistance, "how many blocks in the past L1 block is considered final (ignored when using Merge finality)")
	f.Bool(prefix+".require-full-finality", DefaultDelayedSequencerConfig.RequireFullFinality, "whether to wait for full finality before sequencing delayed messages")
	f.Bool(prefix+".use-merge-finality", DefaultDelayedSequencerConfig.UseMergeFinality, "whether to use The Merge's notion of finality before sequencing delayed messages")
}

var DefaultDelayedSequencerConfig = DelayedSequencerConfig{
	Enable:              false,
	FinalizeDistance:    20,
	RequireFullFinality: true,
	UseMergeFinality:    true,
}

var TestDelayedSequencerConfig = DelayedSequencerConfig{
	Enable:              true,
	FinalizeDistance:    20,
	RequireFullFinality: true,
	UseMergeFinality:    true,
}

func NewDelayedSequencer(l1Reader *headerreader.HeaderReader, reader *InboxReader, exec *execution.ExecutionEngine, coordinator *SeqCoordinator, config DelayedSequencerConfigFetcher) (*DelayedSequencer, error) {
	d := &DelayedSequencer{
		l1Reader:    l1Reader,
		bridge:      reader.DelayedBridge(),
		inbox:       reader.Tracker(),
		coordinator: coordinator,
		exec:        exec,
		config:      config,
	}
	if coordinator != nil {
		coordinator.SetDelayedSequencer(d)
	}
	return d, nil
}

func (d *DelayedSequencer) getDelayedMessagesRead() (uint64, error) {
	return d.exec.NextDelayedMessageNumber()
}

func (d *DelayedSequencer) trySequence(ctx context.Context, lastBlockHeader *types.Header) error {
	if d.coordinator != nil && !d.coordinator.CurrentlyChosen() {
		return nil
	}

	return d.sequenceWithoutLockout(ctx, lastBlockHeader)
}

func (d *DelayedSequencer) sequenceWithoutLockout(ctx context.Context, lastBlockHeader *types.Header) error {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	config := d.config()
	if !config.Enable {
		return nil
	}

	var finalized uint64
	if config.UseMergeFinality && lastBlockHeader.Difficulty.Sign() == 0 {
		var err error
		if config.RequireFullFinality {
			finalized, err = d.l1Reader.LatestFinalizedBlockNr(ctx)
		} else {
			finalized, err = d.l1Reader.LatestSafeBlockNr(ctx)
		}
		if err != nil {
			return err
		}
	} else {
		currentNum := lastBlockHeader.Number.Int64()
		if currentNum < config.FinalizeDistance {
			return nil
		}
		finalized = uint64(currentNum - config.FinalizeDistance)
	}

	if d.waitingForFinalizedBlock > finalized {
		return nil
	}

	// Unless we find an unfinalized message (which sets waitingForBlock),
	// we won't find a new finalized message until FinalizeDistance blocks in the future.
	d.waitingForFinalizedBlock = lastBlockHeader.Number.Uint64() + 1

	dbDelayedCount, err := d.inbox.GetDelayedCount()
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
		msg, acc, parentChainBlockNumber, err := d.inbox.GetDelayedMessageAccumulatorAndParentChainBlockNumber(pos)
		if err != nil {
			return err
		}
		if parentChainBlockNumber > finalized {
			// Message isn't finalized yet; stop here
			d.waitingForFinalizedBlock = parentChainBlockNumber
			break
		}
		if lastDelayedAcc != (common.Hash{}) {
			// Ensure that there hasn't been a reorg and this message follows the last
			fullMsg := DelayedInboxMessage{
				BeforeInboxAcc:         lastDelayedAcc,
				Message:                msg,
				ParentChainBlockNumber: parentChainBlockNumber,
			}
			if fullMsg.AfterInboxAcc() != acc {
				return errors.New("delayed message accumulator mismatch while sequencing")
			}
		}
		lastDelayedAcc = acc
		messages = append(messages, msg)
		pos++
	}

	// Sequence the delayed messages, if any
	if len(messages) > 0 {
		delayedBridgeAcc, err := d.bridge.GetAccumulator(ctx, pos-1, new(big.Int).SetUint64(finalized))
		if err != nil {
			return err
		}
		if delayedBridgeAcc != lastDelayedAcc {
			// Probably a reorg that hasn't been picked up by the inbox reader
			return fmt.Errorf("inbox reader at delayed message %v db accumulator %v doesn't match delayed bridge accumulator %v at L1 block %v", pos-1, lastDelayedAcc, delayedBridgeAcc, finalized)
		}
		for i, msg := range messages {
			err = d.exec.SequenceDelayedMessage(msg, startPos+uint64(i))
			if err != nil {
				return err
			}
		}
		log.Info("DelayedSequencer: Sequenced", "msgnum", len(messages), "startpos", startPos)
	}

	return nil
}

// Dangerous: bypasses lockout check!
func (d *DelayedSequencer) ForceSequenceDelayed(ctx context.Context) error {
	lastBlockHeader, err := d.l1Reader.LastHeader(ctx)
	if err != nil {
		return err
	}
	return d.sequenceWithoutLockout(ctx, lastBlockHeader)
}

func (d *DelayedSequencer) run(ctx context.Context) {
	headerChan, cancel := d.l1Reader.Subscribe(false)
	defer cancel()

	for {
		select {
		case nextHeader, ok := <-headerChan:
			if !ok {
				log.Info("delayed sequencer: header channel close")
				return
			}
			if err := d.trySequence(ctx, nextHeader); err != nil {
				log.Error("Delayed sequencer error", "err", err)
			}
		case <-ctx.Done():
			log.Info("delayed sequencer: context done", "err", ctx.Err())
			return
		}
	}
}

func (d *DelayedSequencer) Start(ctxIn context.Context) {
	d.StopWaiter.Start(ctxIn, d)
	d.LaunchThread(d.run)
}
