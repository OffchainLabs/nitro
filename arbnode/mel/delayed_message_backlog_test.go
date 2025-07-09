package mel

import (
	"context"
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
