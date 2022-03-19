package arbnode

import (
	"context"
	"errors"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	flag "github.com/spf13/pflag"

	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/util"
	"github.com/offchainlabs/nitro/util/arbmath"
)

type DelayedSequencer struct {
	util.StopWaiter
	client          arbutil.L1Interface
	bridge          *DelayedBridge
	inbox           *InboxTracker
	txStreamer      *TransactionStreamer
	waitingForBlock *big.Int
	config          *DelayedSequencerConfig
}

type DelayedSequencerConfig struct {
	FinalizeDistance *big.Int      `koanf:"finalize-distance"`
	BlocksAggregate  *big.Int      `koanf:"blocks-aggregate"`
	TimeAggregate    time.Duration `koanf:"time-aggregate"`
}

func DelayedSequencerConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Int64(prefix+".finalize-distance", DefaultDelayedSequencerConfig.FinalizeDistance.Int64(), "how many blocks in the past L1 block is considered final")
	f.Int64(prefix+".blocks-aggregate", DefaultDelayedSequencerConfig.BlocksAggregate.Int64(), "how many blocks we aggregate looking for delayedMessage")
	f.Duration(prefix+".time-aggregate", DefaultDelayedSequencerConfig.TimeAggregate, "polling interval for the delayed sequencer")
}

var DefaultDelayedSequencerConfig = DelayedSequencerConfig{
	FinalizeDistance: big.NewInt(12),
	BlocksAggregate:  big.NewInt(5),
	TimeAggregate:    time.Minute,
}

var TestDelayedSequencerConfig = DelayedSequencerConfig{
	FinalizeDistance: big.NewInt(12),
	BlocksAggregate:  big.NewInt(5),
	TimeAggregate:    time.Second,
}

func NewDelayedSequencer(client arbutil.L1Interface, reader *InboxReader, txStreamer *TransactionStreamer, config *DelayedSequencerConfig) (*DelayedSequencer, error) {
	return &DelayedSequencer{
		client:     client,
		bridge:     reader.DelayedBridge(),
		inbox:      reader.Tracker(),
		txStreamer: txStreamer,
		config:     config,
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

func (d *DelayedSequencer) update(ctx context.Context) error {
	lastBlockHeader, err := d.client.HeaderByNumber(ctx, nil)
	if err != nil {
		return err
	}

	// Unless we find an unfinalized message (which sets waitingForBlock),
	// we won't find a new finalized message until FinalizeDistance blocks in the future.
	d.waitingForBlock = new(big.Int).Add(lastBlockHeader.Number, d.config.FinalizeDistance)
	finalized := new(big.Int).Sub(lastBlockHeader.Number, d.config.FinalizeDistance)
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
			d.waitingForBlock = new(big.Int).Add(blockNumber, d.config.FinalizeDistance)
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

func (d *DelayedSequencer) run(ctx context.Context) error {
	headerChan, cancel := arbutil.HeaderSubscribeWithRetry(ctx, d.client)
	defer cancel()

	for {
		err := d.update(ctx)
		if err != nil {
			return err
		}
		timeout := time.After(d.config.TimeAggregate)
	AggregateWaitLoop:
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-timeout:
				break AggregateWaitLoop
			case newHeader, ok := <-headerChan:
				if ctx.Err() != nil {
					return ctx.Err()
				}
				if !ok {
					return errors.New("header channel closed")
				}
				if d.waitingForBlock == nil || newHeader.Number.Cmp(d.waitingForBlock) >= 0 {
					break AggregateWaitLoop
				}
			}
		}
	}
}

func (d *DelayedSequencer) Start(ctxIn context.Context) {
	d.StopWaiter.Start(ctxIn)
	d.CallIteratively(func(ctx context.Context) time.Duration {
		err := d.run(ctx)
		if err != nil && !errors.Is(err, context.Canceled) {
			log.Error("error reading inbox", "err", err)
		}
		return time.Second
	})
}
