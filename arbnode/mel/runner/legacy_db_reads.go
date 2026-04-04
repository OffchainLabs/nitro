// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package melrunner

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/offchainlabs/nitro/arbnode/db/read"
	"github.com/offchainlabs/nitro/arbnode/db/schema"
	"github.com/offchainlabs/nitro/arbnode/mel"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
)

// legacyFetchDelayedMessage reads a delayed message stored under pre-MEL schema keys.
// It tries RlpDelayedMessagePrefix ("e") first, then falls back to LegacyDelayedMessagePrefix ("d").
// Returns a fully populated DelayedInboxMessage including BeforeInboxAcc (obtained from the
// previous message's AfterInboxAcc, or zero hash for index 0). BlockHash is left as zero.
func legacyFetchDelayedMessage(db ethdb.KeyValueStore, index uint64) (*mel.DelayedInboxMessage, error) {
	msg, parentChainBlockNumber, err := legacyGetDelayedMessageAndParentChainBlockNumber(db, index)
	if err != nil {
		return nil, err
	}
	// BeforeInboxAcc is the AfterInboxAcc of the previous message (zero hash for index 0)
	var beforeInboxAcc common.Hash
	if index > 0 {
		beforeInboxAcc, err = legacyGetDelayedAcc(db, index-1)
		if err != nil {
			return nil, fmt.Errorf("failed to get BeforeInboxAcc for delayed message %d: %w", index, err)
		}
	}
	return &mel.DelayedInboxMessage{
		BeforeInboxAcc:         beforeInboxAcc,
		Message:                msg,
		ParentChainBlockNumber: parentChainBlockNumber,
	}, nil
}

// legacyGetDelayedMessageAndParentChainBlockNumber reads a delayed message and its parent chain
// block number from pre-MEL schema keys. Mirrors InboxTracker.getRawDelayedMessageAccumulatorAndParentChainBlockNumber.
func legacyGetDelayedMessageAndParentChainBlockNumber(db ethdb.KeyValueStore, index uint64) (*arbostypes.L1IncomingMessage, uint64, error) {
	data, isRlp, err := legacyReadRawFromEitherPrefix(db, index)
	if err != nil {
		return nil, 0, fmt.Errorf("delayed message at index %d not found under either prefix: %w", index, err)
	}
	_, payload, err := splitAccAndPayload(data)
	if err != nil {
		return nil, 0, err
	}
	msg, err := legacyDecodeDelayedMessage(payload, isRlp, index)
	if err != nil {
		return nil, 0, err
	}
	if msg.Header == nil {
		return nil, 0, fmt.Errorf("decoded delayed message at index %d has nil header", index)
	}
	if !isRlp {
		// Legacy "d" prefix does not store parent chain block number separately
		return msg, msg.Header.BlockNumber, nil
	}
	parentChainBlockNumber, err := legacyGetParentChainBlockNumber(db, index)
	if err != nil {
		if !rawdb.IsDbErrNotFound(err) {
			return nil, 0, fmt.Errorf("error reading parent chain block number for delayed message %d: %w", index, err)
		}
		log.Warn("Legacy parent chain block number not found, falling back to header block number; this may indicate data inconsistency", "index", index, "headerBlockNumber", msg.Header.BlockNumber)
		return msg, msg.Header.BlockNumber, nil
	}
	return msg, parentChainBlockNumber, nil
}

// splitAccAndPayload extracts the 32-byte AfterInboxAcc prefix and remaining
// payload from a legacy delayed message DB entry.
func splitAccAndPayload(data []byte) (common.Hash, []byte, error) {
	if len(data) < 32 {
		return common.Hash{}, nil, errors.New("delayed message entry missing accumulator")
	}
	return common.BytesToHash(data[:32]), data[32:], nil
}

// legacyReadRawFromEitherPrefix reads the raw bytes for a delayed message index
// from RlpDelayedMessagePrefix ("e") first, falling back to LegacyDelayedMessagePrefix ("d").
// Returns the raw data and which prefix was used (true = RLP prefix, false = legacy prefix).
func legacyReadRawFromEitherPrefix(db ethdb.KeyValueStore, index uint64) ([]byte, bool, error) {
	data, err := db.Get(read.Key(schema.RlpDelayedMessagePrefix, index))
	if err == nil {
		return data, true, nil
	}
	if !rawdb.IsDbErrNotFound(err) {
		return nil, false, err
	}
	data, err = db.Get(read.Key(schema.LegacyDelayedMessagePrefix, index))
	if err != nil {
		return nil, false, err
	}
	return data, false, nil
}

