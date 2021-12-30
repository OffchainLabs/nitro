//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbnode

import (
	"bytes"
	"context"
	"sync"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/offchainlabs/arbstate/arbos"
	"github.com/offchainlabs/arbstate/arbstate"
	"github.com/offchainlabs/arbstate/validator"
	"github.com/pkg/errors"
)

type InboxTracker struct {
	db         ethdb.Database
	txStreamer *TransactionStreamer
	mutex      sync.Mutex
	validator  *validator.BlockValidator
}

func NewInboxTracker(raw ethdb.Database, txStreamer *TransactionStreamer) (*InboxTracker, error) {
	db := &InboxTracker{
		db:         rawdb.NewTable(raw, arbitrumPrefix),
		txStreamer: txStreamer,
	}
	return db, nil
}

func (t *InboxTracker) SetBlockValidator(validator *validator.BlockValidator) {
	t.validator = validator
}

func (t *InboxTracker) Initialize() error {
	batch := t.db.NewBatch()

	hasKey, err := t.db.Has(delayedMessageCountKey)
	if err != nil {
		return err
	}
	if !hasKey {
		value, err := rlp.EncodeToBytes(uint64(0))
		if err != nil {
			return err
		}
		err = batch.Put(delayedMessageCountKey, value)
		if err != nil {
			return err
		}
	}

	hasKey, err = t.db.Has(sequencerBatchCountKey)
	if err != nil {
		return err
	}
	if !hasKey {
		value, err := rlp.EncodeToBytes(uint64(0))
		if err != nil {
			return err
		}
		err = batch.Put(sequencerBatchCountKey, value)
		if err != nil {
			return err
		}
		log.Info("InboxTracker", "SequencerBatchCount", 0)
	}

	return batch.Write()
}

var accumulatorNotFound error = errors.New("accumulator not found")

func (t *InboxTracker) GetDelayedAcc(seqNum uint64) (common.Hash, error) {
	key := dbKey(delayedMessagePrefix, seqNum)
	hasKey, err := t.db.Has(key)
	if err != nil {
		return common.Hash{}, err
	}
	if !hasKey {
		return common.Hash{}, accumulatorNotFound
	}
	data, err := t.db.Get(key)
	if err != nil {
		return common.Hash{}, err
	}
	if len(data) < 32 {
		return common.Hash{}, errors.New("delayed message entry missing accumulator")
	}
	var hash common.Hash
	copy(hash[:], data[:32])
	return hash, nil
}

