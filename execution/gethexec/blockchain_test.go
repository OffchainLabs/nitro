// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package gethexec

import (
	"errors"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
)

func TestGetStateHistory(t *testing.T) {
	maxBlockSpeed := time.Millisecond * 250
	expectedStateHistory := uint64(345600)
	actualStateHistory := GetStateHistory(maxBlockSpeed)
	if actualStateHistory != expectedStateHistory {
		t.Errorf("Expected state history to be %d, but got %d", expectedStateHistory, actualStateHistory)
	}
}

// TestSequencerWrapperMutexReleasedOnPanic verifies that createBlocksMutex is
// properly released even when sequencerFunc panics. Without defer, a panic
// would bypass Unlock() and leave the mutex locked, causing a deadlock on the
// next call (e.g. after createBlock's recover() catches the panic).
func TestSequencerWrapperMutexReleasedOnPanic(t *testing.T) {
	engine := &ExecutionEngine{
		cachedL1PriceData: NewL1PriceData(),
	}

	// Mirrors what createBlock does: call into sequencerWrapper and recover.
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected a panic but got none")
			}
		}()
		_, _ = engine.sequencerWrapper(func() (*types.Block, error) {
			panic("simulated sequencer panic")
		})
	}()

	// The mutex must be unlocked after the panic is recovered upstream.
	if !engine.createBlocksMutex.TryLock() {
		t.Fatal("createBlocksMutex is still locked after panic recovery; would deadlock on next call")
	}
	engine.createBlocksMutex.Unlock()
}

// TestSequencerWrapperMutexReleasedOnSuccess verifies that normal (non-panic)
// returns also leave the mutex unlocked.
func TestSequencerWrapperMutexReleasedOnSuccess(t *testing.T) {
	engine := &ExecutionEngine{
		cachedL1PriceData: NewL1PriceData(),
	}

	sentinel := errors.New("stop retrying")
	_, err := engine.sequencerWrapper(func() (*types.Block, error) {
		return nil, sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("unexpected error: %v", err)
	}

	if !engine.createBlocksMutex.TryLock() {
		t.Fatal("createBlocksMutex is still locked after normal return")
	}
	engine.createBlocksMutex.Unlock()
}
