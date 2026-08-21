// Copyright 2024-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package gethexec

import (
	"testing"
	"time"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
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
		_, _, _ = engine.SequenceTransactions(nil, nil)
	}()

	// The mutex must be unlocked after the panic is recovered upstream.
	if !engine.createBlocksMutex.TryLock() {
		t.Fatal("createBlocksMutex is still locked after panic recovery; would deadlock on next call")
	}
	engine.createBlocksMutex.Unlock()
}

func TestEnqueueDelayedMessagesRejectsMisalignedBatches(t *testing.T) {
	// Queue seeded with idx 5 => expected next idx is 6, without needing a
	// blockchain for the empty-queue header fallback.
	tests := []struct {
		name        string
		firstMsgIdx uint64
		numMsgs     int
		wantNextIdx uint64
	}{
		{"aligned batch appended", 6, 2, 8},
		{"overlapping batch trimmed", 4, 4, 8},
		{"fully duplicate batch dropped", 3, 3, 6},
		{"gapped batch dropped", 8, 2, 6},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := &ExecutionEngine{}
			engine.delayedMsgs.Push(&delayedMsg{msgIdx: 5})

			msgs := make([]*arbostypes.L1IncomingMessage, tt.numMsgs)
			engine.EnqueueDelayedMessages(msgs, tt.firstMsgIdx)

			nextIdx, err := engine.NextDelayedMessageNumber()
			if err != nil {
				t.Fatal(err)
			}
			if nextIdx != tt.wantNextIdx {
				t.Errorf("NextDelayedMessageNumber() = %d, want %d", nextIdx, tt.wantNextIdx)
			}
			// The queue must stay contiguous from idx 5.
			for wantIdx := uint64(5); engine.delayedMsgs.Len() > 0; wantIdx++ {
				if gotIdx := engine.delayedMsgs.Pop().msgIdx; gotIdx != wantIdx {
					t.Fatalf("queue entry msgIdx = %d, want %d", gotIdx, wantIdx)
				}
			}
		})
	}
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
