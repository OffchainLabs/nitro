//
// Copyright 2021-2022, Offchain Labs, Inc. All rights reserved.
//

package arbnode

import (
	"bytes"
	"context"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/validator"
	"github.com/pkg/errors"
)

type InboxTracker struct {
	db         ethdb.Database
	txStreamer *TransactionStreamer
	mutex      sync.Mutex
	validator  *validator.BlockValidator
	das        arbstate.DataAvailabilityServiceReader
}

func NewInboxTracker(raw ethdb.Database, txStreamer *TransactionStreamer, das arbstate.DataAvailabilityServiceReader) (*InboxTracker, error) {
	db := &InboxTracker{
		db:         rawdb.NewTable(raw, arbitrumPrefix),
		txStreamer: txStreamer,
		das:        das,
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
	MessageCount        arbutil.MessageIndex
	DelayedMessageCount uint64
	L1Block             uint64
}

func (t *InboxTracker) GetBatchMetadata(seqNum uint64) (BatchMetadata, error) {
	key := dbKey(sequencerBatchMetaPrefix, seqNum)
	hasKey, err := t.db.Has(key)
	if err != nil {
		return BatchMetadata{}, err
	}
	if !hasKey {
		return BatchMetadata{}, accumulatorNotFound
	}
	data, err := t.db.Get(key)
	if err != nil {
		return BatchMetadata{}, err
	}
	var metadata BatchMetadata
	err = rlp.DecodeBytes(data, &metadata)
	return metadata, err
}

func (t *InboxTracker) GetBatchMessageCount(seqNum uint64) (arbutil.MessageIndex, error) {
	metadata, err := t.GetBatchMetadata(seqNum)
	return metadata.MessageCount, err
}

// Convenience function wrapping GetBatchMetadata
func (t *InboxTracker) GetBatchAcc(seqNum uint64) (common.Hash, error) {
	metadata, err := t.GetBatchMetadata(seqNum)
	return metadata.Accumulator, err
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
		seqBatchIter.Next()
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
		var prevMesssageCount arbutil.MessageIndex
		if count > 0 {
			prevMesssageCount, err = t.GetBatchMessageCount(count - 1)
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
	client arbutil.L1Interface
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

func (t *InboxTracker) AddSequencerBatches(ctx context.Context, client arbutil.L1Interface, batches []*SequencerInboxBatch) error {
	if len(batches) == 0 {
		return nil
	}
	t.mutex.Lock()
	defer t.mutex.Unlock()

	pos := batches[0].SequenceNumber
	startPos := pos
	var nextAcc common.Hash
	var prevbatchmeta BatchMetadata
	if pos > 0 {
		var err error
		prevbatchmeta, err = t.GetBatchMetadata(pos - 1)
		nextAcc = prevbatchmeta.Accumulator
		if errors.Is(err, accumulatorNotFound) {
			return errors.New("missing previous sequencer batch")
		} else if err != nil {
			return err
		}
	}

	dbBatch := t.db.NewBatch()
	err := deleteStartingAt(t.db, dbBatch, delayedSequencedPrefix, uint64ToBytes(prevbatchmeta.DelayedMessageCount))
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
	multiplexer := arbstate.NewInboxMultiplexer(backend, prevbatchmeta.DelayedMessageCount, t.das)
	batchMessageCounts := make(map[uint64]arbutil.MessageIndex)
	currentpos := prevbatchmeta.MessageCount + 1
	for {
		if len(backend.batches) == 0 {
			break
		}
		batchSeqNum := backend.batches[0].SequenceNumber
		msg, err := multiplexer.Pop(ctx)
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
			L1Block:             batch.BlockNumber,
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
	log.Info("InboxTracker", "sequencerBatchCount", pos, "prevMessageCount", prevbatchmeta.MessageCount, "newMessageCount", int(prevbatchmeta.MessageCount)+len(messages))

	if t.validator != nil {
		t.validator.ReorgToBatchCount(startPos)
	}

	// This also writes the batch
	err = t.txStreamer.AddMessagesAndEndBatch(prevbatchmeta.MessageCount, true, messages, dbBatch)
	if err != nil {
		return err
	}

	if t.validator != nil {
		batchBytes := make([][]byte, 0, len(batches))
		for _, batch := range batches {
			msg, err := batch.Serialize(ctx, client)
			if err != nil {
				return err
			}
			batchBytes = append(batchBytes, msg)
		}
		t.validator.ProcessBatches(startPos, batchBytes)
	}

	if t.txStreamer.broadcastServer != nil && prevbatchmeta.MessageCount > 0 {
		t.txStreamer.broadcastServer.Confirm(prevbatchmeta.MessageCount - 1)
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

	var prevBatchMeta BatchMetadata
	if count > 0 {
		var err error
		prevBatchMeta, err = t.GetBatchMetadata(count - 1)
		if errors.Is(err, accumulatorNotFound) {
			return errors.New("attempted to reorg to future batch count")
		} else if err != nil {
			return err
		}
	}

	dbBatch := t.db.NewBatch()

	err := deleteStartingAt(t.db, dbBatch, delayedSequencedPrefix, uint64ToBytes(prevBatchMeta.DelayedMessageCount))
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
	return t.txStreamer.ReorgToAndEndBatch(dbBatch, prevBatchMeta.MessageCount)
}
