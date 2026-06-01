// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package melrunner

import (
	"encoding/binary"
	"errors"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/offchainlabs/nitro/arbnode/db/read"
	"github.com/offchainlabs/nitro/arbnode/db/schema"
	"github.com/offchainlabs/nitro/arbnode/mel"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
)

// --- Test helpers ---

func makeDelayedL1Msg(index uint64) *arbostypes.L1IncomingMessage {
	// #nosec G115
	reqID := common.BigToHash(big.NewInt(int64(index + 1)))
	return &arbostypes.L1IncomingMessage{
		Header: &arbostypes.L1IncomingMessageHeader{
			Kind:        arbostypes.L1MessageType_EndOfBlock,
			Poster:      [20]byte{},
			BlockNumber: index * 10,
			Timestamp:   index * 100,
			RequestId:   &reqID,
			L1BaseFee:   common.Big0,
		},
		L2msg: []byte{},
	}
}

func storeRlpDelayedMessage(t *testing.T, db ethdb.KeyValueStore, index uint64, acc common.Hash, msg *arbostypes.L1IncomingMessage) {
	t.Helper()
	msgBytes, err := rlp.EncodeToBytes(msg)
	require.NoError(t, err)
	data := append(acc.Bytes(), msgBytes...)
	require.NoError(t, db.Put(read.Key(schema.RlpDelayedMessagePrefix, index), data))
}

func storeLegacyDelayedMessage(t *testing.T, db ethdb.KeyValueStore, index uint64, acc common.Hash, msg *arbostypes.L1IncomingMessage) {
	t.Helper()
	msgBytes, err := msg.Serialize()
	require.NoError(t, err)
	data := append(acc.Bytes(), msgBytes...)
	require.NoError(t, db.Put(read.Key(schema.LegacyDelayedMessagePrefix, index), data))
}

func storeParentChainBlockNumber(t *testing.T, db ethdb.KeyValueStore, index uint64, blockNum uint64) {
	t.Helper()
	data := make([]byte, 8)
	binary.BigEndian.PutUint64(data, blockNum)
	require.NoError(t, db.Put(read.Key(schema.ParentChainBlockNumberPrefix, index), data))
}

func storeBatchMetadata(t *testing.T, db ethdb.KeyValueStore, seqNum uint64, meta mel.BatchMetadata) {
	t.Helper()
	data, err := rlp.EncodeToBytes(meta)
	require.NoError(t, err)
	require.NoError(t, db.Put(read.Key(schema.SequencerBatchMetaPrefix, seqNum), data))
}

func storeBatchCount(t *testing.T, db ethdb.KeyValueStore, count uint64) {
	t.Helper()
	data, err := rlp.EncodeToBytes(count)
	require.NoError(t, err)
	require.NoError(t, db.Put(schema.SequencerBatchCountKey, data))
}

// --- Tests ---

func TestLegacyGetDelayedMessageFromRlpPrefix(t *testing.T) {
	t.Parallel()

	t.Run("HappyPath", func(t *testing.T) {
		db := rawdb.NewMemoryDatabase()
		msg := makeDelayedL1Msg(5)
		acc := common.HexToHash("0xdeadbeef")
		storeRlpDelayedMessage(t, db, 5, acc, msg)

		gotMsg, gotAcc, err := legacyGetDelayedMessageFromRlpPrefix(db, 5)
		require.NoError(t, err)
		require.Equal(t, acc, gotAcc)
		require.Equal(t, msg.Header.Kind, gotMsg.Header.Kind)
		require.Equal(t, msg.Header.BlockNumber, gotMsg.Header.BlockNumber)
		require.Equal(t, *msg.Header.RequestId, *gotMsg.Header.RequestId)
	})

	t.Run("NotFound", func(t *testing.T) {
		db := rawdb.NewMemoryDatabase()
		_, _, err := legacyGetDelayedMessageFromRlpPrefix(db, 0)
		require.Error(t, err)
	})

	t.Run("DataTooShort", func(t *testing.T) {
		db := rawdb.NewMemoryDatabase()
		require.NoError(t, db.Put(read.Key(schema.RlpDelayedMessagePrefix, 0), make([]byte, 16)))
		_, _, err := legacyGetDelayedMessageFromRlpPrefix(db, 0)
		require.ErrorContains(t, err, "missing accumulator")
	})

	t.Run("MalformedRLP", func(t *testing.T) {
		db := rawdb.NewMemoryDatabase()
		data := append(make([]byte, 32), 0xFF, 0xFF, 0xFF)
		require.NoError(t, db.Put(read.Key(schema.RlpDelayedMessagePrefix, 0), data))
		_, _, err := legacyGetDelayedMessageFromRlpPrefix(db, 0)
		require.ErrorContains(t, err, "error decoding RLP")
	})
}

