package blocksreexecutor

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"

	"github.com/offchainlabs/nitro/util/stopwaiter"
)

func newTestReExecutor(fatalCh chan error) *BlocksReExecutor {
	return &BlocksReExecutor{
		StopWaiter:    stopwaiter.StopWaiter{},
		config:        nil,
		db:            nil,
		blockchain:    nil,
		stateFor:      nil,
		done:          nil,
		fatalErrChan:  fatalCh,
		fatalReported: atomic.Bool{},
		blocks:        nil,
		mutex:         sync.Mutex{},
		success:       nil,
	}
}

func TestReportFatalErrSetsFatalReported(t *testing.T) {
	fatalCh := make(chan error, 1)
	s := newTestReExecutor(fatalCh)

	realErr := errors.New("disk corruption")
	s.reportFatalErr(realErr)

	if !s.fatalReported.Load() {
		t.Fatal("expected fatalReported to be set")
	}
	select {
	case err := <-fatalCh:
		if !errors.Is(err, realErr) {
			t.Fatalf("expected realErr, got: %v", err)
		}
	default:
		t.Fatal("expected error in fatalErrChan")
	}
}

func TestReportFatalErrDoesNotBlockOnFullChannel(t *testing.T) {
	fatalCh := make(chan error, 1)
	s := newTestReExecutor(fatalCh)

	// Fill the channel
	s.reportFatalErr(errors.New("first"))
	// Second call should not block (exercises the default branch)
	s.reportFatalErr(errors.New("second"))

	if !s.fatalReported.Load() {
		t.Fatal("expected fatalReported to be set")
	}
	// Only the first error should be in the channel
	err := <-fatalCh
	if !strings.Contains(err.Error(), "first") {
		t.Fatalf("expected first error preserved, got: %v", err)
	}
	select {
	case extra := <-fatalCh:
		t.Fatalf("expected channel to be empty after drain, got: %v", extra)
	default:
	}
}

func TestReportFatalErrMultipleErrorTypes(t *testing.T) {
	fatalCh := make(chan error, 4)
	s := newTestReExecutor(fatalCh)

	for _, err := range []error{
		errors.New("disk corruption"),
		fmt.Errorf("wrapped: %w", errors.New("inner")),
		errors.New("another error"),
	} {
		s.reportFatalErr(err)
		select {
		case fatal := <-fatalCh:
			if fatal == nil {
				t.Fatalf("expected non-nil error for input: %v", err)
			}
		default:
			t.Fatalf("expected error in channel for input: %v", err)
		}
	}
}

func newTestConfig() *Config {
	return &Config{
		Enable:             false,
		Mode:               "",
		Blocks:             "",
		CommitStateToDisk:  false,
		Room:               0,
		MinBlocksPerThread: 0,
		TrieCleanLimit:     0,
		ValidateMultiGas:   false,
		blocks:             nil,
	}
}

func TestAdvanceStateUpToBlockCancelledContext(t *testing.T) {
	// When the context is already cancelled, advanceStateUpToBlock should
	// return ctx.Err() immediately without entering the loop, and still
	// call lastRelease via defer.
	s := newTestReExecutor(nil)
	s.config = newTestConfig()
	targetHeader := &types.Header{Number: big.NewInt(10)}
	lastAvailableHeader := &types.Header{Number: big.NewInt(5)}
	released := false
	release := func() { released = true }

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := s.advanceStateUpToBlock(ctx, nil, targetHeader, lastAvailableHeader, release)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got: %v", err)
	}
	if !released {
		t.Fatal("expected lastRelease to be called even with cancelled context")
	}
}

