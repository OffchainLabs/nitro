// Copyright 2023-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package backlog

import (
	"errors"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/broadcaster/message"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/containers"
)

func validateBacklog(t *testing.T, b *backlog, count, start, end uint64, lookupKeys []arbutil.MessageIndex) {
	t.Helper()
	if b.Count() != count {
		t.Errorf("backlog message count (%d) does not equal expected message count (%d)", b.Count(), count)
	}

	head := b.head.Load()
	if start != 0 && head.Start() != start {
		t.Errorf("head of backlog (%d) does not equal expected head (%d)", head.Start(), start)
	}

	tail := b.tail.Load()
	if end != 0 && tail.End() != end {
		t.Errorf("tail of backlog (%d) does not equal expected tail (%d)", tail.End(), end)
	}

	// Must be exactly these keys: a stale entry keeps its whole forward segment
	// chain reachable, and checking only for presence would miss it.
	expKeys := make([]uint64, 0, len(lookupKeys))
	for _, k := range lookupKeys {
		expKeys = append(expKeys, uint64(k))
	}
	actualKeys := b.lookupByIndex.Keys()
	slices.Sort(expKeys)
	slices.Sort(actualKeys)
	if !slices.Equal(expKeys, actualKeys) {
		t.Errorf("lookupByIndex keys (%v) do not equal expected keys (%v)", actualKeys, expKeys)
	}

	if uint64(len(lookupKeys)) != b.Count() {
		t.Errorf("expected number of messages (%d) does not equal backlog message count (%d)", len(lookupKeys), b.Count())
	}
}

func validateBroadcastMessage(t *testing.T, bm *message.BroadcastMessage, expectedCount int, start, end uint64) {
	actualCount := len(bm.Messages)
	if actualCount != expectedCount {
		t.Errorf("number of messages returned (%d) does not equal the expected number of messages (%d)", actualCount, expectedCount)
		return
	}

	s := arbmath.MaxInt(start, 40)
	for i := s; i <= end; i++ {
		msg := bm.Messages[i-s]
		if uint64(msg.SequenceNumber) != i {
			t.Errorf("unexpected sequence number (%d) in %d returned message", i, i-s)
		}
	}
}

func createDummyBacklog(indexes []arbutil.MessageIndex) (*backlog, error) {
	b := &backlog{
		config: func() *Config { return &DefaultTestConfig },
	}
	b.lookupByIndex = containers.SyncMap[uint64, *backlogSegment]{}
	bm := &message.BroadcastMessage{Messages: message.CreateDummyBroadcastMessages(indexes)}
	err := b.Append(bm)
	return b, err
}

