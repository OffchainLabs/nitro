// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbnode

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

// stubBatchDataProvider returns getErr from the GetSequencerMessageBytes* methods
// (the only ones exercised by these tests). All other methods return zero values
// to satisfy the BatchDataProvider interface. Tests can flip getErr to nil to
// simulate the inbox tracker catching up.
type stubBatchDataProvider struct {
	getErr error
}

func (p *stubBatchDataProvider) GetBatchCount() (uint64, error) { return 0, nil }
func (p *stubBatchDataProvider) GetBatchMessageCount(seqNum uint64) (arbutil.MessageIndex, error) {
	return 0, nil
}
func (p *stubBatchDataProvider) GetDelayedAcc(seqNum uint64) (common.Hash, error) {
	return common.Hash{}, nil
}
func (p *stubBatchDataProvider) GetSequencerMessageBytes(ctx context.Context, seqNum uint64) ([]byte, common.Hash, error) {
	return nil, common.Hash{}, p.getErr
}
func (p *stubBatchDataProvider) GetSequencerMessageBytesForParentBlock(ctx context.Context, seqNum uint64, parentChainBlock uint64) ([]byte, common.Hash, error) {
	return nil, common.Hash{}, p.getErr
}
func (p *stubBatchDataProvider) FindParentChainBlockContainingDelayed(ctx context.Context, index uint64) (uint64, error) {
	return 0, nil
}

// buildBatchPostingReportL2msg constructs a minimal L2msg payload that
// arbostypes.ParseBatchPostingReportMessageFields can decode. Layout:
//
//	batchTimestamp(32) | batchPosterAddr(20) | dataHash(32) | batchNum(32) | l1BaseFee(32)
//
// extraGas (uint64) is optional and intentionally omitted; the parser tolerates
// EOF after l1BaseFee.
func buildBatchPostingReportL2msg(batchNum uint64) []byte {
	out := make([]byte, 32+20+32+32+32)
	// batchNum sits in the low 8 bytes of the 4th 32-byte slot (offset 84+24=108).
	binary.BigEndian.PutUint64(out[108:], batchNum)
	return out
}

// buildL2NormalMessage returns a non-BatchPostingReport message that GetMessage
// will read without hitting batchDataProvider. Used to exercise the read-success
// branch of ExecuteNextMsg.
func buildL2NormalMessage() arbostypes.MessageWithMetadata {
	return arbostypes.MessageWithMetadata{
		Message: &arbostypes.L1IncomingMessage{
			Header: &arbostypes.L1IncomingMessageHeader{
				Kind:      arbostypes.L1MessageType_L2Message,
				RequestId: &common.Hash{},
				L1BaseFee: common.Big0,
			},
			L2msg: []byte{0x00},
		},
		DelayedMessagesRead: 1,
	}
}

// TestAccumulatorNotFoundErrSubstring guards the substring match in
// EphemeralErrorHandler against drift in AccumulatorNotFoundErr's text. The
// throttle in TransactionStreamer is initialised with AccumulatorNotFoundErr.Error()
// (i.e. "accumulator not found"); changing that string without updating the
// EphemeralErrorHandler initializer would silently break the throttle.
func TestAccumulatorNotFoundErrSubstring(t *testing.T) {
	const want = "accumulator not found"
	if !strings.Contains(AccumulatorNotFoundErr.Error(), want) {
		t.Fatalf("AccumulatorNotFoundErr.Error() must contain %q for the EphemeralErrorHandler substring match in ExecuteNextMsg to engage; got %q",
			want, AccumulatorNotFoundErr.Error())
	}
}

// TestLogReadMessageErrPicksMessage verifies that logReadMessageErr selects the
// descriptive message for AccumulatorNotFoundErr (via errors.Is, so wrapping is
// preserved) and the generic message for any other error.
func TestLogReadMessageErrPicksMessage(t *testing.T) {
	logHandler := testhelpers.InitTestLog(t, slog.LevelDebug)

	_, streamer, _, _ := NewTransactionStreamerForTest(t, t.Context(), common.Address{})

	// Wrapped sentinel → descriptive message.
	wrapped := fmt.Errorf("failed to fetch batch: %w: no metadata for batch 1225668", AccumulatorNotFoundErr)
	streamer.logReadMessageErr(wrapped, 42)
	if !logHandler.WasLogged("waiting for inbox tracker") {
		t.Fatal("expected descriptive 'waiting for inbox tracker' message for wrapped AccumulatorNotFoundErr")
	}
	if logHandler.WasLogged("failed to readMessage") {
		t.Fatal("did not expect generic 'failed to readMessage' message for AccumulatorNotFoundErr")
	}

	logHandler.Clear()
	streamer.accNotFoundErrHandler.Reset()

	// Unrelated error → generic message.
	streamer.logReadMessageErr(errors.New("some unrelated db error"), 42)
	if !logHandler.WasLogged("failed to readMessage") {
		t.Fatal("expected generic 'failed to readMessage' for non-accumulator error")
	}
	if logHandler.WasLogged("waiting for inbox tracker") {
		t.Fatal("did not expect descriptive message for unrelated error")
	}
}

