//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package broadcaster

import (
	"context"
	"testing"
	"time"

	"github.com/offchainlabs/arbitrum/packages/arb-util/configuration"
	"github.com/offchainlabs/arbstate/arbstate"
)

func waitUntilUpdated(t *testing.T, testFn func() bool, errText string) {
	updateTimeout := time.After(2 * time.Second)
	for {
		if testFn() {
			break
		}
		select {
		case <-updateTimeout:
			t.Fatalf("Failed waiting for %s", errText)
		case <-time.After(10 * time.Millisecond):
		}
	}
}

func TestBroadcasterMessagesRemovedOnConfirmation(t *testing.T) {
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	broadcasterSettings := configuration.FeedOutput{
		Addr:          "0.0.0.0",
		IOTimeout:     2 * time.Second,
		Port:          "9642",
		Ping:          5 * time.Second,
		ClientTimeout: 30 * time.Second,
		Queue:         1,
		Workers:       128,
	}

	b := NewBroadcaster(broadcasterSettings)
	err := b.Start(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Stop()

	dummyMessage := arbstate.MessageWithMetadata{}
	expectMessageCount := func(count int) func() bool {
		return (func() bool { return b.catchupBuffer.GetMessageCount() == count })
	}

	b.BroadcastSingle(dummyMessage, 1)
	waitUntilUpdated(t, expectMessageCount(1), "update count after 1 message")
	b.BroadcastSingle(dummyMessage, 2)
	waitUntilUpdated(t, expectMessageCount(2), "update count after 2 message")
	b.BroadcastSingle(dummyMessage, 3)
	waitUntilUpdated(t, expectMessageCount(3), "update count after 3 message")
	b.BroadcastSingle(dummyMessage, 4)
	waitUntilUpdated(t, expectMessageCount(4), "update count after 4 message")

	b.Confirm(1)
	waitUntilUpdated(t, expectMessageCount(3),
		"update count after 4 message, 1 cleared")

	b.Confirm(3)
	waitUntilUpdated(t, expectMessageCount(1),
		"update count after 4 message, 3 cleared")

	b.BroadcastSingle(dummyMessage, 5)
	waitUntilUpdated(t, expectMessageCount(2),
		"update count after 5 message, 3 cleared")

	b.Confirm(10)
	waitUntilUpdated(t, expectMessageCount(0),
		"update count after messages confirmed beyond latest")

	b.BroadcastSingle(dummyMessage, 6)
	b.BroadcastSingle(dummyMessage, 7)
	b.BroadcastSingle(dummyMessage, 8)
	b.BroadcastSingle(dummyMessage, 9)
	b.Confirm(2)

	waitUntilUpdated(t, expectMessageCount(4),
		"don't update count after confirming already confirmed messages")

	b.Confirm(7)
	waitUntilUpdated(t, expectMessageCount(2),
		"update count after 4 mesages, 2 cleared")

	b.Confirm(9)
	waitUntilUpdated(t, expectMessageCount(0),
		"update count after messages cleared up to latest")
}
