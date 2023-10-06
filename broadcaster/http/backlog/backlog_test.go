package backlog

import (
	"errors"
	"sync/atomic"
	"testing"

	"github.com/offchainlabs/nitro/arbutil"
	m "github.com/offchainlabs/nitro/broadcaster/message"
	"github.com/offchainlabs/nitro/util/arbmath"
)

func validateBacklog(t *testing.T, b *backlog, count int, start, end arbutil.MessageIndex, lookupKeys []arbutil.MessageIndex) {
	if b.MessageCount() != count {
		t.Errorf("backlog message count (%d) does not equal expected message count (%d)", b.MessageCount(), count)
	}

	head := b.head.Load()
	if start != 0 && head.start != start {
		t.Errorf("head of backlog (%d) does not equal expected head (%d)", head.start, start)
	}

	tail := b.tail.Load()
	if end != 0 && tail.end != end {
		t.Errorf("tail of backlog (%d) does not equal expected tail (%d)", tail.end, end)
	}

	for _, k := range lookupKeys {
		if _, ok := b.lookupByIndex[k]; !ok {
			t.Errorf("failed to find message (%d) in lookup", k)
		}
	}
}

func createDummyBacklog(indexes []arbutil.MessageIndex) (*backlog, error) {
	b := &backlog{
		lookupByIndex: map[arbutil.MessageIndex]atomic.Pointer[backlogSegment]{},
		config:        func() *Config { return &DefaultTestConfig },
	}
	bm := &m.BroadcastMessage{Messages: m.CreateDummyBroadcastMessages(indexes)}
	err := b.Append(bm)
	return b, err
}

func TestAppend(t *testing.T) {
	testcases := []struct {
		name               string
		backlogIndexes     []arbutil.MessageIndex
		newIndexes         []arbutil.MessageIndex
		expectedCount      int
		expectedStart      arbutil.MessageIndex
		expectedEnd        arbutil.MessageIndex
		expectedLookupKeys []arbutil.MessageIndex
	}{
		{
			"EmptyBacklog",
			[]arbutil.MessageIndex{},
			[]arbutil.MessageIndex{40, 41, 42, 43, 44, 45, 46},
			7,
			40,
			46,
			[]arbutil.MessageIndex{40, 41, 42, 43, 44, 45, 46},
		},
		{
			"NonEmptyBacklog",
			[]arbutil.MessageIndex{40, 41},
			[]arbutil.MessageIndex{42, 43, 44, 45, 46},
			7,
			40,
			46,
			[]arbutil.MessageIndex{40, 41, 42, 43, 44, 45, 46},
		},
		{
			"NonSequential",
			[]arbutil.MessageIndex{40, 41},
			[]arbutil.MessageIndex{42, 43, 45, 46},
			2, // Message 45 is non sequential, the previous messages will be dropped from the backlog
			45,
			46,
			[]arbutil.MessageIndex{45, 46},
		},
		{
			"MessageSeen",
			[]arbutil.MessageIndex{40, 41},
			[]arbutil.MessageIndex{42, 43, 44, 45, 46, 41},
			7, // Message 41 is already present in the backlog, it will be ignored
			40,
			46,
			[]arbutil.MessageIndex{40, 41, 42, 43, 44, 45, 46},
		},
		{
			"NonSequentialFirstSegmentMessage",
			[]arbutil.MessageIndex{40, 41},
			[]arbutil.MessageIndex{42, 44, 45, 46},
			3, // Message 44 is non sequential and the first message in a new segment, the previous messages will be dropped from the backlog
			44,
			46,
			[]arbutil.MessageIndex{45, 46},
		},
		{
			"MessageSeenFirstSegmentMessage",
			[]arbutil.MessageIndex{40, 41},
			[]arbutil.MessageIndex{42, 43, 44, 45, 41, 46},
			7, // Message 41 is already present in the backlog and the first message in a new segment, it will be ignored
			40,
			46,
			[]arbutil.MessageIndex{40, 41, 42, 43, 44, 45, 46},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			// The segment limit is 3, the above test cases have been created
			// to include testing certain actions on the first message of a
			// new segment.
			b, err := createDummyBacklog(tc.backlogIndexes)
			if err != nil {
				t.Fatalf("error creating dummy backlog: %s", err)
			}

			bm := &m.BroadcastMessage{Messages: m.CreateDummyBroadcastMessages(tc.newIndexes)}
			err = b.Append(bm)
			if err != nil {
				t.Fatalf("error appending BroadcastMessage: %s", err)
			}

			validateBacklog(t, b, tc.expectedCount, tc.expectedStart, tc.expectedEnd, tc.expectedLookupKeys)
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

	b := &backlog{
		lookupByIndex: map[arbutil.MessageIndex]atomic.Pointer[backlogSegment]{40: p},
		config:        func() *Config { return &DefaultTestConfig },
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

	validateBacklog(t, b, 0, 0, 0, []arbutil.MessageIndex{})
}

func TestDelete(t *testing.T) {
	testcases := []struct {
		name               string
		backlogIndexes     []arbutil.MessageIndex
		confirmed          arbutil.MessageIndex
		expectedCount      int
		expectedStart      arbutil.MessageIndex
		expectedEnd        arbutil.MessageIndex
		expectedLookupKeys []arbutil.MessageIndex
	}{
		{
			"EmptyBacklog",
			[]arbutil.MessageIndex{},
			0, // no segements in backlog so nothing to delete
			0,
			0,
			0,
			[]arbutil.MessageIndex{},
		},
		{
			"MsgBeforeBacklog",
			[]arbutil.MessageIndex{40, 41, 42, 43, 44, 45, 46},
			39, // no segments will be deleted
			7,
			40,
			46,
			[]arbutil.MessageIndex{40, 41, 42, 43, 44, 45, 46},
		},
		{
			"MsgInBacklog",
			[]arbutil.MessageIndex{40, 41, 42, 43, 44, 45, 46},
			43, // only the first segment will be deleted
			4,
			43,
			46,
			[]arbutil.MessageIndex{43, 44, 45, 46},
		},
		{
			"MsgInFirstSegmentInBacklog",
			[]arbutil.MessageIndex{40, 41, 42, 43, 44, 45, 46},
			42, // first segment will not be deleted as confirmed message is there
			7,
			40,
			46,
			[]arbutil.MessageIndex{40, 41, 42, 43, 44, 45, 46},
		},
		{
			"MsgAfterBacklog",
			[]arbutil.MessageIndex{40, 41, 42, 43, 44, 45, 46},
			47, // all segments will be deleted
			0,
			0,
			0,
			[]arbutil.MessageIndex{},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			b, err := createDummyBacklog(tc.backlogIndexes)
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

			validateBacklog(t, b, tc.expectedCount, tc.expectedStart, tc.expectedEnd, tc.expectedLookupKeys)
		})
	}
}

// make sure that an append, then delete, then append ends up with the correct messageCounts

func TestGetEmptyBacklog(t *testing.T) {
	b, err := createDummyBacklog([]arbutil.MessageIndex{})
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
	b, err := createDummyBacklog(indexes)
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
			nil,
			4,
		},
		{
			"LowerBoundClose",
			39,
			43,
			nil,
			4,
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

				start := arbmath.MaxInt(tc.start, 40)
				for i := start; i <= tc.end; i++ {
					msg := bm.Messages[i-start]
					if msg.SequenceNumber != i {
						t.Fatalf("unexpected sequence number (%d) in %d returned message", i, i-tc.start)
					}
				}
			}
		})
	}
}
