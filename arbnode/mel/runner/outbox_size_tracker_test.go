// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package melrunner

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOutboxSizeTracker_RecordAndLookup(t *testing.T) {
	t.Parallel()
	tracker := NewOutboxSizeTracker(100, 5, nil)

	// Initial entry
	val, ok := tracker.Lookup(100)
	require.True(t, ok)
	require.Equal(t, 5, val)

	// Record sequential blocks
	tracker.Record(101, 5)
	tracker.Record(102, 8) // pour happened
	tracker.Record(103, 7) // pop happened

	val, ok = tracker.Lookup(101)
	require.True(t, ok)
	require.Equal(t, 5, val)

	val, ok = tracker.Lookup(102)
	require.True(t, ok)
	require.Equal(t, 8, val)

	val, ok = tracker.Lookup(103)
	require.True(t, ok)
	require.Equal(t, 7, val)

	// Out of range lookups
	_, ok = tracker.Lookup(99)
	require.False(t, ok)
	_, ok = tracker.Lookup(104)
	require.False(t, ok)
}

func TestOutboxSizeTracker_NonSequentialRecord(t *testing.T) {
	t.Parallel()
	tracker := NewOutboxSizeTracker(100, 5, nil)
	tracker.Record(101, 6)

	// Non-sequential record resets
	tracker.Record(200, 10)

	_, ok := tracker.Lookup(100)
	require.False(t, ok)
	_, ok = tracker.Lookup(101)
	require.False(t, ok)

	val, ok := tracker.Lookup(200)
	require.True(t, ok)
	require.Equal(t, 10, val)
}

func TestOutboxSizeTracker_TrimLeft(t *testing.T) {
	t.Parallel()
	tracker := NewOutboxSizeTracker(100, 0, nil)
	for i := uint64(101); i <= 110; i++ {
		// #nosec G115
		tracker.Record(i, int(i-100))
	}

	// Trim up to block 105
	tracker.TrimLeft(105)

	_, ok := tracker.Lookup(105)
	require.False(t, ok)

	val, ok := tracker.Lookup(106)
	require.True(t, ok)
	require.Equal(t, 6, val)

	val, ok = tracker.Lookup(110)
	require.True(t, ok)
	require.Equal(t, 10, val)
}

func TestOutboxSizeTracker_TrimLeft_BeyondAll(t *testing.T) {
	t.Parallel()
	tracker := NewOutboxSizeTracker(100, 5, nil)
	tracker.Record(101, 6)

	// Trim beyond all entries — sizes becomes empty
	tracker.TrimLeft(200)

	_, ok := tracker.Lookup(100)
	require.False(t, ok)
	_, ok = tracker.Lookup(101)
	require.False(t, ok)
}

func TestOutboxSizeTracker_TrimRight(t *testing.T) {
	t.Parallel()
	tracker := NewOutboxSizeTracker(100, 0, nil)
	for i := uint64(101); i <= 110; i++ {
		// #nosec G115
		tracker.Record(i, int(i-100))
	}

	// Trim from block 108 onwards
	tracker.TrimRight(108)

	val, ok := tracker.Lookup(107)
	require.True(t, ok)
	require.Equal(t, 7, val)

	_, ok = tracker.Lookup(108)
	require.False(t, ok)
	_, ok = tracker.Lookup(110)
	require.False(t, ok)
}

func TestOutboxSizeTracker_TrimRight_AtStart(t *testing.T) {
	t.Parallel()
	tracker := NewOutboxSizeTracker(100, 5, nil)
	tracker.Record(101, 6)
	tracker.Record(102, 7)

	// Trim at startBlock empties the array
	tracker.TrimRight(100)

	_, ok := tracker.Lookup(100)
	require.False(t, ok)
	_, ok = tracker.Lookup(101)
	require.False(t, ok)
}

func TestOutboxSizeTracker_Reset(t *testing.T) {
	t.Parallel()
	tracker := NewOutboxSizeTracker(100, 5, nil)
	tracker.Record(101, 6)
	tracker.Record(102, 7)

	tracker.Reset(200, 42)

	_, ok := tracker.Lookup(100)
	require.False(t, ok)

	val, ok := tracker.Lookup(200)
	require.True(t, ok)
	require.Equal(t, 42, val)
}

func TestOutboxSizeTracker_TrimToFinalized(t *testing.T) {
	t.Parallel()
	finalizedBlock := uint64(100)
	tracker := NewOutboxSizeTracker(95, 0, func() (uint64, error) {
		return finalizedBlock, nil
	})
	for i := uint64(96); i <= 110; i++ {
		// #nosec G115
		tracker.Record(i, int(i-95))
	}

	tracker.TrimToFinalized()

	_, ok := tracker.Lookup(100)
	require.False(t, ok)

	val, ok := tracker.Lookup(101)
	require.True(t, ok)
	require.Equal(t, 6, val)
}

func TestOutboxSizeTracker_NegativeValues(t *testing.T) {
	t.Parallel()
	tracker := NewOutboxSizeTracker(100, -1, nil)

	val, ok := tracker.Lookup(100)
	require.True(t, ok)
	require.Equal(t, -1, val)
}
