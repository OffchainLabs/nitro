// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

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

	exec, streamer, db, _ := NewTransactionStreamerForTest(t, common.Address{})
	tracker, err := NewInboxTracker(db, streamer, nil)
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
	err = tracker.AddDelayedMessages([]*DelayedInboxMessage{initMsgDelayed, userDelayed}, false)
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
		TimeBounds:             bridgegen.ISequencerInboxTimeBounds{},
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
		AfterDelayedAcc:        userDelayed.AfterInboxAcc(),
		AfterDelayedCount:      2,
		TimeBounds:             bridgegen.ISequencerInboxTimeBounds{},
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
		AfterDelayedAcc:        userDelayed.AfterInboxAcc(),
		AfterDelayedCount:      2,
		TimeBounds:             bridgegen.ISequencerInboxTimeBounds{},
		RawLog:                 types.Log{},
		DataLocation:           0,
		BridgeAddress:          [20]byte{},
		Serialized:             serializedUserMsgBatch,
	}
	err = tracker.AddSequencerBatches(ctx, nil, []*SequencerInboxBatch{initMsgBatch, userMsgBatch, emptyBatch})
	Require(t, err)

	// Reorg out the user delayed message
	err = tracker.ReorgDelayedTo(1, true)
	Require(t, err)

	msgCount, err := streamer.GetMessageCount()
	Require(t, err)
	if msgCount != 1 {
		Fail(t, "Unexpected tx streamer message count", msgCount, "(expected 1)")
	}

	delayedCount, err := tracker.GetDelayedCount()
	Require(t, err)
	if delayedCount != 1 {
		Fail(t, "Unexpected tracker delayed message count", delayedCount, "(expected 1)")
	}

	batchCount, err := tracker.GetBatchCount()
	Require(t, err)
	if batchCount != 1 {
		Fail(t, "Unexpected tracker batch count", batchCount, "(expected 1)")
	}

	emptyBatch = &SequencerInboxBatch{
		BlockHash:              [32]byte{},
		ParentChainBlockNumber: 0,
		SequenceNumber:         1,
		BeforeInboxAcc:         [32]byte{1},
		AfterInboxAcc:          [32]byte{2},
		AfterDelayedAcc:        initMsgDelayed.AfterInboxAcc(),
		AfterDelayedCount:      1,
		TimeBounds:             bridgegen.ISequencerInboxTimeBounds{},
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
