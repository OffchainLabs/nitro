// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbnode

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/broadcaster"
	"github.com/offchainlabs/nitro/staker"
	"github.com/offchainlabs/nitro/util/containers"
	"github.com/pkg/errors"
)

type InboxTracker struct {
	db         ethdb.Database
	txStreamer *TransactionStreamer
	mutex      sync.Mutex
	validator  *staker.BlockValidator
	das        arbstate.DataAvailabilityReader

	batchMetaMutex sync.Mutex
	batchMeta      *containers.LruCache[uint64, BatchMetadata]
}

func NewInboxTracker(db ethdb.Database, txStreamer *TransactionStreamer, das arbstate.DataAvailabilityReader) (*InboxTracker, error) {
	// We support a nil txStreamer for the pruning code
	if txStreamer != nil && txStreamer.chainConfig.ArbitrumChainParams.DataAvailabilityCommittee && das == nil {
		return nil, errors.New("data availability service required but unconfigured")
	}
	tracker := &InboxTracker{
		db:         db,
		txStreamer: txStreamer,
		das:        das,
		batchMeta:  containers.NewLruCache[uint64, BatchMetadata](1000),
	}
	return tracker, nil
}

func (t *InboxTracker) SetBlockValidator(validator *staker.BlockValidator) {
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

	err = batch.Write()
	if err != nil {
		return err
	}

	return nil
}

var AccumulatorNotFoundErr = errors.New("accumulator not found")

