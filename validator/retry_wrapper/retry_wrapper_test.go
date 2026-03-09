// Copyright 2026, Offchain Labs, Inc.
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

var (
	testModuleRoot = common.Hash{1}
	successState   = validator.GoGlobalState{Batch: 1, PosInBatch: 1}
)

type mockSpawner struct {
	launches atomic.Int64
	results  []launchResult // consumed in order
}

type launchResult struct {
	state validator.GoGlobalState
	err   error
}

func (m *mockSpawner) Launch(_ *validator.ValidationInput, moduleRoot common.Hash) validator.ValidationRun {
	idx := int(m.launches.Add(1)) - 1
	r := m.results[idx]
	return server_common.NewValRun(containers.NewReadyPromise(r.state, r.err), moduleRoot)
}

func (m *mockSpawner) WasmModuleRoots() ([]common.Hash, error) { return nil, nil }
func (m *mockSpawner) Start(context.Context) error             { return nil }
func (m *mockSpawner) Stop()                                   {}
func (m *mockSpawner) Name() string                            { return "mock" }
func (m *mockSpawner) StylusArchs() []rawdb.WasmTarget         { return nil }
func (m *mockSpawner) Capacity() int                           { return 1 }

func TestRetryWrapper(t *testing.T) {
	timeoutErr := context.DeadlineExceeded
	genericErr := errors.New("validation failed")

	tests := []struct {
		name            string
		allowedAttempts uint64
		allowedTimeouts uint64
		results         []launchResult
		wantErr         error
		wantState       validator.GoGlobalState
		wantLaunchCount int
	}{
		{
			name:            "success on first attempt",
			allowedAttempts: 1,
			allowedTimeouts: 1,
			results: []launchResult{
				{state: successState},
			},
			wantState:       successState,
			wantLaunchCount: 1,
		},
		{
			name:            "success after timeout retries",
			allowedAttempts: 0,
			allowedTimeouts: 2,
			results: []launchResult{
				{err: timeoutErr},
				{err: timeoutErr},
				{state: successState},
			},
			wantState:       successState,
			wantLaunchCount: 3,
		},
		{
			name:            "success after non-timeout retries",
			allowedAttempts: 2,
			allowedTimeouts: 0,
			results: []launchResult{
				{err: genericErr},
				{err: genericErr},
				{state: successState},
			},
			wantState:       successState,
			wantLaunchCount: 3,
		},
		{
			name:            "timeout retries exhausted",
			allowedAttempts: 0,
			allowedTimeouts: 2,
			results: []launchResult{
				{err: timeoutErr},
				{err: timeoutErr},
				{err: timeoutErr}, // 3rd timeout, exceeds allowedTimeouts=2
			},
			wantErr:         timeoutErr,
			wantLaunchCount: 3,
		},
		{
			name:            "non-timeout retries exhausted",
			allowedAttempts: 1,
			allowedTimeouts: 0,
			results: []launchResult{
				{err: genericErr},
				{err: genericErr}, // 2nd failure, exceeds allowedAttempts=1
			},
			wantErr:         genericErr,
			wantLaunchCount: 2,
		},
		{
			name:            "zero allowed means no retries for timeouts",
			allowedAttempts: 0,
			allowedTimeouts: 0,
			results: []launchResult{
				{err: timeoutErr},
			},
			wantErr:         timeoutErr,
			wantLaunchCount: 1,
		},
		{
			name:            "zero allowed means no retries for non-timeouts",
			allowedAttempts: 0,
			allowedTimeouts: 0,
			results: []launchResult{
				{err: genericErr},
			},
			wantErr:         genericErr,
			wantLaunchCount: 1,
		},
		{
			name:            "timeout and non-timeout counters are independent",
			allowedAttempts: 1,
			allowedTimeouts: 1,
			results: []launchResult{
				{err: timeoutErr}, // timeoutAttempts=1, within limit
				{err: genericErr}, // nonTimeoutAttempts=1, within limit
				{err: timeoutErr}, // timeoutAttempts=2, exceeds allowedTimeouts=1
			},
			wantErr:         timeoutErr,
			wantLaunchCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockSpawner{results: tt.results}
			wrapper := NewValidationSpawnerRetryWrapper(mock)
			wrapper.StopWaiter.Start(t.Context(), wrapper)

			run := wrapper.LaunchWithNAllowedAttempts(nil, testModuleRoot, tt.allowedAttempts, tt.allowedTimeouts)
			got, err := run.Await(t.Context())

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if got != tt.wantState {
					t.Fatalf("expected state %v, got %v", tt.wantState, got)
				}
			}

			if int(mock.launches.Load()) != tt.wantLaunchCount {
				t.Fatalf("expected %d launches, got %d", tt.wantLaunchCount, mock.launches.Load())
			}
		})
	}
}

func TestRetryWrapperContextCancellation(t *testing.T) {
	mock := &mockSpawner{
		results: []launchResult{
			{err: context.Canceled},
		},
	}
	wrapper := NewValidationSpawnerRetryWrapper(mock)
	ctx, cancel := context.WithCancel(context.Background())
	wrapper.StopWaiter.Start(ctx, wrapper)
	cancel()

	run := wrapper.LaunchWithNAllowedAttempts(nil, testModuleRoot, 5, 5)
	_, err := run.Await(context.Background())
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
}
