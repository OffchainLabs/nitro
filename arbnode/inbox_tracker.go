// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbnode

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/broadcaster"
	m "github.com/offchainlabs/nitro/broadcaster/message"
	"github.com/offchainlabs/nitro/staker"
	"github.com/offchainlabs/nitro/util/containers"
)

var (
	inboxLatestBatchGauge        = metrics.NewRegisteredGauge("arb/inbox/latest/batch", nil)
	inboxLatestBatchMessageGauge = metrics.NewRegisteredGauge("arb/inbox/latest/batch/message", nil)
)

type InboxTracker struct {
	db               ethdb.Database
	txStreamer       *TransactionStreamer
	mutex            sync.Mutex
	validator        *staker.BlockValidator
	das              arbstate.DataAvailabilityReader
	blobReader       arbstate.BlobReader
	firstBatchToKeep uint64

	batchMetaMutex sync.Mutex
	batchMeta      *containers.LruCache[uint64, BatchMetadata]
}

func NewInboxTracker(db ethdb.Database, txStreamer *TransactionStreamer, das arbstate.DataAvailabilityReader, blobReader arbstate.BlobReader, firstBatchToKeep uint64) (*InboxTracker, error) {
	// We support a nil txStreamer for the pruning code
	if txStreamer != nil && txStreamer.chainConfig.ArbitrumChainParams.DataAvailabilityCommittee && das == nil {
		return nil, errors.New("data availability service required but unconfigured")
	}
	tracker := &InboxTracker{
		db:               db,
		txStreamer:       txStreamer,
		das:              das,
		blobReader:       blobReader,
		batchMeta:        containers.NewLruCache[uint64, BatchMetadata](1000),
		firstBatchToKeep: firstBatchToKeep,
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

func (t *InboxTracker) deleteBatchMetadataStartingAt(dbBatch ethdb.Batch, startIndex uint64) error {
	t.batchMetaMutex.Lock()
	defer t.batchMetaMutex.Unlock()
	iter := t.db.NewIterator(sequencerBatchMetaPrefix, uint64ToKey(startIndex))
	defer iter.Release()
	for iter.Next() {
		curKey := iter.Key()
		err := dbBatch.Delete(curKey)
		if err != nil {
			return err
		}
		curIndex := binary.BigEndian.Uint64(bytes.TrimPrefix(curKey, sequencerBatchMetaPrefix))
		t.batchMeta.Remove(curIndex)
	}
	return iter.Error()
}

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
	ParentChainBlock    uint64
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

func (t *InboxTracker) GetBatchParentChainBlock(seqNum uint64) (uint64, error) {
	metadata, err := t.GetBatchMetadata(seqNum)
	return metadata.ParentChainBlock, err
}

func (t *InboxTracker) GetFirstBatchToKeep() uint64 {
	return t.firstBatchToKeep
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

// err will return unexpected/internal errors
// bool will be false if batch not found (meaning, block not yet posted on a batch)
func (t *InboxTracker) FindInboxBatchContainingMessage(pos arbutil.MessageIndex) (uint64, bool, error) {
	batchCount, err := t.GetBatchCount()
	if err != nil {
		return 0, false, err
	}
	low := uint64(0)
	high := batchCount - 1
	lastBatchMessageCount, err := t.GetBatchMessageCount(high)
	if err != nil {
		return 0, false, err
	}
	if lastBatchMessageCount <= pos {
		return 0, false, nil
	}
	// Iteration preconditions:
	// - high >= low
	// - msgCount(low - 1) <= pos implies low <= target
	// - msgCount(high) > pos implies high >= target
	// Therefore, if low == high, then low == high == target
	for {
		// Due to integer rounding, mid >= low && mid < high
		mid := (low + high) / 2
		count, err := t.GetBatchMessageCount(mid)
		if err != nil {
			return 0, false, err
		}
		if count < pos {
			// Must narrow as mid >= low, therefore mid + 1 > low, therefore newLow > oldLow
			// Keeps low precondition as msgCount(mid) < pos
			low = mid + 1
		} else if count == pos {
			return mid + 1, true, nil
		} else if count == pos+1 || mid == low { // implied: count > pos
			return mid, true, nil
		} else {
			// implied: count > pos + 1
			// Must narrow as mid < high, therefore newHigh < oldHigh
			// Keeps high precondition as msgCount(mid) > pos
			high = mid
		}
		if high == low {
			return high, true, nil
		}
	}
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
	var feedMessages []*m.BroadcastFeedMessage
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
	var nextAcc common.Hash
	for len(messages) > 0 {
		pos, err := messages[0].Message.Header.SeqNum()
		if err != nil {
			return err
		}
		if pos+1 == t.firstBatchToKeep {
			nextAcc = messages[0].AfterInboxAcc()
		}
		if pos < t.firstBatchToKeep {
			messages = messages[1:]
		} else {
			break
		}
	}
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

	if pos > t.firstBatchToKeep {
		var err error
		nextAcc, err = t.GetDelayedAcc(pos - 1)
		if err != nil {
			if errors.Is(err, AccumulatorNotFoundErr) {
				return errors.New("missing previous delayed message")
			}
			return err
		}
	}

	batch := t.db.NewBatch()
	for _, message := range messages {
		seqNum, err := message.Message.Header.SeqNum()
		if err != nil {
			return err
		}

		if seqNum != pos {
			return fmt.Errorf("unexpected delayed sequence number %v, expected %v", seqNum, pos)
		}

		if nextAcc != message.BeforeInboxAcc {
			return fmt.Errorf("previous delayed accumulator mismatch for message %v", seqNum)
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
		if err := rlp.DecodeBytes(seqBatchIter.Value(), &batchSeqNum); err != nil {
			return err
		}
		if !canReorgBatches {
			return fmt.Errorf("reorging of sequencer batch number %v via delayed messages reorg to count %v disabled in this instance", batchSeqNum, newDelayedCount)
		}
		if err := batch.Delete(seqBatchIter.Key()); err != nil {
			return err
		}
		if reorgSeqBatchesToCount == nil {
			// Set the count to the first deleted sequence number.
			// E.g. if the deleted sequence number is 1, set the count to 1,
			// meaning that the last and only batch is at sequence number 0.
			reorgSeqBatchesToCount = &batchSeqNum
		}
	}
	if err := seqBatchIter.Error(); err != nil {
		return err
	}
	// Release the iterator early.
	// It's fine to call Release multiple times,
	// which we'll do because of the defer.
	seqBatchIter.Release()
	if reorgSeqBatchesToCount == nil {
		return batch.Write()
	}

	count := *reorgSeqBatchesToCount
	if t.validator != nil {
		t.validator.ReorgToBatchCount(count)
	}
	countData, err = rlp.EncodeToBytes(count)
	if err != nil {
		return err
	}
	if err := batch.Put(sequencerBatchCountKey, countData); err != nil {
		return err
	}
	log.Warn("InboxTracker delayed message reorg is causing a sequencer batch reorg", "sequencerBatchCount", count, "delayedCount", newDelayedCount)

	if err := t.deleteBatchMetadataStartingAt(batch, count); err != nil {
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
}

type multiplexerBackend struct {
	batchSeqNum           uint64
	batches               []*SequencerInboxBatch
	positionWithinMessage uint64

	ctx    context.Context
	client arbutil.L1Interface
	inbox  *InboxTracker
}

func (b *multiplexerBackend) PeekSequencerInbox() ([]byte, common.Hash, error) {
	if len(b.batches) == 0 {
		return nil, common.Hash{}, errors.New("read past end of specified sequencer batches")
	}
	bytes, err := b.batches[0].Serialize(b.ctx, b.client)
	return bytes, b.batches[0].BlockHash, err
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
	var nextAcc common.Hash
	var prevbatchmeta BatchMetadata
	sequenceNumberToKeep := t.firstBatchToKeep
	if sequenceNumberToKeep > 0 {
		sequenceNumberToKeep--
	}
	for len(batches) > 0 {
		if batches[0].SequenceNumber+1 == sequenceNumberToKeep {
			nextAcc = batches[0].AfterInboxAcc
			prevbatchmeta = BatchMetadata{
				Accumulator:         batches[0].AfterInboxAcc,
				DelayedMessageCount: batches[0].AfterDelayedCount,
				//MessageCount:        batchMessageCounts[batches[0].SequenceNumber],
				ParentChainBlock: batches[0].ParentChainBlockNumber,
			}
		}
		if batches[0].SequenceNumber < sequenceNumberToKeep {
			batches = batches[1:]
		} else {
			break
		}
	}
	if len(batches) == 0 {
		return nil
	}
	t.mutex.Lock()
	defer t.mutex.Unlock()

	pos := batches[0].SequenceNumber
	startPos := pos

	if pos > sequenceNumberToKeep {
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

		if batch.AfterDelayedCount > 0 && t.firstBatchToKeep == 0 {
			haveDelayedAcc, err := t.GetDelayedAcc(batch.AfterDelayedCount - 1)
			if errors.Is(err, AccumulatorNotFoundErr) {
				// We somehow missed a referenced delayed message; go back and look for it
				return delayedMessagesMismatch
			}
			if err != nil {
				return err
			}
			if haveDelayedAcc != batch.AfterDelayedAcc {
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
	var daProviders []arbstate.DataAvailabilityProvider
	if t.das != nil {
		daProviders = append(daProviders, arbstate.NewDAProviderDAS(t.das))
	}
	if t.blobReader != nil {
		daProviders = append(daProviders, arbstate.NewDAProviderBlobReader(t.blobReader))
	}
	multiplexer := arbstate.NewInboxMultiplexer(backend, prevbatchmeta.DelayedMessageCount, daProviders, arbstate.KeysetValidate)
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
			ParentChainBlock:    batch.ParentChainBlockNumber,
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

	err = t.deleteBatchMetadataStartingAt(dbBatch, pos)
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
		latestL1Block = batches[len(batches)-1].ParentChainBlockNumber
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
	inboxLatestBatchGauge.Update(int64(pos))
	inboxLatestBatchMessageGauge.Update(int64(newMessageCount))

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
		}
		if err != nil {
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
	}
	if currentCount < count {
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
		}
		if err != nil {
			return err
		}
	}

	if t.validator != nil {
		t.validator.ReorgToBatchCount(count)
	}

	dbBatch := t.db.NewBatch()

	err := deleteStartingAt(t.db, dbBatch, delayedSequencedPrefix, uint64ToKey(prevBatchMeta.DelayedMessageCount+1))
	if err != nil {
		return err
	}
	err = t.deleteBatchMetadataStartingAt(dbBatch, count)
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
