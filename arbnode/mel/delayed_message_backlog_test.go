// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package mel

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDelayedMessageBacklog(t *testing.T) {
	backlog, err := NewDelayedMessageBacklog(100, func() (uint64, error) { return 0, nil })
	require.NoError(t, err)

	// Verify handling of dirties
	for i := uint64(0); i < 2; i++ {
		require.NoError(t, backlog.Add(&DelayedMessageBacklogEntry{Index: i}))
	}
	backlog.CommitDirties()
	require.True(t, backlog.dirtiesStartPos == 2)
	// Add dirties and verify that calling a clone returns a new struct without dirty entries,
	// leaving the original unchanged.
	for i := uint64(2); i < 5; i++ {
		require.NoError(t, backlog.Add(&DelayedMessageBacklogEntry{Index: i}))
	}
	cloneEarly := backlog.clone() // should return new struct with only committed entries
	require.True(t, len(cloneEarly.entries) == 2)
	require.True(t, len(backlog.entries) == 5) // original is unchanged
	numEntries := uint64(25)
	for i := uint64(5); i < numEntries; i++ { // continue from 5 since 2,3,4 are still in backlog
		require.NoError(t, backlog.Add(&DelayedMessageBacklogEntry{Index: i}))
	}
	backlog.CommitDirties()
	// #nosec G115
	require.True(t, uint64(backlog.dirtiesStartPos) == numEntries)

	// Test that clone works - compare comparable fields (finalizedAndReadIndexFetcher is a func and cannot be compared with reflect.DeepEqual)
	cloned := backlog.clone()
	require.Equal(t, cloned.targetBufferSize, backlog.targetBufferSize)
	require.Equal(t, cloned.dirtiesStartPos, backlog.dirtiesStartPos)
	require.Equal(t, cloned.initMessage, backlog.initMessage)
	require.Equal(t, len(cloned.entries), len(backlog.entries))
	for i, entry := range cloned.entries {
		require.Equal(t, entry, backlog.entries[i])
	}

	// Test failures with Get
	// Entry not found
	_, err = backlog.Get(numEntries + 1)
	if err == nil {
		t.Fatal("backlog Get function should've errored for an invalid index query")
	}
	if !strings.Contains(err.Error(), "out of bounds") {
		t.Fatalf("unexpected error: %s", err.Error())
	}
	// Index mismatch
	failIndex := uint64(3)
	backlog.entries[failIndex].Index = failIndex + 1 // shouldnt match
	_, err = backlog.Get(failIndex)
	if err == nil {
		t.Fatal("backlog Get function should've errored for an invalid entry in the backlog")
	}
	if !strings.Contains(err.Error(), "index mismatch in the delayed message backlog entry") {
		t.Fatalf("unexpected error: %s", err.Error())
	}

	// Verify that advancing the finalizedAndRead will trim the delayedMessageBacklogEntry while keeping the unread ones
	finalizedAndRead := uint64(7)
	backlog.finalizedAndReadIndexFetcher = func() (uint64, error) { return finalizedAndRead, nil }
	backlog.trimFinalizedAndReadEntries()
	require.True(t, len(backlog.entries) == int(numEntries-finalizedAndRead)) // #nosec G115
	require.True(t, backlog.entries[0].Index == finalizedAndRead)

	// Verify that Reorg handling works as expected, reorg of 5 indexes
	newSeen := numEntries - 5
	require.NoError(t, backlog.reorg(newSeen))
	// as newDelayedMessageBacklog hasnt updated with new finalized info, its starting elements remain unchanged, just that the right parts are trimmed till (newSeen-1) delayed index
	require.True(t, len(backlog.entries) == int(newSeen-finalizedAndRead)) // #nosec G115
	require.True(t, backlog.entries[len(backlog.entries)-1].Index == newSeen-1)
}
