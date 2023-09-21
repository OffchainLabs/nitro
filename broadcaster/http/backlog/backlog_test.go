package backlog

import (
	"sync/atomic"
	"testing"

	"github.com/offchainlabs/nitro/arbutil"
	m "github.com/offchainlabs/nitro/broadcaster/message"
)

func TestAppend(t *testing.T) {
	testcases := []struct {
		name           string
		backlogIndexes []arbutil.MessageIndex
		newIndexes     []arbutil.MessageIndex
		expectedCount  int
	}{
		{
			"EmptyBacklog",
			[]arbutil.MessageIndex{},
			[]arbutil.MessageIndex{40, 41, 42, 43, 44, 45},
			6,
		},
		{
			"NonEmptyBacklog",
			[]arbutil.MessageIndex{40, 41},
			[]arbutil.MessageIndex{42, 43, 44, 45},
			6,
		},
		{
			"NonSequential",
			[]arbutil.MessageIndex{40, 41},
			[]arbutil.MessageIndex{42, 43, 45, 46},
			2, // Message 45 is non sequential, the previous messages will be dropped from the backlog
		},
		{
			"MessageSeen",
			[]arbutil.MessageIndex{40, 41},
			[]arbutil.MessageIndex{42, 43, 44, 45, 46, 41},
			7, // Message 41 is already present in the backlog, it will be ignored
		},
		{
			"NonSequentialFirstSegmentMessage",
			[]arbutil.MessageIndex{40, 41},
			[]arbutil.MessageIndex{42, 44, 45},
			2, // Message 44 is non sequential and the first message in a new segment, the previous messages will be dropped from the backlog
		},
		{
			"MessageSeenFirstSegmentMessage",
			[]arbutil.MessageIndex{40, 41},
			[]arbutil.MessageIndex{42, 43, 44, 45, 41},
			6, // Message 41 is already present in the backlog and the first message in a new segment, it will be ignored
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			// The segment limit is 3, the above test cases have been created
			// to include testing certain actions on the first message of a
			// new segment.
			b, err := createDummyBacklog(tc.backlogIndexes, 3)
			if err != nil {
				t.Fatalf("error creating backlog: %s", err.Error())
			}

			bm := &m.BroadcastMessage{Messages: m.CreateDummyBroadcastMessages(tc.newIndexes)}
			err = b.Append(bm)
			if err != nil {
				t.Fatalf("error appending BroadcastMessage: %s", err)
			}

			t.Logf("%v", b.lookupByIndex)
			if b.MessageCount() != tc.expectedCount {
				t.Fatalf("backlog message count (%d) does not equal expected message count (%d)", b.MessageCount(), tc.expectedCount)
			}
		})
	}
}

// TestDelete

// TestGet

// make sure that an append, then delete, then append ends up with the correct messageCounts

func createDummyBacklog(indexes []arbutil.MessageIndex, segmentLimit int) (*Backlog, error) {
	b := &Backlog{
		lookupByIndex: map[arbutil.MessageIndex]atomic.Pointer[backlogSegment]{},
		segmentLimit:  func() int { return segmentLimit },
	}
	bm := &m.BroadcastMessage{Messages: m.CreateDummyBroadcastMessages(indexes)}
	err := b.Append(bm)
	return b, err
}
