//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package broadcaster

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/offchainlabs/arbstate/arbstate"
	"github.com/offchainlabs/arbstate/util/testhelpers"
	"github.com/offchainlabs/arbstate/wsbroadcastserver"
)

type predicate interface {
	Test() bool
	Error() string
}

func waitUntilUpdated(t *testing.T, p predicate) {
	updateTimeout := time.After(2 * time.Second)
	for {
		if p.Test() {
			break
		}
		select {
		case <-updateTimeout:
			t.Fatalf("%s", p.Error())
		case <-time.After(10 * time.Millisecond):
		}
	}
}

type messageCountPredicate struct {
	b              *Broadcaster
	expected       int
	contextMessage string
	was            int
}

func (p *messageCountPredicate) Test() bool {
	p.was = p.b.catchupBuffer.GetMessageCount()
	return p.was == p.expected
}

func (p *messageCountPredicate) Error() string {
	return fmt.Sprintf("Expected %d, was %d: %s", p.expected, p.was, p.contextMessage)
}

func TestBroadcasterMessagesRemovedOnConfirmation(t *testing.T) {
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	broadcasterSettings := wsbroadcastserver.BroadcasterConfig{
		Addr:          "0.0.0.0",
		IOTimeout:     2 * time.Second,
		Port:          "9642",
		Ping:          5 * time.Second,
		ClientTimeout: 30 * time.Second,
		Queue:         1,
		Workers:       128,
	}

	b := NewBroadcaster(broadcasterSettings)
	Require(t, b.Start(ctx))
	defer b.StopAndWait()

	dummyMessage := arbstate.MessageWithMetadata{}
	expectMessageCount := func(count int, contextMessage string) predicate {
		return &messageCountPredicate{b, count, contextMessage, 0}
	}

	// Normal broadcasting and confirming
	b.BroadcastSingle(dummyMessage, 1)
	waitUntilUpdated(t, expectMessageCount(1, "after 1 message"))
	b.BroadcastSingle(dummyMessage, 2)
	waitUntilUpdated(t, expectMessageCount(2, "after 2 messages"))
	b.BroadcastSingle(dummyMessage, 3)
	waitUntilUpdated(t, expectMessageCount(3, "after 3 messages"))
	b.BroadcastSingle(dummyMessage, 4)
	waitUntilUpdated(t, expectMessageCount(4, "after 4 messages"))

	b.Confirm(1)
	waitUntilUpdated(t, expectMessageCount(3,
		"after 4 messages, 1 cleared"))

	b.Confirm(3)
	waitUntilUpdated(t, expectMessageCount(1,
		"after 4 messages, 3 cleared"))

	b.BroadcastSingle(dummyMessage, 5)
	waitUntilUpdated(t, expectMessageCount(2,
		"after 5 messages, 3 cleared"))

	// Confirm not-yet-seen or already confirmed/cleared sequence numbers
	b.Confirm(7)
	waitUntilUpdated(t, expectMessageCount(0,
		"clear all messages after confirmed 1 beyond latest"))

	b.BroadcastSingle(dummyMessage, 3)
	b.BroadcastSingle(dummyMessage, 4)
	b.BroadcastSingle(dummyMessage, 5)
	b.BroadcastSingle(dummyMessage, 6)
	b.Confirm(2)
	waitUntilUpdated(t, expectMessageCount(4,
		"don't update count after confirming already confirmed messages"))

	b.Confirm(4)
	waitUntilUpdated(t, expectMessageCount(2,
		"update count after 4 mesages, 2 cleared"))

	b.Confirm(9)
	waitUntilUpdated(t, expectMessageCount(0,
		"clear all messages after confirmed 3 beyond latest"))

	// Duplicates and messages already seen
	b.BroadcastSingle(dummyMessage, 2)
	b.BroadcastSingle(dummyMessage, 0)
	b.BroadcastSingle(dummyMessage, 1)
	b.BroadcastSingle(dummyMessage, 2)
	waitUntilUpdated(t, expectMessageCount(1,
		"1 message after duplicates and already seen messages"))

}

func Require(t *testing.T, err error, text ...string) {
	t.Helper()
	testhelpers.RequireImpl(t, err, text...)
}

func Fail(t *testing.T, printables ...interface{}) {
	t.Helper()
	testhelpers.FailImpl(t, printables...)
}