// legacyDecodeDelayedMessage decodes raw data (after the 32-byte acc prefix) into
// an L1IncomingMessage. isRlp controls whether the payload is RLP-encoded or L1-serialized.
func legacyDecodeDelayedMessage(payload []byte, isRlp bool, index uint64) (*arbostypes.L1IncomingMessage, error) {
	if isRlp {
		var msg *arbostypes.L1IncomingMessage
		if err := rlp.DecodeBytes(payload, &msg); err != nil {
			return nil, fmt.Errorf("error decoding RLP delayed message at index %d: %w", index, err)
		}
		return msg, nil
	}
	msg, err := arbostypes.ParseIncomingL1Message(bytes.NewReader(payload), nil)
	if err != nil {
		return nil, fmt.Errorf("error parsing legacy delayed message at index %d: %w", index, err)
	}
	return msg, nil
}

// legacyGetParentChainBlockNumber reads the parent chain block number stored under
// ParentChainBlockNumberPrefix ("p") for a given delayed message index.
func legacyGetParentChainBlockNumber(db ethdb.KeyValueStore, index uint64) (uint64, error) {
	key := read.Key(schema.ParentChainBlockNumberPrefix, index)
	data, err := db.Get(key)
	if err != nil {
		return 0, err
	}
	if len(data) != 8 {
		return 0, fmt.Errorf("parent chain block number data has unexpected length %d for index %d, expected 8", len(data), index)
	}
	return binary.BigEndian.Uint64(data), nil
}

// legacyFetchBatchMetadata reads batch metadata stored under pre-MEL SequencerBatchMetaPrefix ("s").
func legacyFetchBatchMetadata(db ethdb.KeyValueStore, seqNum uint64) (*mel.BatchMetadata, error) {
	batchMetadata, err := read.BatchMetadata(db, seqNum)
	if err != nil {
		return nil, err
	}
	return &batchMetadata, nil
}

// legacyGetDelayedAcc reads the delayed message accumulator (AfterInboxAcc) from pre-MEL keys.
// Tries RlpDelayedMessagePrefix ("e") first, then LegacyDelayedMessagePrefix ("d").
func legacyGetDelayedAcc(db ethdb.KeyValueStore, seqNum uint64) (common.Hash, error) {
	data, _, err := legacyReadRawFromEitherPrefix(db, seqNum)
	if err != nil {
		if rawdb.IsDbErrNotFound(err) {
			return common.Hash{}, fmt.Errorf("%w: delayed accumulator not found for index %d", mel.ErrAccumulatorNotFound, seqNum)
		}
		return common.Hash{}, err
	}
	acc, _, err := splitAccAndPayload(data)
	return acc, err
}

// legacyFindBatchCountAtBlock finds the number of batches posted at or before
// the given parent chain block number using binary search on the monotonically
// non-decreasing ParentChainBlock field.
func legacyFindBatchCountAtBlock(db ethdb.KeyValueStore, totalBatchCount uint64, blockNum uint64) (uint64, error) {
	if totalBatchCount == 0 {
		return 0, nil
	}
	// Check if the last batch is at or before blockNum (common case during migration).
	lastMeta, err := legacyFetchBatchMetadata(db, totalBatchCount-1)
	if err != nil {
		return 0, fmt.Errorf("failed to read batch metadata %d: %w", totalBatchCount-1, err)
	}
	if lastMeta.ParentChainBlock <= blockNum {
		return totalBatchCount, nil
	}
	// Check if even the first batch is after blockNum.
	firstMeta, err := legacyFetchBatchMetadata(db, 0)
	if err != nil {
		return 0, fmt.Errorf("failed to read batch metadata 0: %w", err)
	}
	if firstMeta.ParentChainBlock > blockNum {
		return 0, nil
	}
	// Binary search: find the largest i such that batch[i].ParentChainBlock <= blockNum.
	// Invariant: batch[low].ParentChainBlock <= blockNum < batch[high].ParentChainBlock
	low := uint64(0)
	high := totalBatchCount - 1
	for low < high {
		mid := low + (high-low+1)/2 // Rounds up to avoid infinite loop when high == low+1
		meta, err := legacyFetchBatchMetadata(db, mid)
		if err != nil {
			return 0, fmt.Errorf("failed to read batch metadata %d: %w", mid, err)
		}
		if meta.ParentChainBlock <= blockNum {
			low = mid
		} else {
			high = mid - 1
		}
	}
	return low + 1, nil // batch count = last valid index + 1
}

