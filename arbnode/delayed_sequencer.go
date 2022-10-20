// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbnode

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	flag "github.com/spf13/pflag"

	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

type DelayedSequencer struct {
	stopwaiter.StopWaiter
	l1Reader                 *headerreader.HeaderReader
	bridge                   *DelayedBridge
	inbox                    *InboxTracker
	txStreamer               *TransactionStreamer
	coordinator              *SeqCoordinator
	waitingForFinalizedBlock *big.Int
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

func NewDelayedSequencer(l1Reader *headerreader.HeaderReader, reader *InboxReader, txStreamer *TransactionStreamer, coordinator *SeqCoordinator, config DelayedSequencerConfigFetcher) (*DelayedSequencer, error) {
	return &DelayedSequencer{
		l1Reader:    l1Reader,
		bridge:      reader.DelayedBridge(),
		inbox:       reader.Tracker(),
		coordinator: coordinator,
		txStreamer:  txStreamer,
		config:      config,
	}, nil
}

func (d *DelayedSequencer) getDelayedMessagesRead() (uint64, error) {
	pos, err := d.txStreamer.GetMessageCount()
	if err != nil || pos == 0 {
		return 0, err
	}
	lastMsg, err := d.txStreamer.GetMessage(pos - 1)
	if err != nil {
		return 0, err
	}
	return lastMsg.DelayedMessagesRead, nil
}

func (d *DelayedSequencer) update(ctx context.Context, lastBlockHeader *types.Header) error {
	if d.coordinator != nil && !d.coordinator.CurrentlyChosen() {
		return nil
	}

	config := d.config()
	if !config.Enable {
		return nil
	}

	finalized := arbmath.BigSub(lastBlockHeader.Number, big.NewInt(config.FinalizeDistance))
	if finalized.Sign() < 0 {
		finalized.SetInt64(0)
	}

	// Once the merge is live, we can directly query for the latest finalized block number
	if lastBlockHeader.Difficulty.Sign() == 0 && config.UseMergeFinality {
		var header *types.Header
		var err error
		if config.RequireFullFinality {
			header, err = d.l1Reader.LatestFinalizedHeader()
		} else {
			header, err = d.l1Reader.LatestSafeHeader()
		}
		if err != nil {
			return fmt.Errorf("Failed to get latest header: %w", err)
		}
		finalized = header.Number
	}

	if d.waitingForFinalizedBlock != nil && arbmath.BigLessThan(finalized, d.waitingForFinalizedBlock) {
		return nil
	}

	// Unless we find an unfinalized message (which sets waitingForBlock),
	// we won't find a new finalized message until FinalizeDistance blocks in the future.
	d.waitingForFinalizedBlock = arbmath.BigAddByUint(lastBlockHeader.Number, 1)

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
	var messages []*arbos.L1IncomingMessage
	for pos < dbDelayedCount {
		msg, acc, err := d.inbox.GetDelayedMessageAndAccumulator(pos)
		if err != nil {
			return err
		}
		blockNumber := arbmath.UintToBig(msg.Header.BlockNumber)
		if blockNumber.Cmp(finalized) > 0 {
			// Message isn't finalized yet; stop here
			d.waitingForFinalizedBlock = blockNumber
			break
		}
		if lastDelayedAcc != (common.Hash{}) {
			// Ensure that there hasn't been a reorg and this message follows the last
			fullMsg := DelayedInboxMessage{
				BeforeInboxAcc: lastDelayedAcc,
				Message:        msg,
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
		delayedBridgeAcc, err := d.bridge.GetAccumulator(ctx, pos-1, finalized)
		if err != nil {
			return err
		}
		if delayedBridgeAcc != lastDelayedAcc {
			// Probably a reorg that hasn't been picked up by the inbox reader
			return fmt.Errorf("inbox reader at delayed message %v db accumulator %v doesn't match delayed bridge accumulator %v at L1 block %v", pos-1, lastDelayedAcc, delayedBridgeAcc, finalized)
		}

		err = d.txStreamer.SequenceDelayedMessages(ctx, messages, startPos)
		if err != nil {
			return err
		}
		log.Info("DelayedSequencer: Sequenced", "msgnum", len(messages), "startpos", startPos)
	}

	return nil
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
			if err := d.update(ctx, nextHeader); err != nil {
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
