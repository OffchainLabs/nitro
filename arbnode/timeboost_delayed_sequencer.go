// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbnode

import (
	"context"
	"errors"

	flag "github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/execution"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

type TimeboostDelayedSequencer struct {
	stopwaiter.StopWaiter
	inbox              *InboxTracker
	reader             *InboxReader
	exec               execution.ExecutionSequencer
	config             TimeboostDelayedSequencerConfigFetcher
	delayedMessageChan chan gethexec.DelayedMessageCommand
}

type TimeboostDelayedSequencerConfig struct {
	Enable bool `koanf:"enable" reload:"hot"`
}

type TimeboostDelayedSequencerConfigFetcher func() *TimeboostDelayedSequencerConfig

func TimeboostDelayedSequencerConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".enable", DefaultTimeboostDelayedSequencerConfig.Enable, "enable delayed sequencer")
}

var DefaultTimeboostDelayedSequencerConfig = TimeboostDelayedSequencerConfig{
	Enable: false,
}

var TestTimeboostDelayedSequencerConfig = TimeboostDelayedSequencerConfig{
	Enable: false,
}

func NewTimeboostDelayedSequencer(reader *InboxReader, exec execution.ExecutionSequencer, config TimeboostDelayedSequencerConfigFetcher) (*TimeboostDelayedSequencer, chan gethexec.DelayedMessageCommand, error) {
	delayedChannel := make(chan gethexec.DelayedMessageCommand, 1)
	d := &TimeboostDelayedSequencer{
		inbox:              reader.Tracker(),
		reader:             reader,
		exec:               exec,
		config:             config,
		delayedMessageChan: delayedChannel,
	}
	return d, delayedChannel, nil
}

func (d *TimeboostDelayedSequencer) getDelayedMessagesRead() (uint64, error) {
	return d.exec.NextDelayedMessageNumber()
}

func (d *TimeboostDelayedSequencer) sequence(ctx context.Context, delayedCount uint64) error {
	config := d.config()
	if !config.Enable {
		return nil
	}

	startPos, err := d.getDelayedMessagesRead()
	if err != nil {
		return err
	}

	// Retrieve all finalized delayed messages
	pos := startPos
	var messages []*arbostypes.L1IncomingMessage
	for pos < delayedCount {
		msg, _, _, err := d.inbox.GetDelayedMessageAccumulatorAndParentChainBlockNumber(ctx, pos)
		if err != nil {
			return err
		}
		err = msg.FillInBatchGasCost(func(batchNum uint64) ([]byte, error) {
			data, _, err := d.reader.GetSequencerMessageBytes(ctx, batchNum)
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
		for i, msg := range messages {
			// #nosec G115
			err = d.exec.SequenceDelayedMessage(msg, startPos+uint64(i))
			if err != nil {
				return err
			}
		}
		log.Info("DelayedSequencer: Sequenced", "msgnum", len(messages), "startpos", startPos)
	}

	return nil
}

func (d *TimeboostDelayedSequencer) run(ctx context.Context) {
	for {
		select {
		case command := <-d.delayedMessageChan:
			if err := d.sequence(ctx, command.DelayedMessagesRead); err != nil {
				if errors.Is(err, gethexec.ExecutionEngineBlockCreationStopped) {
					log.Info("stopping block creation in delayed sequencer because execution engine has stopped")
					return
				}
				log.Error("Delayed sequencer error", "err", err)
			}
		case <-ctx.Done():
			log.Debug("delayed sequencer: context done", "err", ctx.Err())
			return
		}
	}
}

func (d *TimeboostDelayedSequencer) Start(ctxIn context.Context) {
	d.StopWaiter.Start(ctxIn, d)
	d.LaunchThread(d.run)
}
