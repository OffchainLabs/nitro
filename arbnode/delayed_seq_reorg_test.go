// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbnode

import (
	"context"
	"encoding/binary"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
)

func TestSequencerReorgFromDelayed(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	exec, streamer, db, _ := NewTransactionStreamerForTest(t, ctx, common.Address{})
	tracker, err := NewInboxTracker(db, streamer, nil, DefaultSnapSyncConfig)
	Require(t, err)

	err = streamer.Start(ctx)
	Require(t, err)
	exec.Start(ctx)
	init, err := streamer.GetMessage(0)
	Require(t, err)

	initMsgDelayed := &DelayedInboxMessage{
		BlockHash:      [32]byte{},
		BeforeInboxAcc: [32]byte{},
		Message:        init.Message,
	}
	delayedRequestId := common.BigToHash(common.Big1)
	userDelayed := &DelayedInboxMessage{
		BlockHash:      [32]byte{},
		BeforeInboxAcc: initMsgDelayed.AfterInboxAcc(),
		Message: &arbostypes.L1IncomingMessage{
			Header: &arbostypes.L1IncomingMessageHeader{
				Kind:        arbostypes.L1MessageType_EndOfBlock,
				Poster:      [20]byte{},
				BlockNumber: 0,
				Timestamp:   0,
				RequestId:   &delayedRequestId,
				L1BaseFee:   common.Big0,
			},
		},
	}
	delayedRequestId2 := common.BigToHash(common.Big2)
	userDelayed2 := &DelayedInboxMessage{
		BlockHash:      [32]byte{},
		BeforeInboxAcc: userDelayed.AfterInboxAcc(),
		Message: &arbostypes.L1IncomingMessage{
			Header: &arbostypes.L1IncomingMessageHeader{
				Kind:        arbostypes.L1MessageType_EndOfBlock,
				Poster:      [20]byte{},
				BlockNumber: 0,
				Timestamp:   0,
				RequestId:   &delayedRequestId2,
				L1BaseFee:   common.Big0,
			},
		},
	}
	err = tracker.AddDelayedMessages([]*DelayedInboxMessage{initMsgDelayed, userDelayed, userDelayed2})
	Require(t, err)

	serializedInitMsgBatch := make([]byte, 40)
	binary.BigEndian.PutUint64(serializedInitMsgBatch[32:], 1)
	initMsgBatch := &SequencerInboxBatch{
		BlockHash:              [32]byte{},
		ParentChainBlockNumber: 0,
		SequenceNumber:         0,
		BeforeInboxAcc:         [32]byte{},
		AfterInboxAcc:          [32]byte{1},
		AfterDelayedAcc:        initMsgDelayed.AfterInboxAcc(),
		AfterDelayedCount:      1,
		TimeBounds:             bridgegen.IBridgeTimeBounds{},
		RawLog:                 types.Log{},
		DataLocation:           0,
		BridgeAddress:          [20]byte{},
		Serialized:             serializedInitMsgBatch,
	}
	serializedUserMsgBatch := make([]byte, 40)
	binary.BigEndian.PutUint64(serializedUserMsgBatch[32:], 2)
	userMsgBatch := &SequencerInboxBatch{
		BlockHash:              [32]byte{},
		ParentChainBlockNumber: 0,
		SequenceNumber:         1,
		BeforeInboxAcc:         [32]byte{1},
		AfterInboxAcc:          [32]byte{2},
		AfterDelayedAcc:        userDelayed2.AfterInboxAcc(),
		AfterDelayedCount:      3,
		TimeBounds:             bridgegen.IBridgeTimeBounds{},
		RawLog:                 types.Log{},
		DataLocation:           0,
		BridgeAddress:          [20]byte{},
		Serialized:             serializedUserMsgBatch,
	}
	emptyBatch := &SequencerInboxBatch{
		BlockHash:              [32]byte{},
		ParentChainBlockNumber: 0,
		SequenceNumber:         2,
		BeforeInboxAcc:         [32]byte{2},
		AfterInboxAcc:          [32]byte{3},
		AfterDelayedAcc:        userDelayed2.AfterInboxAcc(),
		AfterDelayedCount:      3,
		TimeBounds:             bridgegen.IBridgeTimeBounds{},
		RawLog:                 types.Log{},
		DataLocation:           0,
		BridgeAddress:          [20]byte{},
		Serialized:             serializedUserMsgBatch,
	}
	err = tracker.AddSequencerBatches(ctx, nil, []*SequencerInboxBatch{initMsgBatch, userMsgBatch, emptyBatch})
	Require(t, err)

	msgCount, err := streamer.GetMessageCount()
	Require(t, err)
	if msgCount != 3 {
		Fail(t, "Unexpected tx streamer message count", msgCount, "(expected 3)")
	}

	delayedCount, err := tracker.GetDelayedCount()
	Require(t, err)
	if delayedCount != 3 {
		Fail(t, "Unexpected tracker delayed message count", delayedCount, "(expected 3)")
	}

	batchCount, err := tracker.GetBatchCount()
	Require(t, err)
	if batchCount != 3 {
		Fail(t, "Unexpected tracker batch count", batchCount, "(expected 3)")
	}

	// By modifying the timestamp of the userDelayed message, and adding it again, we cause a reorg
	userDelayedModified := &DelayedInboxMessage{
		BlockHash:      [32]byte{},
		BeforeInboxAcc: initMsgDelayed.AfterInboxAcc(),
		Message: &arbostypes.L1IncomingMessage{
			Header: &arbostypes.L1IncomingMessageHeader{
				Kind:        arbostypes.L1MessageType_EndOfBlock,
				Poster:      [20]byte{},
				BlockNumber: 0,
				Timestamp:   userDelayed.Message.Header.Timestamp + 1,
				RequestId:   &delayedRequestId,
				L1BaseFee:   common.Big0,
			},
		},
	}
	err = tracker.AddDelayedMessages([]*DelayedInboxMessage{userDelayedModified})
	Require(t, err)

	// userMsgBatch, and emptyBatch will be reorged out
	msgCount, err = streamer.GetMessageCount()
	Require(t, err)
	if msgCount != 1 {
		Fail(t, "Unexpected tx streamer message count", msgCount, "(expected 1)")
	}

	batchCount, err = tracker.GetBatchCount()
	Require(t, err)
	if batchCount != 1 {
		Fail(t, "Unexpected tracker batch count", batchCount, "(expected 1)")
	}

	// userDelayed2 will be deleted
	delayedCount, err = tracker.GetDelayedCount()
	Require(t, err)
	if delayedCount != 2 {
		Fail(t, "Unexpected tracker delayed message count", delayedCount, "(expected 2)")
	}

	// guarantees that delayed msg 1 is userDelayedModified and not userDelayed
	msg, err := tracker.GetDelayedMessage(ctx, 1)
	Require(t, err)
	if msg.Header.RequestId.Cmp(*userDelayedModified.Message.Header.RequestId) != 0 {
		Fail(t, "Unexpected delayed message requestId", msg.Header.RequestId, "(expected", userDelayedModified.Message.Header.RequestId, ")")
	}
	if msg.Header.Timestamp != userDelayedModified.Message.Header.Timestamp {
		Fail(t, "Unexpected delayed message timestamp", msg.Header.Timestamp, "(expected", userDelayedModified.Message.Header.Timestamp, ")")
	}
	if userDelayedModified.Message.Header.Timestamp == userDelayed.Message.Header.Timestamp {
		Fail(t, "Unexpected delayed message timestamp", userDelayedModified.Message.Header.Timestamp, "(expected", userDelayed.Message.Header.Timestamp, ")")
	}

	emptyBatch = &SequencerInboxBatch{
		BlockHash:              [32]byte{},
		ParentChainBlockNumber: 0,
		SequenceNumber:         1,
		BeforeInboxAcc:         [32]byte{1},
		AfterInboxAcc:          [32]byte{2},
		AfterDelayedAcc:        initMsgDelayed.AfterInboxAcc(),
		AfterDelayedCount:      1,
		TimeBounds:             bridgegen.IBridgeTimeBounds{},
		RawLog:                 types.Log{},
		DataLocation:           0,
		BridgeAddress:          [20]byte{},
		Serialized:             serializedInitMsgBatch,
	}
	err = tracker.AddSequencerBatches(ctx, nil, []*SequencerInboxBatch{emptyBatch})
	Require(t, err)

	msgCount, err = streamer.GetMessageCount()
	Require(t, err)
	if msgCount != 2 {
		Fail(t, "Unexpected tx streamer message count", msgCount, "(expected 2)")
	}

	batchCount, err = tracker.GetBatchCount()
	Require(t, err)
	if batchCount != 2 {
		Fail(t, "Unexpected tracker batch count", batchCount, "(expected 2)")
	}
}

