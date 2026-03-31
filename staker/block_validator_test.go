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

// newTestValidator creates a BlockValidator with a buffered fatalErr channel
// and the given config values. Returns the validator and the fatal error channel.
func newTestValidator(failureIsFatal bool, validationPoll time.Duration) (*BlockValidator, chan error) {
	fatalCh := make(chan error, 1)
	v := &BlockValidator{
		fatalErr: fatalCh,
	}
	v.config = func() *BlockValidatorConfig {
		return &BlockValidatorConfig{
			ValidationPoll: validationPoll,
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
	v, fatalCh := newTestValidator(true, 0)
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
	v, fatalCh := newTestValidator(true, 0)
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

func TestReorgGuardAllowsTwo(t *testing.T) {
	// Verify count == 2 also passes the guard (boundary sanity check).
	v := &BlockValidator{}
	err := v.Reorg(context.Background(), 2)
	if err != nil {
		t.Fatalf("expected no error for count == 2, got: %v", err)
	}
}

func TestPossiblyFatalSendsAllErrorsWhenNotStopped(t *testing.T) {
	v, fatalCh := newTestValidator(true, 0)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	v.StopWaiter.Start(ctx, v)

	// possiblyFatal does not suppress any error types — context errors are
	// handled upstream in advanceValidations/handleValidationResult instead.
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
			v, fatalCh := newTestValidator(true, 0)

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
	// not for context.DeadlineExceeded (timeout). With chainCaughtUp=false,
	// Reorg returns nil early without side effects. We verify no fatal error
	// is produced (if the skip guard incorrectly matched DeadlineExceeded,
	// Reorg would not be called, but the observable result is the same here).
	v, fatalCh := newTestValidator(true, 0)

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()

	reorgTarget := arbutil.MessageIndex(5)
	v.handleValidationResult(ctx, &reorgTarget, nil, "test")

	requireNoFatalError(t, fatalCh)
}

func TestPossiblyFatalTreatsCanceledAsFatalWithUnstartedStopWaiter(t *testing.T) {
	v, fatalCh := newTestValidator(true, 0)

	v.possiblyFatal(context.Canceled)
	requireFatalError(t, fatalCh, context.Canceled)
}

func TestPossiblyFatalChannelFull(t *testing.T) {
	// When fatalErr already has an error, the second error is dropped
	// (with a log) rather than blocking. Verify the first error is
	// preserved and the second doesn't panic or block.
	v, fatalCh := newTestValidator(true, 0)

	first := errors.New("first error")
	second := errors.New("second error")
	v.possiblyFatal(first)
	v.possiblyFatal(second) // should not block

	requireFatalError(t, fatalCh, first)
	// Channel should now be empty (second was dropped).
	requireNoFatalError(t, fatalCh)
}

func TestPossiblyFatalNilIsNoop(t *testing.T) {
	v, fatalCh := newTestValidator(true, 0)

	v.possiblyFatal(nil)
	requireNoFatalError(t, fatalCh)
}

func TestPossiblyFatalSuppressesAllErrorsWhenStopped(t *testing.T) {
	// After StopAndWait, possiblyFatal should silently return for any error,
	// including real (non-context) errors. This is the primary shutdown
	// suppression mechanism in possiblyFatal itself.
	v, fatalCh := newTestValidator(true, 0)
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

func TestIsShutdownCancellation(t *testing.T) {
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	deadlineCtx, deadlineCancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer deadlineCancel()
	liveCtx := context.Background()

	tests := []struct {
		name string
		ctx  context.Context
		err  error
		want bool
	}{
		{"canceled ctx + canceled err", canceledCtx, context.Canceled, true},
		{"canceled ctx + wrapped canceled err", canceledCtx, fmt.Errorf("spawner died: %w", context.Canceled), true},
		{"canceled ctx + deadline exceeded err", canceledCtx, context.DeadlineExceeded, false},
		{"canceled ctx + other err", canceledCtx, errors.New("disk full"), false},
		{"canceled ctx + nil err", canceledCtx, nil, false},
		{"deadline exceeded ctx + canceled err", deadlineCtx, context.Canceled, false},
		{"live ctx + canceled err", liveCtx, context.Canceled, false},
		{"live ctx + other err", liveCtx, errors.New("some error"), false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := isShutdownCancellation(tc.ctx, tc.err)
			if got != tc.want {
				t.Fatalf("isShutdownCancellation() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestAdvanceValidationsFailedEntry(t *testing.T) {
	// Table-driven tests for advanceValidations behavior when a validation
	// entry has failed (Success=false). The key axes are:
	//   - validation error type (context.Canceled, wrapped Canceled, other)
	//   - context state (live vs cancelled)
	//
	// Note: with a pre-cancelled context, advanceValidations returns early at
	// the ctx.Err() check before reaching the isShutdownCancellation logic.
	// The isShutdownCancellation function itself is tested in
	// TestIsShutdownCancellation.
	tests := []struct {
		name        string
		cancelCtx   bool // whether to cancel the context before calling
		validErr    error
		wantReorg   bool  // expect non-nil reorg pointer
		wantErr     error // if non-nil, expect errors.Is match on returned err
		wantFatal   bool  // expect an error on fatalCh
		fatalTarget error // if non-nil, the fatal error must match via errors.Is
	}{
		{
			name:      "cancelled context returns early with context error",
			cancelCtx: true,
			validErr:  context.Canceled,
			wantReorg: false,
			wantErr:   context.Canceled,
			wantFatal: false,
		},
		{
			name:        "non-canceled error with live context calls possiblyFatal",
			cancelCtx:   false,
			validErr:    errors.New("validation execution failed"),
			wantReorg:   true,
			wantErr:     nil,
			wantFatal:   true,
			fatalTarget: nil, // checked by requireFatalError with nil target
		},
		{
			name:      "canceled error with live context treated as transient (spawner shutdown race)",
			cancelCtx: false,
			validErr:  context.Canceled,
			wantReorg: true,
			wantErr:   nil,
			wantFatal: false,
		},
		{
			name:      "wrapped canceled error with live context treated as transient",
			cancelCtx: false,
			validErr:  fmt.Errorf("spawner died: %w", context.Canceled),
			wantReorg: true,
			wantErr:   nil,
			wantFatal: false,
		},
		{
			name:        "deadline exceeded with live context calls possiblyFatal",
			cancelCtx:   false,
			validErr:    context.DeadlineExceeded,
			wantReorg:   true,
			wantErr:     nil,
			wantFatal:   true,
			fatalTarget: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			v, fatalCh := newTestValidator(true, 0)
			v.chainCaughtUp = true
			v.validatedA.Store(0)
			v.recordSentA.Store(1)

			status := &validationStatus{}
			status.Status.Store(uint32(ValidationDone))
			status.DoneEntry = &validationDoneEntry{
				Success: false,
				Err:     tc.validErr,
				Start:   validator.GoGlobalState{},
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

// TestAdvanceValidationsSpawnerShutdownRace exercises the race condition where
// validation spawners (tracked children of the block validator) are stopped
// before the block validator's own context is canceled.  During this window,
// in-flight validations fail with context.Canceled while the block validator's
// ctx is still live.  Before the fix, this caused possiblyFatal to fire and
// send a spurious fatal error.
func TestAdvanceValidationsSpawnerShutdownRace(t *testing.T) {
	v, fatalCh := newTestValidator(true, 100*time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	v.StopWaiter.Start(ctx, v)
	v.chainCaughtUp = true
	v.validatedA.Store(0)
	v.recordSentA.Store(1)

	// Simulate a validation that completed with context.Canceled — as happens
	// when a spawner child is stopped while a validation is in flight.
	status := &validationStatus{}
	status.Status.Store(uint32(ValidationDone))
	status.DoneEntry = &validationDoneEntry{
		Success: false,
		Err:     context.Canceled,
		Start:   validator.GoGlobalState{},
	}
	v.validations.Store(arbutil.MessageIndex(0), status)

	// The block validator's context is still live (not canceled) — this is
	// the key condition that triggers the race.  advanceValidations must
	// treat context.Canceled as transient and retry (return &pos) without
	// calling possiblyFatal.
	reorg, err := v.advanceValidations(ctx)
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if reorg == nil {
		t.Fatal("expected non-nil reorg pointer (retry position)")
	}
	if *reorg != 0 {
		t.Fatalf("expected retry at position 0, got %v", *reorg)
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

// TestFailedValidationsCounterNotIncrementedForTransient verifies the
// interaction between classifyValidationError and the counter-increment
// guard in sendValidations' untracked thread.  Before the fix, the guard
// used `!= validationShutdown`, which caused the counter to be incremented
// for transient (spawner-shutdown-race) cancellations.  After the fix it
// uses `== validationFatal`, so the counter is only incremented for real
// failures.
func TestFailedValidationsCounterNotIncrementedForTransient(t *testing.T) {
	liveCtx := context.Background()

	// Simulates the guard condition in the untracked validation thread:
	//   if classifyValidationError(ctx, err, label) == validationFatal { counter.Inc(1) }
	shouldIncrement := func(ctx context.Context, err error) bool {
		return classifyValidationError(ctx, err, "test") == validationFatal
	}

	// Spawner shutdown race: context.Canceled with live ctx should NOT increment.
	if shouldIncrement(liveCtx, context.Canceled) {
		t.Error("counter would be incremented for context.Canceled with live ctx (spawner shutdown race)")
	}
	if shouldIncrement(liveCtx, fmt.Errorf("wrapped: %w", context.Canceled)) {
		t.Error("counter would be incremented for wrapped context.Canceled with live ctx")
	}

	// Real errors SHOULD increment.
	if !shouldIncrement(liveCtx, errors.New("validation execution failed")) {
		t.Error("counter should be incremented for real validation errors")
	}
	if !shouldIncrement(liveCtx, context.DeadlineExceeded) {
		t.Error("counter should be incremented for deadline exceeded")
	}
}

func TestPossiblyFatalNonFatalConfig(t *testing.T) {
	v, fatalCh := newTestValidator(false, 0)

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
	v, fatalCh := newTestValidator(failureIsFatal, 0)
	v.StatelessBlockValidator = &StatelessBlockValidator{db: db}
	return v, fatalCh
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
	// lastValidGS must NOT have been updated since the DB write failed.
	if v.lastValidGS != originalGS {
		t.Fatalf("lastValidGS was updated despite DB failure: expected %v, got %v", originalGS, v.lastValidGS)
	}
	requireFatalError(t, fatalCh, dbErr)
}