// TestExecuteNextMsgEphemeralAccumulatorNotFound verifies that when a
// BatchPostingReport's batch metadata is missing, ExecuteNextMsg engages the
// ephemeral handler so the error doesn't log at ERROR on every retry.
func TestExecuteNextMsgEphemeralAccumulatorNotFound(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	exec, streamer, _, _ := NewTransactionStreamerForTest(t, ctx, common.Address{})

	if streamer.accNotFoundErrHandler == nil {
		t.Fatal("accNotFoundErrHandler should be initialized by NewTransactionStreamer")
	}
	handler := streamer.accNotFoundErrHandler

	stubProvider := &stubBatchDataProvider{
		getErr: fmt.Errorf("%w: no metadata for batch 1225668", AccumulatorNotFoundErr),
	}
	Require(t, streamer.SetBatchDataProvider(stubProvider, nil))

	// StopWaiter context only; skip Start to avoid the executeMessages goroutine
	// racing on FirstOccurrence.
	streamer.StopWaiter.Start(ctx, streamer)
	defer streamer.StopAndWait()
	Require(t, exec.Start(ctx))
	defer exec.StopAndWait()

	// DelayedMessagesRead != 0 plus a non-failing FindParentChainBlockContainingDelayed
	// steers FillInBatchGasFields toward GetSequencerMessageBytesForParentBlock; the
	// stub fails both byte-fetch methods identically so the dispatch detail isn't
	// load-bearing for this test. Init message is auto-added at idx 0 by
	// NewTransactionStreamerForTest → AddFakeInitMessage.
	batchPostingReport := arbostypes.MessageWithMetadata{
		Message: &arbostypes.L1IncomingMessage{
			Header: &arbostypes.L1IncomingMessageHeader{
				Kind:      arbostypes.L1MessageType_BatchPostingReport,
				RequestId: &common.Hash{},
				L1BaseFee: common.Big0,
			},
			L2msg: buildBatchPostingReportL2msg(1225668),
		},
		DelayedMessagesRead: 1,
	}
	if !batchPostingReport.Message.IsBatchGasFieldsMissing() {
		t.Fatal("batch posting report test fixture must have IsBatchGasFieldsMissing() == true; otherwise ExecuteNextMsg won't trigger the batch fetcher")
	}
	Require(t, streamer.AddMessages(1, false, []arbostypes.MessageWithMetadata{batchPostingReport}, nil))

	// Confirm the error chain still produces AccumulatorNotFoundErr; important
	// because the handler matches on substring "accumulator not found".
	_, err := streamer.GetMessage(1)
	if err == nil {
		t.Fatal("expected GetMessage to fail with AccumulatorNotFoundErr")
	}
	if !errors.Is(err, AccumulatorNotFoundErr) {
		t.Fatalf("expected error chain to contain AccumulatorNotFoundErr, got: %v", err)
	}

	// exec head is 0 (genesis built from init), consensusHead is 1 (batch posting
	// report), so msgIdxToExecute = 1: ExecuteNextMsg attempts the read and fails.
	streamer.ExecuteNextMsg(ctx)
	if handler.FirstOccurrence.Equal(time.Time{}) {
		t.Fatal("expected handler to engage after AccumulatorNotFoundErr from ExecuteNextMsg, but FirstOccurrence is still zero")
	}
}

// TestExecuteNextMsgResetsHandlerOnReadSuccess verifies the production-side
// Reset placement in ExecuteNextMsg: when both reads succeed, FirstOccurrence
// is cleared before DigestMessage. A regression that moved the Reset to end-of-function
// would leave FirstOccurrence stale if DigestMessage failed, and this test would catch it.
func TestExecuteNextMsgResetsHandlerOnReadSuccess(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	exec, streamer, _, _ := NewTransactionStreamerForTest(t, ctx, common.Address{})

	streamer.StopWaiter.Start(ctx, streamer)
	defer streamer.StopAndWait()
	Require(t, exec.Start(ctx))
	defer exec.StopAndWait()

	// Add a non-batch-posting-report message so GetMessage succeeds without
	// hitting batchDataProvider.
	Require(t, streamer.AddMessages(1, false, []arbostypes.MessageWithMetadata{buildL2NormalMessage()}, nil))

	// Pre-engage the handler as if a prior accumulator-not-found error had fired.
	handler := streamer.accNotFoundErrHandler
	*handler.FirstOccurrence = time.Now().Add(-30 * time.Second)

	// ExecuteNextMsg: reads idx 1 successfully (no batch fetching), so the
	// production Reset at the post-read-success site fires. DigestMessage may
	// fail on the synthetic L2 message, but Reset has already happened.
	streamer.ExecuteNextMsg(ctx)

	if !handler.FirstOccurrence.Equal(time.Time{}) {
		t.Fatal("expected ExecuteNextMsg's post-read-success Reset to clear FirstOccurrence")
	}
}
