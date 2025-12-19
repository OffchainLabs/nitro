// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package gethexec

import (
	"testing"
	"time"

	"github.com/offchainlabs/nitro/arbutil"
)

func TestSyncHistory_Add(t *testing.T) {
	msgLag := 100 * time.Millisecond
	h := newSyncHistory(msgLag)

	now := time.Now()

	// Add some entries
	h.add(arbutil.MessageIndex(100), now)
	h.add(arbutil.MessageIndex(200), now.Add(50*time.Millisecond))
	h.add(arbutil.MessageIndex(300), now.Add(100*time.Millisecond))

	// Check that all entries are present
	if len(h.entries) != 3 {
		t.Errorf("Expected 3 entries, got %d", len(h.entries))
	}

	// Add an entry that should trigger trimming (more than msgLag later)
	// The new entry is at now+250ms, so we keep entries from (now+250ms - 100ms) = now+150ms onwards
	// All previous entries are before now+150ms, so they get trimmed
	h.add(arbutil.MessageIndex(400), now.Add(250*time.Millisecond))

	// Only the newest entry should remain after trimming
	if len(h.entries) != 1 {
		t.Errorf("Expected 1 entry after trimming, got %d", len(h.entries))
	}

	// Verify only the newest entry remains
	if h.entries[0].maxMessageCount != 400 {
		t.Errorf("Expected first entry to have maxMessageCount 400, got %d", h.entries[0].maxMessageCount)
	}
}

func TestSyncHistory_GetSyncTarget(t *testing.T) {
	msgLag := 100 * time.Millisecond
	h := newSyncHistory(msgLag)

	now := time.Now()

	// Test empty history
	target := h.getSyncTarget(now)
	if target != 0 {
		t.Errorf("Expected 0 for empty history, got %d", target)
	}

	// Add entries at various times
	h.add(arbutil.MessageIndex(100), now.Add(-250*time.Millisecond)) // Too old (beyond msgLag)
	h.add(arbutil.MessageIndex(200), now.Add(-180*time.Millisecond)) // Too old (beyond msgLag)
	h.add(arbutil.MessageIndex(300), now.Add(-150*time.Millisecond)) // Too old (beyond msgLag)
	h.add(arbutil.MessageIndex(400), now.Add(-120*time.Millisecond)) // Too old (beyond msgLag)
	h.add(arbutil.MessageIndex(500), now.Add(-80*time.Millisecond))  // In window (less than msgLag ago)
	h.add(arbutil.MessageIndex(600), now.Add(-50*time.Millisecond))  // In window (less than msgLag ago)

	// Should return the oldest entry in the window (500)
	target = h.getSyncTarget(now)
	if target != 500 {
		t.Errorf("Expected sync target 500, got %d", target)
	}
}

func TestSyncHistory_GetSyncTarget_NoValidEntries(t *testing.T) {
	msgLag := 100 * time.Millisecond
	h := newSyncHistory(msgLag)

	now := time.Now()

	// Add only entries outside the valid window
	h.add(arbutil.MessageIndex(100), now.Add(-250*time.Millisecond)) // Too old
	h.add(arbutil.MessageIndex(200), now.Add(-250*time.Millisecond)) // Too old

	// Should return 0 as no entries are in the valid window
	target := h.getSyncTarget(now)
	if target != 0 {
		t.Errorf("Expected sync target 0, got %d", target)
	}
}

func TestSyncHistory_GetSyncTarget_ExactBoundaries(t *testing.T) {
	msgLag := 100 * time.Millisecond
	h := newSyncHistory(msgLag)

	now := time.Now()

	// Add entries exactly at the boundaries
	h.add(arbutil.MessageIndex(100), now.Add(-msgLag))   // Exactly at msgLag ago (included)
	h.add(arbutil.MessageIndex(200), now.Add(-msgLag/2)) // More recent than msgLag ago

	// Both entries are in the window, return the oldest (100)
	target := h.getSyncTarget(now)
	if target != 100 {
		t.Errorf("Expected sync target 100, got %d", target)
	}
}

func TestSyncHistory_Trimming(t *testing.T) {
	msgLag := 100 * time.Millisecond
	h := newSyncHistory(msgLag)

	baseTime := time.Now()

	// Add many entries - they will get trimmed as we go
	// With msgLag=100ms, we keep entries within 100ms of the newest
	for i := 0; i < 10; i++ {
		// #nosec G115
		h.add(arbutil.MessageIndex(i*100), baseTime.Add(time.Duration(i*50)*time.Millisecond))
	}

	// After adding entry at 450ms, we keep entries from 350ms onwards
	// That's entries at 350ms, 400ms, 450ms = 3 entries
	if len(h.entries) != 3 {
		t.Errorf("Expected 3 entries after incremental adds, got %d", len(h.entries))
	}

	// Verify the first entry is the one at 350ms (index 7)
	if h.entries[0].maxMessageCount != 700 {
		t.Errorf("Expected first entry to be 700, got %d", h.entries[0].maxMessageCount)
	}

	// Add an entry much later that should trigger more aggressive trimming
	futureTime := baseTime.Add(1 * time.Second)
	h.add(arbutil.MessageIndex(1000), futureTime)

	// Should have trimmed all old entries (keeping only the new one since all others are > 100ms old)
	if len(h.entries) != 1 {
		t.Errorf("Expected 1 entry after adding future entry, got %d", len(h.entries))
	}

	if h.entries[0].maxMessageCount != 1000 {
		t.Errorf("Expected remaining entry to be 1000, got %d", h.entries[0].maxMessageCount)
	}
}

func TestSyncHistory_ConcurrentAccess(t *testing.T) {
	msgLag := 10 * time.Millisecond
	h := newSyncHistory(msgLag)

	done := make(chan bool)
	now := time.Now()

	// Concurrent adds
	go func() {
		for i := 0; i < 100; i++ {
			// #nosec G115
			h.add(arbutil.MessageIndex(i), now.Add(time.Duration(i)*time.Millisecond))
			time.Sleep(time.Microsecond)
		}
		done <- true
	}()

	// Concurrent reads
	go func() {
		for i := 0; i < 100; i++ {
			h.getSyncTarget(now.Add(time.Duration(i) * time.Millisecond))
			time.Sleep(time.Microsecond)
		}
		done <- true
	}()

	// Wait for both goroutines
	<-done
	<-done

	// Just verify we didn't panic and have some entries
	if len(h.entries) == 0 {
		t.Error("Expected some entries after concurrent operations")
	}
}

func TestSyncHistory_EdgeCases(t *testing.T) {
	msgLag := 100 * time.Millisecond
	h := newSyncHistory(msgLag)

	now := time.Now()

	// Test with single entry in window
	h.add(arbutil.MessageIndex(100), now.Add(-50*time.Millisecond))
	target := h.getSyncTarget(now)
	if target != 100 {
		t.Errorf("Expected sync target 100 for single entry, got %d", target)
	}

	// Test with msgLag = 0 (edge case)
	h2 := newSyncHistory(0)
	h2.add(arbutil.MessageIndex(200), now)
	target2 := h2.getSyncTarget(now)
	// With msgLag=0, the window is from 0 to 0 ago, so current entry should match
	if target2 != 200 {
		t.Errorf("Expected sync target 200 for msgLag=0, got %d", target2)
	}
}
