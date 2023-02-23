// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package broadcaster

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/util/testhelpers"
	"github.com/offchainlabs/nitro/wsbroadcastserver"
)

type predicate interface {
	Test() bool
	Error() string
}

func waitUntilUpdated(t *testing.T, p predicate) {
	t.Helper()
	updateTimer := time.NewTimer(2 * time.Second)
	defer updateTimer.Stop()
	for {
		if p.Test() {
			break
		}
		select {
		case <-updateTimer.C:
			t.Fatalf("%s", p.Error())
		default:
		}
		time.Sleep(10 * time.Millisecond)
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

	config := wsbroadcastserver.DefaultTestBroadcasterConfig

	chainId := uint64(5555)
	feedErrChan := make(chan error, 10)
	b, err := NewBroadcaster(func() *wsbroadcastserver.BroadcasterConfig { return &config }, chainId, feedErrChan, nil)
	Require(t, err)
	Require(t, b.Initialize())
	Require(t, b.Start(ctx))
	defer b.StopAndWait()

	expectMessageCount := func(count int, contextMessage string) predicate {
		return &messageCountPredicate{b, count, contextMessage, 0}
	}

	// Normal broadcasting and confirming
	Require(t, b.BroadcastSingle(arbstate.EmptyTestMessageWithMetadata, 1))
	waitUntilUpdated(t, expectMessageCount(1, "after 1 message"))
	Require(t, b.BroadcastSingle(arbstate.EmptyTestMessageWithMetadata, 2))
	waitUntilUpdated(t, expectMessageCount(2, "after 2 messages"))
	Require(t, b.BroadcastSingle(arbstate.EmptyTestMessageWithMetadata, 3))
	waitUntilUpdated(t, expectMessageCount(3, "after 3 messages"))
	Require(t, b.BroadcastSingle(arbstate.EmptyTestMessageWithMetadata, 4))
	waitUntilUpdated(t, expectMessageCount(4, "after 4 messages"))

	b.Confirm(1)
	waitUntilUpdated(t, expectMessageCount(3,
		"after 4 messages, 1 cleared by confirm"))

	b.Confirm(2)
	waitUntilUpdated(t, expectMessageCount(2,
		"after 4 messages, 2 cleared by confirm"))

	b.Confirm(1)
	waitUntilUpdated(t, expectMessageCount(2,
		"nothing changed because confirmed sequence number before cache"))

	b.Confirm(2)
	Require(t, b.BroadcastSingle(arbstate.EmptyTestMessageWithMetadata, 5))
	waitUntilUpdated(t, expectMessageCount(3,
		"after 5 messages, 2 cleared by confirm"))

	// Confirm not-yet-seen or already confirmed/cleared sequence numbers twice to force clearing cache
	b.Confirm(6)
	waitUntilUpdated(t, expectMessageCount(0,
		"clear all messages after confirmed 1 beyond latest"))
}

func Require(t *testing.T, err error, printables ...interface{}) {
	t.Helper()
	testhelpers.RequireImpl(t, err, printables...)
}

func Fail(t *testing.T, printables ...interface{}) {
	t.Helper()
	testhelpers.FailImpl(t, printables...)
}
