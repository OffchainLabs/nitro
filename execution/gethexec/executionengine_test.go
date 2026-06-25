// Copyright 2024-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package gethexec

import (
	"testing"
	"time"
)

func TestSequenceTransactionsMutexReleasedOnPanic(t *testing.T) {
	// Zero-value engine: sequenceTransactionsWithBlockMutex calls
	// getCurrentHeader -> s.bc.CurrentBlock() on a nil bc, which panics.
	engine := &ExecutionEngine{}

	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected a panic but got none")
			}
		}()
		_, _, _ = engine.SequenceTransactions(nil, nil, nil)
	}()

	// The mutex must be unlocked after the panic is recovered upstream.
	if !engine.createBlocksMutex.TryLock() {
		t.Fatal("createBlocksMutex is still locked after panic recovery; would deadlock on next call")
	}
	engine.createBlocksMutex.Unlock()
}

// Uses a zero-value ExecutionEngine so createBlockFromNextMessage nil-derefs;
// the test asserts the recover swallows it and the goroutine returns.
func TestPrefetchNextBlockRecoversFromPanic(t *testing.T) {
	engine := &ExecutionEngine{}

	done := make(chan struct{})
	var recovered any
	go func() {
		defer func() {
			recovered = recover()
			close(done)
		}()
		engine.prefetchNextBlock(nil)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("prefetch goroutine did not complete; panic was not recovered")
	}

	if recovered != nil {
		t.Fatalf("panic sneaked out: %v", recovered)
	}
}
