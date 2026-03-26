package staker

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/execution"
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

func TestReorgGuardRejectsZero(t *testing.T) {
	v := &BlockValidator{}
	err := v.Reorg(context.Background(), 0)
	if err == nil {
		t.Fatal("expected error for count == 0")
	}
	if err.Error() != "cannot reorg out genesis" {
		t.Fatalf("unexpected error: %v", err)
	}
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
	fatalCh := make(chan error, 1)
	v := &BlockValidator{
		StatelessBlockValidator: &StatelessBlockValidator{
			streamer: streamer,
		},
		chainCaughtUp:   true,
		createNodesChan: make(chan struct{}, 1),
		fatalErr:        fatalCh,
	}
	v.config = func() *BlockValidatorConfig {
		return &BlockValidatorConfig{FailureIsFatal: true}
	}
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

	// No fatal error should have been produced.
	select {
	case err := <-fatalCh:
		t.Fatalf("unexpected fatal error: %v", err)
	default:
	}
}

func TestReorgGuardAllowsTwo(t *testing.T) {
	// Verify count == 2 also passes the guard (boundary sanity check).
	v := &BlockValidator{}
	err := v.Reorg(context.Background(), 2)
	if err != nil {
		t.Fatalf("expected no error for count == 2, got: %v", err)
	}
}

func TestPossiblyFatalSuppressesContextErrors(t *testing.T) {
	fatalCh := make(chan error, 1)
	v := &BlockValidator{
		fatalErr: fatalCh,
	}
	v.config = func() *BlockValidatorConfig {
		return &BlockValidatorConfig{FailureIsFatal: true}
	}
	// Start the embedded StopWaiter with a cancelled context so
	// possiblyFatal's GetContextSafe check sees ctx.Err() != nil.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	v.StopWaiter.Start(ctx, v)

	// context.Canceled should be suppressed when lifecycle context is cancelled
	v.possiblyFatal(context.Canceled)
	select {
	case err := <-fatalCh:
		t.Fatalf("context.Canceled should not be fatal, got: %v", err)
	default:
	}

	// Wrapped context.Canceled should also be suppressed (errors.Is handles wrapping)
	v.possiblyFatal(fmt.Errorf("validation failed: %w", context.Canceled))
	select {
	case err := <-fatalCh:
		t.Fatalf("wrapped context.Canceled should not be fatal, got: %v", err)
	default:
	}

	// context.DeadlineExceeded IS fatal — real timeouts should not be silently suppressed
	v.possiblyFatal(context.DeadlineExceeded)
	select {
	case err := <-fatalCh:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("expected DeadlineExceeded, got: %v", err)
		}
	default:
		t.Fatal("expected context.DeadlineExceeded to be sent to fatalErr")
	}

	// Wrapped context.DeadlineExceeded should also be fatal
	wrappedDeadline := fmt.Errorf("timed out: %w", context.DeadlineExceeded)
	v.possiblyFatal(wrappedDeadline)
	select {
	case err := <-fatalCh:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("expected wrapped DeadlineExceeded, got: %v", err)
		}
	default:
		t.Fatal("expected wrapped context.DeadlineExceeded to be sent to fatalErr")
	}

	// A real error should be sent to fatalErr
	realErr := errors.New("validation failed")
	v.possiblyFatal(realErr)
	select {
	case err := <-fatalCh:
		if !errors.Is(err, realErr) {
			t.Fatalf("expected realErr, got: %v", err)
		}
	default:
		t.Fatal("expected real error to be sent to fatalErr")
	}
}

func TestHandleValidationResultSkipsReorgDuringShutdown(t *testing.T) {
	// When the context is cancelled (shutdown), handleValidationResult should
	// skip the reorg and return without calling Reorg or possiblyFatal.
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // simulate shutdown

	fatalCh := make(chan error, 1)
	v := &BlockValidator{
		fatalErr: fatalCh,
	}
	v.config = func() *BlockValidatorConfig {
		return &BlockValidatorConfig{
			ValidationPoll: 0,
			FailureIsFatal: true,
		}
	}

	reorgTarget := arbutil.MessageIndex(5)
	result := v.handleValidationResult(ctx, &reorgTarget, nil, "test")
	if result != 0 {
		t.Errorf("expected ValidationPoll duration (0), got %v", result)
	}

	// No fatal error should have been produced.
	select {
	case err := <-fatalCh:
		t.Fatalf("unexpected fatal error during shutdown reorg skip: %v", err)
	default:
	}
}

