package arbnode

import (
	"context"
	"errors"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/arbstate/arbos"
)

type DelayedSequencer struct {
	client         L1Interface
	bridge         *DelayedBridge
	inboxState     *InboxState
	nextToSequence uint64
	scannedBlockNr *big.Int
	config         *DelayedSequencerConfig
}

type DelayedSequencerConfig struct {
	FinalizeDistance *big.Int      // how many blocks in the past L1 block is considered final
	BlocksAggregate  *big.Int      // how many blocks we aggregate looking for delayedMessage
	TimeAggregate    time.Duration // how many blocks we aggregate looking for delayedMessages
}

var DefaultDelayedSequencerConfig = &DelayedSequencerConfig{
	FinalizeDistance: big.NewInt(12),
	BlocksAggregate:  big.NewInt(5),
	TimeAggregate:    time.Minute,
}

func NewDelayedSequencer(client L1Interface, reader *InboxReader, inboxState *InboxState, config *DelayedSequencerConfig) (*DelayedSequencer, error) {
	pos, err := inboxState.GetMessageCount()
	if err != nil {
		return nil, err
	}
	var delayedMessagesRead uint64
	var delayedBlockNrRead *big.Int
	if pos > 0 {
		lastMsg, err := inboxState.GetMessage(pos - 1)
		if err != nil {
			return nil, err
		}
		delayedMessagesRead = lastMsg.DelayedMessagesRead
		if delayedMessagesRead > 0 {
			msg, err := reader.Database().GetDelayedMessage(delayedMessagesRead - 1)
			if err != nil {
				return nil, err
			}
			delayedBlockNrRead = msg.Header.BlockNumber.Big()
		}
	}
	if delayedBlockNrRead == nil {
		delayedBlockNrRead = reader.DelayedBridge().FirstBlock()
	}
	sequencer := DelayedSequencer{
		client:         client,
		bridge:         reader.DelayedBridge(),
		inboxState:     inboxState,
		nextToSequence: delayedMessagesRead,
		scannedBlockNr: delayedBlockNrRead,
		config:         config,
	}
	return &sequencer, nil
}

func (d *DelayedSequencer) finalizedBlockNr(ctx context.Context) (*big.Int, error) {
	lastBlockHeader, err := d.client.HeaderByNumber(ctx, nil)
	if err != nil {
		return nil, err
	}
	finalized := new(big.Int).Sub(lastBlockHeader.Number, d.config.FinalizeDistance)
	if finalized.Sign() > 0 {
		return finalized, nil
	}
	return big.NewInt(0), nil
}

func (d *DelayedSequencer) sendToSequencer(newMessages []*DelayedInboxMessage) error {
	l1msgs := make([]*arbos.L1IncomingMessage, len(newMessages))
	for i, delayedMsg := range newMessages {
		l1msgs[i] = delayedMsg.Message
	}
	err := d.inboxState.SequenceDelayedMessages(l1msgs, d.nextToSequence)
	if err != nil {
		return err
	}
	d.nextToSequence += uint64(len(newMessages))
	return nil
}

func (d *DelayedSequencer) update(ctx context.Context) error {
	blockNr, err := d.finalizedBlockNr(ctx)
	if err != nil {
		return err
	}
	if blockNr.Cmp(d.scannedBlockNr) <= 0 {
		// make sure there wasn't a deep reorg
		messageNrReorged, err := d.bridge.GetMessageCount(ctx, d.scannedBlockNr)
		if err != nil {
			return err
		}
		if messageNrReorged != d.nextToSequence {
			return errors.New("deep reorg detected")
		}
		return nil
	}
	messagesNow, err := d.bridge.GetMessageCount(ctx, blockNr)
	if err != nil {
		return err
	}
	if messagesNow < d.nextToSequence {
		return errors.New("deep reorg detected")
	}
	if messagesNow == d.nextToSequence {
		d.scannedBlockNr = blockNr
		return nil
	}
	newMessages, err := d.bridge.LookupMessagesInRange(ctx, new(big.Int).Add(d.scannedBlockNr, big.NewInt(1)), blockNr)
	if err != nil {
		return err
	}
	// these messages should be finalized, so we expect different querie to not contradt ech other
	if (d.nextToSequence + uint64(len(newMessages))) != messagesNow {
		return errors.New("fetching messages from delayedbridge error")
	}
	err = d.sendToSequencer(newMessages)
	if err != nil {
		return err
	}
	d.scannedBlockNr = blockNr
	return nil
}

// only if pushed externaly - it's possible that only some of the messages posted in
// a single L1 block were sent to the sequencer inbox.
// handle it by sending a batch completing the delayed messages posted in the same block.
func (d *DelayedSequencer) consumeFullBlock(ctx context.Context) error {
	if d.nextToSequence == 0 {
		return nil
	}
	msgCountScannedBlock, err := d.bridge.GetMessageCount(ctx, d.scannedBlockNr)
	if err != nil {
		return err
	}
	if msgCountScannedBlock < d.nextToSequence {
		return errors.New("either reorg or scanned block not set well")
	}
	if msgCountScannedBlock == d.nextToSequence {
		return nil
	}
	blockMessages, err := d.bridge.LookupMessagesInRange(ctx, d.scannedBlockNr, d.scannedBlockNr)
	if err != nil {
		return err
	}
	indexOfLastScanned := int64(d.nextToSequence) + int64(len(blockMessages)) - int64(msgCountScannedBlock)
	if indexOfLastScanned < 1 {
		return errors.New("either reorg or scanned block not set well")
	}
	blockMessages = blockMessages[indexOfLastScanned+1:]
	err = d.sendToSequencer(blockMessages)
	if err != nil {
		return err
	}
	return nil
}

func (d *DelayedSequencer) run(ctx context.Context) error {
	err := d.consumeFullBlock(ctx)
	if err != nil {
		return err
	}
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
		nextBlockToCheck := new(big.Int).Add(d.scannedBlockNr, d.config.FinalizeDistance)
		nextBlockToCheck.Add(nextBlockToCheck, d.config.BlocksAggregate)
	AggregateWaitLoop:
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-timeout:
				break AggregateWaitLoop
			case newHeader := <-headerChan:
				if newHeader.Number.Cmp(nextBlockToCheck) >= 0 {
					break AggregateWaitLoop
				}
			}
		}
	}
}

func (d *DelayedSequencer) Start(ctx context.Context) {
	go (func() {
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
	})()
}
