//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbnode

import (
	"bytes"
	"context"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/offchainlabs/arbstate/arbos"
	"github.com/offchainlabs/arbstate/arbstate"
	"github.com/pkg/errors"
)

type InboxReaderDb struct {
	db         ethdb.Database
	inboxState *InboxState
	mutex      sync.Mutex
}

func NewInboxReaderDb(raw ethdb.Database, inboxState *InboxState) (*InboxReaderDb, error) {
	db := &InboxReaderDb{
		db:         rawdb.NewTable(raw, arbitrumPrefix),
		inboxState: inboxState,
	}
	err := db.initialize()
	return db, err
}

func (d *InboxReaderDb) initialize() error {
	batch := d.db.NewBatch()

	hasKey, err := d.db.Has(delayedMessageCountKey)
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

	hasKey, err = d.db.Has(sequencerBatchCountKey)
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
	}

	return batch.Write()
}

var accumulatorNotFound error = errors.New("accumulator not found")

func (d *InboxReaderDb) GetDelayedAcc(seqNum uint64) (common.Hash, error) {
	key := dbKey(delayedMessagePrefix, seqNum)
	hasKey, err := d.db.Has(key)
	if err != nil {
		return common.Hash{}, err
	}
	if !hasKey {
		return common.Hash{}, accumulatorNotFound
	}
	data, err := d.db.Get(key)
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

func (d *InboxReaderDb) GetDelayedCount() (uint64, error) {
	data, err := d.db.Get(delayedMessageCountKey)
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
	DelayedMessageCount uint64
	MessageCount        uint64
}

func (d *InboxReaderDb) GetBatchMetadata(seqNum uint64) (BatchMetadata, error) {
	key := dbKey(sequencerBatchMetaPrefix, seqNum)
	hasKey, err := d.db.Has(key)
	if err != nil {
		return BatchMetadata{}, err
	}
	if !hasKey {
		return BatchMetadata{}, accumulatorNotFound
	}
	data, err := d.db.Get(key)
	if err != nil {
		return BatchMetadata{}, err
	}
	var metadata BatchMetadata
	err = rlp.DecodeBytes(data, &metadata)
	return metadata, err
}

// Convenience function wrapping GetBatchMetadata
func (d *InboxReaderDb) GetBatchAcc(seqNum uint64) (common.Hash, error) {
	meta, err := d.GetBatchMetadata(seqNum)
	return meta.Accumulator, err
}

func (d *InboxReaderDb) GetBatchCount() (uint64, error) {
	data, err := d.db.Get(sequencerBatchCountKey)
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

func (d *InboxReaderDb) getDelayedMessageBytesAndAccumulator(seqNum uint64) ([]byte, common.Hash, error) {
	key := dbKey(delayedMessagePrefix, seqNum)
	data, err := d.db.Get(key)
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

func (d *InboxReaderDb) GetDelayedMessageAndAccumulator(seqNum uint64) (*arbos.L1IncomingMessage, common.Hash, error) {
	data, acc, err := d.getDelayedMessageBytesAndAccumulator(seqNum)
	if err != nil {
		return nil, acc, err
	}
	message, err := arbos.ParseIncomingL1Message(bytes.NewReader(data))
	return message, acc, err
}

func (d *InboxReaderDb) GetDelayedMessage(seqNum uint64) (*arbos.L1IncomingMessage, error) {
	msg, _, err := d.GetDelayedMessageAndAccumulator(seqNum)
	return msg, err
}

func (d *InboxReaderDb) addDelayedMessages(messages []*DelayedInboxMessage) error {
	if len(messages) == 0 {
		return nil
	}
	d.mutex.Lock()
	defer d.mutex.Unlock()

	pos, err := messages[0].Message.Header.SeqNum()
	if err != nil {
		return err
	}
	var nextAcc common.Hash
	if pos > 0 {
		var err error
		nextAcc, err = d.GetDelayedAcc(pos - 1)
		if err != nil {
			if errors.Is(err, accumulatorNotFound) {
				return errors.New("missing previous delayed message")
			} else {
				return err
			}
		}
	}

	batch := d.db.NewBatch()
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

	return d.setDelayedCountReorgAndWriteBatch(batch, pos)
}

// All-in-one delayed message count adjuster. Can go forwards or backwards.
// Requires the mutex is held. Sets the delayed count and performs any sequencer batch reorg necessary.
// Also deletes any future delayed messages.
func (d *InboxReaderDb) setDelayedCountReorgAndWriteBatch(batch ethdb.Batch, newDelayedCount uint64) error {
	err := deleteStartingAt(d.db, batch, delayedMessagePrefix, uint64ToBytes(newDelayedCount))
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

	seqBatchIter := d.db.NewIterator(delayedSequencedPrefix, uint64ToBytes(newDelayedCount))
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
		err = deleteStartingAt(d.db, batch, sequencerBatchMetaPrefix, uint64ToBytes(count))
		if err != nil {
			return err
		}
		var prevMeta BatchMetadata
		if count > 0 {
			prevMeta, err = d.GetBatchMetadata(count - 1)
			if err != nil {
				return err
			}
		}
		// Writes batch
		return d.inboxState.ReorgToAndEndBatch(batch, prevMeta.MessageCount)
	} else {
		return batch.Write()
	}
}

type multiplexerBackend struct {
	batchSeqNum           uint64
	batches               []*SequencerInboxBatch
	positionWithinMessage uint64

	ctx    context.Context
	client L1Interface
	db     *InboxReaderDb
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
	data, _, err := b.db.getDelayedMessageBytesAndAccumulator(seqNum)
	return data, err
}

var delayedMessagesMismatch = errors.New("sequencer batch delayed messages missing or different")

func (d *InboxReaderDb) addSequencerBatches(ctx context.Context, client L1Interface, batches []*SequencerInboxBatch) error {
	if len(batches) == 0 {
		return nil
	}
	d.mutex.Lock()
	defer d.mutex.Unlock()

	pos := batches[0].SequenceNumber
	var nextAcc common.Hash
	var prevDelayedMessages uint64
	var startMessagePos uint64
	if pos > 0 {
		meta, err := d.GetBatchMetadata(pos - 1)
		if errors.Is(err, accumulatorNotFound) {
			return errors.New("missing previous sequencer batch")
		} else if err != nil {
			return err
		}
		nextAcc = meta.Accumulator
		prevDelayedMessages = meta.DelayedMessageCount
		startMessagePos = meta.MessageCount
	}

	dbBatch := d.db.NewBatch()
	err := deleteStartingAt(d.db, dbBatch, delayedSequencedPrefix, uint64ToBytes(prevDelayedMessages))
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
			haveDelayedAcc, err := d.GetDelayedAcc(batch.AfterDelayedCount - 1)
			if errors.Is(err, accumulatorNotFound) {
				return delayedMessagesMismatch
			} else if err != nil {
				return err
			} else if haveDelayedAcc != batch.AfterDelayedAcc {
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

		db:     d,
		ctx:    ctx,
		client: client,
	}
	multiplexer := arbstate.NewInboxMultiplexer(backend, prevDelayedMessages)
	batchMessageCounts := make(map[uint64]uint64)
	for {
		if len(backend.batches) == 0 {
			break
		}
		batchSeqNum := backend.batches[0].SequenceNumber
		msg, err := multiplexer.Peek()
		if err != nil {
			return err
		}
		err = multiplexer.Advance()
		if err != nil {
			return err
		}
		messages = append(messages, *msg)
		batchMessageCounts[batchSeqNum] = startMessagePos + uint64(len(messages))
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

		pos++
	}

	err = deleteStartingAt(d.db, dbBatch, sequencerBatchMetaPrefix, uint64ToBytes(pos))
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

	// This also writes the batch
	return d.inboxState.AddMessagesAndEndBatch(startMessagePos, true, messages, dbBatch)
}

func (d *InboxReaderDb) ReorgDelayedTo(count uint64) error {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	currentCount, err := d.GetDelayedCount()
	if err != nil {
		return err
	}
	if currentCount == count {
		return nil
	} else if currentCount < count {
		return errors.New("attempted to reorg to future delayed count")
	}

	return d.setDelayedCountReorgAndWriteBatch(d.db.NewBatch(), count)
}

func (d *InboxReaderDb) ReorgBatchesTo(count uint64) error {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	var prevMeta BatchMetadata
	if count > 0 {
		var err error
		prevMeta, err = d.GetBatchMetadata(count - 1)
		if errors.Is(err, accumulatorNotFound) {
			return errors.New("attempted to reorg to future batch count")
		} else if err != nil {
			return err
		}
	}

	dbBatch := d.db.NewBatch()

	err := deleteStartingAt(d.db, dbBatch, delayedSequencedPrefix, uint64ToBytes(prevMeta.DelayedMessageCount))
	if err != nil {
		return err
	}
	err = deleteStartingAt(d.db, dbBatch, sequencerBatchMetaPrefix, uint64ToBytes(count))
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

	return d.inboxState.ReorgToAndEndBatch(dbBatch, prevMeta.MessageCount)
}
