// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbnode

import (
	"context"
	"encoding/binary"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
)

func TestSequencerReorgFromDelayed(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	streamer, db, _ := NewTransactionStreamerForTest(t, common.Address{})
	tracker, err := NewInboxTracker(db, streamer, nil)
	Require(t, err)

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
		Message: &arbos.L1IncomingMessage{
			Header: &arbos.L1IncomingMessageHeader{
				Kind:        arbos.L1MessageType_EndOfBlock,
				Poster:      [20]byte{},
				BlockNumber: 0,
				Timestamp:   0,
				RequestId:   &delayedRequestId,
				L1BaseFee:   common.Big0,
			},
		},
	}
	err = tracker.AddDelayedMessages([]*DelayedInboxMessage{initMsgDelayed, userDelayed})
	Require(t, err)

	serializedInitMsgBatch := make([]byte, 40)
	binary.BigEndian.PutUint64(serializedInitMsgBatch[32:], 1)
	initMsgBatch := &SequencerInboxBatch{
		BlockHash:         [32]byte{},
		BlockNumber:       0,
		SequenceNumber:    0,
		BeforeInboxAcc:    [32]byte{},
		AfterInboxAcc:     [32]byte{1},
		AfterDelayedAcc:   initMsgDelayed.AfterInboxAcc(),
		AfterDelayedCount: 1,
		TimeBounds:        bridgegen.ISequencerInboxTimeBounds{},
		txIndexInBlock:    0,
		dataLocation:      0,
		bridgeAddress:     [20]byte{},
		serialized:        serializedInitMsgBatch,
	}
	serializedUserMsgBatch := make([]byte, 40)
	binary.BigEndian.PutUint64(serializedUserMsgBatch[32:], 2)
	userMsgBatch := &SequencerInboxBatch{
		BlockHash:         [32]byte{},
		BlockNumber:       0,
		SequenceNumber:    1,
		BeforeInboxAcc:    [32]byte{1},
		AfterInboxAcc:     [32]byte{2},
		AfterDelayedAcc:   userDelayed.AfterInboxAcc(),
		AfterDelayedCount: 2,
		TimeBounds:        bridgegen.ISequencerInboxTimeBounds{},
		txIndexInBlock:    0,
		dataLocation:      0,
		bridgeAddress:     [20]byte{},
		serialized:        serializedUserMsgBatch,
	}
	emptyBatch := &SequencerInboxBatch{
		BlockHash:         [32]byte{},
		BlockNumber:       0,
		SequenceNumber:    2,
		BeforeInboxAcc:    [32]byte{2},
		AfterInboxAcc:     [32]byte{3},
		AfterDelayedAcc:   userDelayed.AfterInboxAcc(),
		AfterDelayedCount: 2,
		TimeBounds:        bridgegen.ISequencerInboxTimeBounds{},
		txIndexInBlock:    0,
		dataLocation:      0,
		bridgeAddress:     [20]byte{},
		serialized:        serializedUserMsgBatch,
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
		BlockHash:         [32]byte{},
		BlockNumber:       0,
		SequenceNumber:    1,
		BeforeInboxAcc:    [32]byte{1},
		AfterInboxAcc:     [32]byte{2},
		AfterDelayedAcc:   initMsgDelayed.AfterInboxAcc(),
		AfterDelayedCount: 1,
		TimeBounds:        bridgegen.ISequencerInboxTimeBounds{},
		txIndexInBlock:    0,
		dataLocation:      0,
		bridgeAddress:     [20]byte{},
		serialized:        serializedInitMsgBatch,
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
