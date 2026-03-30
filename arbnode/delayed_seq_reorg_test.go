// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbnode

import (
	"context"
	"encoding/binary"
	"errors"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"

	"github.com/offchainlabs/nitro/arbnode/mel"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
)

func TestSequencerReorgFromDelayed(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	exec, streamer, db, _ := NewTransactionStreamerForTest(t, ctx, common.Address{})
	tracker, err := NewInboxTracker(db, streamer, nil)
	Require(t, err)

	err = streamer.Start(ctx)
	Require(t, err)
	err = exec.Start(ctx)
	Require(t, err)
	init, err := streamer.GetMessage(0)
	Require(t, err)

	initMsgDelayed := &mel.DelayedInboxMessage{
		BlockHash:      [32]byte{},
		BeforeInboxAcc: [32]byte{},
		Message:        init.Message,
	}
	delayedRequestId := common.BigToHash(common.Big1)
	userDelayed := &mel.DelayedInboxMessage{
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
	userDelayed2 := &mel.DelayedInboxMessage{
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
	err = tracker.AddDelayedMessages([]*mel.DelayedInboxMessage{initMsgDelayed, userDelayed, userDelayed2})
	Require(t, err)

	serializedInitMsgBatch := make([]byte, 40)
	binary.BigEndian.PutUint64(serializedInitMsgBatch[32:], 1)
	initMsgBatch := &mel.SequencerInboxBatch{
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
	userMsgBatch := &mel.SequencerInboxBatch{
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
	emptyBatch := &mel.SequencerInboxBatch{
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
	err = tracker.AddSequencerBatches(ctx, nil, []*mel.SequencerInboxBatch{initMsgBatch, userMsgBatch, emptyBatch})
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
	userDelayedModified := &mel.DelayedInboxMessage{
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
	err = tracker.AddDelayedMessages([]*mel.DelayedInboxMessage{userDelayedModified})
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

	emptyBatch = &mel.SequencerInboxBatch{
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
	err = tracker.AddSequencerBatches(ctx, nil, []*mel.SequencerInboxBatch{emptyBatch})
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
	tracker, err := NewInboxTracker(db, streamer, nil)
	Require(t, err)

	err = streamer.Start(ctx)
	Require(t, err)
	err = exec.Start(ctx)
	Require(t, err)
	init, err := streamer.GetMessage(0)
	Require(t, err)

	initMsgDelayed := &mel.DelayedInboxMessage{
		BlockHash:      [32]byte{},
		BeforeInboxAcc: [32]byte{},
		Message:        init.Message,
	}
	delayedRequestId := common.BigToHash(common.Big1)
	userDelayed := &mel.DelayedInboxMessage{
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
	userDelayed2 := &mel.DelayedInboxMessage{
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
	err = tracker.AddDelayedMessages([]*mel.DelayedInboxMessage{initMsgDelayed, userDelayed, userDelayed2})
	Require(t, err)

	serializedInitMsgBatch := make([]byte, 40)
	binary.BigEndian.PutUint64(serializedInitMsgBatch[32:], 1)
	initMsgBatch := &mel.SequencerInboxBatch{
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
	userMsgBatch := &mel.SequencerInboxBatch{
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
	emptyBatch := &mel.SequencerInboxBatch{
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
	err = tracker.AddSequencerBatches(ctx, nil, []*mel.SequencerInboxBatch{initMsgBatch, userMsgBatch, emptyBatch})
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
	userDelayed3 := &mel.DelayedInboxMessage{
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
	err = tracker.AddDelayedMessages([]*mel.DelayedInboxMessage{userDelayed2, userDelayed3})
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
	userDelayed2Modified := &mel.DelayedInboxMessage{
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
	err = tracker.AddDelayedMessages([]*mel.DelayedInboxMessage{userDelayed2Modified})
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

	emptyBatch = &mel.SequencerInboxBatch{
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
	err = tracker.AddSequencerBatches(ctx, nil, []*mel.SequencerInboxBatch{emptyBatch})
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

// mismatchTestFixture holds the shared state for delayed-mismatch tests.
type mismatchTestFixture struct {
	ctx           context.Context
	tracker       *InboxTracker
	initDelayed   *mel.DelayedInboxMessage
	userDelayed   *mel.DelayedInboxMessage
	mismatchBatch *mel.SequencerInboxBatch
}

// newMismatchTestFixture creates a tracker with one init delayed message
// committed to the DB (delayed count = 1) and prepares a second delayed
// message and a batch whose AfterDelayedAcc is intentionally wrong.
func newMismatchTestFixture(t *testing.T, ctx context.Context) *mismatchTestFixture {
	t.Helper()
	exec, streamer, db, _ := NewTransactionStreamerForTest(t, ctx, common.Address{})
	tracker, err := NewInboxTracker(db, streamer, nil)
	Require(t, err)

	err = streamer.Start(ctx)
	Require(t, err)
	err = exec.Start(ctx)
	Require(t, err)
	init, err := streamer.GetMessage(0)
	Require(t, err)

	initDelayed := &mel.DelayedInboxMessage{
		BlockHash:      [32]byte{},
		BeforeInboxAcc: [32]byte{},
		Message:        init.Message,
	}
	delayedRequestId := common.BigToHash(common.Big1)
	userDelayed := &mel.DelayedInboxMessage{
		BlockHash:      [32]byte{},
		BeforeInboxAcc: initDelayed.AfterInboxAcc(),
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

	err = tracker.AddDelayedMessages([]*mel.DelayedInboxMessage{initDelayed})
	Require(t, err)

	serializedBatch := make([]byte, 40)
	binary.BigEndian.PutUint64(serializedBatch[32:], 1)
	mismatchBatch := &mel.SequencerInboxBatch{
		BlockHash:              [32]byte{},
		ParentChainBlockNumber: 0,
		SequenceNumber:         0,
		BeforeInboxAcc:         [32]byte{},
		AfterInboxAcc:          [32]byte{1},
		AfterDelayedAcc:        common.Hash{0xff}, // wrong accumulator
		AfterDelayedCount:      2,
		TimeBounds:             bridgegen.IBridgeTimeBounds{},
		RawLog:                 types.Log{},
		DataLocation:           0,
		BridgeAddress:          [20]byte{},
		Serialized:             serializedBatch,
	}

	return &mismatchTestFixture{
		ctx:           ctx,
		tracker:       tracker,
		initDelayed:   initDelayed,
		userDelayed:   userDelayed,
		mismatchBatch: mismatchBatch,
	}
}

// TestDelayedMismatchRollsBackDelayedMessages verifies that addMessages rolls
// back delayed messages when AddSequencerBatches fails with a delayed
// accumulator mismatch. Without the rollback, delayed messages would be
// committed to the DB without corresponding batches.
func TestDelayedMismatchRollsBackDelayedMessages(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	f := newMismatchTestFixture(t, ctx)

	// addMessages should roll back delayed messages on mismatch
	reader := &InboxReader{tracker: f.tracker}
	delayedMismatch, err := reader.addMessages(
		ctx,
		[]*mel.SequencerInboxBatch{f.mismatchBatch},
		[]*mel.DelayedInboxMessage{f.userDelayed},
	)
	Require(t, err)
	if !delayedMismatch {
		Fail(t, "Expected delayedMismatch to be true")
	}

	// Delayed count should be rolled back to 1 (the init message only).
	// Before the fix, this would be 2 — an orphaned delayed message.
	delayedCount, err := f.tracker.GetDelayedCount()
	Require(t, err)
	if delayedCount != 1 {
		Fail(t, "Delayed count not rolled back after mismatch", delayedCount, "(expected 1)")
	}
}

// TestDelayedMismatchNoOpRollback verifies that addMessages handles a mismatch
// correctly even when no new delayed messages were provided. The rollback
// should be a no-op (rolling back to the current count) without errors.
func TestDelayedMismatchNoOpRollback(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	f := newMismatchTestFixture(t, ctx)

	reader := &InboxReader{tracker: f.tracker}
	delayedMismatch, err := reader.addMessages(
		ctx,
		[]*mel.SequencerInboxBatch{f.mismatchBatch},
		nil, // no new delayed messages
	)
	Require(t, err)
	if !delayedMismatch {
		Fail(t, "Expected delayedMismatch to be true")
	}

	// Count should remain 1 (init message only, no rollback needed).
	delayedCount, err := f.tracker.GetDelayedCount()
	Require(t, err)
	if delayedCount != 1 {
		Fail(t, "Delayed count changed unexpectedly", delayedCount, "(expected 1)")
	}
}

// TestDelayedMismatchAtTrackerLevel verifies that calling AddDelayedMessages
// then AddSequencerBatches with a mismatched accumulator returns
// delayedMessagesMismatch and leaves delayed messages in the DB. This
// documents the low-level behavior that addMessages must compensate for.
func TestDelayedMismatchAtTrackerLevel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	f := newMismatchTestFixture(t, ctx)

	// Add the second delayed message — now count = 2
	err := f.tracker.AddDelayedMessages([]*mel.DelayedInboxMessage{f.userDelayed})
	Require(t, err)

	delayedCount, err := f.tracker.GetDelayedCount()
	Require(t, err)
	if delayedCount != 2 {
		Fail(t, "Unexpected delayed count", delayedCount, "(expected 2)")
	}

	// AddSequencerBatches should return delayedMessagesMismatch
	err = f.tracker.AddSequencerBatches(ctx, nil, []*mel.SequencerInboxBatch{f.mismatchBatch})
	if !errors.Is(err, delayedMessagesMismatch) {
		Fail(t, "Expected delayedMessagesMismatch error, got", err)
	}

	// Delayed messages are still in the DB (AddSequencerBatches does not roll them back)
	delayedCount, err = f.tracker.GetDelayedCount()
	Require(t, err)
	if delayedCount != 2 {
		Fail(t, "Delayed messages should still be in DB", delayedCount, "(expected 2)")
	}

	// ReorgDelayedTo cleans up the orphaned messages
	err = f.tracker.ReorgDelayedTo(1)
	Require(t, err)

	delayedCount, err = f.tracker.GetDelayedCount()
	Require(t, err)
	if delayedCount != 1 {
		Fail(t, "ReorgDelayedTo did not clean up orphaned messages", delayedCount, "(expected 1)")
	}
}

// TestAddMessages_GetDelayedCountError verifies that addMessages returns a
// wrapped error when the initial GetDelayedCount call fails (e.g. closed DB).
func TestAddMessages_GetDelayedCountError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	f := newMismatchTestFixture(t, ctx)

	// Close the underlying DB so that GetDelayedCount fails.
	f.tracker.db.Close()

	reader := &InboxReader{tracker: f.tracker}
	_, err := reader.addMessages(ctx, nil, nil)
	if err == nil {
		Fail(t, "Expected error from addMessages when GetDelayedCount fails")
	}
	if !strings.Contains(err.Error(), "getting delayed message count before adding messages") {
		Fail(t, "Expected wrapped error, got:", err)
	}
}

// TestAddMessages_ReorgDelayedToError verifies that when addMessages detects a
// delayed accumulator mismatch and the subsequent ReorgDelayedTo fails, the
// returned error wraps the rollback error and includes the original mismatch.
func TestAddMessages_ReorgDelayedToError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	f := newMismatchTestFixture(t, ctx)

	// Wrap the DB so that the second batch.Write (ReorgDelayedTo) fails.
	// First batch.Write (AddDelayedMessages) succeeds normally.
	injectedErr := errors.New("injected write failure")
	f.tracker.db = &failingBatchDB{
		Database:          f.tracker.db,
		writesBeforeFail:  1, // allow 1 successful Write, then fail
		writeErr:          injectedErr,
	}

	reader := &InboxReader{tracker: f.tracker}
	_, err := reader.addMessages(
		ctx,
		[]*mel.SequencerInboxBatch{f.mismatchBatch},
		[]*mel.DelayedInboxMessage{f.userDelayed},
	)
	if err == nil {
		Fail(t, "Expected error when ReorgDelayedTo fails during rollback")
	}
	if !errors.Is(err, injectedErr) {
		Fail(t, "Returned error should wrap the rollback error, got:", err)
	}
	if !strings.Contains(err.Error(), "failed to rollback delayed messages") {
		Fail(t, "Returned error should describe rollback failure, got:", err)
	}
	if !strings.Contains(err.Error(), "original mismatch") {
		Fail(t, "Returned error should include original mismatch error, got:", err)
	}
	if !errors.Is(err, delayedMessagesMismatch) {
		Fail(t, "Returned error should wrap the original mismatch error, got:", err)
	}
}

// failingBatchDB wraps an ethdb.Database and makes batch Write() calls fail
// after a configurable number of successful writes.
type failingBatchDB struct {
	ethdb.Database
	writesBeforeFail int
	writeErr         error
	writeCount       atomic.Int32
}

func (f *failingBatchDB) NewBatch() ethdb.Batch {
	return &failingBatch{Batch: f.Database.NewBatch(), parent: f}
}

func (f *failingBatchDB) NewBatchWithSize(size int) ethdb.Batch {
	return &failingBatch{Batch: f.Database.NewBatchWithSize(size), parent: f}
}

type failingBatch struct {
	ethdb.Batch
	parent *failingBatchDB
}

func (b *failingBatch) Write() error {
	n := int(b.parent.writeCount.Add(1))
	if n > b.parent.writesBeforeFail {
		return b.parent.writeErr
	}
	return b.Batch.Write()
}