// CreateInitialMELStateFromLegacyDB constructs an initial MEL state from pre-MEL
// database entries (legacy inbox reader/tracker data). The caller must provide:
//   - startBlockNum: the parent chain block to anchor the initial state (should be finalized)
//   - delayedSeenAtBlock: the on-chain delayed message count at startBlockNum (from bridge contract)
//
// Batch count, msg count, and delayed read are derived from legacy batch metadata.
// The MEL inbox accumulator is reconstructed for any unread delayed messages.
func CreateInitialMELStateFromLegacyDB(
	db ethdb.KeyValueStore,
	sequencerInbox common.Address,
	bridgeAddr common.Address,
	parentChainId uint64,
	fetchBlock func(blockNum uint64) (hash, parentHash common.Hash, err error),
	startBlockNum uint64,
	delayedSeenAtBlock uint64,
) (*mel.State, error) {
	totalBatchCount, err := read.Value[uint64](db, schema.SequencerBatchCountKey)
	if err != nil {
		if rawdb.IsDbErrNotFound(err) {
			totalBatchCount = 0
		} else {
			return nil, fmt.Errorf("failed to read legacy batch count: %w", err)
		}
	}

	// Find batch count at or before the start block
	batchCount, err := legacyFindBatchCountAtBlock(db, totalBatchCount, startBlockNum)
	if err != nil {
		return nil, fmt.Errorf("failed to find batch count at block %d: %w", startBlockNum, err)
	}

	// Determine msgCount and delayedRead from the last batch at/before start block
	var msgCount uint64
	var delayedRead uint64
	if batchCount > 0 {
		batchMeta, err := legacyFetchBatchMetadata(db, batchCount-1)
		if err != nil {
			return nil, fmt.Errorf("failed to read batch metadata %d: %w", batchCount-1, err)
		}
		msgCount = uint64(batchMeta.MessageCount)
		delayedRead = batchMeta.DelayedMessageCount
	}

	blockHash, parentHash, err := fetchBlock(startBlockNum)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch block %d: %w", startBlockNum, err)
	}

	state := &mel.State{
		BatchPostingTargetAddress:          sequencerInbox,
		DelayedMessagePostingTargetAddress: bridgeAddr,
		ParentChainId:                      parentChainId,
		ParentChainBlockNumber:             startBlockNum,
		ParentChainBlockHash:               blockHash,
		ParentChainPreviousBlockHash:       parentHash,
		DelayedMessagesSeen:                delayedRead,
		DelayedMessagesRead:                delayedRead,
		MsgCount:                           msgCount,
		BatchCount:                         batchCount,
	}

	if delayedRead > delayedSeenAtBlock {
		return nil, fmt.Errorf("delayedRead (%d) exceeds delayedSeenAtBlock (%d) at block %d; batch metadata is inconsistent with on-chain delayed count", delayedRead, delayedSeenAtBlock, startBlockNum)
	}

	// Reconstruct MEL inbox accumulator for unread delayed messages
	// (messages seen but not yet consumed by a batch at the start block)
	for i := delayedRead; i < delayedSeenAtBlock; i++ {
		delayedMsg, err := legacyFetchDelayedMessage(db, i)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch legacy delayed message %d: %w", i, err)
		}
		if err := state.AccumulateDelayedMessage(delayedMsg); err != nil {
			return nil, fmt.Errorf("failed to accumulate delayed message %d: %w", i, err)
		}
	}

	return state, nil
}