func TestHandleValidationResultLogsButDoesNotFatal(t *testing.T) {
	// Errors from advanceValidations/sendValidations are transient and retried
	// on the next poll. They are intentionally NOT sent to fatalErr — only
	// Reorg errors go through possiblyFatal. This matches the original upstream
	// behavior where iterativeValidationProgress just called log.Error.
	fatalCh := make(chan error, 1)
	v := &BlockValidator{
		fatalErr: fatalCh,
	}
	v.config = func() *BlockValidatorConfig {
		return &BlockValidatorConfig{
			ValidationPoll: 0,
			FailureIsFatal: true,
		}
	}

	realErr := errors.New("validation data corruption")
	result := v.handleValidationResult(context.Background(), nil, realErr, "test")
	if result != 0 {
		t.Errorf("expected ValidationPoll duration (0), got %v", result)
	}

	// The error should NOT be sent to fatalErr — it is logged and retried.
	select {
	case err := <-fatalCh:
		t.Fatalf("transient error should not be fatal, got: %v", err)
	default:
	}
}

func TestHandleValidationResultSuppressesCanceledDuringShutdown(t *testing.T) {
	// When the lifecycle context is cancelled (shutdown) and the error is
	// context.Canceled, handleValidationResult should take the Debug-level
	// suppression path (not the Error-level path). Neither path sends to
	// fatalErr, but the distinction matters for log noise during shutdown.
	fatalCh := make(chan error, 1)
	v := &BlockValidator{
		fatalErr: fatalCh,
	}
	v.config = func() *BlockValidatorConfig {
		return &BlockValidatorConfig{
			ValidationPoll: 0,
			FailureIsFatal: true,
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // simulate shutdown

	// context.Canceled with cancelled ctx hits the log.Debug suppression path.
	v.handleValidationResult(ctx, nil, context.Canceled, "test")
	select {
	case err := <-fatalCh:
		t.Fatalf("context.Canceled during shutdown should not be fatal, got: %v", err)
	default:
	}

	// Wrapped context.Canceled with cancelled ctx also hits the suppression path.
	v.handleValidationResult(ctx, nil, fmt.Errorf("wrapped: %w", context.Canceled), "test")
	select {
	case err := <-fatalCh:
		t.Fatalf("wrapped context.Canceled during shutdown should not be fatal, got: %v", err)
	default:
	}
}

func TestHandleValidationResultDoesNotEscalateNonCanceledErrors(t *testing.T) {
	// handleValidationResult never sends errors to fatalErr (they are
	// transient and retried). This test verifies that context.DeadlineExceeded
	// and other errors are logged (at Error level) but not escalated.
	fatalCh := make(chan error, 1)
	v := &BlockValidator{
		fatalErr: fatalCh,
	}
	v.config = func() *BlockValidatorConfig {
		return &BlockValidatorConfig{
			ValidationPoll: 0,
			FailureIsFatal: true,
		}
	}

	// DeadlineExceeded is NOT suppressed — it goes through log.Error —
	// but it is still not fatal because handleValidationResult never
	// sends to fatalErr regardless.
	v.handleValidationResult(context.Background(), nil, context.DeadlineExceeded, "test")
	select {
	case err := <-fatalCh:
		t.Fatalf("context.DeadlineExceeded should not be fatal in handleValidationResult, got: %v", err)
	default:
	}

	// context.Canceled with a LIVE context also hits log.Error (not suppressed),
	// because the suppression requires ctx.Err() != nil.
	v.handleValidationResult(context.Background(), nil, context.Canceled, "test")
	select {
	case err := <-fatalCh:
		t.Fatalf("context.Canceled with live context should not be fatal, got: %v", err)
	default:
	}
}

func TestHandleValidationResultReorgSucceeds(t *testing.T) {
	// Exercises the happy-path: handleValidationResult receives a reorg
	// pointer on a live context, calls Reorg, and Reorg succeeds.
	// Uses chainCaughtUp=false so Reorg returns nil early after the guard.
	fatalCh := make(chan error, 1)
	v := &BlockValidator{
		fatalErr: fatalCh,
	}
	v.config = func() *BlockValidatorConfig {
		return &BlockValidatorConfig{
			ValidationPoll: 0,
			FailureIsFatal: true,
		}
	}

	reorgTarget := arbutil.MessageIndex(5)
	result := v.handleValidationResult(context.Background(), &reorgTarget, nil, "test")
	if result != 0 {
		t.Errorf("expected ValidationPoll duration (0), got %v", result)
	}

	// No fatal error should have been produced.
	select {
	case err := <-fatalCh:
		t.Fatalf("unexpected fatal error from successful reorg: %v", err)
	default:
	}
}

func TestHandleValidationResultReorgFailure(t *testing.T) {
	// When Reorg fails, the error should go through possiblyFatal.
	// Use count=1 (which has a fast path in GlobalStatePositionsAtCount)
	// but leave the streamer empty so ResultAtMessageIndex(0) fails.
	streamer := &mockStreamer{
		results:  map[arbutil.MessageIndex]*execution.MessageResult{},
		messages: map[arbutil.MessageIndex]*arbostypes.MessageWithMetadata{},
	}
	fatalCh := make(chan error, 1)
	v := &BlockValidator{
		StatelessBlockValidator: &StatelessBlockValidator{
			streamer: streamer,
		},
		chainCaughtUp:   true,
		createNodesChan: make(chan struct{}, 1),
		fatalErr:        fatalCh,
	}
	v.config = func() *BlockValidatorConfig {
		return &BlockValidatorConfig{
			ValidationPoll: 0,
			FailureIsFatal: true,
		}
	}
	// Set createdA >= count so we enter the full Reorg path.
	v.createdA.Store(1)

	reorgTarget := arbutil.MessageIndex(1)
	result := v.handleValidationResult(context.Background(), &reorgTarget, nil, "test")
	if result != 0 {
		t.Errorf("expected ValidationPoll duration (0), got %v", result)
	}

	// The Reorg failure (streamer missing result at index 0) should have
	// been sent through possiblyFatal to fatalErr.
	select {
	case err := <-fatalCh:
		if err == nil {
			t.Fatal("expected non-nil fatal error from reorg failure")
		}
	default:
		t.Fatal("expected reorg failure to produce a fatal error")
	}
}

func TestPossiblyFatalTreatsCanceledAsFatalWithLiveContext(t *testing.T) {
	// When the lifecycle context is still active (not shutting down),
	// context.Canceled should NOT be suppressed — it indicates a bug or
	// unexpected cancellation, not a clean shutdown.
	fatalCh := make(chan error, 1)
	v := &BlockValidator{
		fatalErr: fatalCh,
	}
	v.config = func() *BlockValidatorConfig {
		return &BlockValidatorConfig{FailureIsFatal: true}
	}
	// Start the StopWaiter with a live (non-cancelled) context.
	v.StopWaiter.Start(context.Background(), v)

	v.possiblyFatal(context.Canceled)
	select {
	case err := <-fatalCh:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled, got: %v", err)
		}
	default:
		t.Fatal("context.Canceled with live lifecycle context should be fatal")
	}
}

func TestPossiblyFatalTreatsCanceledAsFatalWithUnstartedStopWaiter(t *testing.T) {
	// When GetContextSafe fails (StopWaiter not started), the suppression
	// condition is false, so context.Canceled falls through to the fatal path.
	// This guards against unconditional suppression of context.Canceled.
	fatalCh := make(chan error, 1)
	v := &BlockValidator{
		fatalErr: fatalCh,
	}
	v.config = func() *BlockValidatorConfig {
		return &BlockValidatorConfig{FailureIsFatal: true}
	}
	// Do NOT start StopWaiter — GetContextSafe will return an error.

	v.possiblyFatal(context.Canceled)
	select {
	case err := <-fatalCh:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled, got: %v", err)
		}
	default:
		t.Fatal("context.Canceled with unstarted StopWaiter should be fatal")
	}
}

