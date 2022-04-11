// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbnode

import (
	"context"
	"errors"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	flag "github.com/spf13/pflag"

	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/util"
	"github.com/offchainlabs/nitro/util/arbmath"
)

type DelayedSequencer struct {
	util.StopWaiter
	l1Reader        *L1Reader
	bridge          *DelayedBridge
	inbox           *InboxTracker
	txStreamer      *TransactionStreamer
	coordinator     *SeqCoordinator
	waitingForBlock *big.Int
	config          *DelayedSequencerConfig
}

type DelayedSequencerConfig struct {
	Enable           bool          `koanf:"enable"`
	FinalizeDistance int64         `koanf:"finalize-distance"`
	TimeAggregate    time.Duration `koanf:"time-aggregate"`
}

func DelayedSequencerConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".enable", DefaultSeqCoordinatorConfig.Enable, "enable sequence coordinator")
	f.Int64(prefix+".finalize-distance", DefaultDelayedSequencerConfig.FinalizeDistance, "how many blocks in the past L1 block is considered final")
	f.Duration(prefix+".time-aggregate", DefaultDelayedSequencerConfig.TimeAggregate, "polling interval for the delayed sequencer")
}

var DefaultDelayedSequencerConfig = DelayedSequencerConfig{
	Enable:           true,
	FinalizeDistance: 12,
	TimeAggregate:    time.Minute,
}

var TestDelayedSequencerConfig = DelayedSequencerConfig{
	Enable:           true,
	FinalizeDistance: 12,
	TimeAggregate:    time.Second,
}

func NewDelayedSequencer(l1Reader *L1Reader, reader *InboxReader, txStreamer *TransactionStreamer, coordinator *SeqCoordinator, config *DelayedSequencerConfig) (*DelayedSequencer, error) {
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
	if d.waitingForBlock != nil && lastBlockHeader.Number.Cmp(d.waitingForBlock) < 0 {
		return nil
	}

	// Unless we find an unfinalized message (which sets waitingForBlock),
	// we won't find a new finalized message until FinalizeDistance blocks in the future.
	d.waitingForBlock = new(big.Int).Add(lastBlockHeader.Number, big.NewInt(d.config.FinalizeDistance))
	finalized := new(big.Int).Sub(lastBlockHeader.Number, big.NewInt(d.config.FinalizeDistance))
	if finalized.Sign() < 0 {
		finalized.SetInt64(0)
	}

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
			d.waitingForBlock = new(big.Int).Add(blockNumber, big.NewInt(d.config.FinalizeDistance))
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
			return errors.New("inbox reader db accumulator doesn't match delayed bridge")
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
	d.StopWaiter.Start(ctxIn)
	d.LaunchThread(d.run)
}