func TestLegacyGetDelayedMessageFromLegacyPrefix(t *testing.T) {
	t.Parallel()

	t.Run("HappyPath", func(t *testing.T) {
		db := rawdb.NewMemoryDatabase()
		msg := makeDelayedL1Msg(3)
		acc := common.HexToHash("0xcafebabe")
		storeLegacyDelayedMessage(t, db, 3, acc, msg)

		gotMsg, gotAcc, err := legacyGetDelayedMessageFromLegacyPrefix(db, 3)
		require.NoError(t, err)
		require.Equal(t, acc, gotAcc)
		require.Equal(t, msg.Header.Kind, gotMsg.Header.Kind)
		require.Equal(t, msg.Header.BlockNumber, gotMsg.Header.BlockNumber)
	})

	t.Run("NotFound", func(t *testing.T) {
		db := rawdb.NewMemoryDatabase()
		_, _, err := legacyGetDelayedMessageFromLegacyPrefix(db, 0)
		require.Error(t, err)
	})

	t.Run("DataTooShort", func(t *testing.T) {
		db := rawdb.NewMemoryDatabase()
		require.NoError(t, db.Put(read.Key(schema.LegacyDelayedMessagePrefix, 0), make([]byte, 10)))
		_, _, err := legacyGetDelayedMessageFromLegacyPrefix(db, 0)
		require.ErrorContains(t, err, "missing accumulator")
	})
}

func TestLegacyGetParentChainBlockNumber(t *testing.T) {
	t.Parallel()

	t.Run("HappyPath", func(t *testing.T) {
		db := rawdb.NewMemoryDatabase()
		storeParentChainBlockNumber(t, db, 7, 12345)

		got, err := legacyGetParentChainBlockNumber(db, 7)
		require.NoError(t, err)
		require.Equal(t, uint64(12345), got)
	})

	t.Run("NotFound", func(t *testing.T) {
		db := rawdb.NewMemoryDatabase()
		_, err := legacyGetParentChainBlockNumber(db, 0)
		require.Error(t, err)
	})

	t.Run("DataTooShort", func(t *testing.T) {
		db := rawdb.NewMemoryDatabase()
		require.NoError(t, db.Put(read.Key(schema.ParentChainBlockNumberPrefix, 0), make([]byte, 4)))
		_, err := legacyGetParentChainBlockNumber(db, 0)
		require.ErrorContains(t, err, "too short")
	})
}

func TestLegacyGetDelayedAcc(t *testing.T) {
	t.Parallel()

	t.Run("FoundUnderRlpPrefix", func(t *testing.T) {
		db := rawdb.NewMemoryDatabase()
		msg := makeDelayedL1Msg(0)
		acc := common.HexToHash("0x1111")
		storeRlpDelayedMessage(t, db, 0, acc, msg)

		got, err := legacyGetDelayedAcc(db, 0)
		require.NoError(t, err)
		require.Equal(t, acc, got)
	})

	t.Run("FallbackToLegacyPrefix", func(t *testing.T) {
		db := rawdb.NewMemoryDatabase()
		msg := makeDelayedL1Msg(0)
		acc := common.HexToHash("0x2222")
		storeLegacyDelayedMessage(t, db, 0, acc, msg)

		got, err := legacyGetDelayedAcc(db, 0)
		require.NoError(t, err)
		require.Equal(t, acc, got)
	})

	t.Run("NotFoundUnderEither", func(t *testing.T) {
		db := rawdb.NewMemoryDatabase()
		_, err := legacyGetDelayedAcc(db, 0)
		require.ErrorContains(t, err, "not found for index")
	})
}

