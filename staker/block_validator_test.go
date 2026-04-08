package staker

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/execution"
	"github.com/offchainlabs/nitro/validator"
)

// Compile-time check that mockStreamer satisfies TransactionStreamerInterface.
var _ TransactionStreamerInterface = (*mockStreamer)(nil)

// mockStreamer is a minimal mock of TransactionStreamerInterface for unit tests.
type mockStreamer struct {
	results  map[arbutil.MessageIndex]*execution.MessageResult
	messages map[arbutil.MessageIndex]*arbostypes.MessageWithMetadata
}

func (m *mockStreamer) SetBlockValidator(*BlockValidator)                       {}
func (m *mockStreamer) GetProcessedMessageCount() (arbutil.MessageIndex, error) { return 0, nil }
func (m *mockStreamer) PauseReorgs()                                            {}
func (m *mockStreamer) ResumeReorgs()                                           {}
func (m *mockStreamer) ChainConfig() *params.ChainConfig                        { return nil }
func (m *mockStreamer) ResultAtMessageIndex(idx arbutil.MessageIndex) (*execution.MessageResult, error) {
	if r, ok := m.results[idx]; ok {
		return r, nil
	}
	return nil, fmt.Errorf("no result at index %d", idx)
}
func (m *mockStreamer) GetMessage(idx arbutil.MessageIndex) (*arbostypes.MessageWithMetadata, error) {
	if msg, ok := m.messages[idx]; ok {
		return msg, nil
	}
	return nil, fmt.Errorf("no message at index %d", idx)
}

// Compile-time check that mockInboxTracker satisfies InboxTrackerInterface.
var _ InboxTrackerInterface = (*mockInboxTracker)(nil)

// mockInboxTracker is a minimal mock of InboxTrackerInterface for unit tests.
// batchMsgCounts maps batch number → cumulative message count at end of that batch.
type mockInboxTracker struct {
	batchMsgCounts map[uint64]arbutil.MessageIndex
}

func (m *mockInboxTracker) SetBlockValidator(*BlockValidator) {}
func (m *mockInboxTracker) GetBatchCount() (uint64, error) {
	return uint64(len(m.batchMsgCounts)), nil
}
func (m *mockInboxTracker) GetBatchMessageCount(seqNum uint64) (arbutil.MessageIndex, error) {
	if c, ok := m.batchMsgCounts[seqNum]; ok {
		return c, nil
	}
	return 0, fmt.Errorf("no batch %d", seqNum)
}
func (m *mockInboxTracker) FindInboxBatchContainingMessage(pos arbutil.MessageIndex) (uint64, bool, error) {
	for batch := uint64(0); batch < uint64(len(m.batchMsgCounts)); batch++ {
		if m.batchMsgCounts[batch] > pos {
			return batch, true, nil
		}
	}
	return 0, false, nil
}
func (m *mockInboxTracker) GetBatchAcc(seqNum uint64) (common.Hash, error) {
	return common.Hash{}, nil
}
func (m *mockInboxTracker) GetDelayedMessageBytes(_ context.Context, seqNum uint64) ([]byte, error) {
	return nil, nil
}

// newTestValidator creates a BlockValidator with a buffered fatalErr channel
// and the given config value. Returns the validator and the fatal error channel.
func newTestValidator(failureIsFatal bool) (*BlockValidator, chan error) {
	fatalCh := make(chan error, 1)
	v := &BlockValidator{
		fatalErr: fatalCh,
	}
	v.config = func() *BlockValidatorConfig {
		return &BlockValidatorConfig{
			FailureIsFatal: failureIsFatal,
		}
	}
	return v, fatalCh
}

func requireNoFatalError(t *testing.T, fatalCh chan error) {
	t.Helper()
	select {
	case err := <-fatalCh:
		t.Fatalf("unexpected fatal error: %v", err)
	default:
	}
}

func requireFatalError(t *testing.T, fatalCh chan error, target error) {
	t.Helper()
	select {
	case err := <-fatalCh:
		if target != nil && !errors.Is(err, target) {
			t.Fatalf("expected fatal error matching %v, got: %v", target, err)
		}
		if err == nil {
			t.Fatal("expected non-nil fatal error")
		}
	default:
		t.Fatalf("expected fatal error (matching %v), but channel was empty", target)
	}
}

func TestReorgGuardRejectsZero(t *testing.T) {
	v, fatalCh := newTestValidator(true)
	err := v.Reorg(context.Background(), 0)
	if err == nil {
		t.Fatal("expected error for count == 0")
	}
	if err.Error() != "cannot reorg out genesis" {
		t.Fatalf("unexpected error: %v", err)
	}
	requireFatalError(t, fatalCh, err)
}

func TestReorgGuardAllowsOne(t *testing.T) {
	// With chainCaughtUp=false (zero value), Reorg returns nil early after
	// the guard, confirming that count==1 is accepted.
	v := &BlockValidator{}
	err := v.Reorg(context.Background(), 1)
	if err != nil {
		t.Fatalf("expected no error for count == 1, got: %v", err)
	}
}

func TestReorgToGenesisWithCaughtUpValidator(t *testing.T) {
	// Exercises Reorg(ctx, 1) through the full chainCaughtUp=true path,
	// verifying count==1 works end-to-end with mock streamer data.
	streamer := &mockStreamer{
		results: map[arbutil.MessageIndex]*execution.MessageResult{
			0: {BlockHash: common.HexToHash("0xaa"), SendRoot: common.HexToHash("0xbb")},
		},
		messages: map[arbutil.MessageIndex]*arbostypes.MessageWithMetadata{
			0: {DelayedMessagesRead: 1},
		},
	}
	v, fatalCh := newTestValidator(true)
	v.StatelessBlockValidator = &StatelessBlockValidator{streamer: streamer}
	v.chainCaughtUp = true
	v.createNodesChan = make(chan struct{}, 1)
	// Set createdA >= count so we don't hit the early "created < count" return.
	v.createdA.Store(1)

	err := v.Reorg(context.Background(), 1)
	if err != nil {
		t.Fatalf("Reorg(ctx, 1) with chainCaughtUp=true failed: %v", err)
	}

	// Verify the validator state was reset correctly.
	if v.createdA.Load() != 1 {
		t.Errorf("expected createdA=1, got %d", v.createdA.Load())
	}
	// nextCreateStartGS should be built from the genesis result and position {1, 0}.
	expectedGS := BuildGlobalState(
		execution.MessageResult{BlockHash: common.HexToHash("0xaa"), SendRoot: common.HexToHash("0xbb")},
		GlobalStatePosition{BatchNumber: 1, PosInBatch: 0},
	)
	if v.nextCreateStartGS != expectedGS {
		t.Errorf("expected nextCreateStartGS=%v, got %v", expectedGS, v.nextCreateStartGS)
	}
	if v.nextCreatePrevDelayed != 1 {
		t.Errorf("expected nextCreatePrevDelayed=1, got %d", v.nextCreatePrevDelayed)
	}

	requireNoFatalError(t, fatalCh)
}

