package backlog

import (
	"errors"
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
			[]arbutil.MessageIndex{40, 41, 42, 43, 44, 45, 46},
			7,
		},
		{
			"NonEmptyBacklog",
			[]arbutil.MessageIndex{40, 41},
			[]arbutil.MessageIndex{42, 43, 44, 45, 46},
			7,
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
			[]arbutil.MessageIndex{42, 44, 45, 46},
			3, // Message 44 is non sequential and the first message in a new segment, the previous messages will be dropped from the backlog
		},
		{
			"MessageSeenFirstSegmentMessage",
			[]arbutil.MessageIndex{40, 41},
			[]arbutil.MessageIndex{42, 43, 44, 45, 41, 46},
			7, // Message 41 is already present in the backlog and the first message in a new segment, it will be ignored
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			// The segment limit is 3, the above test cases have been created
			// to include testing certain actions on the first message of a
			// new segment.
			b, err := createDummyBacklog(tc.backlogIndexes, 3)
			if err != nil {
				t.Fatalf("error creating dummy backlog: %s", err)
			}

			bm := &m.BroadcastMessage{Messages: m.CreateDummyBroadcastMessages(tc.newIndexes)}
			err = b.Append(bm)
			if err != nil {
				t.Fatalf("error appending BroadcastMessage: %s", err)
			}

			if b.MessageCount() != tc.expectedCount {
				t.Fatalf("backlog message count (%d) does not equal expected message count (%d)", b.MessageCount(), tc.expectedCount)
			}
		})
	}
}

func TestDeleteInvalidBacklog(t *testing.T) {
	// Create a backlog with an invalid sequence
	s := &backlogSegment{
		start:    40,
		end:      42,
		messages: m.CreateDummyBroadcastMessages([]arbutil.MessageIndex{40, 42}),
	}
	s.messageCount.Store(2)

	p := atomic.Pointer[backlogSegment]{}
	p.Store(s)

	b := &Backlog{
		lookupByIndex: map[arbutil.MessageIndex]atomic.Pointer[backlogSegment]{40: p},
		segmentLimit:  func() int { return 3 },
	}
	b.messageCount.Store(2)
	b.head.Store(s)
	b.tail.Store(s)

	bm := &m.BroadcastMessage{
		Messages:                       nil,
		ConfirmedSequenceNumberMessage: &m.ConfirmedSequenceNumberMessage{41},
	}

	err := b.Append(bm)
	if err != nil {
		t.Fatalf("error appending BroadcastMessage: %s", err)
	}

	if b.MessageCount() != 0 {
		t.Fatalf("backlog message count (%d) does not equal expected message count (0)", b.MessageCount())
	}
}

// TestDelete
func TestDelete(t *testing.T) {
	testcases := []struct {
		name           string
		backlogIndexes []arbutil.MessageIndex
		confirmed      arbutil.MessageIndex
		expectedCount  int
	}{
		// empty backlog, delete should do nothing and just leave it the same
		{
			"EmptyBacklog",
			[]arbutil.MessageIndex{},
			0,
			0,
		},
		// delete message that appears before backlog, do nothing, leave backlog the same
		{
			"MsgBeforeBacklog",
			[]arbutil.MessageIndex{40, 41, 42, 43, 44, 45, 46},
			39,
			7,
		},
		// delete message that appears in backlog, deletes everything before that message in the backlog
		{
			"MsgInBacklog",
			[]arbutil.MessageIndex{40, 41, 42, 43, 44, 45, 46},
			43, // only the first segment will be deleted
			4,
		},
		{
			"MsgInFirstSegmentInBacklog",
			[]arbutil.MessageIndex{40, 41, 42, 43, 44, 45, 46},
			42,
			7,
		},
		// delete message that appears after backlog, deletes everything in the backlog, no error
		{
			"MsgAfterBacklog",
			[]arbutil.MessageIndex{40, 41, 42, 43, 44, 45, 46},
			47,
			0,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			b, err := createDummyBacklog(tc.backlogIndexes, 3)
			if err != nil {
				t.Fatalf("error creating dummy backlog: %s", err)
			}

			bm := &m.BroadcastMessage{
				Messages:                       nil,
				ConfirmedSequenceNumberMessage: &m.ConfirmedSequenceNumberMessage{tc.confirmed},
			}

			err = b.Append(bm)
			if err != nil {
				t.Fatalf("error appending BroadcastMessage: %s", err)
			}

			if b.MessageCount() != tc.expectedCount {
				t.Fatalf("backlog message count (%d) does not equal expected message count (%d)", b.MessageCount(), tc.expectedCount)
			}
		})
	}
}

// make sure that an append, then delete, then append ends up with the correct messageCounts

func TestGetEmptyBacklog(t *testing.T) {
	b, err := createDummyBacklog([]arbutil.MessageIndex{}, 3)
	if err != nil {
		t.Fatalf("error creating dummy backlog: %s", err)
	}

	_, err = b.Get(1, 2)
	if !errors.Is(err, errOutOfBounds) {
		t.Fatalf("unexpected error: %s", err)
	}
}

func TestGet(t *testing.T) {
	indexes := []arbutil.MessageIndex{40, 41, 42, 43, 44, 45, 46}
	b, err := createDummyBacklog(indexes, 3)
	if err != nil {
		t.Fatalf("error creating dummy backlog: %s", err)
	}

	testcases := []struct {
		name          string
		start         arbutil.MessageIndex
		end           arbutil.MessageIndex
		expectedErr   error
		expectedCount int
	}{
		{
			"LowerBoundFar",
			0,
			43,
			errOutOfBounds,
			0,
		},
		{
			"LowerBoundClose",
			39,
			43,
			errOutOfBounds,
			0,
		},
		{
			"UpperBoundFar",
			43,
			18446744073709551615,
			errOutOfBounds,
			0,
		},
		{
			"UpperBoundClose",
			0,
			47,
			errOutOfBounds,
			0,
		},
		{
			"AllMessages",
			40,
			46,
			nil,
			7,
		},
		{
			"SomeMessages",
			42,
			44,
			nil,
			3,
		},
		{
			"FirstMessage",
			40,
			40,
			nil,
			1,
		},
		{
			"LastMessage",
			46,
			46,
			nil,
			1,
		},
		{
			"SingleMessage",
			43,
			43,
			nil,
			1,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			bm, err := b.Get(tc.start, tc.end)
			if !errors.Is(err, tc.expectedErr) {
				t.Fatalf("unexpected error: %s", err)
			}

			// Some of the tests are checking the correct error is returned
			// Do not check bm if an error should be returned
			if err == nil {
				actualCount := len(bm.Messages)
				if actualCount != tc.expectedCount {
					t.Fatalf("number of messages returned (%d) does not equal the expected number of messages (%d)", actualCount, tc.expectedCount)
				}

				for i := tc.start; i <= tc.end; i++ {
					msg := bm.Messages[i-tc.start]
					if msg.SequenceNumber != i {
						t.Fatalf("unexpected sequence number (%d) in %d returned message", i, i-tc.start)
					}
				}
			}
		})
	}
}

func createDummyBacklog(indexes []arbutil.MessageIndex, segmentLimit int) (*Backlog, error) {
	b := &Backlog{
		lookupByIndex: map[arbutil.MessageIndex]atomic.Pointer[backlogSegment]{},
		segmentLimit:  func() int { return segmentLimit },
	}
	bm := &m.BroadcastMessage{Messages: m.CreateDummyBroadcastMessages(indexes)}
	err := b.Append(bm)
	return b, err
}