func TestHandleValidationResultDoesNotSkipReorgOnDeadlineExceeded(t *testing.T) {
	// The reorg skip should only trigger for context.Canceled (clean shutdown),
	// not for context.DeadlineExceeded (timeout). With chainCaughtUp=false,
	// Reorg returns nil early, so we just verify it was attempted (no skip).
	fatalCh := make(chan error, 1)
	v := &BlockValidator{
		fatalErr: fatalCh,
	}
	v.config = func() *BlockValidatorConfig {
		return &BlockValidatorConfig{
			ValidationPoll: 0,
			FailureIsFatal: true,
		}
	}

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()

	reorgTarget := arbutil.MessageIndex(5)
	// With DeadlineExceeded context, reorg should NOT be skipped.
	// chainCaughtUp=false means Reorg returns nil immediately.
	v.handleValidationResult(ctx, &reorgTarget, nil, "test")

	// No fatal error — Reorg succeeded (no-op due to chainCaughtUp=false).
	select {
	case err := <-fatalCh:
		t.Fatalf("unexpected fatal error: %v", err)
	default:
	}
}

func TestHandleValidationResultErrorTakesPrecedenceOverReorg(t *testing.T) {
	// When both err and reorg are non-nil, the error path should execute
	// and the reorg should be ignored. This guards against refactoring
	// the if/else if into separate if blocks.
	fatalCh := make(chan error, 1)
	v := &BlockValidator{
		fatalErr: fatalCh,
	}
	v.config = func() *BlockValidatorConfig {
		return &BlockValidatorConfig{
			ValidationPoll: 0,
			FailureIsFatal: true,
		}
	}

	reorgTarget := arbutil.MessageIndex(5)
	realErr := errors.New("something broke")
	v.handleValidationResult(context.Background(), &reorgTarget, realErr, "test")

	// Error path only logs — no fatal. If reorg were also processed,
	// it would call Reorg on a zero-value validator and potentially panic.
	select {
	case err := <-fatalCh:
		t.Fatalf("unexpected fatal error: %v", err)
	default:
	}
}