func TestSequencerReorgFromLastDelayedMsg(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	exec, streamer, db, _ := NewTransactionStreamerForTest(t, ctx, common.Address{})
	tracker, err := NewInboxTracker(db, streamer, nil, DefaultSnapSyncConfig)
	Require(t, err)

	err = streamer.Start(ctx)
	Require(t, err)
	exec.Start(ctx)
	init, err := streamer.GetMessage(0)
	Require(t, err)

	initMsgDelayed := &DelayedInboxMessage{
		BlockHash:      [32]byte{},
		BeforeInboxAcc: [32]byte{},
		Message:        init.Message,
	}
	delayedRequestId := common.BigToHash(common.Big1)
	userDelayed := &DelayedInboxMessage{
		BlockHash:      [32]byte{},
		BeforeInboxAcc: initMsgDelayed.AfterInboxAcc(),
		Message: &arbostypes.L1IncomingMessage{
			Header: &arbostypes.L1IncomingMessageHeader{
				Kind:        arbostypes.L1MessageType_EndOfBlock,
				Poster:      [20]byte{},
				BlockNumber: 0,
				Timestamp:   0,
				RequestId:   &delayedRequestId,
				L1BaseFee:   common.Big0,
			},
		},
	}
	delayedRequestId2 := common.BigToHash(common.Big2)
	userDelayed2 := &DelayedInboxMessage{
		BlockHash:      [32]byte{},
		BeforeInboxAcc: userDelayed.AfterInboxAcc(),
		Message: &arbostypes.L1IncomingMessage{
			Header: &arbostypes.L1IncomingMessageHeader{
				Kind:        arbostypes.L1MessageType_EndOfBlock,
				Poster:      [20]byte{},
				BlockNumber: 0,
				Timestamp:   0,
				RequestId:   &delayedRequestId2,
				L1BaseFee:   common.Big0,
			},
		},
	}
	err = tracker.AddDelayedMessages([]*DelayedInboxMessage{initMsgDelayed, userDelayed, userDelayed2})
	Require(t, err)

	serializedInitMsgBatch := make([]byte, 40)
	binary.BigEndian.PutUint64(serializedInitMsgBatch[32:], 1)
	initMsgBatch := &SequencerInboxBatch{
		BlockHash:              [32]byte{},
		ParentChainBlockNumber: 0,
		SequenceNumber:         0,
		BeforeInboxAcc:         [32]byte{},
		AfterInboxAcc:          [32]byte{1},
		AfterDelayedAcc:        initMsgDelayed.AfterInboxAcc(),
		AfterDelayedCount:      1,
		TimeBounds:             bridgegen.IBridgeTimeBounds{},
		RawLog:                 types.Log{},
		DataLocation:           0,
		BridgeAddress:          [20]byte{},
		Serialized:             serializedInitMsgBatch,
	}
	serializedUserMsgBatch := make([]byte, 40)
	binary.BigEndian.PutUint64(serializedUserMsgBatch[32:], 2)
	userMsgBatch := &SequencerInboxBatch{
		BlockHash:              [32]byte{},
		ParentChainBlockNumber: 0,
		SequenceNumber:         1,
		BeforeInboxAcc:         [32]byte{1},
		AfterInboxAcc:          [32]byte{2},
		AfterDelayedAcc:        userDelayed2.AfterInboxAcc(),
		AfterDelayedCount:      3,
		TimeBounds:             bridgegen.IBridgeTimeBounds{},
		RawLog:                 types.Log{},
		DataLocation:           0,
		BridgeAddress:          [20]byte{},
		Serialized:             serializedUserMsgBatch,
	}
	emptyBatch := &SequencerInboxBatch{
		BlockHash:              [32]byte{},
		ParentChainBlockNumber: 0,
		SequenceNumber:         2,
		BeforeInboxAcc:         [32]byte{2},
		AfterInboxAcc:          [32]byte{3},
		AfterDelayedAcc:        userDelayed2.AfterInboxAcc(),
		AfterDelayedCount:      3,
		TimeBounds:             bridgegen.IBridgeTimeBounds{},
		RawLog:                 types.Log{},
		DataLocation:           0,
		BridgeAddress:          [20]byte{},
		Serialized:             serializedUserMsgBatch,
	}
	err = tracker.AddSequencerBatches(ctx, nil, []*SequencerInboxBatch{initMsgBatch, userMsgBatch, emptyBatch})
	Require(t, err)

	msgCount, err := streamer.GetMessageCount()
	Require(t, err)
	if msgCount != 3 {
		Fail(t, "Unexpected tx streamer message count", msgCount, "(expected 3)")
	}

	delayedCount, err := tracker.GetDelayedCount()
	Require(t, err)
	if delayedCount != 3 {
		Fail(t, "Unexpected tracker delayed message count", delayedCount, "(expected 3)")
	}

	batchCount, err := tracker.GetBatchCount()
	Require(t, err)
	if batchCount != 3 {
		Fail(t, "Unexpected tracker batch count", batchCount, "(expected 3)")
	}

	// Adding an already existing message alongside a new one shouldn't cause a reorg
	delayedRequestId3 := common.BigToHash(common.Big3)
	userDelayed3 := &DelayedInboxMessage{
		BlockHash:      [32]byte{},
		BeforeInboxAcc: userDelayed2.AfterInboxAcc(),
		Message: &arbostypes.L1IncomingMessage{
			Header: &arbostypes.L1IncomingMessageHeader{
				Kind:        arbostypes.L1MessageType_EndOfBlock,
				Poster:      [20]byte{},
				BlockNumber: 0,
				Timestamp:   0,
				RequestId:   &delayedRequestId3,
				L1BaseFee:   common.Big0,
			},
		},
	}
	err = tracker.AddDelayedMessages([]*DelayedInboxMessage{userDelayed2, userDelayed3})
	Require(t, err)

	msgCount, err = streamer.GetMessageCount()
	Require(t, err)
	if msgCount != 3 {
		Fail(t, "Unexpected tx streamer message count", msgCount, "(expected 3)")
	}

	batchCount, err = tracker.GetBatchCount()
	Require(t, err)
	if batchCount != 3 {
		Fail(t, "Unexpected tracker batch count", batchCount, "(expected 3)")
	}

	// By modifying the timestamp of the userDelayed2 message, and adding it again, we cause a reorg
	userDelayed2Modified := &DelayedInboxMessage{
		BlockHash:      [32]byte{},
		BeforeInboxAcc: userDelayed.AfterInboxAcc(),
		Message: &arbostypes.L1IncomingMessage{
			Header: &arbostypes.L1IncomingMessageHeader{
				Kind:        arbostypes.L1MessageType_EndOfBlock,
				Poster:      [20]byte{},
				BlockNumber: 0,
				Timestamp:   userDelayed2.Message.Header.Timestamp + 1,
				RequestId:   &delayedRequestId2,
				L1BaseFee:   common.Big0,
			},
		},
	}
	err = tracker.AddDelayedMessages([]*DelayedInboxMessage{userDelayed2Modified})
	Require(t, err)

	msgCount, err = streamer.GetMessageCount()
	Require(t, err)
	if msgCount != 1 {
		Fail(t, "Unexpected tx streamer message count", msgCount, "(expected 1)")
	}

	batchCount, err = tracker.GetBatchCount()
	Require(t, err)
	if batchCount != 1 {
		Fail(t, "Unexpected tracker batch count", batchCount, "(expected 1)")
	}

	delayedCount, err = tracker.GetDelayedCount()
	Require(t, err)
	if delayedCount != 3 {
		Fail(t, "Unexpected tracker delayed message count", delayedCount, "(expected 3)")
	}

	// guarantees that delayed msg 2 is userDelayedModified and not userDelayed
	msg, err := tracker.GetDelayedMessage(ctx, 2)
	Require(t, err)
	if msg.Header.RequestId.Cmp(*userDelayed2Modified.Message.Header.RequestId) != 0 {
		Fail(t, "Unexpected delayed message requestId", msg.Header.RequestId, "(expected", userDelayed2Modified.Message.Header.RequestId, ")")
	}
	if msg.Header.Timestamp != userDelayed2Modified.Message.Header.Timestamp {
		Fail(t, "Unexpected delayed message timestamp", msg.Header.Timestamp, "(expected", userDelayed2Modified.Message.Header.Timestamp, ")")
	}
	if userDelayed2Modified.Message.Header.Timestamp == userDelayed2.Message.Header.Timestamp {
		Fail(t, "Unexpected delayed message timestamp", userDelayed2Modified.Message.Header.Timestamp, "(expected", userDelayed2.Message.Header.Timestamp, ")")
	}

	emptyBatch = &SequencerInboxBatch{
		BlockHash:              [32]byte{},
		ParentChainBlockNumber: 0,
		SequenceNumber:         1,
		BeforeInboxAcc:         [32]byte{1},
		AfterInboxAcc:          [32]byte{2},
		AfterDelayedAcc:        initMsgDelayed.AfterInboxAcc(),
		AfterDelayedCount:      1,
		TimeBounds:             bridgegen.IBridgeTimeBounds{},
		RawLog:                 types.Log{},
		DataLocation:           0,
		BridgeAddress:          [20]byte{},
		Serialized:             serializedInitMsgBatch,
	}
	err = tracker.AddSequencerBatches(ctx, nil, []*SequencerInboxBatch{emptyBatch})
	Require(t, err)

	msgCount, err = streamer.GetMessageCount()
	Require(t, err)
	if msgCount != 2 {
		Fail(t, "Unexpected tx streamer message count", msgCount, "(expected 2)")
	}

	batchCount, err = tracker.GetBatchCount()
	Require(t, err)
	if batchCount != 2 {
		Fail(t, "Unexpected tracker batch count", batchCount, "(expected 2)")
	}
}