func TestLegacyGetDelayedMessageAndParentChainBlockNumber(t *testing.T) {
	t.Parallel()

	t.Run("RlpPrefixWithParentChainBlockNum", func(t *testing.T) {
		db := rawdb.NewMemoryDatabase()
		msg := makeDelayedL1Msg(0)
		msg.Header.BlockNumber = 100
		storeRlpDelayedMessage(t, db, 0, common.Hash{}, msg)
		storeParentChainBlockNumber(t, db, 0, 200)

		gotMsg, gotBlock, err := legacyGetDelayedMessageAndParentChainBlockNumber(db, 0)
		require.NoError(t, err)
		require.Equal(t, uint64(200), gotBlock)
		require.Equal(t, msg.Header.Kind, gotMsg.Header.Kind)
	})

	t.Run("RlpPrefixWithoutParentChainBlockNum", func(t *testing.T) {
		db := rawdb.NewMemoryDatabase()
		msg := makeDelayedL1Msg(0)
		msg.Header.BlockNumber = 100
		storeRlpDelayedMessage(t, db, 0, common.Hash{}, msg)

		_, gotBlock, err := legacyGetDelayedMessageAndParentChainBlockNumber(db, 0)
		require.NoError(t, err)
		require.Equal(t, uint64(100), gotBlock)
	})

	t.Run("FallbackToLegacyPrefix", func(t *testing.T) {
		db := rawdb.NewMemoryDatabase()
		msg := makeDelayedL1Msg(0)
		msg.Header.BlockNumber = 50
		storeLegacyDelayedMessage(t, db, 0, common.Hash{}, msg)

		_, gotBlock, err := legacyGetDelayedMessageAndParentChainBlockNumber(db, 0)
		require.NoError(t, err)
		require.Equal(t, uint64(50), gotBlock)
	})

	t.Run("NotFoundUnderEither", func(t *testing.T) {
		db := rawdb.NewMemoryDatabase()
		_, _, err := legacyGetDelayedMessageAndParentChainBlockNumber(db, 0)
		require.ErrorContains(t, err, "not found under either prefix")
	})
}

func TestLegacyFetchDelayedMessage(t *testing.T) {
	t.Parallel()

	t.Run("IndexZero", func(t *testing.T) {
		db := rawdb.NewMemoryDatabase()
		msg := makeDelayedL1Msg(0)
		acc := common.HexToHash("0xaaaa")
		storeRlpDelayedMessage(t, db, 0, acc, msg)
		storeParentChainBlockNumber(t, db, 0, 42)

		got, err := legacyFetchDelayedMessage(db, 0)
		require.NoError(t, err)
		require.Equal(t, common.Hash{}, got.BeforeInboxAcc)
		require.Equal(t, uint64(42), got.ParentChainBlockNumber)
		require.Equal(t, msg.Header.Kind, got.Message.Header.Kind)
	})

	t.Run("IndexGreaterThanZero", func(t *testing.T) {
		db := rawdb.NewMemoryDatabase()
		acc0 := common.HexToHash("0xbbbb")
		acc1 := common.HexToHash("0xcccc")
		storeRlpDelayedMessage(t, db, 0, acc0, makeDelayedL1Msg(0))
		storeRlpDelayedMessage(t, db, 1, acc1, makeDelayedL1Msg(1))
		storeParentChainBlockNumber(t, db, 1, 99)

		got, err := legacyFetchDelayedMessage(db, 1)
		require.NoError(t, err)
		require.Equal(t, acc0, got.BeforeInboxAcc)
		require.Equal(t, uint64(99), got.ParentChainBlockNumber)
	})

	t.Run("PreviousMessageMissing", func(t *testing.T) {
		db := rawdb.NewMemoryDatabase()
		storeRlpDelayedMessage(t, db, 1, common.Hash{}, makeDelayedL1Msg(1))
		storeParentChainBlockNumber(t, db, 1, 10)

		_, err := legacyFetchDelayedMessage(db, 1)
		require.ErrorContains(t, err, "failed to get BeforeInboxAcc")
	})
}

