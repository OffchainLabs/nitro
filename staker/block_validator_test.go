package staker

import (
	"context"
	"errors"
	"fmt"
	"testing"

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
	// the guard. The chainCaughtUp=false path is the relevant one for the
	// shutdown scenario that triggered "cannot reorg out genesis" errors:
	// the validator hasn't caught up yet when the context is cancelled.
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

	// context.Canceled should be suppressed
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

	// context.DeadlineExceeded should be logged but not fatal
	v.possiblyFatal(context.DeadlineExceeded)
	select {
	case err := <-fatalCh:
		t.Fatalf("context.DeadlineExceeded should not be fatal, got: %v", err)
	default:
	}

	// Wrapped context.DeadlineExceeded should also be suppressed
	v.possiblyFatal(fmt.Errorf("timed out: %w", context.DeadlineExceeded))
	select {
	case err := <-fatalCh:
		t.Fatalf("wrapped context.DeadlineExceeded should not be fatal, got: %v", err)
	default:
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

func TestHandleValidationResultSuppressesContextErrors(t *testing.T) {
	// Context errors should be suppressed (not sent to fatalErr).
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

	v.handleValidationResult(context.Background(), nil, context.Canceled, "test")
	select {
	case err := <-fatalCh:
		t.Fatalf("context.Canceled should not be fatal, got: %v", err)
	default:
	}

	v.handleValidationResult(context.Background(), nil, context.DeadlineExceeded, "test")
	select {
	case err := <-fatalCh:
		t.Fatalf("context.DeadlineExceeded should not be fatal, got: %v", err)
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
