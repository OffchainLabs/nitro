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