func TestLegacyFetchBatchMetadata(t *testing.T) {
	t.Parallel()

	t.Run("HappyPath", func(t *testing.T) {
		db := rawdb.NewMemoryDatabase()
		want := mel.BatchMetadata{
			Accumulator:         common.HexToHash("0xdead"),
			MessageCount:        10,
			DelayedMessageCount: 5,
			ParentChainBlock:    100,
		}
		storeBatchMetadata(t, db, 0, want)

		got, err := legacyFetchBatchMetadata(db, 0)
		require.NoError(t, err)
		require.Equal(t, want, *got)
	})

	t.Run("NotFound", func(t *testing.T) {
		db := rawdb.NewMemoryDatabase()
		_, err := legacyFetchBatchMetadata(db, 0)
		require.Error(t, err)
	})
}

func TestLegacyFindBatchCountAtBlock(t *testing.T) {
	t.Parallel()

	setupBatches := func(t *testing.T, db ethdb.KeyValueStore, blocks []uint64) {
		t.Helper()
		for i, block := range blocks {
			// #nosec G115
			storeBatchMetadata(t, db, uint64(i), mel.BatchMetadata{ParentChainBlock: block})
		}
	}

	t.Run("AllBatchesBeforeTarget", func(t *testing.T) {
		db := rawdb.NewMemoryDatabase()
		setupBatches(t, db, []uint64{10, 20, 30})
		got, err := legacyFindBatchCountAtBlock(db, 3, 50)
		require.NoError(t, err)
		require.Equal(t, uint64(3), got)
	})

	t.Run("SomeBatchesAfterTarget", func(t *testing.T) {
		db := rawdb.NewMemoryDatabase()
		setupBatches(t, db, []uint64{10, 20, 30})
		got, err := legacyFindBatchCountAtBlock(db, 3, 25)
		require.NoError(t, err)
		require.Equal(t, uint64(2), got)
	})

	t.Run("NoBatchesBeforeTarget", func(t *testing.T) {
		db := rawdb.NewMemoryDatabase()
		setupBatches(t, db, []uint64{10, 20, 30})
		got, err := legacyFindBatchCountAtBlock(db, 3, 5)
		require.NoError(t, err)
		require.Equal(t, uint64(0), got)
	})

	t.Run("ZeroTotalBatchCount", func(t *testing.T) {
		db := rawdb.NewMemoryDatabase()
		got, err := legacyFindBatchCountAtBlock(db, 0, 100)
		require.NoError(t, err)
		require.Equal(t, uint64(0), got)
	})

	t.Run("ExactBlockMatch", func(t *testing.T) {
		db := rawdb.NewMemoryDatabase()
		setupBatches(t, db, []uint64{10, 20, 30})
		got, err := legacyFindBatchCountAtBlock(db, 3, 20)
		require.NoError(t, err)
		require.Equal(t, uint64(2), got)
	})
}

