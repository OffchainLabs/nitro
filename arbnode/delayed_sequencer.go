// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbnode

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"

	"github.com/offchainlabs/nitro/arbnode/mel"
	"github.com/offchainlabs/nitro/arbnode/mel/runner"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/execution"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

var delayedSequencerFilteredTxWaitSeconds = metrics.NewRegisteredGauge(
	"arb/delayedsequencer/filtered_tx_wait_seconds", nil)

// FilteredTxWaitState tracks a halt while waiting for filtered transactions
// to be added to the onchain filter
type FilteredTxWaitState struct {
	TxHashes      []common.Hash
	DelayedMsgIdx uint64
	FirstSeen     time.Time
	LastLogTime   time.Time
	LastFullRetry time.Time
}

type DelayedSequencer struct {
	stopwaiter.StopWaiter
	l1Reader                 *headerreader.HeaderReader
	bridge                   *DelayedBridge
	inbox                    *InboxTracker
	reader                   *InboxReader
	msgExtractor             *melrunner.MessageExtractor
	exec                     execution.ExecutionSequencer
	coordinator              *SeqCoordinator
	waitingForFinalizedBlock *uint64
	waitingForFilteredTx     *FilteredTxWaitState
	mutex                    sync.Mutex
	config                   DelayedSequencerConfigFetcher
}

type DelayedSequencerConfig struct {
	Enable                      bool          `koanf:"enable" reload:"hot"`
	FinalizeDistance            int64         `koanf:"finalize-distance" reload:"hot"`
	RequireFullFinality         bool          `koanf:"require-full-finality" reload:"hot"`
	UseMergeFinality            bool          `koanf:"use-merge-finality" reload:"hot"`
	RescanInterval              time.Duration `koanf:"rescan-interval" reload:"hot"`
	FilteredTxFullRetryInterval time.Duration `koanf:"filtered-tx-full-retry-interval" reload:"hot"`
}

type DelayedSequencerConfigFetcher func() *DelayedSequencerConfig

func DelayedSequencerConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Bool(prefix+".enable", DefaultDelayedSequencerConfig.Enable, "enable delayed sequencer")
	f.Int64(prefix+".finalize-distance", DefaultDelayedSequencerConfig.FinalizeDistance, "how many blocks in the past L1 block is considered final (ignored when using Merge finality)")
	f.Bool(prefix+".require-full-finality", DefaultDelayedSequencerConfig.RequireFullFinality, "whether to wait for full finality before sequencing delayed messages")
	f.Bool(prefix+".use-merge-finality", DefaultDelayedSequencerConfig.UseMergeFinality, "whether to use The Merge's notion of finality before sequencing delayed messages")
	f.Duration(prefix+".rescan-interval", DefaultDelayedSequencerConfig.RescanInterval, "frequency to rescan for new delayed messages (the parent chain reader's poll-interval config is more important than this)")
	f.Duration(prefix+".filtered-tx-full-retry-interval", DefaultDelayedSequencerConfig.FilteredTxFullRetryInterval, "how often to do a full re-execution when halted on a filtered delayed message")
}

var DefaultDelayedSequencerConfig = DelayedSequencerConfig{
	Enable:                      false,
	FinalizeDistance:            20,
	RequireFullFinality:         false,
	UseMergeFinality:            true,
	RescanInterval:              time.Second,
	FilteredTxFullRetryInterval: 30 * time.Second,
}

var TestDelayedSequencerConfig = DelayedSequencerConfig{
	Enable:                      true,
	FinalizeDistance:            20,
	RequireFullFinality:         false,
	UseMergeFinality:            false,
	RescanInterval:              time.Millisecond * 100,
	FilteredTxFullRetryInterval: 1 * time.Second,
}