func TestPossiblyFatalSendsAllErrorsWhenNotStopped(t *testing.T) {
	v, fatalCh := newTestValidator(true)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	v.StopWaiter.Start(ctx, v)

	// possiblyFatal does not distinguish error types — callers that need to
	// filter shutdown/transient errors use classifyValidationError first.
	v.possiblyFatal(context.Canceled)
	requireFatalError(t, fatalCh, context.Canceled)

	v.possiblyFatal(context.DeadlineExceeded)
	requireFatalError(t, fatalCh, context.DeadlineExceeded)

	realErr := errors.New("validation failed")
	v.possiblyFatal(realErr)
	requireFatalError(t, fatalCh, realErr)
}

func TestHandleValidationResult(t *testing.T) {
	reorgIdx := arbutil.MessageIndex(5)
	reorgIdx1 := arbutil.MessageIndex(1)

	tests := []struct {
		name      string
		cancelCtx bool
		reorg     *arbutil.MessageIndex
		err       error
		wantFatal bool
		// setupReorgFailure configures the validator so Reorg will fail
		setupReorgFailure bool
	}{
		{
			name:  "skips reorg during shutdown",
			reorg: &reorgIdx, cancelCtx: true,
		},
		{
			name:      "non-canceled error escalates to possiblyFatal",
			err:       errors.New("validation data corruption"),
			wantFatal: true,
		},
		{
			name: "canceled error with live ctx logs warn (not shutdown)",
			err:  context.Canceled,
		},
		{
			name: "wrapped canceled error with live ctx logs warn",
			err:  fmt.Errorf("wrapped: %w", context.Canceled),
		},
		{
			name: "canceled error during shutdown suppressed to debug",
			err:  context.Canceled, cancelCtx: true,
		},
		{
			name:      "deadline exceeded escalates to possiblyFatal",
			err:       context.DeadlineExceeded,
			wantFatal: true,
		},
		{
			name:  "reorg succeeds on live context",
			reorg: &reorgIdx,
		},
		{
			name:              "reorg failure sends to possiblyFatal",
			reorg:             &reorgIdx1,
			setupReorgFailure: true,
			wantFatal:         true,
		},
		{
			name:      "error takes precedence over reorg",
			reorg:     &reorgIdx,
			err:       errors.New("something broke"),
			wantFatal: true,
		},
		{
			name: "nil err and nil reorg is noop",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			v, fatalCh := newTestValidator(true)

			if tc.setupReorgFailure {
				streamer := &mockStreamer{
					results:  map[arbutil.MessageIndex]*execution.MessageResult{},
					messages: map[arbutil.MessageIndex]*arbostypes.MessageWithMetadata{},
				}
				v.StatelessBlockValidator = &StatelessBlockValidator{streamer: streamer}
				v.chainCaughtUp = true
				v.createNodesChan = make(chan struct{}, 1)
				v.createdA.Store(1)
			}

			ctx := context.Background()
			if tc.cancelCtx {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}

			result := v.handleValidationResult(ctx, tc.reorg, tc.err, "test")
			if result != 0 {
				t.Errorf("expected ValidationPoll duration (0), got %v", result)
			}

			if tc.wantFatal {
				requireFatalError(t, fatalCh, nil)
			} else {
				requireNoFatalError(t, fatalCh)
			}
		})
	}
}

func TestHandleValidationResultDoesNotSkipReorgOnDeadlineExceeded(t *testing.T) {
	// The reorg skip guard only triggers for context.Canceled (clean shutdown),
	// not for context.DeadlineExceeded (timeout). With chainCaughtUp=true and
	// mock data, we verify Reorg actually executes (modifying validator state)
	// rather than being skipped.
	streamer := &mockStreamer{
		results: map[arbutil.MessageIndex]*execution.MessageResult{
			0: {BlockHash: common.HexToHash("0xaa"), SendRoot: common.HexToHash("0xbb")},
		},
		messages: map[arbutil.MessageIndex]*arbostypes.MessageWithMetadata{
			0: {DelayedMessagesRead: 1},
		},
	}
	v, fatalCh := newTestValidator(true)
	v.StatelessBlockValidator = &StatelessBlockValidator{streamer: streamer}
	v.chainCaughtUp = true
	v.createNodesChan = make(chan struct{}, 1)
	v.createdA.Store(2)

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()

	reorgTarget := arbutil.MessageIndex(1)
	v.handleValidationResult(ctx, &reorgTarget, nil, "test")

	// Reorg was executed (not skipped): createdA should be reset to count.
	if v.createdA.Load() != 1 {
		t.Errorf("expected createdA=1 after reorg, got %d", v.createdA.Load())
	}
	requireNoFatalError(t, fatalCh)
}

func TestPossiblyFatalChannelFull(t *testing.T) {
	// When fatalErr already has an error, the second error is dropped
	// (with a log) rather than blocking. Verify the first error is
	// preserved and the second doesn't panic or block.
	v, fatalCh := newTestValidator(true)

	first := errors.New("first error")
	second := errors.New("second error")
	v.possiblyFatal(first)
	v.possiblyFatal(second) // should not block

	requireFatalError(t, fatalCh, first)
	// Channel should now be empty (second was dropped).
	requireNoFatalError(t, fatalCh)
}

func TestPossiblyFatalNilIsNoop(t *testing.T) {
	v, fatalCh := newTestValidator(true)

	v.possiblyFatal(nil)
	requireNoFatalError(t, fatalCh)
}