func TestCreateInitialMELStateFromLegacyDB(t *testing.T) {
	t.Parallel()

	seqInbox := common.HexToAddress("0xAABB")
	bridgeAddr := common.HexToAddress("0xCCDD")
	parentChainId := uint64(1)
	blockHash := common.HexToHash("0x1234")
	parentHash := common.HexToHash("0x5678")
	fetchBlock := func(blockNum uint64) (common.Hash, common.Hash, error) {
		return blockHash, parentHash, nil
	}

	t.Run("ZeroBatchCount", func(t *testing.T) {
		db := rawdb.NewMemoryDatabase()
		storeBatchCount(t, db, 0)

		state, err := CreateInitialMELStateFromLegacyDB(db, seqInbox, bridgeAddr, parentChainId, fetchBlock, 100, 0)
		require.NoError(t, err)
		require.Equal(t, uint64(0), state.BatchCount)
		require.Equal(t, uint64(0), state.MsgCount)
		require.Equal(t, uint64(0), state.DelayedMessagesRead)
		require.Equal(t, uint64(0), state.DelayedMessagesSeen)
		require.Equal(t, common.Hash{}, state.DelayedMessageInboxAcc)
		require.Equal(t, seqInbox, state.BatchPostingTargetAddress)
		require.Equal(t, bridgeAddr, state.DelayedMessagePostingTargetAddress)
		require.Equal(t, parentChainId, state.ParentChainId)
		require.Equal(t, uint64(100), state.ParentChainBlockNumber)
		require.Equal(t, blockHash, state.ParentChainBlockHash)
		require.Equal(t, parentHash, state.ParentChainPreviousBlockHash)
	})

	t.Run("NoUnreadDelayedMessages", func(t *testing.T) {
		db := rawdb.NewMemoryDatabase()
		storeBatchCount(t, db, 2)
		storeBatchMetadata(t, db, 0, mel.BatchMetadata{
			MessageCount:        5,
			DelayedMessageCount: 2,
			ParentChainBlock:    10,
		})
		storeBatchMetadata(t, db, 1, mel.BatchMetadata{
			MessageCount:        10,
			DelayedMessageCount: 3,
			ParentChainBlock:    20,
		})

		state, err := CreateInitialMELStateFromLegacyDB(db, seqInbox, bridgeAddr, parentChainId, fetchBlock, 25, 3)
		require.NoError(t, err)
		require.Equal(t, uint64(2), state.BatchCount)
		require.Equal(t, uint64(10), state.MsgCount)
		require.Equal(t, uint64(3), state.DelayedMessagesRead)
		require.Equal(t, uint64(3), state.DelayedMessagesSeen)
		require.Equal(t, common.Hash{}, state.DelayedMessageInboxAcc)
	})

	t.Run("WithUnreadDelayedMessages", func(t *testing.T) {
		db := rawdb.NewMemoryDatabase()
		storeBatchCount(t, db, 1)
		storeBatchMetadata(t, db, 0, mel.BatchMetadata{
			MessageCount:        5,
			DelayedMessageCount: 1,
			ParentChainBlock:    10,
		})

		// Store delayed messages 1 and 2 under "e" prefix with known accumulators.
		// Message at index 0 has acc0 (used as BeforeInboxAcc for index 1).
		acc0 := common.HexToHash("0xa0a0")
		acc1 := common.HexToHash("0xb1b1")
		acc2 := common.HexToHash("0xc2c2")
		storeRlpDelayedMessage(t, db, 0, acc0, makeDelayedL1Msg(0))
		storeRlpDelayedMessage(t, db, 1, acc1, makeDelayedL1Msg(1))
		storeRlpDelayedMessage(t, db, 2, acc2, makeDelayedL1Msg(2))
		storeParentChainBlockNumber(t, db, 1, 12)
		storeParentChainBlockNumber(t, db, 2, 14)

		state, err := CreateInitialMELStateFromLegacyDB(db, seqInbox, bridgeAddr, parentChainId, fetchBlock, 15, 3)
		require.NoError(t, err)
		require.Equal(t, uint64(1), state.BatchCount)
		require.Equal(t, uint64(5), state.MsgCount)
		require.Equal(t, uint64(1), state.DelayedMessagesRead)
		require.Equal(t, uint64(3), state.DelayedMessagesSeen)
		require.NotEqual(t, common.Hash{}, state.DelayedMessageInboxAcc)
	})

	t.Run("FetchBlockError", func(t *testing.T) {
		db := rawdb.NewMemoryDatabase()
		storeBatchCount(t, db, 0)
		failFetchBlock := func(blockNum uint64) (common.Hash, common.Hash, error) {
			return common.Hash{}, common.Hash{}, errors.New("rpc error")
		}

		_, err := CreateInitialMELStateFromLegacyDB(db, seqInbox, bridgeAddr, parentChainId, failFetchBlock, 100, 0)
		require.ErrorContains(t, err, "failed to fetch block")
	})

	t.Run("MissingBatchCount", func(t *testing.T) {
		db := rawdb.NewMemoryDatabase()
		_, err := CreateInitialMELStateFromLegacyDB(db, seqInbox, bridgeAddr, parentChainId, fetchBlock, 100, 0)
		require.ErrorContains(t, err, "failed to read legacy batch count")
	})
}
