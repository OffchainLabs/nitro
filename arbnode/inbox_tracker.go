// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

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
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/broadcaster"
	"github.com/offchainlabs/nitro/broadcaster/message"
	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/staker"
	"github.com/offchainlabs/nitro/util/containers"
)

var (
	inboxLatestBatchGauge        = metrics.NewRegisteredGauge("arb/inbox/latest/batch", nil)
	inboxLatestBatchMessageGauge = metrics.NewRegisteredGauge("arb/inbox/latest/batch/message", nil)
)

type InboxTracker struct {
	db             ethdb.Database
	txStreamer     *TransactionStreamer
	mutex          sync.Mutex
	validator      *staker.BlockValidator
	dapReaders     *daprovider.ReaderRegistry
	snapSyncConfig SnapSyncConfig

	batchMetaMutex sync.Mutex
	batchMeta      *containers.LruCache[uint64, BatchMetadata]
}

func NewInboxTracker(db ethdb.Database, txStreamer *TransactionStreamer, dapReaders *daprovider.ReaderRegistry, snapSyncConfig SnapSyncConfig) (*InboxTracker, error) {
	tracker := &InboxTracker{
		db:             db,
		txStreamer:     txStreamer,
		dapReaders:     dapReaders,
		batchMeta:      containers.NewLruCache[uint64, BatchMetadata](1000),
		snapSyncConfig: snapSyncConfig,
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

// Database returns the underlying database for test purposes
func (t *InboxTracker) Database() ethdb.Database {
	return t.db
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
	var feedMessages []*message.BroadcastFeedMessage
	for seqNum := startMessage; seqNum < messageCount; seqNum++ {
		message, err := t.txStreamer.GetMessage(seqNum)
		if err != nil {
			return fmt.Errorf("error getting message %v: %w", seqNum, err)
		}

		msgResult, err := t.txStreamer.ResultAtMessageIndex(seqNum)
		var blockHash *common.Hash
		if err == nil {
			blockHash = &msgResult.BlockHash
		}

		blockMetadata, err := t.txStreamer.BlockMetadataAtMessageIndex(seqNum)
		if err != nil {
			log.Warn("Error getting blockMetadata byte array from tx streamer", "err", err)
		}

		feedMessage, err := broadcastServer.NewBroadcastFeedMessage(*message, seqNum, blockHash, blockMetadata)
		if err != nil {
			return fmt.Errorf("error creating broadcast feed message %v: %w", seqNum, err)
		}
		feedMessages = append(feedMessages, feedMessage)
	}
	return broadcastServer.PopulateFeedBacklog(feedMessages)
}

func (t *InboxTracker) legacyGetDelayedMessageAndAccumulator(ctx context.Context, seqNum uint64) (*arbostypes.L1IncomingMessage, common.Hash, error) {
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
	if err != nil {
		return nil, common.Hash{}, err
	}

	err = msg.FillInBatchGasFields(func(batchNum uint64) ([]byte, error) {
		data, _, err := t.txStreamer.inboxReader.GetSequencerMessageBytes(ctx, batchNum)
		return data, err
	})

	return msg, acc, err
}

func (t *InboxTracker) GetDelayedMessageAccumulatorAndParentChainBlockNumber(ctx context.Context, seqNum uint64) (*arbostypes.L1IncomingMessage, common.Hash, uint64, error) {
	msg, acc, blockNum, err := t.getRawDelayedMessageAccumulatorAndParentChainBlockNumber(ctx, seqNum)
	if err != nil {
		return msg, acc, blockNum, err
	}
	err = msg.FillInBatchGasFields(func(batchNum uint64) ([]byte, error) {
		data, _, err := t.txStreamer.inboxReader.GetSequencerMessageBytes(ctx, batchNum)
		return data, err
	})
	return msg, acc, blockNum, err
}

// does not return message, so does not need to fill in batchGasFields
func (t *InboxTracker) GetParentChainBlockNumberFor(ctx context.Context, seqNum uint64) (uint64, error) {
	_, _, blockNum, err := t.getRawDelayedMessageAccumulatorAndParentChainBlockNumber(ctx, seqNum)
	return blockNum, err
}

// this function will not error
func (t *InboxTracker) getRawDelayedMessageAccumulatorAndParentChainBlockNumber(ctx context.Context, seqNum uint64) (*arbostypes.L1IncomingMessage, common.Hash, uint64, error) {
	delayedMessageKey := dbKey(rlpDelayedMessagePrefix, seqNum)
	exists, err := t.db.Has(delayedMessageKey)
	if err != nil {
		return nil, common.Hash{}, 0, err
	}
	if !exists {
		msg, acc, err := t.legacyGetDelayedMessageAndAccumulator(ctx, seqNum)
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

func (t *InboxTracker) GetDelayedMessage(ctx context.Context, seqNum uint64) (*arbostypes.L1IncomingMessage, error) {
	msg, _, _, err := t.GetDelayedMessageAccumulatorAndParentChainBlockNumber(ctx, seqNum)
	return msg, err
}

func (t *InboxTracker) GetDelayedMessageBytes(ctx context.Context, seqNum uint64) ([]byte, error) {
	msg, err := t.GetDelayedMessage(ctx, seqNum)
	if err != nil {
		return nil, err
	}
	return msg.Serialize()
}

// AddDelayedMessages adds delayed messages to the database using the provided batch.
// It does not commit the batch - the caller is responsible for committing.
// Returns a validator reorg callback that should be executed after successful commit.
func (t *InboxTracker) AddDelayedMessages(batch ethdb.Batch, messages []*DelayedInboxMessage) (validatorReorgFunc, map[uint64]common.Hash, error) {
	var nextAcc common.Hash
	firstDelayedMsgToKeep := uint64(0)
	uncommittedAccs := make(map[uint64]common.Hash)
	if len(messages) == 0 {
		return nil, uncommittedAccs, nil
	}
	pos, err := messages[0].Message.Header.SeqNum()
	if err != nil {
		return nil, uncommittedAccs, err
	}
	if t.snapSyncConfig.Enabled && pos < t.snapSyncConfig.DelayedCount {
		firstDelayedMsgToKeep = t.snapSyncConfig.DelayedCount
		if firstDelayedMsgToKeep > 0 {
			firstDelayedMsgToKeep--
		}
		for {
			if len(messages) == 0 {
				return nil, uncommittedAccs, nil
			}
			pos, err = messages[0].Message.Header.SeqNum()
			if err != nil {
				return nil, uncommittedAccs, err
			}
			if pos+1 == firstDelayedMsgToKeep {
				nextAcc = messages[0].AfterInboxAcc()
			}
			if pos < firstDelayedMsgToKeep {
				messages = messages[1:]
			} else {
				break
			}
		}
	}
	t.mutex.Lock()
	defer t.mutex.Unlock()

	// This math is safe to do as we know len(messages) > 0
	haveLastAcc, err := t.GetDelayedAcc(pos + uint64(len(messages)) - 1)
	if err == nil {
		if haveLastAcc == messages[len(messages)-1].AfterInboxAcc() {
			// We already have these delayed messages
			return nil, uncommittedAccs, nil
		}
	} else if !errors.Is(err, AccumulatorNotFoundErr) {
		return nil, uncommittedAccs, err
	}

	if pos > firstDelayedMsgToKeep {
		var err error
		nextAcc, err = t.GetDelayedAcc(pos - 1)
		if err != nil {
			if errors.Is(err, AccumulatorNotFoundErr) {
				return nil, uncommittedAccs, errors.New("missing previous delayed message")
			}
			return nil, uncommittedAccs, err
		}
	}

	firstPos := pos
	for _, message := range messages {
		seqNum, err := message.Message.Header.SeqNum()
		if err != nil {
			return nil, uncommittedAccs, err
		}

		if seqNum != pos {
			return nil, uncommittedAccs, fmt.Errorf("unexpected delayed sequence number %v, expected %v", seqNum, pos)
		}

		if nextAcc != message.BeforeInboxAcc {
			return nil, uncommittedAccs, fmt.Errorf("previous delayed accumulator mismatch for message %v", seqNum)
		}
		nextAcc = message.AfterInboxAcc()

		if firstPos == pos {
			// Check if this message is a duplicate
			haveAcc, err := t.GetDelayedAcc(seqNum)
			if err == nil {
				if haveAcc == nextAcc {
					// Skip this message, as we already have it in our database
					pos++
					firstPos++
					messages = messages[1:]
					continue
				}
			} else if !errors.Is(err, AccumulatorNotFoundErr) {
				return nil, uncommittedAccs, err
			}
		}

		delayedMsgKey := dbKey(rlpDelayedMessagePrefix, seqNum)

		msgData, err := rlp.EncodeToBytes(message.Message)
		if err != nil {
			return nil, uncommittedAccs, err
		}
		data := nextAcc.Bytes()
		data = append(data, msgData...)
		err = batch.Put(delayedMsgKey, data)
		if err != nil {
			return nil, uncommittedAccs, err
		}

		if message.ParentChainBlockNumber != message.Message.Header.BlockNumber {
			parentChainBlockNumberKey := dbKey(parentChainBlockNumberPrefix, seqNum)
			parentChainBlockNumberByte := make([]byte, 8)
			binary.BigEndian.PutUint64(parentChainBlockNumberByte, message.ParentChainBlockNumber)
			err = batch.Put(parentChainBlockNumberKey, parentChainBlockNumberByte)
			if err != nil {
				return nil, uncommittedAccs, err
			}
		}

		// Track the accumulator for this uncommitted delayed message
		uncommittedAccs[seqNum] = nextAcc

		pos++
	}

	validatorReorg, err := t.setDelayedCountReorgToBatch(batch, firstPos, pos, true, true)
	if err != nil {
		return nil, uncommittedAccs, err
	}
	return validatorReorg, uncommittedAccs, nil
}

// validatorReorgFunc is a callback to execute validator reorg after batch commit
type validatorReorgFunc func()

// BatchSideEffects holds callbacks for side effects that should be executed after batch commit
type BatchSideEffects struct {
	ValidatorReorg   validatorReorgFunc
	CacheUpdate      func()
	BroadcastConfirm func()
}

// setDelayedCountReorgToBatch is like setDelayedCountReorgAndWriteBatch but doesn't commit the batch.
// It returns a callback function for validator reorg side effects that should be executed after commit.
// Requires the mutex is held. Sets the delayed count and performs any sequencer batch reorg necessary.
// Also deletes any future delayed messages.
func (t *InboxTracker) setDelayedCountReorgToBatch(batch ethdb.Batch, firstNewDelayedMessagePos uint64, newDelayedCount uint64, canReorgBatches bool, deferCommit bool) (validatorReorgFunc, error) {
	if firstNewDelayedMessagePos > newDelayedCount {
		return nil, fmt.Errorf("firstNewDelayedMessagePos %v is after newDelayedCount %v", firstNewDelayedMessagePos, newDelayedCount)
	}
	err := deleteStartingAt(t.db, batch, rlpDelayedMessagePrefix, uint64ToKey(newDelayedCount))
	if err != nil {
		return nil, err
	}
	err = deleteStartingAt(t.db, batch, parentChainBlockNumberPrefix, uint64ToKey(newDelayedCount))
	if err != nil {
		return nil, err
	}
	err = deleteStartingAt(t.db, batch, legacyDelayedMessagePrefix, uint64ToKey(newDelayedCount))
	if err != nil {
		return nil, err
	}

	countData, err := rlp.EncodeToBytes(newDelayedCount)
	if err != nil {
		return nil, err
	}
	err = batch.Put(delayedMessageCountKey, countData)
	if err != nil {
		return nil, err
	}

	seqBatchIter := t.db.NewIterator(delayedSequencedPrefix, uint64ToKey(firstNewDelayedMessagePos+1))
	defer seqBatchIter.Release()
	var reorgSeqBatchesToCount *uint64
	for seqBatchIter.Next() {
		var batchSeqNum uint64
		if err := rlp.DecodeBytes(seqBatchIter.Value(), &batchSeqNum); err != nil {
			return nil, err
		}
		if !canReorgBatches {
			return nil, fmt.Errorf("reorging of sequencer batch number %v via delayed messages reorg to count %v disabled in this instance", batchSeqNum, newDelayedCount)
		}
		if err := batch.Delete(seqBatchIter.Key()); err != nil {
			return nil, err
		}
		if reorgSeqBatchesToCount == nil {
			// Set the count to the first deleted sequence number.
			// E.g. if the deleted sequence number is 1, set the count to 1,
			// meaning that the last and only batch is at sequence number 0.
			reorgSeqBatchesToCount = &batchSeqNum
		}
	}
	if err := seqBatchIter.Error(); err != nil {
		return nil, err
	}
	// Release the iterator early.
	// It's fine to call Release multiple times,
	// which we'll do because of the defer.
	seqBatchIter.Release()
	if reorgSeqBatchesToCount == nil {
		if !deferCommit {
			return nil, batch.Write()
		}
		return nil, nil
	}

	count := *reorgSeqBatchesToCount

	// Capture validator reorg as side effect
	validatorReorg := func() {
		if t.validator != nil {
			t.validator.ReorgToBatchCount(count)
		}
	}

	countData, err = rlp.EncodeToBytes(count)
	if err != nil {
		return nil, err
	}
	if err := batch.Put(sequencerBatchCountKey, countData); err != nil {
		return nil, err
	}
	log.Warn("InboxTracker delayed message reorg is causing a sequencer batch reorg", "sequencerBatchCount", count, "delayedCount", newDelayedCount)

	if err := t.deleteBatchMetadataStartingAt(batch, count); err != nil {
		return nil, err
	}
	var prevMessageCount arbutil.MessageIndex
	if count > 0 {
		prevMessageCount, err = t.GetBatchMessageCount(count - 1)
		if err != nil {
			return nil, err
		}
	}

	if deferCommit {
		// Prepare reorg but don't commit - caller will commit
		// Don't call addMessagesAndReorg if prevMessageCount == 0 as it cannot reorg out the init message
		if prevMessageCount > 0 {
			err = t.txStreamer.addMessagesAndReorg(batch, prevMessageCount, nil)
			if err != nil {
				return nil, err
			}
		}
		return validatorReorg, nil
	}

	// Original behavior: commit via ReorgAtAndEndBatch
	return validatorReorg, t.txStreamer.ReorgAtAndEndBatch(batch, prevMessageCount)
}

// All-in-one delayed message count adjuster. Can go forwards or backwards.
// Requires the mutex is held. Sets the delayed count and performs any sequencer batch reorg necessary.
// Also deletes any future delayed messages.
// This is the legacy version that commits the batch immediately.
func (t *InboxTracker) setDelayedCountReorgAndWriteBatch(batch ethdb.Batch, firstNewDelayedMessagePos uint64, newDelayedCount uint64, canReorgBatches bool) error {
	validatorReorg, err := t.setDelayedCountReorgToBatch(batch, firstNewDelayedMessagePos, newDelayedCount, canReorgBatches, false)
	if err != nil {
		return err
	}
	// Execute validator reorg side effect if needed
	if validatorReorg != nil {
		validatorReorg()
	}
	return nil
}

type multiplexerBackend struct {
	batchSeqNum           uint64
	batches               []*SequencerInboxBatch
	positionWithinMessage uint64

	ctx                        context.Context
	client                     *ethclient.Client
	inbox                      *InboxTracker
	uncommittedDelayedMessages []*DelayedInboxMessage // For reading uncommitted delayed messages
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

	// First check uncommitted delayed messages
	for _, dm := range b.uncommittedDelayedMessages {
		msgSeqNum, err := dm.Message.Header.SeqNum()
		if err != nil {
			return nil, err
		}
		if msgSeqNum == seqNum {
			return dm.Message, nil
		}
	}

	// Fall back to database
	return b.inbox.GetDelayedMessage(b.ctx, seqNum)
}

var delayedMessagesMismatch = errors.New("sequencer batch delayed messages missing or different")

// AddSequencerBatches adds sequencer batches to the database using the provided batch.
// It does not commit the batch - the caller is responsible for committing.
// The uncommittedDelayedMessages contains delayed messages that were written to the batch but not yet committed.
// The uncommittedDelayedAccs map contains accumulators for those messages, allowing validation against uncommitted data.
// Returns side effect callbacks that should be executed after successful commit.
func (t *InboxTracker) AddSequencerBatches(batch ethdb.Batch, ctx context.Context, client *ethclient.Client, batches []*SequencerInboxBatch, uncommittedDelayedMessages []*DelayedInboxMessage, uncommittedDelayedAccs map[uint64]common.Hash) (*BatchSideEffects, error) {
	var nextAcc common.Hash
	var prevbatchmeta BatchMetadata
	sequenceNumberToKeep := uint64(0)
	if len(batches) == 0 {
		return nil, nil
	}
	if t.snapSyncConfig.Enabled && batches[0].SequenceNumber < t.snapSyncConfig.BatchCount {
		sequenceNumberToKeep = t.snapSyncConfig.BatchCount
		if sequenceNumberToKeep > 0 {
			sequenceNumberToKeep--
		}
		for {
			if len(batches) == 0 {
				return nil, nil
			}
			if batches[0].SequenceNumber+1 == sequenceNumberToKeep {
				nextAcc = batches[0].AfterInboxAcc
				prevbatchmeta = BatchMetadata{
					Accumulator:         batches[0].AfterInboxAcc,
					DelayedMessageCount: batches[0].AfterDelayedCount,
					MessageCount:        arbutil.MessageIndex(t.snapSyncConfig.PrevBatchMessageCount),
					ParentChainBlock:    batches[0].ParentChainBlockNumber,
				}
			}
			if batches[0].SequenceNumber < sequenceNumberToKeep {
				batches = batches[1:]
			} else {
				break
			}
		}
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
			return nil, errors.New("missing previous sequencer batch")
		} else if err != nil {
			return nil, err
		}
	}

	err := deleteStartingAt(t.db, batch, delayedSequencedPrefix, uint64ToKey(prevbatchmeta.DelayedMessageCount+1))
	if err != nil {
		return nil, err
	}

	for _, batchItem := range batches {
		if batchItem.SequenceNumber != pos {
			return nil, fmt.Errorf("unexpected batch sequence number %v expected %v", batchItem.SequenceNumber, pos)
		}
		if nextAcc != batchItem.BeforeInboxAcc {
			return nil, fmt.Errorf("previous batch accumulator %v mismatch expected %v", batchItem.BeforeInboxAcc, nextAcc)
		}

		if batchItem.AfterDelayedCount > 0 {
			delayedSeqNum := batchItem.AfterDelayedCount - 1

			// First check if this delayed message is in the uncommitted batch
			haveDelayedAcc, inUncommitted := uncommittedDelayedAccs[delayedSeqNum]
			var notFound bool

			if !inUncommitted {
				// Not in uncommitted data, check the database
				var err error
				haveDelayedAcc, err = t.GetDelayedAcc(delayedSeqNum)
				notFound = errors.Is(err, AccumulatorNotFoundErr)
				if err != nil && !notFound {
					return nil, err
				}
			}

			if (!inUncommitted && notFound) || haveDelayedAcc != batchItem.AfterDelayedAcc {
				log.Debug(
					"Delayed message accumulator doesn't match sequencer batch",
					"batch", batchItem.SequenceNumber,
					"delayedPosition", delayedSeqNum,
					"haveDelayedAcc", haveDelayedAcc,
					"batchDelayedAcc", batchItem.AfterDelayedAcc,
					"inUncommitted", inUncommitted,
					"notFound", notFound,
				)
				// We somehow missed a delayed message reorg; go back and look for it
				return nil, delayedMessagesMismatch
			}
		}

		nextAcc = batchItem.AfterInboxAcc
		pos++
	}

	var messages []arbostypes.MessageWithMetadata
	backend := &multiplexerBackend{
		batchSeqNum: batches[0].SequenceNumber,
		batches:     batches,

		inbox:                      t,
		ctx:                        ctx,
		client:                     client,
		uncommittedDelayedMessages: uncommittedDelayedMessages,
	}
	multiplexer := arbstate.NewInboxMultiplexer(backend, prevbatchmeta.DelayedMessageCount, t.dapReaders, daprovider.KeysetValidate)
	batchMessageCounts := make(map[uint64]arbutil.MessageIndex)
	currentPos := prevbatchmeta.MessageCount + 1
	for {
		if len(backend.batches) == 0 {
			break
		}
		batchSeqNum := backend.batches[0].SequenceNumber
		msg, err := multiplexer.Pop(ctx)
		if err != nil {
			return nil, err
		}
		messages = append(messages, *msg)
		batchMessageCounts[batchSeqNum] = currentPos
		currentPos += 1
	}

	lastBatchMeta := prevbatchmeta
	batchMetas := make(map[uint64]BatchMetadata, len(batches))
	for _, batchItem := range batches {
		meta := BatchMetadata{
			Accumulator:         batchItem.AfterInboxAcc,
			DelayedMessageCount: batchItem.AfterDelayedCount,
			MessageCount:        batchMessageCounts[batchItem.SequenceNumber],
			ParentChainBlock:    batchItem.ParentChainBlockNumber,
		}
		batchMetas[batchItem.SequenceNumber] = meta
		metaBytes, err := rlp.EncodeToBytes(meta)
		if err != nil {
			return nil, err
		}
		err = batch.Put(dbKey(sequencerBatchMetaPrefix, batchItem.SequenceNumber), metaBytes)
		if err != nil {
			return nil, err
		}

		seqNumData, err := rlp.EncodeToBytes(batchItem.SequenceNumber)
		if err != nil {
			return nil, err
		}
		if batchItem.AfterDelayedCount < lastBatchMeta.DelayedMessageCount {
			return nil, errors.New("batch delayed message count went backwards")
		}
		if batchItem.AfterDelayedCount > lastBatchMeta.DelayedMessageCount {
			err = batch.Put(dbKey(delayedSequencedPrefix, batchItem.AfterDelayedCount), seqNumData)
			if err != nil {
				return nil, err
			}
		}
		lastBatchMeta = meta
	}

	err = t.deleteBatchMetadataStartingAt(batch, pos)
	if err != nil {
		return nil, err
	}
	countData, err := rlp.EncodeToBytes(pos)
	if err != nil {
		return nil, err
	}
	err = batch.Put(sequencerBatchCountKey, countData)
	if err != nil {
		return nil, err
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
	// #nosec G115
	log.Info(
		"InboxTracker",
		"sequencerBatchCount", pos,
		"messageCount", newMessageCount,
		"l1Block", latestL1Block,
		"l1Timestamp", time.Unix(int64(latestTimestamp), 0),
	)
	// #nosec G115
	inboxLatestBatchGauge.Update(int64(pos))
	// #nosec G115
	inboxLatestBatchMessageGauge.Update(int64(newMessageCount))

	// Capture side effects as closures - execute these AFTER batch commit succeeds
	sideEffects := &BatchSideEffects{
		ValidatorReorg: func() {
			if t.validator != nil {
				t.validator.ReorgToBatchCount(startPos)
			}
		},
		CacheUpdate: func() {
			t.batchMetaMutex.Lock()
			for seqNum, meta := range batchMetas {
				t.batchMeta.Add(seqNum, meta)
			}
			t.batchMetaMutex.Unlock()
		},
		BroadcastConfirm: func() {
			if t.txStreamer.broadcastServer != nil && pos > 1 {
				prevprevbatchmeta, err := t.GetBatchMetadata(pos - 2)
				if errors.Is(err, AccumulatorNotFoundErr) {
					log.Error("missing previous previous sequencer batch during broadcast confirm")
					return
				}
				if err != nil {
					log.Error("failed to get previous previous batch metadata", "err", err)
					return
				}
				if prevprevbatchmeta.MessageCount > 0 {
					// Confirm messages from batch before last batch
					t.txStreamer.broadcastServer.Confirm(prevprevbatchmeta.MessageCount - 1)
				}
			}
		},
	}

	// This also writes to the batch but does NOT commit
	err = t.txStreamer.AddMessagesAndEndBatchWithDeferredCommit(prevbatchmeta.MessageCount, true, messages, nil, batch, true)
	if err != nil {
		return nil, err
	}

	return sideEffects, nil
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
	}
	if currentCount < count {
		return errors.New("attempted to reorg to future delayed count")
	}

	return t.setDelayedCountReorgAndWriteBatch(t.db.NewBatch(), count, count, false)
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
	return t.txStreamer.ReorgAtAndEndBatch(dbBatch, prevBatchMeta.MessageCount)
}