func TestPossiblyFatalSuppressesAllErrorsWhenStopped(t *testing.T) {
	// After StopAndWait, possiblyFatal should log but not escalate any error
	// to fatalErr. This is the primary shutdown suppression mechanism.
	v, fatalCh := newTestValidator(true)
	ctx, cancel := context.WithCancel(context.Background())
	v.StopWaiter.Start(ctx, v)
	cancel()
	v.StopWaiter.StopAndWait()

	v.possiblyFatal(errors.New("real error after stop"))
	requireNoFatalError(t, fatalCh)

	v.possiblyFatal(context.Canceled)
	requireNoFatalError(t, fatalCh)

	v.possiblyFatal(context.DeadlineExceeded)
	requireNoFatalError(t, fatalCh)
}

func TestAdvanceValidationsFailedEntry(t *testing.T) {
	// Table-driven tests for advanceValidations behavior when a validation
	// entry has failed (Success=false). The key axes are:
	//   - validation error type (context.Canceled, wrapped Canceled, other)
	//   - context state (live vs cancelled)
	//
	// Note: with a pre-cancelled context, advanceValidations returns early at
	// the ctx.Err() check before reaching the ErrSeverity switch.
	tests := []struct {
		name        string
		cancelCtx   bool                    // whether to cancel the context before calling
		validErr    error                   // error stored in DoneEntry.Err
		severity    validationErrorSeverity // pre-classified severity stored in DoneEntry.ErrSeverity
		wantReorg   bool                    // expect non-nil reorg pointer
		wantErr     error                   // if non-nil, expect errors.Is match on returned err
		wantFatal   bool                    // expect an error on fatalCh
		fatalTarget error                   // if non-nil, the fatal error must match via errors.Is
	}{
		{
			name:      "cancelled context returns early with context error",
			cancelCtx: true,
			validErr:  context.Canceled,
			severity:  validationShutdown,
			wantReorg: false,
			wantErr:   context.Canceled,
			wantFatal: false,
		},
		{
			name:        "non-canceled error with live context calls possiblyFatal",
			cancelCtx:   false,
			validErr:    errors.New("validation execution failed"),
			severity:    validationFatal,
			wantReorg:   true,
			wantErr:     nil,
			wantFatal:   true,
			fatalTarget: nil, // checked by requireFatalError with nil target
		},
		{
			name:      "canceled error with live context treated as transient (spawner shutdown race)",
			cancelCtx: false,
			validErr:  context.Canceled,
			severity:  validationTransient,
			wantReorg: true,
			wantErr:   nil,
			wantFatal: false,
		},
		{
			name:      "wrapped canceled error with live context treated as transient",
			cancelCtx: false,
			validErr:  fmt.Errorf("spawner died: %w", context.Canceled),
			severity:  validationTransient,
			wantReorg: true,
			wantErr:   nil,
			wantFatal: false,
		},
		{
			name:        "deadline exceeded with live context calls possiblyFatal",
			cancelCtx:   false,
			validErr:    context.DeadlineExceeded,
			severity:    validationFatal,
			wantReorg:   true,
			wantErr:     nil,
			wantFatal:   true,
			fatalTarget: nil,
		},
		{
			// Exercises the validationShutdown branch with a live ctx.
			// The error was already classified and logged at the source;
			// advanceValidations returns (nil, nil) so the caller does not
			// re-classify or log duplicates.
			name:      "shutdown severity with live context returns nil without possiblyFatal",
			cancelCtx: false,
			validErr:  context.Canceled,
			severity:  validationShutdown,
			wantReorg: false,
			wantErr:   nil,
			wantFatal: false,
		},
		{
			// Safety-net test: if ErrSeverity is never set (validationUnclassified,
			// the zero value), the default branch must call possiblyFatal.
			name:        "unclassified severity hits default branch and calls possiblyFatal",
			cancelCtx:   false,
			validErr:    errors.New("some error"),
			severity:    validationUnclassified,
			wantReorg:   true,
			wantErr:     nil,
			wantFatal:   true,
			fatalTarget: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			v, fatalCh := newTestValidator(true)
			v.chainCaughtUp = true
			v.validatedA.Store(0)
			v.recordSentA.Store(1)

			status := &validationStatus{}
			status.Status.Store(uint32(ValidationDone))
			status.DoneEntry = &validationDoneEntry{
				Success:     false,
				Err:         tc.validErr,
				ErrSeverity: tc.severity,
				Start:       validator.GoGlobalState{},
			}
			v.validations.Store(arbutil.MessageIndex(0), status)

			ctx := context.Background()
			if tc.cancelCtx {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}

			reorg, err := v.advanceValidations(ctx)

			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("expected error matching %v, got: %v", tc.wantErr, err)
				}
			} else if err != nil {
				t.Fatalf("expected nil error, got: %v", err)
			}

			if tc.wantReorg {
				if reorg == nil {
					t.Fatal("expected non-nil reorg pointer")
				}
				if *reorg != 0 {
					t.Fatalf("expected reorg at position 0, got %v", *reorg)
				}
			} else if reorg != nil {
				t.Fatalf("expected nil reorg pointer, got %v", *reorg)
			}

			if tc.wantFatal {
				requireFatalError(t, fatalCh, tc.fatalTarget)
			} else {
				requireNoFatalError(t, fatalCh)
			}
		})
	}
}

func TestAdvanceValidationsStartStateMismatchTriggersReorg(t *testing.T) {
	v, fatalCh := newTestValidator(true)
	v.chainCaughtUp = true
	v.validatedA.Store(0)
	v.recordSentA.Store(1)

	cancelCalled := false
	status := &validationStatus{}
	status.Status.Store(uint32(ValidationDone))
	status.Cancel = func() { cancelCalled = true }
	status.DoneEntry = &validationDoneEntry{
		Success: true,
		Start:   validator.GoGlobalState{Batch: 99, PosInBatch: 0},
		End:     validator.GoGlobalState{Batch: 100, PosInBatch: 0},
	}
	v.validations.Store(arbutil.MessageIndex(0), status)
	v.lastValidGS = validator.GoGlobalState{Batch: 0, PosInBatch: 0}

	reorg, err := v.advanceValidations(context.Background())
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if reorg == nil {
		t.Fatal("expected non-nil reorg pointer for start-state mismatch")
	}
	if *reorg != 0 {
		t.Fatalf("expected reorg at position 0, got %v", *reorg)
	}
	if !cancelCalled {
		t.Fatal("expected Cancel to be called on mismatched entry")
	}
	requireNoFatalError(t, fatalCh)
}