func TestPossiblyFatalChannelFull(t *testing.T) {
	// When fatalErr already has an error, the second error is dropped
	// (with a log) rather than blocking. Verify the first error is
	// preserved and the second doesn't panic or block.
	fatalCh := make(chan error, 1)
	v := &BlockValidator{
		fatalErr: fatalCh,
	}
	v.config = func() *BlockValidatorConfig {
		return &BlockValidatorConfig{FailureIsFatal: true}
	}

	first := errors.New("first error")
	second := errors.New("second error")
	v.possiblyFatal(first)
	v.possiblyFatal(second) // should not block

	err := <-fatalCh
	if !errors.Is(err, first) {
		t.Fatalf("expected first error preserved, got: %v", err)
	}
	// Channel should now be empty (second was dropped).
	select {
	case err := <-fatalCh:
		t.Fatalf("second error should have been dropped, got: %v", err)
	default:
	}
}

func TestPossiblyFatalNilIsNoop(t *testing.T) {
	fatalCh := make(chan error, 1)
	v := &BlockValidator{
		fatalErr: fatalCh,
	}
	v.config = func() *BlockValidatorConfig {
		return &BlockValidatorConfig{FailureIsFatal: true}
	}

	v.possiblyFatal(nil)
	select {
	case err := <-fatalCh:
		t.Fatalf("nil error should be a no-op, got: %v", err)
	default:
	}
}

func TestHandleValidationResultNoopOnNilErrNilReorg(t *testing.T) {
	// When both err and reorg are nil, handleValidationResult should
	// return ValidationPoll with no side effects.
	fatalCh := make(chan error, 1)
	v := &BlockValidator{
		fatalErr: fatalCh,
	}
	v.config = func() *BlockValidatorConfig {
		return &BlockValidatorConfig{
			ValidationPoll: 0,
			FailureIsFatal: true,
		}
	}

	result := v.handleValidationResult(context.Background(), nil, nil, "test")
	if result != 0 {
		t.Errorf("expected ValidationPoll duration (0), got %v", result)
	}
	select {
	case err := <-fatalCh:
		t.Fatalf("unexpected fatal error: %v", err)
	default:
	}
}

func TestPossiblyFatalNonFatalConfig(t *testing.T) {
	fatalCh := make(chan error, 1)
	v := &BlockValidator{
		fatalErr: fatalCh,
	}
	v.config = func() *BlockValidatorConfig {
		return &BlockValidatorConfig{FailureIsFatal: false}
	}

	// With FailureIsFatal=false, a real error should be logged but not sent to fatalErr.
	v.possiblyFatal(errors.New("non-fatal validation error"))
	select {
	case err := <-fatalCh:
		t.Fatalf("error should not be fatal when FailureIsFatal=false, got: %v", err)
	default:
	}
}
