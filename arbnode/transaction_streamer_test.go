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
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"

	"github.com/offchainlabs/nitro/arbnode/db/schema"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/execution"
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

type checkResultFixture struct {
	streamer     *TransactionStreamer
	db           ethdb.Database
	fatalErrChan chan error
	info         *arbostypes.MessageWithMetadataAndBlockInfo
	msgResult    *execution.MessageResult
}

// Wires a streamer with the given config and pre-populates the cleanup-branch preconditions
func newCheckResultMismatchFixture(t *testing.T, cfg TransactionStreamerConfig) checkResultFixture {
	t.Helper()
	fatalErrChan := make(chan error, 1)
	db := rawdb.NewMemoryDatabase()
	s := &TransactionStreamer{
		db:                     db,
		fatalErrChan:           fatalErrChan,
		config:                 func() *TransactionStreamerConfig { return &cfg },
		trackBlockMetadataFrom: 1,
	}
	Require(t, db.Put(dbKey(schema.BlockMetadataInputFeedPrefix, 42), []byte{0xcd}))
	feedHash := common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111")
	localHash := common.HexToHash("0x2222222222222222222222222222222222222222222222222222222222222222")
	return checkResultFixture{
		streamer:     s,
		db:           db,
		fatalErrChan: fatalErrChan,
		info: &arbostypes.MessageWithMetadataAndBlockInfo{
			MessageWithMeta: arbostypes.MessageWithMetadata{},
			BlockHash:       &feedHash,
			BlockMetadata:   common.BlockMetadata{0xab},
		},
		msgResult: &execution.MessageResult{BlockHash: localHash},
	}
}

// BlockMetadata + trackBlockMetadataFrom are set so the cleanup branch is
// reachable; the assertion is that the shutdown path skips it (no mutation).
func TestCheckResultShutdownOnMismatch(t *testing.T) {
	logHandler := testhelpers.InitTestLog(t, slog.LevelDebug)

	if !DefaultTransactionStreamerConfig.ShutdownOnBlockhashMismatch {
		t.Fatal("DefaultTransactionStreamerConfig.ShutdownOnBlockhashMismatch must be true; the safe default was flipped intentionally")
	}

	f := newCheckResultMismatchFixture(t, DefaultTransactionStreamerConfig)

	if f.streamer.checkResult(42, f.msgResult, f.info) {
		t.Fatal("expected checkResult to return false (halt) on mismatch with shutdown flag set")
	}

	if !logHandler.WasLogged(BlockHashMismatchLogMsg) {
		t.Error("expected BlockHashMismatchLogMsg to be logged")
	}
	if !logHandler.WasLogged("shutdown-on-blockhash-mismatch=false") {
		t.Error("expected override-hint log naming the override flag")
	}

	select {
	case err := <-f.fatalErrChan:
		msg := err.Error()
		for _, want := range []string{BlockHashMismatchLogMsg, "42", "1111", "2222"} {
			if !strings.Contains(msg, want) {
				t.Errorf("fatal err missing %q; got: %v", want, err)
			}
		}
	case <-time.After(time.Second):
		t.Fatal("expected a fatal error on fatalErrChan")
	}

	hadOld, err := f.db.Has(dbKey(schema.BlockMetadataInputFeedPrefix, 42))
	Require(t, err)
	if !hadOld {
		t.Error("shutdown path must not delete pre-existing blockMetadata")
	}
	hasMissing, err := f.db.Has(dbKey(schema.MissingBlockMetadataInputFeedPrefix, 42))
	Require(t, err)
	if hasMissing {
		t.Error("shutdown path must not write the Missing marker")
	}
}

func TestCheckResultOverrideContinues(t *testing.T) {
	cfg := DefaultTransactionStreamerConfig
	cfg.ShutdownOnBlockhashMismatch = false
	f := newCheckResultMismatchFixture(t, cfg)

	if !f.streamer.checkResult(42, f.msgResult, f.info) {
		t.Fatal("expected checkResult to return true (continue) when shutdown flag is off")
	}

	select {
	case err := <-f.fatalErrChan:
		t.Fatalf("did not expect a fatal error in override mode: %v", err)
	default:
	}

	hadOld, err := f.db.Has(dbKey(schema.BlockMetadataInputFeedPrefix, 42))
	Require(t, err)
	if hadOld {
		t.Error("expected stale blockMetadata to be deleted by override path")
	}
	hasMissing, err := f.db.Has(dbKey(schema.MissingBlockMetadataInputFeedPrefix, 42))
	Require(t, err)
	if !hasMissing {
		t.Error("expected MissingBlockMetadata marker to be written by override path")
	}
}

func TestCheckResultMatchingHashes(t *testing.T) {
	fatalErrChan := make(chan error, 1)
	cfg := DefaultTransactionStreamerConfig
	s := &TransactionStreamer{
		db:           rawdb.NewMemoryDatabase(),
		fatalErrChan: fatalErrChan,
		config:       func() *TransactionStreamerConfig { return &cfg },
	}
	h := common.HexToHash("0xabcd000000000000000000000000000000000000000000000000000000000000")
	info := &arbostypes.MessageWithMetadataAndBlockInfo{
		MessageWithMeta: arbostypes.MessageWithMetadata{},
		BlockHash:       &h,
		BlockMetadata:   nil,
	}
	msgResult := &execution.MessageResult{BlockHash: h}

	if !s.checkResult(42, msgResult, info) {
		t.Fatal("matching hashes should return true")
	}
	select {
	case err := <-fatalErrChan:
		t.Fatalf("did not expect fatal err on matching hashes: %v", err)
	default:
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
