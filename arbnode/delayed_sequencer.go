package arbnode

import (
	"context"
	"errors"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/arbstate/arbos"
)

type DelayedSequencer struct {
	client          L1Interface
	bridge          *DelayedBridge
	inbox           *InboxTracker
	txStreamer      *TransactionStreamer
	waitingForBlock *big.Int
	config          *DelayedSequencerConfig
}

type DelayedSequencerConfig struct {
	FinalizeDistance *big.Int      // how many blocks in the past L1 block is considered final
	BlocksAggregate  *big.Int      // how many blocks we aggregate looking for delayedMessage
	TimeAggregate    time.Duration // how many blocks we aggregate looking for delayedMessages
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

func NewDelayedSequencer(client L1Interface, reader *InboxReader, txStreamer *TransactionStreamer, config *DelayedSequencerConfig) (*DelayedSequencer, error) {
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
		blockNumber := msg.Header.BlockNumber.Big()
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

		err = d.txStreamer.SequenceDelayedMessages(messages, startPos)
		if err != nil {
			return err
		}
		log.Info("DelayedSequencer: Sequenced", "msgnum", len(messages), "startpos", startPos)
	}

	return nil
}

func (d *DelayedSequencer) run(ctx context.Context) error {
	headerChan := make(chan *types.Header)
	headSubscribe, err := d.client.SubscribeNewHead(ctx, headerChan)
	if err != nil {
		return err
	}
	defer headSubscribe.Unsubscribe()
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
			case newHeader := <-headerChan:
				if d.waitingForBlock == nil || newHeader.Number.Cmp(d.waitingForBlock) >= 0 {
					break AggregateWaitLoop
				}
			}
		}
	}
}

func (d *DelayedSequencer) Start(parentCtx context.Context) *Stopper {
	stopper, ctx := NewStopper(parentCtx, "Delayed sequencer")
	go func() {
		defer stopper.Close()
		for {
			err := d.run(ctx)
			if err != nil && !errors.Is(err, context.Canceled) {
				log.Error("error reading inbox", "err", err)
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second):
			}
		}
	}()
	return stopper
}
