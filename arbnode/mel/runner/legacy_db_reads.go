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
	msg, _, err := legacyGetDelayedMessageFromRlpPrefix(db, index)
	if err != nil {
		if !rawdb.IsDbErrNotFound(err) {
			return nil, 0, err
		}
		// Fall back to legacy "d" prefix
		msg, _, err = legacyGetDelayedMessageFromLegacyPrefix(db, index)
		if err != nil {
			return nil, 0, fmt.Errorf("delayed message at index %d not found under either prefix: %w", index, err)
		}
		// Legacy "d" prefix does not store parent chain block number separately
		return msg, msg.Header.BlockNumber, nil
	}
	parentChainBlockNumber, err := legacyGetParentChainBlockNumber(db, index)
	if err != nil {
		if !rawdb.IsDbErrNotFound(err) {
			return nil, 0, err
		}
		return msg, msg.Header.BlockNumber, nil
	}
	return msg, parentChainBlockNumber, nil
}

// legacyGetDelayedMessageFromRlpPrefix reads from RlpDelayedMessagePrefix ("e").
// Format: [32-byte AfterInboxAcc | RLP(L1IncomingMessage)]
// Returns the decoded message and the AfterInboxAcc stored alongside it.
func legacyGetDelayedMessageFromRlpPrefix(db ethdb.KeyValueStore, index uint64) (*arbostypes.L1IncomingMessage, common.Hash, error) {
	key := read.Key(schema.RlpDelayedMessagePrefix, index)
	data, err := db.Get(key)
	if err != nil {
		return nil, common.Hash{}, err
	}
	if len(data) < 32 {
		return nil, common.Hash{}, errors.New("delayed message RLP entry missing accumulator")
	}
	var acc common.Hash
	copy(acc[:], data[:32])
	var msg *arbostypes.L1IncomingMessage
	if err := rlp.DecodeBytes(data[32:], &msg); err != nil {
		return nil, common.Hash{}, fmt.Errorf("error decoding RLP delayed message at index %d: %w", index, err)
	}
	return msg, acc, nil
}

// legacyGetDelayedMessageFromLegacyPrefix reads from LegacyDelayedMessagePrefix ("d").
// Format: [32-byte AfterInboxAcc | L1-serialized message]
// Returns the decoded message and the AfterInboxAcc stored alongside it.
func legacyGetDelayedMessageFromLegacyPrefix(db ethdb.KeyValueStore, index uint64) (*arbostypes.L1IncomingMessage, common.Hash, error) {
	key := read.Key(schema.LegacyDelayedMessagePrefix, index)
	data, err := db.Get(key)
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
		return nil, common.Hash{}, fmt.Errorf("error parsing legacy delayed message at index %d: %w", index, err)
	}
	return msg, acc, nil
}

// legacyGetParentChainBlockNumber reads the parent chain block number stored under
// ParentChainBlockNumberPrefix ("p") for a given delayed message index.
func legacyGetParentChainBlockNumber(db ethdb.KeyValueStore, index uint64) (uint64, error) {
	key := read.Key(schema.ParentChainBlockNumberPrefix, index)
	data, err := db.Get(key)
	if err != nil {
		return 0, err
	}
	if len(data) < 8 {
		return 0, fmt.Errorf("parent chain block number data too short for index %d", index)
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
	key := read.Key(schema.RlpDelayedMessagePrefix, seqNum)
	has, err := db.Has(key)
	if err != nil {
		return common.Hash{}, err
	}
	if !has {
		key = read.Key(schema.LegacyDelayedMessagePrefix, seqNum)
		has, err = db.Has(key)
		if err != nil {
			return common.Hash{}, err
		}
		if !has {
			return common.Hash{}, fmt.Errorf("delayed accumulator not found for index %d", seqNum)
		}
	}
	data, err := db.Get(key)
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

// legacyFindBatchCountAtBlock finds the number of batches posted at or before
// the given parent chain block number by scanning backwards from totalBatchCount.
func legacyFindBatchCountAtBlock(db ethdb.KeyValueStore, totalBatchCount uint64, blockNum uint64) (uint64, error) {
	for i := totalBatchCount; i > 0; i-- {
		meta, err := legacyFetchBatchMetadata(db, i-1)
		if err != nil {
			return 0, fmt.Errorf("failed to read batch metadata %d: %w", i-1, err)
		}
		if meta.ParentChainBlock <= blockNum {
			return i, nil
		}
	}
	return 0, nil
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
		return nil, fmt.Errorf("failed to read legacy batch count: %w", err)
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
		Version:                            0,
		BatchPostingTargetAddress:          sequencerInbox,
		DelayedMessagePostingTargetAddress: bridgeAddr,
		ParentChainId:                      parentChainId,
		ParentChainBlockNumber:             startBlockNum,
		ParentChainBlockHash:               blockHash,
		ParentChainPreviousBlockHash:       parentHash,
		DelayedMessagesSeen:                delayedRead, // will be incremented during accumulation
		DelayedMessagesRead:                delayedRead,
		MsgCount:                           msgCount,
		BatchCount:                         batchCount,
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
		state.DelayedMessagesSeen++
	}

	return state, nil
}