func (t *InboxTracker) GetDelayedAcc(seqNum uint64) (common.Hash, error) {
	key := dbKey(rlpDelayedMessagePrefix, seqNum)
	hasKey, err := t.db.Has(key)
	if err != nil {
		return common.Hash{}, err
	}
	if !hasKey {
		key = dbKey(legacyDelayedMessagePrefix, seqNum)
		hasKey, err = t.db.Has(key)
		if err != nil {
			return common.Hash{}, err
		}
		if !hasKey {
			return common.Hash{}, fmt.Errorf("%w: not found delayed %d", AccumulatorNotFoundErr, seqNum)
		}
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
	t.batchMetaMutex.Lock()
	defer t.batchMetaMutex.Unlock()
	metadata, exist := t.batchMeta.Get(seqNum)
	if exist {
		return metadata, nil
	}
	key := dbKey(sequencerBatchMetaPrefix, seqNum)
	hasKey, err := t.db.Has(key)
	if err != nil {
		return BatchMetadata{}, err
	}
	if !hasKey {
		return BatchMetadata{}, fmt.Errorf("%w: no metadata for batch %d", AccumulatorNotFoundErr, seqNum)
	}
	data, err := t.db.Get(key)
	if err != nil {
		return BatchMetadata{}, err
	}
	err = rlp.DecodeBytes(data, &metadata)
	if err != nil {
		return BatchMetadata{}, err
	}
	t.batchMeta.Add(seqNum, metadata)
	return metadata, nil
}

func (t *InboxTracker) GetBatchMessageCount(seqNum uint64) (arbutil.MessageIndex, error) {
	metadata, err := t.GetBatchMetadata(seqNum)
	return metadata.MessageCount, err
}

// GetBatchAcc is a convenience function wrapping GetBatchMetadata
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

func (t *InboxTracker) PopulateFeedBacklog(broadcastServer *broadcaster.Broadcaster) error {
	batchCount, err := t.GetBatchCount()
	if err != nil {
		return fmt.Errorf("error getting batch count: %w", err)
	}
	var startMessage arbutil.MessageIndex
	if batchCount >= 2 {
		// As in AddSequencerBatches, we want to keep the most recent batch's messages.
		// This prevents issues if a user's L1 is a bit behind or an L1 reorg occurs.
		// `batchCount - 2` is the index of the batch before the last batch.
		batchIndex := batchCount - 2
		startMessage, err = t.GetBatchMessageCount(batchIndex)
		if err != nil {
			return fmt.Errorf("error getting batch %v message count: %w", batchIndex, err)
		}
	}
	messageCount, err := t.txStreamer.GetMessageCount()
	if err != nil {
		return fmt.Errorf("error getting tx streamer message count: %w", err)
	}
	var feedMessages []*broadcaster.BroadcastFeedMessage
	for seqNum := startMessage; seqNum < messageCount; seqNum++ {
		message, err := t.txStreamer.GetMessage(seqNum)
		if err != nil {
			return fmt.Errorf("error getting message %v: %w", seqNum, err)
		}
		feedMessage, err := broadcastServer.NewBroadcastFeedMessage(*message, seqNum)
		if err != nil {
			return fmt.Errorf("error creating broadcast feed message %v: %w", seqNum, err)
		}
		feedMessages = append(feedMessages, feedMessage)
	}
	broadcastServer.BroadcastFeedMessages(feedMessages)
	return nil
}

func (t *InboxTracker) legacyGetDelayedMessageAndAccumulator(seqNum uint64) (*arbostypes.L1IncomingMessage, common.Hash, error) {
	key := dbKey(legacyDelayedMessagePrefix, seqNum)
	data, err := t.db.Get(key)
	if err != nil {
		return nil, common.Hash{}, err
	}
	if len(data) < 32 {
		return nil, common.Hash{}, errors.New("delayed message legacy entry missing accumulator")
	}
	var acc common.Hash
	copy(acc[:], data[:32])
	msg, err := arbostypes.ParseIncomingL1Message(bytes.NewReader(data[32:]), nil)
	return msg, acc, err
}

func (t *InboxTracker) GetDelayedMessageAccumulatorAndParentChainBlockNumber(seqNum uint64) (*arbostypes.L1IncomingMessage, common.Hash, uint64, error) {
	delayedMessageKey := dbKey(rlpDelayedMessagePrefix, seqNum)
	exists, err := t.db.Has(delayedMessageKey)
	if err != nil {
		return nil, common.Hash{}, 0, err
	}
	if !exists {
		msg, acc, err := t.legacyGetDelayedMessageAndAccumulator(seqNum)
		return msg, acc, 0, err
	}
	data, err := t.db.Get(delayedMessageKey)
	if err != nil {
		return nil, common.Hash{}, 0, err
	}
	if len(data) < 32 {
		return nil, common.Hash{}, 0, errors.New("delayed message new entry missing accumulator")
	}
	var acc common.Hash
	copy(acc[:], data[:32])
	var msg *arbostypes.L1IncomingMessage
	err = rlp.DecodeBytes(data[32:], &msg)
	if err != nil {
		return msg, acc, 0, err
	}

	parentChainBlockNumberKey := dbKey(parentChainBlockNumberPrefix, seqNum)
	exists, err = t.db.Has(parentChainBlockNumberKey)
	if err != nil {
		return msg, acc, 0, err
	}
	if !exists {
		return msg, acc, msg.Header.BlockNumber, nil
	}
	data, err = t.db.Get(parentChainBlockNumberKey)
	if err != nil {
		return msg, acc, 0, err
	}

	return msg, acc, binary.BigEndian.Uint64(data), nil

}

func (t *InboxTracker) GetDelayedMessage(seqNum uint64) (*arbostypes.L1IncomingMessage, error) {
	msg, _, _, err := t.GetDelayedMessageAccumulatorAndParentChainBlockNumber(seqNum)
	return msg, err
}

func (t *InboxTracker) GetDelayedMessageBytes(seqNum uint64) ([]byte, error) {
	msg, err := t.GetDelayedMessage(seqNum)
	if err != nil {
		return nil, err
	}
	return msg.Serialize()
}

func (t *InboxTracker) AddDelayedMessages(messages []*DelayedInboxMessage, hardReorg bool) error {
	if len(messages) == 0 {
		return nil
	}
	t.mutex.Lock()
	defer t.mutex.Unlock()

	pos, err := messages[0].Message.Header.SeqNum()
	if err != nil {
		return err
	}

	if !hardReorg {
		// This math is safe to do as we know len(messages) > 0
		haveLastAcc, err := t.GetDelayedAcc(pos + uint64(len(messages)) - 1)
		if err == nil {
			if haveLastAcc == messages[len(messages)-1].AfterInboxAcc() {
				// We already have these delayed messages
				return nil
			}
		} else if !errors.Is(err, AccumulatorNotFoundErr) {
			return err
		}
	}

	var nextAcc common.Hash
	if pos > 0 {
		var err error
		nextAcc, err = t.GetDelayedAcc(pos - 1)
		if err != nil {
			if errors.Is(err, AccumulatorNotFoundErr) {
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

		delayedMsgKey := dbKey(rlpDelayedMessagePrefix, seqNum)

		msgData, err := rlp.EncodeToBytes(message.Message)
		if err != nil {
			return err
		}
		data := nextAcc.Bytes()
		data = append(data, msgData...)
		err = batch.Put(delayedMsgKey, data)
		if err != nil {
			return err
		}

		if message.ParentChainBlockNumber != message.Message.Header.BlockNumber {
			parentChainBlockNumberKey := dbKey(parentChainBlockNumberPrefix, seqNum)
			parentChainBlockNumberByte := make([]byte, 8)
			binary.BigEndian.PutUint64(parentChainBlockNumberByte, message.ParentChainBlockNumber)
			err = batch.Put(parentChainBlockNumberKey, parentChainBlockNumberByte)
			if err != nil {
				return err
			}
		}

		pos++
	}

	return t.setDelayedCountReorgAndWriteBatch(batch, pos, true)
}

func (t *InboxTracker) clearBatchMetaCache() {
	t.batchMetaMutex.Lock()
	defer t.batchMetaMutex.Unlock()
	t.batchMeta.Clear()
}

// All-in-one delayed message count adjuster. Can go forwards or backwards.
// Requires the mutex is held. Sets the delayed count and performs any sequencer batch reorg necessary.
// Also deletes any future delayed messages.
func (t *InboxTracker) setDelayedCountReorgAndWriteBatch(batch ethdb.Batch, newDelayedCount uint64, canReorgBatches bool) error {
	err := deleteStartingAt(t.db, batch, rlpDelayedMessagePrefix, uint64ToKey(newDelayedCount))
	if err != nil {
		return err
	}
	err = deleteStartingAt(t.db, batch, parentChainBlockNumberPrefix, uint64ToKey(newDelayedCount))
	if err != nil {
		return err
	}
	err = deleteStartingAt(t.db, batch, legacyDelayedMessagePrefix, uint64ToKey(newDelayedCount))
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

	seqBatchIter := t.db.NewIterator(delayedSequencedPrefix, uint64ToKey(newDelayedCount+1))
	defer seqBatchIter.Release()
	var reorgSeqBatchesToCount *uint64
	for seqBatchIter.Next() {
		var batchSeqNum uint64
		err := rlp.DecodeBytes(seqBatchIter.Value(), &batchSeqNum)
		if err != nil {
			return err
		}
		if !canReorgBatches {
			return fmt.Errorf("reorging of sequencer batch number %v via delayed messages reorg to count %v disabled in this instance", batchSeqNum, newDelayedCount)
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
	err = seqBatchIter.Error()
	if err != nil {
		return err
	}
	// Release the iterator early.
	// It's fine to call Release multiple times,
	// which we'll do because of the defer.
	seqBatchIter.Release()
	if reorgSeqBatchesToCount != nil {
		// Clear the batchMeta cache after writing the reorg to disk
		defer t.clearBatchMetaCache()

		count := *reorgSeqBatchesToCount
		if t.validator != nil {
			t.validator.ReorgToBatchCount(count)
		}
		countData, err := rlp.EncodeToBytes(count)
		if err != nil {
			return err
		}
		err = batch.Put(sequencerBatchCountKey, countData)
		if err != nil {
			return err
		}
		log.Warn("InboxTracker delayed message reorg is causing a sequencer batch reorg", "sequencerBatchCount", count, "delayedCount", newDelayedCount)
		err = deleteStartingAt(t.db, batch, sequencerBatchMetaPrefix, uint64ToKey(count))
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

func (b *multiplexerBackend) ReadDelayedInbox(seqNum uint64) (*arbostypes.L1IncomingMessage, error) {
	if len(b.batches) == 0 || seqNum >= b.batches[0].AfterDelayedCount {
		return nil, errors.New("attempted to read past end of sequencer batch delayed messages")
	}
	return b.inbox.GetDelayedMessage(seqNum)
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
		if errors.Is(err, AccumulatorNotFoundErr) {
			return errors.New("missing previous sequencer batch")
		} else if err != nil {
			return err
		}
	}

	dbBatch := t.db.NewBatch()
	err := deleteStartingAt(t.db, dbBatch, delayedSequencedPrefix, uint64ToKey(prevbatchmeta.DelayedMessageCount+1))
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
			if errors.Is(err, AccumulatorNotFoundErr) {
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

	var messages []arbostypes.MessageWithMetadata
	backend := &multiplexerBackend{
		batchSeqNum: batches[0].SequenceNumber,
		batches:     batches,

		inbox:  t,
		ctx:    ctx,
		client: client,
	}
	multiplexer := arbstate.NewInboxMultiplexer(backend, prevbatchmeta.DelayedMessageCount, t.das, arbstate.KeysetValidate)
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

	lastBatchMeta := prevbatchmeta
	batchMetas := make(map[uint64]BatchMetadata, len(batches))
	for _, batch := range batches {
		meta := BatchMetadata{
			Accumulator:         batch.AfterInboxAcc,
			DelayedMessageCount: batch.AfterDelayedCount,
			MessageCount:        batchMessageCounts[batch.SequenceNumber],
			L1Block:             batch.BlockNumber,
		}
		batchMetas[batch.SequenceNumber] = meta
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
		if batch.AfterDelayedCount < lastBatchMeta.DelayedMessageCount {
			return errors.New("batch delayed message count went backwards")
		}
		if batch.AfterDelayedCount > lastBatchMeta.DelayedMessageCount {
			err = dbBatch.Put(dbKey(delayedSequencedPrefix, batch.AfterDelayedCount), seqNumData)
			if err != nil {
				return err
			}
		}
		lastBatchMeta = meta
	}

	err = deleteStartingAt(t.db, dbBatch, sequencerBatchMetaPrefix, uint64ToKey(pos))
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

	newMessageCount := prevbatchmeta.MessageCount + arbutil.MessageIndex(len(messages))
	var latestL1Block uint64
	if len(batches) > 0 {
		latestL1Block = batches[len(batches)-1].BlockNumber
	}
	var latestTimestamp uint64
	if len(messages) > 0 {
		latestTimestamp = messages[len(messages)-1].Message.Header.Timestamp
	}
	log.Info(
		"InboxTracker",
		"sequencerBatchCount", pos,
		"messageCount", newMessageCount,
		"l1Block", latestL1Block,
		"l1Timestamp", time.Unix(int64(latestTimestamp), 0),
	)

	if t.validator != nil {
		t.validator.ReorgToBatchCount(startPos)
	}

	// This also writes the batch
	err = t.txStreamer.AddMessagesAndEndBatch(prevbatchmeta.MessageCount, true, messages, dbBatch)
	if err != nil {
		return err
	}

	// Update the batchMeta cache immediately after writing the batch
	t.batchMetaMutex.Lock()
	for seqNum, meta := range batchMetas {
		t.batchMeta.Add(seqNum, meta)
	}
	t.batchMetaMutex.Unlock()

	if t.txStreamer.broadcastServer != nil && pos > 1 {
		prevprevbatchmeta, err := t.GetBatchMetadata(pos - 2)
		if errors.Is(err, AccumulatorNotFoundErr) {
			return errors.New("missing previous previous sequencer batch")
		} else if err != nil {
			return err
		}
		if prevprevbatchmeta.MessageCount > 0 {
			// Confirm messages from batch before last batch
			t.txStreamer.broadcastServer.Confirm(prevprevbatchmeta.MessageCount - 1)
		}
	}

	return nil
}

func (t *InboxTracker) ReorgDelayedTo(count uint64, canReorgBatches bool) error {
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

	return t.setDelayedCountReorgAndWriteBatch(t.db.NewBatch(), count, canReorgBatches)
}

func (t *InboxTracker) ReorgBatchesTo(count uint64) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	var prevBatchMeta BatchMetadata
	if count > 0 {
		var err error
		prevBatchMeta, err = t.GetBatchMetadata(count - 1)
		if errors.Is(err, AccumulatorNotFoundErr) {
			return errors.New("attempted to reorg to future batch count")
		} else if err != nil {
			return err
		}
	}

	if t.validator != nil {
		t.validator.ReorgToBatchCount(count)
	}

	// Clear the batchMeta cache after writing the reorg to disk
	defer t.clearBatchMetaCache()

	dbBatch := t.db.NewBatch()

	err := deleteStartingAt(t.db, dbBatch, delayedSequencedPrefix, uint64ToKey(prevBatchMeta.DelayedMessageCount+1))
	if err != nil {
		return err
	}
	err = deleteStartingAt(t.db, dbBatch, sequencerBatchMetaPrefix, uint64ToKey(count))
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
