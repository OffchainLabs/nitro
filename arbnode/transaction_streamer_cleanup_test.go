// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package arbnode

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/core/rawdb"

	"github.com/offchainlabs/nitro/arbnode/db/schema"
)

// TestDeleteTrailingEntries_RemovesOrphans simulates a crash that left
// orphaned message entries beyond the persisted MessageCount.
// deleteTrailingEntries must delete those trailing entries without
// touching anything below the count, and without corrupting unrelated
// prefixes in the same database.
func TestDeleteTrailingEntries_RemovesOrphans(t *testing.T) {
	t.Parallel()
	db := rawdb.NewMemoryDatabase()

	// The prefixes that deleteTrailingEntries cleans.
	cleanedPrefixes := [][]byte{
		schema.MessagePrefix,
		schema.MessageResultPrefix,
		schema.BlockHashInputFeedPrefix,
		schema.BlockMetadataInputFeedPrefix,
		schema.MissingBlockMetadataInputFeedPrefix,
	}

	// Write entries at positions 0-7 for every cleaned prefix.
	// Positions 0-4 are valid; 5-7 are orphaned "crash" entries.
	for _, prefix := range cleanedPrefixes {
		for i := uint64(0); i < 8; i++ {
			key := append(append([]byte{}, prefix...), uint64ToKey(i)...)
			require.NoError(t, db.Put(key, []byte("data")))
		}
	}

	// Write entries under unrelated prefixes that must NOT be touched.
	unrelatedPrefixes := []struct {
		prefix []byte
		name   string
	}{
		{schema.LegacyDelayedMessagePrefix, "LegacyDelayed"},
		{schema.RlpDelayedMessagePrefix, "RlpDelayed"},
		{schema.SequencerBatchMetaPrefix, "BatchMeta"},
		{schema.DelayedSequencedPrefix, "DelayedSequenced"},
		{schema.MelDelayedMessagePrefix, "MelDelayed"},
		{schema.MelSequencerBatchMetaPrefix, "MelBatchMeta"},
	}
	for _, u := range unrelatedPrefixes {
		for i := uint64(0); i < 10; i++ {
			key := append(append([]byte{}, u.prefix...), uint64ToKey(i)...)
			require.NoError(t, db.Put(key, []byte("unrelated")))
		}
	}

	// Also write singleton keys that must survive.
	require.NoError(t, db.Put(schema.DelayedMessageCountKey, []byte("singleton")))
	require.NoError(t, db.Put(schema.SequencerBatchCountKey, []byte("singleton")))

	// Delete trailing entries beyond position 5.
	require.NoError(t, deleteTrailingEntries(db, 5))

	// Entries 0-4 must still exist for all cleaned prefixes.
	for _, prefix := range cleanedPrefixes {
		for i := uint64(0); i < 5; i++ {
			key := append(append([]byte{}, prefix...), uint64ToKey(i)...)
			has, err := db.Has(key)
			require.NoError(t, err)
			require.True(t, has, "entry at position %d under prefix %x should still exist", i, prefix)
		}
	}

	// Entries 5-7 must be gone for all cleaned prefixes.
	for _, prefix := range cleanedPrefixes {
		for i := uint64(5); i < 8; i++ {
			key := append(append([]byte{}, prefix...), uint64ToKey(i)...)
			has, err := db.Has(key)
			require.NoError(t, err)
			require.False(t, has, "trailing entry at position %d under prefix %x should be deleted", i, prefix)
		}
	}

	// Unrelated prefixes must be fully intact.
	for _, u := range unrelatedPrefixes {
		for i := uint64(0); i < 10; i++ {
			key := append(append([]byte{}, u.prefix...), uint64ToKey(i)...)
			has, err := db.Has(key)
			require.NoError(t, err)
			require.True(t, has, "%s entry at position %d should not be touched", u.name, i)
		}
	}

	// Singleton keys must be intact.
	for _, key := range [][]byte{schema.DelayedMessageCountKey, schema.SequencerBatchCountKey} {
		has, err := db.Has(key)
		require.NoError(t, err)
		require.True(t, has, "singleton key %s should not be touched", key)
	}
}

// TestDeleteTrailingEntries_NoOpWhenClean verifies the function is a
// no-op when there are no orphaned entries beyond msgCount.
func TestDeleteTrailingEntries_NoOpWhenClean(t *testing.T) {
	t.Parallel()
	db := rawdb.NewMemoryDatabase()

	// Write exactly 3 entries under MessagePrefix — no trailing data.
	for i := uint64(0); i < 3; i++ {
		key := append(append([]byte{}, schema.MessagePrefix...), uint64ToKey(i)...)
		require.NoError(t, db.Put(key, []byte("data")))
	}

	require.NoError(t, deleteTrailingEntries(db, 3))

	// All 3 entries must still exist.
	for i := uint64(0); i < 3; i++ {
		key := append(append([]byte{}, schema.MessagePrefix...), uint64ToKey(i)...)
		has, err := db.Has(key)
		require.NoError(t, err)
		require.True(t, has, "entry at position %d should survive", i)
	}
}

// TestDeleteTrailingEntries_SparseEntriesBelowCount verifies that entries
// below msgCount survive even when there are gaps (not every position has
// an entry). Also verifies entries at exactly msgCount are deleted.
func TestDeleteTrailingEntries_SparseEntriesBelowCount(t *testing.T) {
	t.Parallel()
	db := rawdb.NewMemoryDatabase()

	// Write sparse entries at positions 0, 3, 4, 7, 9 under MessagePrefix.
	// With msgCount=5, positions 0, 3, 4 should survive; 7, 9 should be deleted.
	// Also write an entry at exactly position 5 (== msgCount) — should be deleted.
	for _, pos := range []uint64{0, 3, 4, 5, 7, 9} {
		key := append(append([]byte{}, schema.MessagePrefix...), uint64ToKey(pos)...)
		require.NoError(t, db.Put(key, []byte("data")))
	}

	require.NoError(t, deleteTrailingEntries(db, 5))

	// Positions below msgCount must survive.
	for _, pos := range []uint64{0, 3, 4} {
		key := append(append([]byte{}, schema.MessagePrefix...), uint64ToKey(pos)...)
		has, err := db.Has(key)
		require.NoError(t, err)
		require.True(t, has, "entry at position %d should survive (below msgCount)", pos)
	}

	// Positions at or above msgCount must be gone.
	for _, pos := range []uint64{5, 7, 9} {
		key := append(append([]byte{}, schema.MessagePrefix...), uint64ToKey(pos)...)
		has, err := db.Has(key)
		require.NoError(t, err)
		require.False(t, has, "entry at position %d should be deleted (>= msgCount)", pos)
	}
}

// TestDeleteTrailingEntries_ZeroCount treats all entries as trailing.
func TestDeleteTrailingEntries_ZeroCount(t *testing.T) {
	t.Parallel()
	db := rawdb.NewMemoryDatabase()

	for i := uint64(0); i < 3; i++ {
		key := append(append([]byte{}, schema.MessagePrefix...), uint64ToKey(i)...)
		require.NoError(t, db.Put(key, []byte("orphan")))
	}

	require.NoError(t, deleteTrailingEntries(db, 0))

	for i := uint64(0); i < 3; i++ {
		key := append(append([]byte{}, schema.MessagePrefix...), uint64ToKey(i)...)
		has, err := db.Has(key)
		require.NoError(t, err)
		require.False(t, has, "entry at position %d should be deleted when count is 0", i)
	}
}