// TestClassifyValidationErrorSpawnerShutdownRace verifies that
// classifyValidationError returns the correct severity when a validation
// spawner is stopped before the block validator's context is canceled.
// During this window the error is context.Canceled but ctx.Err() is nil.
// The function should return validationTransient (not validationFatal),
// so that the caller does NOT increment the failed-validations counter.
func TestClassifyValidationErrorSpawnerShutdownRace(t *testing.T) {
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	deadlineCtx, deadlineCancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer deadlineCancel()
	liveCtx := context.Background()

	tests := []struct {
		name string
		ctx  context.Context
		err  error
		want validationErrorSeverity
	}{
		{
			name: "canceled ctx + canceled err = shutdown",
			ctx:  canceledCtx,
			err:  context.Canceled,
			want: validationShutdown,
		},
		{
			name: "canceled ctx + wrapped canceled err = shutdown",
			ctx:  canceledCtx,
			err:  fmt.Errorf("spawner: %w", context.Canceled),
			want: validationShutdown,
		},
		{
			name: "live ctx + canceled err = transient (spawner shutdown race)",
			ctx:  liveCtx,
			err:  context.Canceled,
			want: validationTransient,
		},
		{
			name: "live ctx + wrapped canceled err = transient",
			ctx:  liveCtx,
			err:  fmt.Errorf("spawner: %w", context.Canceled),
			want: validationTransient,
		},
		{
			name: "live ctx + real error = fatal",
			ctx:  liveCtx,
			err:  errors.New("validation execution failed"),
			want: validationFatal,
		},
		{
			name: "canceled ctx + real error = fatal",
			ctx:  canceledCtx,
			err:  errors.New("disk full"),
			want: validationFatal,
		},
		{
			name: "canceled ctx + deadline exceeded err = fatal",
			ctx:  canceledCtx,
			err:  context.DeadlineExceeded,
			want: validationFatal,
		},
		{
			name: "deadline ctx + canceled err = transient",
			ctx:  deadlineCtx,
			err:  context.Canceled,
			want: validationTransient,
		},
		{
			name: "live ctx + deadline exceeded = fatal",
			ctx:  liveCtx,
			err:  context.DeadlineExceeded,
			want: validationFatal,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := classifyValidationError(tc.ctx, tc.err, "test")
			if got != tc.want {
				t.Fatalf("classifyValidationError() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestPossiblyFatalNonFatalConfig(t *testing.T) {
	v, fatalCh := newTestValidator(false)

	// With FailureIsFatal=false, a real error should be logged but not sent to fatalErr.
	v.possiblyFatal(errors.New("non-fatal validation error"))
	requireNoFatalError(t, fatalCh)
}

// failingDB wraps an ethdb.Database and makes Put return an error.
type failingDB struct {
	ethdb.Database
	putErr error
}

func (f *failingDB) Put(key []byte, value []byte) error {
	return f.putErr
}

func newTestValidatorWithDB(t *testing.T, failureIsFatal bool, db ethdb.Database) (*BlockValidator, chan error) {
	t.Helper()
	v, fatalCh := newTestValidator(failureIsFatal)
	v.StatelessBlockValidator = &StatelessBlockValidator{db: db}
	return v, fatalCh
}

func TestInitAssumeValidDBFailure(t *testing.T) {
	dbErr := errors.New("disk full")
	db := &failingDB{Database: rawdb.NewMemoryDatabase(), putErr: dbErr}
	v, _ := newTestValidatorWithDB(t, true, db)

	gs := validator.GoGlobalState{Batch: 5, PosInBatch: 3}
	err := v.InitAssumeValid(gs)
	if err == nil {
		t.Fatal("expected error from InitAssumeValid when DB write fails")
	}
	if !errors.Is(err, dbErr) {
		t.Fatalf("expected error wrapping dbErr, got: %v", err)
	}
}

func TestWriteLastValidatedOrderingOnSuccess(t *testing.T) {
	db := rawdb.NewMemoryDatabase()
	v, _ := newTestValidatorWithDB(t, true, db)

	gs := validator.GoGlobalState{Batch: 5, PosInBatch: 3}
	err := v.writeLastValidated(gs, nil)
	if err != nil {
		t.Fatalf("writeLastValidated failed: %v", err)
	}
	if v.lastValidGS != gs {
		t.Fatalf("expected lastValidGS=%v, got %v", gs, v.lastValidGS)
	}

	// Verify the DB has the data by reading it back.
	info, err := ReadLastValidatedInfo(db)
	if err != nil {
		t.Fatalf("ReadLastValidatedInfo failed: %v", err)
	}
	if info.GlobalState != gs {
		t.Fatalf("expected persisted GlobalState=%v, got %v", gs, info.GlobalState)
	}
}

func TestWriteLastValidatedOrderingOnDBFailure(t *testing.T) {
	dbErr := errors.New("disk full")
	db := &failingDB{Database: rawdb.NewMemoryDatabase(), putErr: dbErr}
	v, _ := newTestValidatorWithDB(t, true, db)

	originalGS := validator.GoGlobalState{Batch: 1, PosInBatch: 0}
	v.lastValidGS = originalGS

	newGS := validator.GoGlobalState{Batch: 5, PosInBatch: 3}
	err := v.writeLastValidated(newGS, nil)
	if !errors.Is(err, dbErr) {
		t.Fatalf("expected db error, got: %v", err)
	}
	// lastValidGS must NOT have been updated since the DB write failed.
	if v.lastValidGS != originalGS {
		t.Fatalf("lastValidGS was updated despite DB failure: expected %v, got %v", originalGS, v.lastValidGS)
	}
}

func TestAdvanceValidationsDBWriteFailureEscalates(t *testing.T) {
	dbErr := errors.New("disk full")
	db := &failingDB{Database: rawdb.NewMemoryDatabase(), putErr: dbErr}
	v, fatalCh := newTestValidatorWithDB(t, true, db)
	v.chainCaughtUp = true
	v.validatedA.Store(0)
	v.recordSentA.Store(1)

	// Set up a successful validation entry.
	gs := validator.GoGlobalState{Batch: 1, PosInBatch: 0}
	status := &validationStatus{}
	status.Status.Store(uint32(ValidationDone))
	status.DoneEntry = &validationDoneEntry{
		Success:         true,
		Start:           gs,
		End:             validator.GoGlobalState{Batch: 1, PosInBatch: 1},
		WasmModuleRoots: []common.Hash{},
	}
	v.validations.Store(arbutil.MessageIndex(0), status)
	v.lastValidGS = gs

	reorg, err := v.advanceValidations(context.Background())
	if err == nil {
		t.Fatal("expected error from advanceValidations when DB write fails")
	}
	if !errors.Is(err, dbErr) {
		t.Fatalf("expected error wrapping dbErr, got: %v", err)
	}
	if reorg != nil {
		t.Fatalf("expected nil reorg on DB error, got: %v", *reorg)
	}

	// The error should propagate to possiblyFatal via handleValidationResult.
	// Call handleValidationResult to verify the full chain.
	v.handleValidationResult(context.Background(), reorg, err, "test")
	requireFatalError(t, fatalCh, dbErr)
}

func TestReorgDBWriteFailureEscalates(t *testing.T) {
	// When validatedA > count, Reorg calls writeLastValidated. If the DB
	// write fails, Reorg must call possiblyFatal and return the error.
	dbErr := errors.New("disk full")
	db := &failingDB{Database: rawdb.NewMemoryDatabase(), putErr: dbErr}

	streamer := &mockStreamer{
		results: map[arbutil.MessageIndex]*execution.MessageResult{
			0: {BlockHash: common.HexToHash("0xaa"), SendRoot: common.HexToHash("0xbb")},
		},
		messages: map[arbutil.MessageIndex]*arbostypes.MessageWithMetadata{
			0: {DelayedMessagesRead: 1},
		},
	}

	v, fatalCh := newTestValidatorWithDB(t, true, db)
	v.StatelessBlockValidator = &StatelessBlockValidator{
		db:       db,
		streamer: streamer,
	}
	v.chainCaughtUp = true
	v.createNodesChan = make(chan struct{}, 1)
	// Set validatedA > count to trigger the writeLastValidated path.
	v.createdA.Store(2)
	v.validatedA.Store(2)

	originalGS := validator.GoGlobalState{Batch: 3, PosInBatch: 0}
	v.lastValidGS = originalGS

	err := v.Reorg(context.Background(), 1)
	if err == nil {
		t.Fatal("expected error from Reorg when DB write fails")
	}
	if !errors.Is(err, dbErr) {
		t.Fatalf("expected error wrapping dbErr, got: %v", err)
	}
	// lastValidGS and validatedA must NOT have been updated since the DB write failed.
	if v.lastValidGS != originalGS {
		t.Fatalf("lastValidGS was updated despite DB failure: expected %v, got %v", originalGS, v.lastValidGS)
	}
	if v.validatedA.Load() != 2 {
		t.Fatalf("validatedA was changed despite DB failure: expected 2, got %d", v.validatedA.Load())
	}
	requireFatalError(t, fatalCh, dbErr)
}

// failAfterNPutsDB wraps an ethdb.Database and makes Put return an error
// after the first N successful calls.
type failAfterNPutsDB struct {
	ethdb.Database
	putErr      error
	allowedPuts int
	putCount    int
}

func (f *failAfterNPutsDB) Put(key []byte, value []byte) error {
	f.putCount++
	if f.putCount > f.allowedPuts {
		return f.putErr
	}
	return f.Database.Put(key, value)
}

func TestAdvanceValidationsMultipleEntries(t *testing.T) {
	// Verifies the advanceValidations loop correctly chains lastValidGS
	// across multiple entries in a single call. The second entry's Start
	// must match the first entry's End (which becomes lastValidGS after
	// the first iteration's writeLastValidated).
	db := rawdb.NewMemoryDatabase()
	v, fatalCh := newTestValidatorWithDB(t, true, db)
	v.chainCaughtUp = true
	v.validatedA.Store(0)
	v.recordSentA.Store(2)
	v.createNodesChan = make(chan struct{}, 1)
	v.sendRecordChan = make(chan struct{}, 1)
	v.sendValidationsChan = make(chan struct{}, 1)

	gs0 := validator.GoGlobalState{Batch: 0, PosInBatch: 0}
	gs1 := validator.GoGlobalState{Batch: 1, PosInBatch: 0}
	gs2 := validator.GoGlobalState{Batch: 2, PosInBatch: 0}

	v.lastValidGS = gs0

	// Entry at pos 0: Start=gs0, End=gs1
	status0 := &validationStatus{}
	status0.Status.Store(uint32(ValidationDone))
	status0.DoneEntry = &validationDoneEntry{
		Success: true,
		Start:   gs0,
		End:     gs1,
	}
	v.validations.Store(arbutil.MessageIndex(0), status0)

	// Entry at pos 1: Start=gs1 (must match End of entry 0), End=gs2
	status1 := &validationStatus{}
	status1.Status.Store(uint32(ValidationDone))
	status1.DoneEntry = &validationDoneEntry{
		Success: true,
		Start:   gs1,
		End:     gs2,
	}
	v.validations.Store(arbutil.MessageIndex(1), status1)

	reorg, err := v.advanceValidations(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reorg != nil {
		t.Fatalf("unexpected reorg: %v", *reorg)
	}

	// Both entries should have been processed.
	if v.validated() != 2 {
		t.Fatalf("expected validated=2, got %d", v.validated())
	}
	if v.lastValidGS != gs2 {
		t.Fatalf("expected lastValidGS=%v, got %v", gs2, v.lastValidGS)
	}
	requireNoFatalError(t, fatalCh)
}

func TestAdvanceValidationsMultiEntryDBFailureOnSecond(t *testing.T) {
	// First entry succeeds (DB write ok), second entry's DB write fails.
	// Verifies that the first entry's state is committed and the error
	// from the second is returned.
	dbErr := errors.New("disk full")
	db := &failAfterNPutsDB{
		Database:    rawdb.NewMemoryDatabase(),
		putErr:      dbErr,
		allowedPuts: 1, // first writeLastValidated succeeds, second fails
	}
	v, fatalCh := newTestValidatorWithDB(t, true, db)
	v.chainCaughtUp = true
	v.validatedA.Store(0)
	v.recordSentA.Store(2)
	v.createNodesChan = make(chan struct{}, 1)
	v.sendRecordChan = make(chan struct{}, 1)
	v.sendValidationsChan = make(chan struct{}, 1)

	gs0 := validator.GoGlobalState{Batch: 0, PosInBatch: 0}
	gs1 := validator.GoGlobalState{Batch: 1, PosInBatch: 0}
	gs2 := validator.GoGlobalState{Batch: 2, PosInBatch: 0}

	v.lastValidGS = gs0

	status0 := &validationStatus{}
	status0.Status.Store(uint32(ValidationDone))
	status0.DoneEntry = &validationDoneEntry{
		Success: true,
		Start:   gs0,
		End:     gs1,
	}
	v.validations.Store(arbutil.MessageIndex(0), status0)

	status1 := &validationStatus{}
	status1.Status.Store(uint32(ValidationDone))
	status1.DoneEntry = &validationDoneEntry{
		Success: true,
		Start:   gs1,
		End:     gs2,
	}
	v.validations.Store(arbutil.MessageIndex(1), status1)

	reorg, err := v.advanceValidations(context.Background())
	if err == nil {
		t.Fatal("expected error from second entry's DB write failure")
	}
	if !errors.Is(err, dbErr) {
		t.Fatalf("expected error wrapping dbErr, got: %v", err)
	}
	if reorg != nil {
		t.Fatalf("expected nil reorg on DB error, got: %v", *reorg)
	}

	// First entry should have been committed before the second failed.
	if v.validated() != 1 {
		t.Fatalf("expected validated=1 (first entry committed), got %d", v.validated())
	}
	if v.lastValidGS != gs1 {
		t.Fatalf("expected lastValidGS=%v (first entry's End), got %v", gs1, v.lastValidGS)
	}

	// The error propagates to handleValidationResult which calls possiblyFatal.
	v.handleValidationResult(context.Background(), reorg, err, "test")
	requireFatalError(t, fatalCh, dbErr)
}

func TestUpdateLatestStakedDBWriteFailureNotCaughtUp(t *testing.T) {
	// When chainCaughtUp=false, UpdateLatestStaked calls writeLastValidated
	// directly. On failure it must call possiblyFatal.
	dbErr := errors.New("disk full")
	db := &failingDB{Database: rawdb.NewMemoryDatabase(), putErr: dbErr}
	v, fatalCh := newTestValidatorWithDB(t, true, db)
	v.chainCaughtUp = false
	v.validatedA.Store(0)
	// lastValidGS at zero value, so any non-zero globalState is "new".

	gs := validator.GoGlobalState{Batch: 1, PosInBatch: 0}
	v.UpdateLatestStaked(1, gs)

	requireFatalError(t, fatalCh, dbErr)
	// lastValidGS must NOT have been updated since the DB write failed.
	if v.lastValidGS != (validator.GoGlobalState{}) {
		t.Fatalf("lastValidGS was updated despite DB failure: got %v", v.lastValidGS)
	}
}

func TestUpdateLatestStakedGetMessageFailureCaughtUp(t *testing.T) {
	// When chainCaughtUp=true, if GetMessage fails, UpdateLatestStaked must
	// call possiblyFatal rather than silently abandoning the operation.
	streamer := &mockStreamer{
		// No messages — GetMessage will fail for any index.
		messages: map[arbutil.MessageIndex]*arbostypes.MessageWithMetadata{},
	}
	v, fatalCh := newTestValidator(true)
	v.StatelessBlockValidator = &StatelessBlockValidator{streamer: streamer}
	v.chainCaughtUp = true
	v.validatedA.Store(0)

	gs := validator.GoGlobalState{Batch: 1, PosInBatch: 0}
	v.UpdateLatestStaked(1, gs)

	requireFatalError(t, fatalCh, nil)
}

func TestUpdateLatestStakedDBWriteFailureCaughtUp(t *testing.T) {
	// When chainCaughtUp=true, UpdateLatestStaked calls writeLastValidated
	// before storing validatedA. On DB failure it must call possiblyFatal
	// and return early without advancing validatedA.
	dbErr := errors.New("disk full")
	db := &failingDB{Database: rawdb.NewMemoryDatabase(), putErr: dbErr}

	streamer := &mockStreamer{
		messages: map[arbutil.MessageIndex]*arbostypes.MessageWithMetadata{
			0: {DelayedMessagesRead: 1},
		},
	}
	v, fatalCh := newTestValidatorWithDB(t, true, db)
	v.StatelessBlockValidator = &StatelessBlockValidator{
		db:       db,
		streamer: streamer,
	}
	v.chainCaughtUp = true
	v.validatedA.Store(0)
	v.createNodesChan = make(chan struct{}, 1)

	gs := validator.GoGlobalState{Batch: 1, PosInBatch: 0}
	v.UpdateLatestStaked(1, gs)

	requireFatalError(t, fatalCh, dbErr)
	// validatedA is stored after writeLastValidated; DB failure returns early.
	if v.validatedA.Load() != 0 {
		t.Fatalf("expected validatedA=0 (DB write failed before store), got %d", v.validatedA.Load())
	}
}

func TestReorgWithMultiplePipelineStages(t *testing.T) {
	// Realistic reorg scenario: validator has 5 entries at different pipeline
	// stages, then reorgs to keep only the first 3 (positions 0-2).
	//
	// Pipeline state before reorg:
	//   pos 0, 1: already validated and removed from map (validatedA=2)
	//   pos 2:    SendingValidation (in-flight)
	//   pos 3:    Prepared (ready to send)
	//   pos 4:    Created (just created)
	//
	// After Reorg(ctx, 3):
	//   - Entries at positions 3 and 4 deleted, Cancel called
	//   - Entry at position 2 preserved (below reorg point)
	//   - Counters capped: createdA=3, recordSentA=3, lastValidationSentA=3
	//   - validatedA=2 unchanged (already below count)
	//   - nextCreateStartGS rebuilt from streamer data at position 2
	db := rawdb.NewMemoryDatabase()

	// Batch 0 contains messages 0-4 (5 messages total).
	tracker := &mockInboxTracker{
		batchMsgCounts: map[uint64]arbutil.MessageIndex{0: 5},
	}
	streamer := &mockStreamer{
		results: map[arbutil.MessageIndex]*execution.MessageResult{
			0: {BlockHash: common.HexToHash("0x10"), SendRoot: common.HexToHash("0x11")},
			1: {BlockHash: common.HexToHash("0x20"), SendRoot: common.HexToHash("0x21")},
			2: {BlockHash: common.HexToHash("0x30"), SendRoot: common.HexToHash("0x31")},
		},
		messages: map[arbutil.MessageIndex]*arbostypes.MessageWithMetadata{
			0: {DelayedMessagesRead: 0},
			1: {DelayedMessagesRead: 0},
			2: {DelayedMessagesRead: 2},
		},
	}

	v, fatalCh := newTestValidatorWithDB(t, true, db)
	v.StatelessBlockValidator = &StatelessBlockValidator{
		db:           db,
		streamer:     streamer,
		inboxTracker: tracker,
	}
	v.chainCaughtUp = true
	v.createNodesChan = make(chan struct{}, 1)

	// Set pipeline counters: 5 created, 4 recorded, 3 sent, 2 validated.
	v.createdA.Store(5)
	v.recordSentA.Store(4)
	v.lastValidationSentA.Store(3)
	v.validatedA.Store(2)
	v.lastValidGS = validator.GoGlobalState{Batch: 0, PosInBatch: 2}

	// Populate entries at positions 2-4 (0 and 1 already advanced past).
	canceled := map[arbutil.MessageIndex]bool{}

	status2 := &validationStatus{}
	status2.Status.Store(uint32(SendingValidation))
	status2.Cancel = func() { canceled[2] = true }
	v.validations.Store(arbutil.MessageIndex(2), status2)

	status3 := &validationStatus{}
	status3.Status.Store(uint32(Prepared))
	status3.Cancel = func() { canceled[3] = true }
	v.validations.Store(arbutil.MessageIndex(3), status3)

	status4 := &validationStatus{}
	status4.Status.Store(uint32(Created))
	status4.Cancel = func() { canceled[4] = true }
	v.validations.Store(arbutil.MessageIndex(4), status4)

	// Reorg to keep positions 0-2.
	err := v.Reorg(context.Background(), 3)
	if err != nil {
		t.Fatalf("Reorg(ctx, 3) failed: %v", err)
	}

	// Entries at positions 3 and 4 should be deleted and canceled.
	if !canceled[3] {
		t.Error("expected Cancel called on position 3")
	}
	if !canceled[4] {
		t.Error("expected Cancel called on position 4")
	}
	// Entry at position 2 should NOT be canceled (below reorg point).
	if canceled[2] {
		t.Error("position 2 should not have been canceled")
	}

	// Verify map: positions 3 and 4 deleted, position 2 preserved.
	if _, found := v.validations.Load(arbutil.MessageIndex(2)); !found {
		t.Error("position 2 should be preserved (below reorg point)")
	}
	if _, found := v.validations.Load(arbutil.MessageIndex(3)); found {
		t.Error("position 3 should have been deleted")
	}
	if _, found := v.validations.Load(arbutil.MessageIndex(4)); found {
		t.Error("position 4 should have been deleted")
	}

	// Verify counters are capped to count=3.
	if v.createdA.Load() != 3 {
		t.Errorf("expected createdA=3, got %d", v.createdA.Load())
	}
	if v.recordSentA.Load() != 3 {
		t.Errorf("expected recordSentA=3 (capped from 4), got %d", v.recordSentA.Load())
	}
	if v.lastValidationSentA.Load() != 3 {
		t.Errorf("expected lastValidationSentA=3 (unchanged), got %d", v.lastValidationSentA.Load())
	}
	// validatedA was 2, below count=3, so it stays at 2.
	if v.validatedA.Load() != 2 {
		t.Errorf("expected validatedA=2 (unchanged, below count), got %d", v.validatedA.Load())
	}

	// Verify nextCreateStartGS is rebuilt from result at position 2.
	// count=3, batch 0 has 5 msgs, so endPosition = {batch=0, posInBatch=3}.
	expectedGS := BuildGlobalState(
		execution.MessageResult{BlockHash: common.HexToHash("0x30"), SendRoot: common.HexToHash("0x31")},
		GlobalStatePosition{BatchNumber: 0, PosInBatch: 3},
	)
	if v.nextCreateStartGS != expectedGS {
		t.Errorf("expected nextCreateStartGS=%v, got %v", expectedGS, v.nextCreateStartGS)
	}
	if v.nextCreatePrevDelayed != 2 {
		t.Errorf("expected nextCreatePrevDelayed=2, got %d", v.nextCreatePrevDelayed)
	}
	if !v.nextCreateBatchReread {
		t.Error("expected nextCreateBatchReread=true")
	}

	requireNoFatalError(t, fatalCh)
}

func TestReorgPastValidatedRollsBackPersistedState(t *testing.T) {
	// Deep reorg where validatedA > count: the validator has validated through
	// position 4, but a reorg rolls back to count=2 (keep only positions 0-1).
	// This triggers writeLastValidated to persist the rolled-back state and
	// resets validatedA. Verifies the DB roundtrip and in-memory consistency.
	db := rawdb.NewMemoryDatabase()

	tracker := &mockInboxTracker{
		batchMsgCounts: map[uint64]arbutil.MessageIndex{0: 5},
	}
	streamer := &mockStreamer{
		results: map[arbutil.MessageIndex]*execution.MessageResult{
			1: {BlockHash: common.HexToHash("0x20"), SendRoot: common.HexToHash("0x21")},
		},
		messages: map[arbutil.MessageIndex]*arbostypes.MessageWithMetadata{
			1: {DelayedMessagesRead: 1},
		},
	}

	v, fatalCh := newTestValidatorWithDB(t, true, db)
	v.StatelessBlockValidator = &StatelessBlockValidator{
		db:           db,
		streamer:     streamer,
		inboxTracker: tracker,
	}
	v.chainCaughtUp = true
	v.createNodesChan = make(chan struct{}, 1)

	// All 5 entries validated and advanced past.
	v.createdA.Store(5)
	v.recordSentA.Store(5)
	v.lastValidationSentA.Store(5)
	v.validatedA.Store(5)
	v.lastValidGS = validator.GoGlobalState{Batch: 0, PosInBatch: 5}

	// Reorg to count=2: keep only positions 0 and 1.
	err := v.Reorg(context.Background(), 2)
	if err != nil {
		t.Fatalf("Reorg(ctx, 2) failed: %v", err)
	}

	// validatedA was 5 > count=2, so writeLastValidated was called.
	if v.validatedA.Load() != 2 {
		t.Errorf("expected validatedA=2, got %d", v.validatedA.Load())
	}

	// lastValidGS should be rebuilt from result at position 1 (count-1).
	// count=2, batch 0, posInBatch = 2-0-1 = 1, endPos = {0, 2}.
	expectedGS := BuildGlobalState(
		execution.MessageResult{BlockHash: common.HexToHash("0x20"), SendRoot: common.HexToHash("0x21")},
		GlobalStatePosition{BatchNumber: 0, PosInBatch: 2},
	)
	if v.lastValidGS != expectedGS {
		t.Errorf("expected lastValidGS=%v, got %v", expectedGS, v.lastValidGS)
	}

	// Verify DB was updated: read back and compare.
	info, err := ReadLastValidatedInfo(db)
	if err != nil {
		t.Fatalf("ReadLastValidatedInfo failed: %v", err)
	}
	if info.GlobalState != expectedGS {
		t.Errorf("persisted GlobalState=%v, want %v", info.GlobalState, expectedGS)
	}

	requireNoFatalError(t, fatalCh)
}

func TestReorgDepthScaling(t *testing.T) {
	// Verify Reorg correctly cleans up entries, resets counters, persists
	// rolled-back state, and preserves entries below the reorg point at
	// varying depths across batch boundaries.
	for _, depth := range []uint64{10, 100, 500} {
		t.Run(fmt.Sprintf("depth_%d", depth), func(t *testing.T) {
			db := rawdb.NewMemoryDatabase()
			total := depth + 5        // entries before reorg
			var keep uint64 = 5       // entries to keep
			var batchSize uint64 = 50 // messages per batch

			// Spread messages across multiple batches.
			// This exercises GlobalStatePositionsAtCount across batch boundaries.
			batchMsgCounts := map[uint64]arbutil.MessageIndex{}
			for b := uint64(0); b*batchSize < total; b++ {
				end := (b + 1) * batchSize
				if end > total {
					end = total
				}
				batchMsgCounts[b] = arbutil.MessageIndex(end)
			}

			tracker := &mockInboxTracker{batchMsgCounts: batchMsgCounts}
			keepResult := &execution.MessageResult{
				BlockHash: common.HexToHash("0xaa"),
				SendRoot:  common.HexToHash("0xbb"),
			}
			keepIdx := arbutil.MessageIndex(keep - 1)
			streamer := &mockStreamer{
				results: map[arbutil.MessageIndex]*execution.MessageResult{
					keepIdx: keepResult,
				},
				messages: map[arbutil.MessageIndex]*arbostypes.MessageWithMetadata{
					keepIdx: {DelayedMessagesRead: 3},
				},
			}

			v, fatalCh := newTestValidatorWithDB(t, true, db)
			v.StatelessBlockValidator = &StatelessBlockValidator{
				db:           db,
				streamer:     streamer,
				inboxTracker: tracker,
			}
			v.chainCaughtUp = true
			v.createNodesChan = make(chan struct{}, 1)
			v.prevBatchCache = map[uint64][]byte{99: {1, 2, 3}} // should be cleared

			v.createdA.Store(total)
			v.recordSentA.Store(total)
			v.lastValidationSentA.Store(total)
			v.validatedA.Store(total)
			v.lastValidGS = validator.GoGlobalState{Batch: 99, PosInBatch: 99}

			// Populate entries below keep (should survive).
			for i := arbutil.MessageIndex(0); i < arbutil.MessageIndex(keep); i++ {
				s := &validationStatus{}
				s.Status.Store(uint32(ValidationDone))
				v.validations.Store(i, s)
			}

			// Populate entries from keep to total-1 with Cancel tracking.
			cancelCount := uint64(0)
			for i := arbutil.MessageIndex(keep); i < arbutil.MessageIndex(total); i++ {
				s := &validationStatus{}
				s.Status.Store(uint32(Prepared))
				s.Cancel = func() { cancelCount++ }
				v.validations.Store(i, s)
			}

			err := v.Reorg(context.Background(), arbutil.MessageIndex(keep))
			if err != nil {
				t.Fatalf("Reorg failed: %v", err)
			}

			// All entries above keep canceled and deleted.
			if cancelCount != depth {
				t.Errorf("expected %d cancels, got %d", depth, cancelCount)
			}
			for i := arbutil.MessageIndex(keep); i < arbutil.MessageIndex(total); i++ {
				if _, found := v.validations.Load(i); found {
					t.Fatalf("entry at position %d should have been deleted", i)
				}
			}

			// Entries below keep survive.
			for i := arbutil.MessageIndex(0); i < arbutil.MessageIndex(keep); i++ {
				if _, found := v.validations.Load(i); !found {
					t.Fatalf("entry at position %d should have survived", i)
				}
			}

			// Counters reset.
			if v.createdA.Load() != keep {
				t.Errorf("expected createdA=%d, got %d", keep, v.createdA.Load())
			}
			if v.validatedA.Load() != keep {
				t.Errorf("expected validatedA=%d, got %d", keep, v.validatedA.Load())
			}
			if v.recordSentA.Load() != keep {
				t.Errorf("expected recordSentA=%d, got %d", keep, v.recordSentA.Load())
			}
			if v.lastValidationSentA.Load() != keep {
				t.Errorf("expected lastValidationSentA=%d, got %d", keep, v.lastValidationSentA.Load())
			}

			// lastValidGS rebuilt correctly from chain data.
			// keep=5, batch 0 has 50 msgs, so posInBatch = 5-0-1 = 4, endPos = {0, 5}.
			expectedGS := BuildGlobalState(*keepResult, GlobalStatePosition{BatchNumber: 0, PosInBatch: uint64(keep)})
			if v.lastValidGS != expectedGS {
				t.Errorf("expected lastValidGS=%v, got %v", expectedGS, v.lastValidGS)
			}

			// DB persisted correctly — roundtrip check.
			info, err := ReadLastValidatedInfo(db)
			if err != nil {
				t.Fatalf("ReadLastValidatedInfo failed: %v", err)
			}
			if info.GlobalState != expectedGS {
				t.Errorf("persisted GlobalState=%v, want %v", info.GlobalState, expectedGS)
			}

			// nextCreatePrevDelayed set from chain data.
			if v.nextCreatePrevDelayed != 3 {
				t.Errorf("expected nextCreatePrevDelayed=3, got %d", v.nextCreatePrevDelayed)
			}

			// prevBatchCache cleared.
			if len(v.prevBatchCache) != 0 {
				t.Errorf("expected prevBatchCache cleared, got %d entries", len(v.prevBatchCache))
			}
			if !v.nextCreateBatchReread {
				t.Error("expected nextCreateBatchReread=true")
			}

			requireNoFatalError(t, fatalCh)
		})
	}
}