func TestAppend(t *testing.T) {
	testcases := []struct {
		name               string
		backlogIndexes     []arbutil.MessageIndex
		newIndexes         []arbutil.MessageIndex
		expectedCount      uint64
		expectedStart      uint64
		expectedEnd        uint64
		expectedLookupKeys []arbutil.MessageIndex
	}{
		{
			name:               "EmptyBacklog",
			backlogIndexes:     []arbutil.MessageIndex{},
			newIndexes:         []arbutil.MessageIndex{40, 41, 42, 43, 44, 45, 46},
			expectedCount:      7,
			expectedStart:      40,
			expectedEnd:        46,
			expectedLookupKeys: []arbutil.MessageIndex{40, 41, 42, 43, 44, 45, 46},
		},
		{
			name:               "NonEmptyBacklog",
			backlogIndexes:     []arbutil.MessageIndex{40, 41},
			newIndexes:         []arbutil.MessageIndex{42, 43, 44, 45, 46},
			expectedCount:      7,
			expectedStart:      40,
			expectedEnd:        46,
			expectedLookupKeys: []arbutil.MessageIndex{40, 41, 42, 43, 44, 45, 46},
		},
		{
			name:               "NonSequential",
			backlogIndexes:     []arbutil.MessageIndex{40, 41},
			newIndexes:         []arbutil.MessageIndex{42, 43, 45, 46},
			expectedCount:      2, // Message 45 is non sequential, the previous messages will be dropped from the backlog
			expectedStart:      45,
			expectedEnd:        46,
			expectedLookupKeys: []arbutil.MessageIndex{45, 46},
		},
		{
			name:               "NonSequentialSingleSegment",
			backlogIndexes:     []arbutil.MessageIndex{40, 41},
			newIndexes:         []arbutil.MessageIndex{45},
			expectedCount:      1, // Message 45 is non sequential and the backlog is a single segment, so head is the segment being emptied
			expectedStart:      45,
			expectedEnd:        45,
			expectedLookupKeys: []arbutil.MessageIndex{45},
		},
		{
			name:               "MessageSeen",
			backlogIndexes:     []arbutil.MessageIndex{40, 41},
			newIndexes:         []arbutil.MessageIndex{42, 43, 44, 45, 46, 41},
			expectedCount:      7, // Message 41 is already present in the backlog, it will be ignored
			expectedStart:      40,
			expectedEnd:        46,
			expectedLookupKeys: []arbutil.MessageIndex{40, 41, 42, 43, 44, 45, 46},
		},
		{
			name:               "NonSequentialFirstSegmentMessage",
			backlogIndexes:     []arbutil.MessageIndex{40, 41},
			newIndexes:         []arbutil.MessageIndex{42, 44, 45, 46},
			expectedCount:      3, // Message 44 is non sequential and the first message in a new segment, the previous messages will be dropped from the backlog
			expectedStart:      44,
			expectedEnd:        46,
			expectedLookupKeys: []arbutil.MessageIndex{44, 45, 46},
		},
		{
			name:               "MessageSeenFirstSegmentMessage",
			backlogIndexes:     []arbutil.MessageIndex{40, 41},
			newIndexes:         []arbutil.MessageIndex{42, 43, 44, 45, 41, 46},
			expectedCount:      7, // Message 41 is already present in the backlog and the first message in a new segment, it will be ignored
			expectedStart:      40,
			expectedEnd:        46,
			expectedLookupKeys: []arbutil.MessageIndex{40, 41, 42, 43, 44, 45, 46},
		},
		// There was a bug where if the last message was a duplicate it could insert an empty segment.
		{
			name:               "MessageSeenFirstSegmentMessageDoesntCreateNewSegment",
			backlogIndexes:     []arbutil.MessageIndex{40, 41},
			newIndexes:         []arbutil.MessageIndex{42, 43, 44, 45, 41},
			expectedCount:      6,
			expectedStart:      40,
			expectedEnd:        45,
			expectedLookupKeys: []arbutil.MessageIndex{40, 41, 42, 43, 44, 45},
		},
		// The above bug could also be used to insert messages out of order.
		{
			name:               "MyMessageSeenFirstSegmentMessageDoesntAllowOutOfOrderInsertion",
			backlogIndexes:     []arbutil.MessageIndex{40, 41},
			newIndexes:         []arbutil.MessageIndex{42, 43, 44, 45, 41, 1},
			expectedCount:      6,
			expectedStart:      40,
			expectedEnd:        45,
			expectedLookupKeys: []arbutil.MessageIndex{40, 41, 42, 43, 44, 45},
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

			bm := &message.BroadcastMessage{Messages: message.CreateDummyBroadcastMessages(tc.newIndexes)}
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
		messages: message.CreateDummyBroadcastMessages([]arbutil.MessageIndex{40, 42}),
	}

	b := &backlog{
		config: func() *Config { return &DefaultTestConfig },
	}
	b.lookupByIndex = containers.SyncMap[uint64, *backlogSegment]{}
	b.lookupByIndex.Store(40, s)
	b.messageCount.Store(2)
	b.head.Store(s)
	b.tail.Store(s)

	bm := &message.BroadcastMessage{
		Messages: nil,
		ConfirmedSequenceNumberMessage: &message.ConfirmedSequenceNumberMessage{
			SequenceNumber: 41,
		},
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
		expectedCount      uint64
		expectedStart      uint64
		expectedEnd        uint64
		expectedLookupKeys []arbutil.MessageIndex
	}{
		{
			name:               "EmptyBacklog",
			backlogIndexes:     []arbutil.MessageIndex{},
			confirmed:          0, // no segements in backlog so nothing to delete
			expectedCount:      0,
			expectedStart:      0,
			expectedEnd:        0,
			expectedLookupKeys: []arbutil.MessageIndex{},
		},
		{
			name:               "MsgBeforeBacklog",
			backlogIndexes:     []arbutil.MessageIndex{40, 41, 42, 43, 44, 45, 46},
			confirmed:          39, // no segments will be deleted
			expectedCount:      7,
			expectedStart:      40,
			expectedEnd:        46,
			expectedLookupKeys: []arbutil.MessageIndex{40, 41, 42, 43, 44, 45, 46},
		},
		{
			name:               "FirstMsgInBacklog",
			backlogIndexes:     []arbutil.MessageIndex{40, 41, 42, 43, 44, 45, 46},
			confirmed:          40, // this is the first message in the backlog
			expectedCount:      6,
			expectedStart:      41,
			expectedEnd:        46,
			expectedLookupKeys: []arbutil.MessageIndex{41, 42, 43, 44, 45, 46},
		},
		{
			name:               "FirstMsgInSegment",
			backlogIndexes:     []arbutil.MessageIndex{40, 41, 42, 43, 44, 45, 46},
			confirmed:          43, // this is the first message in a middle segment of the backlog
			expectedCount:      3,
			expectedStart:      44,
			expectedEnd:        46,
			expectedLookupKeys: []arbutil.MessageIndex{44, 45, 46},
		},
		{
			name:               "MiddleMsgInSegment",
			backlogIndexes:     []arbutil.MessageIndex{40, 41, 42, 43, 44, 45, 46},
			confirmed:          44, // this is a message in the middle of a middle segment of the backlog
			expectedCount:      2,
			expectedStart:      45,
			expectedEnd:        46,
			expectedLookupKeys: []arbutil.MessageIndex{45, 46},
		},
		{
			name:               "LastMsgInSegment",
			backlogIndexes:     []arbutil.MessageIndex{40, 41, 42, 43, 44, 45, 46},
			confirmed:          45, // this is the last message in a middle segment of the backlog, the whole segment should be deleted along with any segments before it
			expectedCount:      1,
			expectedStart:      46,
			expectedEnd:        46,
			expectedLookupKeys: []arbutil.MessageIndex{46},
		},
		{
			name:               "MsgAfterBacklog",
			backlogIndexes:     []arbutil.MessageIndex{40, 41, 42, 43, 44, 45, 46},
			confirmed:          47, // all segments will be deleted
			expectedCount:      0,
			expectedStart:      0,
			expectedEnd:        0,
			expectedLookupKeys: []arbutil.MessageIndex{},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			b, err := createDummyBacklog(tc.backlogIndexes)
			if err != nil {
				t.Fatalf("error creating dummy backlog: %s", err)
			}

			bm := &message.BroadcastMessage{
				Messages: nil,
				ConfirmedSequenceNumberMessage: &message.ConfirmedSequenceNumberMessage{
					SequenceNumber: tc.confirmed,
				},
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
func TestAppendDeleteAppend(t *testing.T) {
	b, err := createDummyBacklog([]arbutil.MessageIndex{40, 41, 42, 43, 44, 45, 46})
	if err != nil {
		t.Fatalf("error creating dummy backlog: %s", err)
	}
	validateBacklog(t, b, 7, 40, 46, []arbutil.MessageIndex{40, 41, 42, 43, 44, 45, 46})

	err = b.Append(&message.BroadcastMessage{
		ConfirmedSequenceNumberMessage: &message.ConfirmedSequenceNumberMessage{SequenceNumber: 43},
	})
	if err != nil {
		t.Fatalf("error appending confirmed sequence number: %s", err)
	}
	validateBacklog(t, b, 3, 44, 46, []arbutil.MessageIndex{44, 45, 46})

	err = b.Append(message.CreateDummyBroadcastMessage([]arbutil.MessageIndex{47, 48}))
	if err != nil {
		t.Fatalf("error appending BroadcastMessage: %s", err)
	}
	validateBacklog(t, b, 5, 44, 48, []arbutil.MessageIndex{44, 45, 46, 47, 48})
}

// TestDropSegmentsReleasesOldSegments covers errDropSegments emptying the head
// segment. Stale lookup entries must not survive: they keep their segment, and
// every segment linked after it, alive while the gauges stay flat.
func TestDropSegmentsReleasesOldSegments(t *testing.T) {
	b, err := createDummyBacklog([]arbutil.MessageIndex{40, 41})
	if err != nil {
		t.Fatalf("error creating dummy backlog: %s", err)
	}
	// The drop empties this segment and reuses it as the new head, so the object
	// outlives the drop and only ages out when the confirm below advances head.
	emptied := b.head.Load()
	if emptied != b.tail.Load() {
		t.Fatal("test requires the backlog to be a single segment so that head is the segment being appended to")
	}

	// 45 is ahead of the expected sequence number, so the segment is emptied.
	err = b.Append(message.CreateDummyBroadcastMessage([]arbutil.MessageIndex{45}))
	if err != nil {
		t.Fatalf("error appending BroadcastMessage: %s", err)
	}
	validateBacklog(t, b, 1, 45, 45, []arbutil.MessageIndex{45})

	// Give the emptied segment a nextSegment, then confirm past it so it leaves
	// the head to tail window.
	err = b.Append(message.CreateDummyBroadcastMessage([]arbutil.MessageIndex{46, 47, 48, 49, 50}))
	if err != nil {
		t.Fatalf("error appending BroadcastMessage: %s", err)
	}
	err = b.Append(&message.BroadcastMessage{
		ConfirmedSequenceNumberMessage: &message.ConfirmedSequenceNumberMessage{SequenceNumber: 47},
	})
	if err != nil {
		t.Fatalf("error appending confirmed sequence number: %s", err)
	}
	if b.head.Load() == emptied {
		t.Fatal("test setup: confirming 47 should have advanced head past the emptied segment")
	}
	validateBacklog(t, b, 3, 48, 50, []arbutil.MessageIndex{48, 49, 50})

	referencedBy := make(map[*backlogSegment][]uint64)
	for _, k := range b.lookupByIndex.Keys() {
		if segment, ok := b.lookupByIndex.Load(k); ok {
			referencedBy[segment] = append(referencedBy[segment], k)
		}
	}
	if keys, found := referencedBy[emptied]; found {
		t.Errorf("lookupByIndex keys %v still reference the segment emptied by the drop, keeping every later segment reachable", keys)
	}
}

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
		start         uint64
		end           uint64
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
			if tc.expectedErr == nil {
				validateBroadcastMessage(t, bm, tc.expectedCount, tc.start, tc.end)
			}
		})
	}
}

// TestBacklogSizeInBytesReleasesLockOnError ensures backlogSizeInBytes does
// not leak the segment RLock on its error paths. A leaked RLock would block
// every future writer on that segment (and, under Go's write-preferring
// RWMutex, every subsequent reader as well), deadlocking the backlog.
func TestBacklogSizeInBytesReleasesLockOnError(t *testing.T) {
	assertLockReleased := func(t *testing.T, name string, seg *backlogSegment) {
		t.Helper()
		if !seg.messagesLock.TryLock() {
			t.Fatalf("%s.messagesLock was left held after backlogSizeInBytes returned an error", name)
		}
		seg.messagesLock.Unlock()
	}

	newBacklog := func() *backlog {
		b := &backlog{config: func() *Config { return &DefaultTestConfig }}
		b.lookupByIndex = containers.SyncMap[uint64, *backlogSegment]{}
		return b
	}

	t.Run("EmptyTail", func(t *testing.T) {
		b := newBacklog()
		head := newBacklogSegment()
		head.messages = message.CreateDummyBroadcastMessages([]arbutil.MessageIndex{40})
		tail := newBacklogSegment()
		head.nextSegment.Store(tail)
		b.head.Store(head)
		b.tail.Store(tail)

		_, err := b.backlogSizeInBytes()
		const want = "tail segment of the feed backlog is empty"
		if err == nil || err.Error() != want {
			t.Fatalf("expected error %q, got %v", want, err)
		}
		assertLockReleased(t, "tail", tail)
		assertLockReleased(t, "head", head)
	})

	t.Run("EmptyHead", func(t *testing.T) {
		b := newBacklog()
		head := newBacklogSegment()
		tail := newBacklogSegment()
		tail.messages = message.CreateDummyBroadcastMessages([]arbutil.MessageIndex{40})
		head.nextSegment.Store(tail)
		b.head.Store(head)
		b.tail.Store(tail)

		_, err := b.backlogSizeInBytes()
		const want = "head segment of the feed backlog is empty"
		if err == nil || err.Error() != want {
			t.Fatalf("expected error %q, got %v", want, err)
		}
		assertLockReleased(t, "head", head)
	})
}

// TestBacklogRaceConditionFlaky performs read & write operations in separate
// goroutines to ensure that the backlog does not have race conditions. The
// `go test -race` command can be used to test this.
func TestBacklogRaceConditionFlaky(t *testing.T) {
	indexes := []arbutil.MessageIndex{40, 41, 42, 43, 44, 45, 46}
	b, err := createDummyBacklog(indexes)
	if err != nil {
		t.Fatalf("error creating dummy backlog: %s", err)
	}

	wg := sync.WaitGroup{}
	newIndexes := []arbutil.MessageIndex{47, 48, 49, 50, 51, 52, 53, 54, 55}

	// Write to backlog in goroutine
	wg.Add(1)
	errs := make(chan error, 15)
	go func(_t *testing.T, b *backlog) {
		defer wg.Done()
		for _, i := range newIndexes {
			bm := message.CreateDummyBroadcastMessage([]arbutil.MessageIndex{i})
			err := b.Append(bm)
			errs <- err
			if err != nil {
				return
			}
			time.Sleep(time.Millisecond)
		}
	}(t, b)

	// Read from backlog in goroutine
	wg.Add(1)
	go func(t *testing.T, b *backlog) {
		defer wg.Done()
		for _, i := range []uint64{42, 43, 44, 45, 46, 47} {
			bm, err := b.Get(i, i+1)
			errs <- err
			if err != nil {
				return
			} else {
				validateBroadcastMessage(t, bm, 2, i, i+1)
			}
			time.Sleep(2 * time.Millisecond)
		}
	}(t, b)

	// Delete from backlog in goroutine. This is normally done via Append with
	// a confirmed sequence number, using delete method for simplicity in test.
	wg.Add(1)
	go func(_t *testing.T, b *backlog) {
		defer wg.Done()
		for _, i := range []uint64{40, 43, 47} {
			b.delete(i)
			time.Sleep(10 * time.Millisecond)
		}
	}(t, b)

	// Wait for all goroutines to finish or return errors
	wg.Wait()
	close(errs)
	for err = range errs {

		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
	}
	// Messages up to 47 were deleted. However the segment that 47 was in is
	// kept, which is why the backlog starts at 46.
	validateBacklog(t, b, 8, 48, 55, newIndexes[1:])
}