func TestAdvanceStateUpToBlockRecoversPanic(t *testing.T) {
	// A nil blockchain causes AdvanceStateByBlock to panic (nil pointer
	// dereference). The panic recovery in advanceStateUpToBlock should
	// catch it and return an error instead of crashing.
	s := newTestReExecutor(nil)
	s.config = newTestConfig()
	targetHeader := &types.Header{Number: big.NewInt(5)}
	lastAvailableHeader := &types.Header{Number: big.NewInt(4)}
	released := false
	release := func() { released = true }

	err := s.advanceStateUpToBlock(context.Background(), nil, targetHeader, lastAvailableHeader, release)
	if err == nil {
		t.Fatal("expected error from panic recovery, got nil")
	}
	if !strings.Contains(err.Error(), "panic during block re-execution") {
		t.Fatalf("expected panic recovery error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "at block 5") {
		t.Fatalf("expected block number in error message, got: %v", err)
	}
	if !released {
		t.Fatal("expected lastRelease to be called")
	}
}

func TestWaitForReExecutionCancelled(t *testing.T) {
	s := newTestReExecutor(make(chan error, 1))
	s.success = make(chan struct{})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := s.WaitForReExecution(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got: %v", err)
	}
}

func TestWaitForReExecutionFatalError(t *testing.T) {
	fatalCh := make(chan error, 1)
	s := newTestReExecutor(fatalCh)
	s.success = make(chan struct{})

	inner := errors.New("disk corruption")
	fatalCh <- inner

	err := s.WaitForReExecution(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, inner) {
		t.Fatalf("expected wrapped error to preserve inner via errors.Is, got: %v", err)
	}
	if !strings.Contains(err.Error(), "shutting BlocksReExecutor down due to fatal error") {
		t.Fatalf("expected wrapped fatal error, got: %v", err)
	}
}

func TestWaitForReExecutionSuccess(t *testing.T) {
	s := newTestReExecutor(make(chan error, 1))
	s.success = make(chan struct{})
	close(s.success)

	err := s.WaitForReExecution(context.Background())
	if err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
}

func TestWaitForReExecutionFatalAndSuccessBothReady(t *testing.T) {
	// When both fatalErrChan and success are ready, select picks one
	// non-deterministically. Both outcomes are acceptable.
	fatalCh := make(chan error, 1)
	s := newTestReExecutor(fatalCh)
	s.success = make(chan struct{})

	fatalCh <- errors.New("fatal")
	close(s.success)

	err := s.WaitForReExecution(context.Background())
	// Either nil (success) or wrapped fatal error is acceptable
	if err != nil && !strings.Contains(err.Error(), "shutting BlocksReExecutor down due to fatal error") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateValidFullConfig(t *testing.T) {
	c := &Config{
		Enable:             true,
		Mode:               "full",
		Blocks:             `[[1,10]]`,
		CommitStateToDisk:  false,
		Room:               4,
		MinBlocksPerThread: 0,
		TrieCleanLimit:     0,
		ValidateMultiGas:   false,
		blocks:             nil,
	}
	if err := c.Validate(); err != nil {
		t.Fatalf("expected valid config, got: %v", err)
	}
	if c.Mode != "full" {
		t.Fatalf("expected mode 'full', got: %s", c.Mode)
	}
	if len(c.blocks) != 1 || c.blocks[0] != [2]uint64{1, 10} {
		t.Fatalf("unexpected parsed blocks: %v", c.blocks)
	}
}

func TestValidateValidRandomConfig(t *testing.T) {
	c := &Config{
		Enable:             true,
		Mode:               "random",
		Blocks:             `[[0,0],[5,20]]`,
		CommitStateToDisk:  false,
		Room:               2,
		MinBlocksPerThread: 0,
		TrieCleanLimit:     0,
		ValidateMultiGas:   false,
		blocks:             nil,
	}
	if err := c.Validate(); err != nil {
		t.Fatalf("expected valid config, got: %v", err)
	}
	if len(c.blocks) != 2 {
		t.Fatalf("expected 2 block ranges, got: %d", len(c.blocks))
	}
}

func TestValidateModeCaseInsensitive(t *testing.T) {
	c := &Config{
		Enable:             true,
		Mode:               "FULL",
		Blocks:             `[[1,10]]`,
		CommitStateToDisk:  false,
		Room:               1,
		MinBlocksPerThread: 0,
		TrieCleanLimit:     0,
		ValidateMultiGas:   false,
		blocks:             nil,
	}
	if err := c.Validate(); err != nil {
		t.Fatalf("expected valid config after lowering mode, got: %v", err)
	}
	if c.Mode != "full" {
		t.Fatalf("expected mode lowered to 'full', got: %s", c.Mode)
	}
}

func TestValidateInvalidMode(t *testing.T) {
	c := &Config{
		Enable:             true,
		Mode:               "turbo",
		Blocks:             `[[1,10]]`,
		CommitStateToDisk:  false,
		Room:               1,
		MinBlocksPerThread: 0,
		TrieCleanLimit:     0,
		ValidateMultiGas:   false,
		blocks:             nil,
	}
	err := c.Validate()
	if err == nil {
		t.Fatal("expected error for invalid mode")
	}
	if !strings.Contains(err.Error(), "invalid mode") {
		t.Fatalf("expected 'invalid mode' error, got: %v", err)
	}
}

func TestValidateEmptyBlocks(t *testing.T) {
	c := &Config{
		Enable:             true,
		Mode:               "full",
		Blocks:             "",
		CommitStateToDisk:  false,
		Room:               1,
		MinBlocksPerThread: 0,
		TrieCleanLimit:     0,
		ValidateMultiGas:   false,
		blocks:             nil,
	}
	err := c.Validate()
	if err == nil {
		t.Fatal("expected error for empty blocks")
	}
	if !strings.Contains(err.Error(), "cannot be empty") {
		t.Fatalf("expected 'cannot be empty' error, got: %v", err)
	}
}

func TestValidateMalformedBlocksJSON(t *testing.T) {
	c := &Config{
		Enable:             true,
		Mode:               "full",
		Blocks:             `not-json`,
		CommitStateToDisk:  false,
		Room:               1,
		MinBlocksPerThread: 0,
		TrieCleanLimit:     0,
		ValidateMultiGas:   false,
		blocks:             nil,
	}
	err := c.Validate()
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
	if !strings.Contains(err.Error(), "failed to parse") {
		t.Fatalf("expected parse error, got: %v", err)
	}
}

func TestValidateInvalidBlockRange(t *testing.T) {
	c := &Config{
		Enable:             true,
		Mode:               "full",
		Blocks:             `[[10,5]]`,
		CommitStateToDisk:  false,
		Room:               1,
		MinBlocksPerThread: 0,
		TrieCleanLimit:     0,
		ValidateMultiGas:   false,
		blocks:             nil,
	}
	err := c.Validate()
	if err == nil {
		t.Fatal("expected error for invalid block range")
	}
	if !strings.Contains(err.Error(), "invalid block range") {
		t.Fatalf("expected 'invalid block range' error, got: %v", err)
	}
}

func TestValidateRoomZero(t *testing.T) {
	c := &Config{
		Enable:             true,
		Mode:               "full",
		Blocks:             `[[1,10]]`,
		CommitStateToDisk:  false,
		Room:               0,
		MinBlocksPerThread: 0,
		TrieCleanLimit:     0,
		ValidateMultiGas:   false,
		blocks:             nil,
	}
	err := c.Validate()
	if err == nil {
		t.Fatal("expected error for room <= 0")
	}
	if !strings.Contains(err.Error(), "room") {
		t.Fatalf("expected room error, got: %v", err)
	}
}

func TestValidateRoomNegative(t *testing.T) {
	c := &Config{
		Enable:             true,
		Mode:               "full",
		Blocks:             `[[1,10]]`,
		CommitStateToDisk:  false,
		Room:               -1,
		MinBlocksPerThread: 0,
		TrieCleanLimit:     0,
		ValidateMultiGas:   false,
		blocks:             nil,
	}
	err := c.Validate()
	if err == nil {
		t.Fatal("expected error for room <= 0")
	}
	if !strings.Contains(err.Error(), "room") {
		t.Fatalf("expected room error, got: %v", err)
	}
}

func TestValidateDisabledSkipsModeCheck(t *testing.T) {
	// When Enable is false, invalid mode should not cause an error
	c := &Config{
		Enable:             false,
		Mode:               "invalid-mode",
		Blocks:             `[[1,10]]`,
		CommitStateToDisk:  false,
		Room:               1,
		MinBlocksPerThread: 0,
		TrieCleanLimit:     0,
		ValidateMultiGas:   false,
		blocks:             nil,
	}
	if err := c.Validate(); err != nil {
		t.Fatalf("expected no error when disabled, got: %v", err)
	}
}

func TestValidateMultipleBlockRanges(t *testing.T) {
	c := &Config{
		Enable:             true,
		Mode:               "full",
		Blocks:             `[[1,10],[20,30],[50,100]]`,
		CommitStateToDisk:  false,
		Room:               2,
		MinBlocksPerThread: 0,
		TrieCleanLimit:     0,
		ValidateMultiGas:   false,
		blocks:             nil,
	}
	if err := c.Validate(); err != nil {
		t.Fatalf("expected valid config, got: %v", err)
	}
	if len(c.blocks) != 3 {
		t.Fatalf("expected 3 block ranges, got: %d", len(c.blocks))
	}
}

func TestValidateSecondRangeInvalid(t *testing.T) {
	c := &Config{
		Enable:             true,
		Mode:               "full",
		Blocks:             `[[1,10],[20,5]]`,
		CommitStateToDisk:  false,
		Room:               1,
		MinBlocksPerThread: 0,
		TrieCleanLimit:     0,
		ValidateMultiGas:   false,
		blocks:             nil,
	}
	err := c.Validate()
	if err == nil {
		t.Fatal("expected error for second invalid range")
	}
	if !strings.Contains(err.Error(), "invalid block range") {
		t.Fatalf("expected 'invalid block range' error, got: %v", err)
	}
}

func TestImplReturnsZeroWhenFatalPreSet(t *testing.T) {
	fatalCh := make(chan error, 1)
	s := newTestReExecutor(fatalCh)
	s.config = &Config{
		Enable:             false,
		Mode:               "",
		Blocks:             "",
		CommitStateToDisk:  false,
		Room:               2,
		MinBlocksPerThread: 0,
		TrieCleanLimit:     0,
		ValidateMultiGas:   false,
		blocks:             nil,
	}
	s.done = make(chan struct{}, 2)
	s.fatalReported.Store(true)

	result := s.Impl(context.Background(), 0, 100, 10)
	if result != 0 {
		t.Fatalf("expected 0 when fatalReported is pre-set, got: %d", result)
	}
}

func TestImplReturnsStartBlockWhenNoWork(t *testing.T) {
	// When startBlock >= currentBlock, no threads are launched and Impl
	// returns currentBlock directly (the success path with no work).
	fatalCh := make(chan error, 1)
	s := newTestReExecutor(fatalCh)
	s.config = &Config{
		Enable:             false,
		Mode:               "",
		Blocks:             "",
		CommitStateToDisk:  false,
		Room:               2,
		MinBlocksPerThread: 0,
		TrieCleanLimit:     0,
		ValidateMultiGas:   false,
		blocks:             nil,
	}
	s.done = make(chan struct{}, 2)

	result := s.Impl(context.Background(), 100, 100, 10)
	if result != 100 {
		t.Fatalf("expected 100 when startBlock == currentBlock, got: %d", result)
	}
}

func TestWrapFatalErr(t *testing.T) {
	s := newTestReExecutor(nil)
	inner := errors.New("something broke")
	wrapped := s.wrapFatalErr(inner)

	if !errors.Is(wrapped, inner) {
		t.Fatal("expected wrapped error to preserve inner via errors.Is")
	}
	if !strings.Contains(wrapped.Error(), "shutting BlocksReExecutor down due to fatal error") {
		t.Fatalf("expected prefix in wrapped error, got: %v", wrapped)
	}
	if !strings.Contains(wrapped.Error(), "something broke") {
		t.Fatalf("expected inner message preserved, got: %v", wrapped)
	}
}
