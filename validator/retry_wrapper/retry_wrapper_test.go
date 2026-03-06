// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package retry_wrapper

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"

	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/validator"
	"github.com/offchainlabs/nitro/validator/server_common"
)

// testSpawner is a mock ValidationSpawner that returns a sequence of
// predetermined results. Each call to Launch pops the next result.
type testSpawner struct {
	results []launchResult
	idx     atomic.Int64
}

type launchResult struct {
	state validator.GoGlobalState
	err   error
}

func (s *testSpawner) Launch(_ *validator.ValidationInput, moduleRoot common.Hash) validator.ValidationRun {
	i := int(s.idx.Add(1) - 1)
	if i >= len(s.results) {
		panic("testSpawner: too many Launch calls")
	}
	r := s.results[i]
	promise := containers.NewPromise[validator.GoGlobalState](nil)
	if r.err != nil {
		promise.ProduceError(r.err)
	} else {
		promise.Produce(r.state)
	}
	return server_common.NewValRun(&promise, moduleRoot)
}

func (s *testSpawner) WasmModuleRoots() ([]common.Hash, error) { return nil, nil }
func (s *testSpawner) Start(context.Context) error              { return nil }
func (s *testSpawner) Stop()                                    {}
func (s *testSpawner) Name() string                             { return "test" }
func (s *testSpawner) StylusArchs() []rawdb.WasmTarget          { return nil }
func (s *testSpawner) Capacity() int                            { return 1 }

var (
	successState = validator.GoGlobalState{Batch: 1, PosInBatch: 1}
	testRoot     = common.Hash{1}
	errTimeout   = context.DeadlineExceeded
	errGeneric   = errors.New("validation failed")
)

func launchAndAwait(t *testing.T, wrapper *ValidationSpawnerRetryWrapper, allowedAttempts, allowedTimeouts uint64) (validator.GoGlobalState, error) {
	t.Helper()
	ctx := context.Background()
	run := wrapper.LaunchWithNAllowedAttempts(nil, testRoot, allowedAttempts, allowedTimeouts)
	return run.Await(ctx)
}

func setupWrapper(t *testing.T, results []launchResult) *ValidationSpawnerRetryWrapper {
	t.Helper()
	spawner := &testSpawner{results: results}
	wrapper := NewValidationSpawnerRetryWrapper(spawner)
	wrapper.StopWaiter.Start(context.Background(), wrapper)
	t.Cleanup(func() { wrapper.StopWaiter.StopAndWait() })
	return wrapper
}

func TestRetryWrapper_SuccessOnFirstAttempt(t *testing.T) {
	wrapper := setupWrapper(t, []launchResult{
		{state: successState},
	})
	state, err := launchAndAwait(t, wrapper, 1, 1)
	if err != nil {
		t.Fatal("unexpected error:", err)
	}
	if state != successState {
		t.Fatalf("unexpected state: got %v, want %v", state, successState)
	}
}

func TestRetryWrapper_TimeoutThenSuccess(t *testing.T) {
	wrapper := setupWrapper(t, []launchResult{
		{err: errTimeout},
		{err: errTimeout},
		{state: successState},
	})
	state, err := launchAndAwait(t, wrapper, 1, 3)
	if err != nil {
		t.Fatal("unexpected error:", err)
	}
	if state != successState {
		t.Fatalf("unexpected state: got %v, want %v", state, successState)
	}
}

func TestRetryWrapper_TimeoutExhausted(t *testing.T) {
	wrapper := setupWrapper(t, []launchResult{
		{err: errTimeout},
		{err: errTimeout},
	})
	_, err := launchAndAwait(t, wrapper, 1, 2)
	if !errors.Is(err, errTimeout) {
		t.Fatalf("expected timeout error, got: %v", err)
	}
}

func TestRetryWrapper_NonTimeoutExhausted(t *testing.T) {
	wrapper := setupWrapper(t, []launchResult{
		{err: errGeneric},
		{err: errGeneric},
	})
	_, err := launchAndAwait(t, wrapper, 2, 3)
	if err == nil || err.Error() != errGeneric.Error() {
		t.Fatalf("expected generic error, got: %v", err)
	}
}

func TestRetryWrapper_CountersAreIndependent(t *testing.T) {
	// Mix of timeout and non-timeout errors: each counter should increment independently.
	wrapper := setupWrapper(t, []launchResult{
		{err: errTimeout},  // timeoutAttempts=1
		{err: errGeneric},  // nonTimeoutAttempts=1
		{err: errTimeout},  // timeoutAttempts=2
		{state: successState},
	})
	state, err := launchAndAwait(t, wrapper, 2, 3)
	if err != nil {
		t.Fatal("unexpected error:", err)
	}
	if state != successState {
		t.Fatalf("unexpected state: got %v, want %v", state, successState)
	}
}

func TestRetryWrapper_MixedErrorsTimeoutExhausted(t *testing.T) {
	// Non-timeout errors should not count toward timeout budget.
	wrapper := setupWrapper(t, []launchResult{
		{err: errGeneric},  // nonTimeoutAttempts=1
		{err: errTimeout},  // timeoutAttempts=1
		{err: errTimeout},  // timeoutAttempts=2, exhausted
	})
	_, err := launchAndAwait(t, wrapper, 3, 2)
	if !errors.Is(err, errTimeout) {
		t.Fatalf("expected timeout error, got: %v", err)
	}
}

func TestRetryWrapper_MixedErrorsNonTimeoutExhausted(t *testing.T) {
	// Timeout errors should not count toward non-timeout budget.
	wrapper := setupWrapper(t, []launchResult{
		{err: errTimeout},  // timeoutAttempts=1
		{err: errGeneric},  // nonTimeoutAttempts=1
		{err: errGeneric},  // nonTimeoutAttempts=2, exhausted
	})
	_, err := launchAndAwait(t, wrapper, 2, 3)
	if err == nil || err.Error() != errGeneric.Error() {
		t.Fatalf("expected generic error, got: %v", err)
	}
}

func TestRetryWrapper_ZeroAllowedTimeouts(t *testing.T) {
	// With 0 allowed timeouts, the first timeout should be immediately fatal.
	wrapper := setupWrapper(t, []launchResult{
		{err: errTimeout},
	})
	_, err := launchAndAwait(t, wrapper, 1, 0)
	if !errors.Is(err, errTimeout) {
		t.Fatalf("expected timeout error, got: %v", err)
	}
}

func TestRetryWrapper_ZeroAllowedAttempts(t *testing.T) {
	// With 0 allowed attempts, the first non-timeout error should be immediately fatal.
	wrapper := setupWrapper(t, []launchResult{
		{err: errGeneric},
	})
	_, err := launchAndAwait(t, wrapper, 0, 3)
	if err == nil || err.Error() != errGeneric.Error() {
		t.Fatalf("expected generic error, got: %v", err)
	}
}

func TestRetryWrapper_ContextCanceled(t *testing.T) {
	// If the context is canceled, the wrapper should return ctx.Err() rather
	// than continuing to retry.
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	spawner := &testSpawner{results: []launchResult{
		// The spawner's result won't matter — ctx is already canceled.
		{err: errTimeout},
	}}
	wrapper := NewValidationSpawnerRetryWrapper(spawner)
	wrapper.StopWaiter.Start(ctx, wrapper)
	defer wrapper.StopWaiter.StopAndWait()

	run := wrapper.LaunchWithNAllowedAttempts(nil, testRoot, 3, 3)
	_, err := run.Await(context.Background())
	if err == nil {
		t.Fatal("expected error from canceled context")
	}
}