func (t *InboxTracker) GetDelayedCount() (uint64, error) {
	data, err := t.db.Get(delayedMessageCountKey)
	if err != nil {
		return 0, err
	}
	var count uint64
	err = rlp.DecodeBytes(data, &count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

type BatchMetadata struct {
	Accumulator         common.Hash
	MessageCount        uint64
	DelayedMessageCount uint64
}

func (t *InboxTracker) GetBatchMetadata(seqNum uint64) (accumulator common.Hash, messageCount uint64, delayedMessageCount uint64, err error) {
	key := dbKey(sequencerBatchMetaPrefix, seqNum)
	var hasKey bool
	hasKey, err = t.db.Has(key)
	if err != nil {
		return
	}
	if !hasKey {
		err = accumulatorNotFound
		return
	}
	data, err := t.db.Get(key)
	if err != nil {
		return
	}
	var metadata BatchMetadata
	err = rlp.DecodeBytes(data, &metadata)
	accumulator = metadata.Accumulator
	delayedMessageCount = metadata.DelayedMessageCount
	messageCount = metadata.MessageCount
	return
}

func (t *InboxTracker) GetBatchMessageCount(seqNum uint64) (uint64, error) {
	_, msgCount, _, err := t.GetBatchMetadata(seqNum)
	return msgCount, err
}

// Convenience function wrapping GetBatchMetadata
func (t *InboxTracker) GetBatchAcc(seqNum uint64) (common.Hash, error) {
	accumulator, _, _, err := t.GetBatchMetadata(seqNum)
	return accumulator, err
}

func (t *InboxTracker) GetBatchCount() (uint64, error) {
	data, err := t.db.Get(sequencerBatchCountKey)
	if err != nil {
		return 0, err
	}
	var count uint64
	err = rlp.DecodeBytes(data, &count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (t *InboxTracker) getDelayedMessageBytesAndAccumulator(seqNum uint64) ([]byte, common.Hash, error) {
	key := dbKey(delayedMessagePrefix, seqNum)
	data, err := t.db.Get(key)
	if err != nil {
		return nil, common.Hash{}, err
	}
	if len(data) < 32 {
		return nil, common.Hash{}, errors.New("delayed message entry missing accumulator")
	}
	var acc common.Hash
	copy(acc[:], data[:32])
	return data[32:], acc, err
}

func (t *InboxTracker) GetDelayedMessageAndAccumulator(seqNum uint64) (*arbos.L1IncomingMessage, common.Hash, error) {
	data, acc, err := t.getDelayedMessageBytesAndAccumulator(seqNum)
	if err != nil {
		return nil, acc, err
	}
	message, err := arbos.ParseIncomingL1Message(bytes.NewReader(data))
	return message, acc, err
}

func (t *InboxTracker) GetDelayedMessage(seqNum uint64) (*arbos.L1IncomingMessage, error) {
	msg, _, err := t.GetDelayedMessageAndAccumulator(seqNum)
	return msg, err
}

func (t *InboxTracker) GetDelayedMessageBytes(seqNum uint64) ([]byte, error) {
	data, _, err := t.getDelayedMessageBytesAndAccumulator(seqNum)
	return data, err
}

func (t *InboxTracker) AddDelayedMessages(messages []*DelayedInboxMessage) error {
	if len(messages) == 0 {
		return nil
	}
	t.mutex.Lock()
	defer t.mutex.Unlock()

	pos, err := messages[0].Message.Header.SeqNum()
	if err != nil {
		return err
	}
	var nextAcc common.Hash
	if pos > 0 {
		var err error
		nextAcc, err = t.GetDelayedAcc(pos - 1)
		if err != nil {
			if errors.Is(err, accumulatorNotFound) {
				return errors.New("missing previous delayed message")
			} else {
				return err
			}
		}
	}

	batch := t.db.NewBatch()
	for _, message := range messages {
		seqNum, err := message.Message.Header.SeqNum()
		if err != nil {
			return err
		}

		if seqNum != pos {
			return errors.New("unexpected delayed sequence number")
		}

		if nextAcc != message.BeforeInboxAcc {
			return errors.New("previous delayed accumulator mismatch")
		}
		nextAcc = message.AfterInboxAcc()

		msgKey := dbKey(delayedMessagePrefix, seqNum)

		msgData, err := message.Message.Serialize()
		if err != nil {
			return err
		}
		data := nextAcc.Bytes()
		data = append(data, msgData...)
		err = batch.Put(msgKey, data)
		if err != nil {
			return err
		}

		pos++
	}

	return t.setDelayedCountReorgAndWriteBatch(batch, pos)
}

// All-in-one delayed message count adjuster. Can go forwards or backwards.
// Requires the mutex is held. Sets the delayed count and performs any sequencer batch reorg necessary.
// Also deletes any future delayed messages.
func (t *InboxTracker) setDelayedCountReorgAndWriteBatch(batch ethdb.Batch, newDelayedCount uint64) error {
	err := deleteStartingAt(t.db, batch, delayedMessagePrefix, uint64ToBytes(newDelayedCount))
	if err != nil {
		return err
	}

	countData, err := rlp.EncodeToBytes(newDelayedCount)
	if err != nil {
		return err
	}
	err = batch.Put(delayedMessageCountKey, countData)
	if err != nil {
		return err
	}

	seqBatchIter := t.db.NewIterator(delayedSequencedPrefix, uint64ToBytes(newDelayedCount))
	defer seqBatchIter.Release()
	var reorgSeqBatchesToCount *uint64
	for {
		err = seqBatchIter.Error()
		if err != nil {
			return err
		}
		if len(seqBatchIter.Key()) == 0 {
			break
		}
		var batchSeqNum uint64
		err := rlp.DecodeBytes(seqBatchIter.Value(), &batchSeqNum)
		if err != nil {
			return err
		}
		err = batch.Delete(seqBatchIter.Key())
		if err != nil {
			return err
		}
		if reorgSeqBatchesToCount == nil {
			// Set the count to the first deleted sequence number.
			// E.g. if the deleted sequence number is 1, set the count to 1,
			// meaning that the last and only batch is at sequence number 0.
			reorgSeqBatchesToCount = &batchSeqNum
		}
	}
	// Release the iterator early.
	// It's fine to call Release multiple times,
	// which we'll do because of the defer.
	seqBatchIter.Release()
	if reorgSeqBatchesToCount != nil {
		count := *reorgSeqBatchesToCount
		err = batch.Put(sequencerBatchCountKey, uint64ToBytes(count))
		if err != nil {
			return err
		}
		log.Info("InboxTracker", "SequencerBatchCount", count)
		err = deleteStartingAt(t.db, batch, sequencerBatchMetaPrefix, uint64ToBytes(count))
		if err != nil {
			return err
		}
		var prevMesssageCount uint64
		if count > 0 {
			_, prevMesssageCount, _, err = t.GetBatchMetadata(count - 1)
			if err != nil {
				return err
			}
		}
		// Writes batch
		return t.txStreamer.ReorgToAndEndBatch(batch, prevMesssageCount)
	} else {
		return batch.Write()
	}
}

type multiplexerBackend struct {
	batchSeqNum           uint64
	batches               []*SequencerInboxBatch
	positionWithinMessage uint64

	ctx    context.Context
	client ethereum.ChainReader
	inbox  *InboxTracker
}

func (b *multiplexerBackend) PeekSequencerInbox() ([]byte, error) {
	if len(b.batches) == 0 {
		return nil, errors.New("read past end of specified sequencer batches")
	}
	return b.batches[0].Serialize(b.ctx, b.client)
}

func (b *multiplexerBackend) GetSequencerInboxPosition() uint64 {
	return b.batchSeqNum
}

func (b *multiplexerBackend) AdvanceSequencerInbox() {
	b.batchSeqNum++
	if len(b.batches) > 0 {
		b.batches = b.batches[1:]
	}
}

func (b *multiplexerBackend) GetPositionWithinMessage() uint64 {
	return b.positionWithinMessage
}

func (b *multiplexerBackend) SetPositionWithinMessage(pos uint64) {
	b.positionWithinMessage = pos
}

func (b *multiplexerBackend) ReadDelayedInbox(seqNum uint64) ([]byte, error) {
	if len(b.batches) == 0 || seqNum >= b.batches[0].AfterDelayedCount {
		return nil, errors.New("attempted to read past end of sequencer batch delayed messages")
	}
	data, _, err := b.inbox.getDelayedMessageBytesAndAccumulator(seqNum)
	return data, err
}

var delayedMessagesMismatch = errors.New("sequencer batch delayed messages missing or different")

func (t *InboxTracker) AddSequencerBatches(ctx context.Context, client ethereum.ChainReader, batches []*SequencerInboxBatch) error {
	if len(batches) == 0 {
		return nil
	}
	t.mutex.Lock()
	defer t.mutex.Unlock()

	pos := batches[0].SequenceNumber
	var nextAcc common.Hash
	var prevDelayedMessages uint64
	var startMessagePos uint64
	if pos > 0 {
		var err error
		nextAcc, startMessagePos, prevDelayedMessages, err = t.GetBatchMetadata(pos - 1)
		if errors.Is(err, accumulatorNotFound) {
			return errors.New("missing previous sequencer batch")
		} else if err != nil {
			return err
		}
	}

	dbBatch := t.db.NewBatch()
	err := deleteStartingAt(t.db, dbBatch, delayedSequencedPrefix, uint64ToBytes(prevDelayedMessages))
	if err != nil {
		return err
	}

	for _, batch := range batches {
		if batch.SequenceNumber != pos {
			return errors.New("unexpected batch sequence number")
		}
		if nextAcc != batch.BeforeInboxAcc {
			return errors.New("previous batch accumulator mismatch")
		}

		if batch.AfterDelayedCount > 0 {
			haveDelayedAcc, err := t.GetDelayedAcc(batch.AfterDelayedCount - 1)
			if errors.Is(err, accumulatorNotFound) {
				// We somehow missed a referenced delayed message; go back and look for it
				return delayedMessagesMismatch
			} else if err != nil {
				return err
			} else if haveDelayedAcc != batch.AfterDelayedAcc {
				// We somehow missed a delayed message reorg; go back and look for it
				return delayedMessagesMismatch
			}
		}

		nextAcc = batch.AfterInboxAcc
		pos++
	}

	var messages []arbstate.MessageWithMetadata
	backend := &multiplexerBackend{
		batchSeqNum: batches[0].SequenceNumber,
		batches:     batches,

		inbox:  t,
		ctx:    ctx,
		client: client,
	}
	multiplexer := arbstate.NewInboxMultiplexer(backend, prevDelayedMessages)
	batchMessageCounts := make(map[uint64]uint64)
	currentpos := startMessagePos + 1
	for {
		if len(backend.batches) == 0 {
			break
		}
		batchSeqNum := backend.batches[0].SequenceNumber
		msg, err := multiplexer.Pop()
		if err != nil {
			return err
		}

		messages = append(messages, *msg)
		batchMessageCounts[batchSeqNum] = currentpos
		currentpos += 1
	}

	for _, batch := range batches {
		meta := BatchMetadata{
			Accumulator:         batch.AfterInboxAcc,
			DelayedMessageCount: batch.AfterDelayedCount,
			MessageCount:        batchMessageCounts[batch.SequenceNumber],
		}
		metaBytes, err := rlp.EncodeToBytes(meta)
		if err != nil {
			return err
		}
		err = dbBatch.Put(dbKey(sequencerBatchMetaPrefix, batch.SequenceNumber), metaBytes)
		if err != nil {
			return err
		}

		seqNumData, err := rlp.EncodeToBytes(batch.SequenceNumber)
		if err != nil {
			return err
		}
		err = dbBatch.Put(dbKey(delayedSequencedPrefix, batch.AfterDelayedCount), seqNumData)
		if err != nil {
			return err
		}
	}

	err = deleteStartingAt(t.db, dbBatch, sequencerBatchMetaPrefix, uint64ToBytes(pos))
	if err != nil {
		return err
	}
	countData, err := rlp.EncodeToBytes(pos)
	if err != nil {
		return err
	}
	err = dbBatch.Put(sequencerBatchCountKey, countData)
	if err != nil {
		return err
	}
	log.Info("InboxTracker", "SequencerBatchCount", pos)

	// This also writes the batch
	err = t.txStreamer.AddMessagesAndEndBatch(startMessagePos, true, messages, dbBatch)
	if err != nil {
		return err
	}

	if t.validator != nil {
		batchMap := make(map[uint64][]byte, len(batches))
		for _, batch := range batches {
			msg, err := batch.Serialize(ctx, client)
			if err != nil {
				return err
			}
			batchMap[batch.SequenceNumber] = msg
		}
		t.validator.ProcessBatches(batchMap)
	}
	return nil
}

func (t *InboxTracker) ReorgDelayedTo(count uint64) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	currentCount, err := t.GetDelayedCount()
	if err != nil {
		return err
	}
	if currentCount == count {
		return nil
	} else if currentCount < count {
		return errors.New("attempted to reorg to future delayed count")
	}

	return t.setDelayedCountReorgAndWriteBatch(t.db.NewBatch(), count)
}

func (t *InboxTracker) ReorgBatchesTo(count uint64) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	var prevCount uint64
	var prevDelayedCount uint64
	if count > 0 {
		var err error
		_, prevCount, prevDelayedCount, err = t.GetBatchMetadata(count - 1)
		if errors.Is(err, accumulatorNotFound) {
			return errors.New("attempted to reorg to future batch count")
		} else if err != nil {
			return err
		}
	}

	dbBatch := t.db.NewBatch()

	err := deleteStartingAt(t.db, dbBatch, delayedSequencedPrefix, uint64ToBytes(prevDelayedCount))
	if err != nil {
		return err
	}
	err = deleteStartingAt(t.db, dbBatch, sequencerBatchMetaPrefix, uint64ToBytes(count))
	if err != nil {
		return err
	}
	countData, err := rlp.EncodeToBytes(count)
	if err != nil {
		return err
	}
	err = dbBatch.Put(sequencerBatchCountKey, countData)
	if err != nil {
		return err
	}
	log.Info("InboxTracker", "SequencerBatchCount", count)
	return t.txStreamer.ReorgToAndEndBatch(dbBatch, prevCount)
}
