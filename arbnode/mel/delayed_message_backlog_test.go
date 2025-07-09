package mel

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDelayedMessageBacklog(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	backlog := NewDelayedMessageBacklog(ctx, 0, nil)
	numEntries := uint64(25)
	for i := uint64(0); i < numEntries; i++ {
		require.NoError(t, backlog.Add(&DelayedMessageBacklogEntry{Index: i}))
	}

	// Test that clone works
	cloned := backlog.clone()
	if !reflect.DeepEqual(backlog, cloned) {
		t.Fatal("cloned doesnt match original")
	}

	// Test failures with Get
	// Entry not found
	_, err := backlog.Get(numEntries + 1)
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
	backlog.finalizedAndReadIndexFetcher = func(context.Context) (uint64, error) { return finalizedAndRead, nil }
	require.NoError(t, backlog.clear())
	require.True(t, len(backlog.entries) == int(numEntries-finalizedAndRead)) // #nosec G115
	require.True(t, backlog.entries[0].Index == finalizedAndRead)

	// Verify that Reorg handling works as expected, reorg of 5 indexes
	newSeen := numEntries - 5
	require.NoError(t, backlog.reorg(newSeen))
	// as newDelayedMessageBacklog hasnt updated with new finalized info, its starting elements remain unchanged, just that the right parts are trimmed till (newSeen-1) delayed index
	require.True(t, len(backlog.entries) == int(newSeen-finalizedAndRead)) // #nosec G115
	require.True(t, backlog.entries[len(backlog.entries)-1].Index == newSeen-1)
}