func NewDelayedSequencer(l1Reader *headerreader.HeaderReader, reader *InboxReader, msgExtractor *melrunner.MessageExtractor, delayedBridge *DelayedBridge, exec execution.ExecutionSequencer, coordinator *SeqCoordinator, config DelayedSequencerConfigFetcher) (*DelayedSequencer, error) {
	d := &DelayedSequencer{
		l1Reader:     l1Reader,
		bridge:       delayedBridge,
		msgExtractor: msgExtractor,
		reader:       reader,
		coordinator:  coordinator,
		exec:         exec,
		config:       config,
	}
	if reader != nil {
		d.bridge = reader.DelayedBridge()
		d.inbox = reader.Tracker()
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

	// Periodic logging if halted waiting for filtered tx to be added to onchain filter
	if d.waitingForFilteredTx != nil {
		now := time.Now()
		waitDuration := now.Sub(d.waitingForFilteredTx.FirstSeen)
		delayedSequencerFilteredTxWaitSeconds.Update(int64(waitDuration.Seconds()))
		if now.Sub(d.waitingForFilteredTx.LastLogTime) >= 5*time.Minute {
			logLevel := log.Warn
			if waitDuration > 1*time.Hour {
				logLevel = log.Error
			}
			logLevel("DelayedSequencer halted on filtered tx - waiting for tx hashes to be added to onchain filter",
				"txHashes", d.waitingForFilteredTx.TxHashes,
				"delayedMsgIdx", d.waitingForFilteredTx.DelayedMsgIdx,
				"waitingSince", d.waitingForFilteredTx.FirstSeen)
			d.waitingForFilteredTx.LastLogTime = now
		}

		// Periodically attempt full re-execution even if the tx hashes aren't in the
		// onchain filter yet. The filtered address set may have changed since the
		// last attempt, which could allow the tx to succeed without needing bypass.
		needsFullRetry := time.Since(d.waitingForFilteredTx.LastFullRetry) >= config.FilteredTxFullRetryInterval
		if !needsFullRetry {
			// Fast-path: check if all filtered tx hashes are now in the onchain filter
			allInFilter := true
			for _, txHash := range d.waitingForFilteredTx.TxHashes {
				isInFilter, err := d.exec.IsTxHashInOnchainFilter(txHash)
				if err != nil {
					log.Error("error checking onchain filter", "err", err, "txHash", txHash)
					allInFilter = false
					break
				}
				if !isInFilter {
					allInFilter = false
					break
				}
			}
			if !allInFilter {
				return nil
			}
		}
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

	var dbDelayedCount uint64
	var err error
	if d.msgExtractor != nil {
		dbDelayedCount, err = d.msgExtractor.GetDelayedCount(ctx, 0)
		if err != nil {
			return err
		}
	} else {
		dbDelayedCount, err = d.inbox.GetDelayedCount()
		if err != nil {
			return err
		}
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
		if d.msgExtractor != nil {
			finalizedPos, err := d.msgExtractor.GetDelayedCount(ctx, finalized)
			if err != nil {
				if !strings.Contains(err.Error(), "not found") {
					return err
				}
				return nil
			}
			if pos > finalizedPos {
				// Message isn't finalized yet; wait for it to be
				d.waitingForFinalizedBlock = &pos
				break
			}
			msg, err := d.msgExtractor.GetDelayedMessage(pos)
			if err != nil {
				return err
			}
			messages = append(messages, msg.Message)
		} else {
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
				fullMsg := mel.DelayedInboxMessage{
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
		}
		pos++
	}

	// Sequence the delayed messages, if any
	if len(messages) > 0 {
		if d.msgExtractor == nil {
			delayedBridgeAcc, err := d.bridge.GetAccumulator(ctx, pos-1, new(big.Int).SetUint64(finalized), finalizedHash)
			if err != nil {
				return err
			}
			if delayedBridgeAcc != lastDelayedAcc {
				// Probably a reorg that hasn't been picked up by the inbox reader
				return fmt.Errorf("inbox reader at delayed message %v db accumulator %v doesn't match delayed bridge accumulator %v at L1 block %v", pos-1, lastDelayedAcc, delayedBridgeAcc, finalized)
			}
		}
		for i, msg := range messages {
			// #nosec G115
			err = d.exec.SequenceDelayedMessage(msg, startPos+uint64(i))
			if err != nil {
				var filteredErr *gethexec.ErrFilteredDelayedMessage
				if errors.As(err, &filteredErr) {
					now := time.Now()
					if d.waitingForFilteredTx == nil {
						// First time hitting filtered tx(es) - log and set waiting state
						log.Error("Delayed message filtered - HALTING delayed sequencer",
							"txHashes", filteredErr.TxHashes,
							"delayedMsgIdx", filteredErr.DelayedMsgIdx)
						d.waitingForFilteredTx = &FilteredTxWaitState{
							TxHashes:      filteredErr.TxHashes,
							DelayedMsgIdx: filteredErr.DelayedMsgIdx,
							FirstSeen:     now,
							LastLogTime:   now,
							LastFullRetry: now,
						}
					} else {
						d.waitingForFilteredTx.TxHashes = filteredErr.TxHashes
						d.waitingForFilteredTx.LastFullRetry = now
					}
					// Return nil to halt without propagating error up - will retry on next interval
					return nil
				}
				return err
			}
			// Success - clear waiting state if we were waiting
			if d.waitingForFilteredTx != nil {
				log.Info("Filtered tx resolved - resuming delayed sequencer",
					"txHashes", d.waitingForFilteredTx.TxHashes,
					"delayedMsgIdx", d.waitingForFilteredTx.DelayedMsgIdx,
					"waitedFor", time.Since(d.waitingForFilteredTx.FirstSeen))
				d.waitingForFilteredTx = nil
				delayedSequencerFilteredTxWaitSeconds.Update(0)
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

// WaitingForFilteredTx returns the tx hashes being waited on, or nil and false if not halted.
// Takes a testing.T to prevent production code from calling this test-only function.
func (d *DelayedSequencer) WaitingForFilteredTx(t *testing.T) ([]common.Hash, bool) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	if d.waitingForFilteredTx == nil {
		return nil, false
	}
	return d.waitingForFilteredTx.TxHashes, true
}
