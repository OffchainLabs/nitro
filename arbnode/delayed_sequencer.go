// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbnode

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/execution"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

// FilteredTxWaitState tracks when we're halted waiting for a filtered tx
// to be added to the on-chain filter
type FilteredTxWaitState struct {
	TxHash        common.Hash
	DelayedMsgIdx uint64
	FirstSeen     time.Time
	LastLogTime   time.Time
}

type DelayedSequencer struct {
	stopwaiter.StopWaiter
	l1Reader                 *headerreader.HeaderReader
	bridge                   *DelayedBridge
	inbox                    *InboxTracker
	reader                   *InboxReader
	exec                     execution.ExecutionSequencer
	coordinator              *SeqCoordinator
	waitingForFinalizedBlock *uint64
	waitingForFilteredTx     *FilteredTxWaitState
	mutex                    sync.Mutex
	config                   DelayedSequencerConfigFetcher
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

func NewDelayedSequencer(l1Reader *headerreader.HeaderReader, reader *InboxReader, exec execution.ExecutionSequencer, coordinator *SeqCoordinator, config DelayedSequencerConfigFetcher) (*DelayedSequencer, error) {
	d := &DelayedSequencer{
		l1Reader:    l1Reader,
		bridge:      reader.DelayedBridge(),
		inbox:       reader.Tracker(),
		reader:      reader,
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

	// Periodic logging if halted waiting for filtered tx to be added to on-chain filter
	if d.waitingForFilteredTx != nil {
		if time.Since(d.waitingForFilteredTx.LastLogTime) > 5*time.Minute {
			log.Error("DelayedSequencer halted on filtered tx - waiting for tx hash to be added to on-chain filter",
				"txHash", d.waitingForFilteredTx.TxHash,
				"delayedMsgIdx", d.waitingForFilteredTx.DelayedMsgIdx,
				"waitingSince", d.waitingForFilteredTx.FirstSeen)
			d.waitingForFilteredTx.LastLogTime = time.Now()
		}
		// Fall through to retry sequencing - will succeed if tx hash was added to filter
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

	if d.waitingForFinalizedBlock != nil && *d.waitingForFinalizedBlock > finalized {
		return nil
	}

	// Reset what block we're waiting for if we've caught up
	d.waitingForFinalizedBlock = nil

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
		msg, acc, parentChainBlockNumber, err := d.inbox.GetDelayedMessageAccumulatorAndParentChainBlockNumber(ctx, pos)
		if err != nil {
			return err
		}
		if parentChainBlockNumber > finalized {
			// Message isn't finalized yet; wait for it to be
			d.waitingForFinalizedBlock = &parentChainBlockNumber
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
		err = msg.FillInBatchGasFields(func(batchNum uint64) ([]byte, error) {
			data, _, err := d.reader.GetSequencerMessageBytesForParentBlock(ctx, batchNum, parentChainBlockNumber)
			return data, err
		})
		if err != nil {
			return err
		}
		messages = append(messages, msg)
		pos++
	}

	// Sequence the delayed messages, if any
	if len(messages) > 0 {
		delayedBridgeAcc, err := d.bridge.GetAccumulator(ctx, pos-1, new(big.Int).SetUint64(finalized), finalizedHash)
		if err != nil {
			return err
		}
		if delayedBridgeAcc != lastDelayedAcc {
			// Probably a reorg that hasn't been picked up by the inbox reader
			return fmt.Errorf("inbox reader at delayed message %v db accumulator %v doesn't match delayed bridge accumulator %v at L1 block %v", pos-1, lastDelayedAcc, delayedBridgeAcc, finalized)
		}
		for i, msg := range messages {
			// #nosec G115
			err = d.exec.SequenceDelayedMessage(msg, startPos+uint64(i))
			if err != nil {
				var filteredErr *gethexec.ErrFilteredDelayedMessage
				if errors.As(err, &filteredErr) {
					if d.waitingForFilteredTx == nil {
						// First time hitting this filtered tx - log and set waiting state
						log.Error("Delayed message filtered - HALTING delayed sequencer",
							"txHash", filteredErr.TxHash,
							"delayedMsgIdx", filteredErr.DelayedMsgIdx)
						d.waitingForFilteredTx = &FilteredTxWaitState{
							TxHash:        filteredErr.TxHash,
							DelayedMsgIdx: filteredErr.DelayedMsgIdx,
							FirstSeen:     time.Now(),
							LastLogTime:   time.Now(),
						}
					}
					// Return nil to halt without propagating error up - will retry on next RescanInterval
					return nil
				}
				return err
			}
			// Success - clear waiting state if we were waiting
			if d.waitingForFilteredTx != nil {
				log.Info("Filtered tx resolved - resuming delayed sequencer",
					"txHash", d.waitingForFilteredTx.TxHash,
					"delayedMsgIdx", d.waitingForFilteredTx.DelayedMsgIdx,
					"waitedFor", time.Since(d.waitingForFilteredTx.FirstSeen))
				d.waitingForFilteredTx = nil
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

	latestHeader, err := d.l1Reader.LastHeader(ctx)
	if err != nil {
		log.Warn("delayed sequencer: failed to get latest header", "err", err)
		latestHeader = nil
	}
	rescanTimer := time.NewTimer(d.config().RescanInterval)
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
		if err := d.trySequence(ctx, latestHeader); err != nil {
			if errors.Is(err, gethexec.ExecutionEngineBlockCreationStopped) {
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

// WaitingForFilteredTx returns the tx hash being waited on, or nil if not halted.
// This is primarily for testing the halt-and-wait behavior.
func (d *DelayedSequencer) WaitingForFilteredTx() *common.Hash {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	if d.waitingForFilteredTx == nil {
		return nil
	}
	hash := d.waitingForFilteredTx.TxHash
	return &hash
}
