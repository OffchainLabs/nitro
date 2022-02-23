//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package broadcastclient

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/offchainlabs/arbstate/arbstate"
	"github.com/offchainlabs/arbstate/arbutil"
	"github.com/offchainlabs/arbstate/broadcaster"
	"github.com/offchainlabs/arbstate/wsbroadcastserver"
)

func TestReceiveMessages(t *testing.T) {
	ctx := context.Background()

	settings := wsbroadcastserver.BroadcasterConfig{
		Addr:          "0.0.0.0",
		IOTimeout:     2 * time.Second,
		Port:          "9742",
		Ping:          5 * time.Second,
		ClientTimeout: 20 * time.Second,
		Queue:         1,
		Workers:       128,
	}

	messageCount := 1000
	clientCount := 2

	b := broadcaster.NewBroadcaster(settings)

	err := b.Start(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Stop()

	var wg sync.WaitGroup
	for i := 0; i < clientCount; i++ {
		wg.Add(1)
		startMakeBroadcastClient(ctx, t, i, messageCount, &wg)
	}

	go func() {
		for i := 0; i < messageCount; i++ {
			b.BroadcastSingle(arbstate.MessageWithMetadata{}, arbutil.MessageIndex(i))
		}
	}()

	wg.Wait()

}

type dummyTransactionStreamer struct {
	messageReceiver chan broadcaster.BroadcastFeedMessage
}

func NewDummyTransactionStreamer() *dummyTransactionStreamer {
	return &dummyTransactionStreamer{
		messageReceiver: make(chan broadcaster.BroadcastFeedMessage),
	}
}

func (ts *dummyTransactionStreamer) AddMessages(pos arbutil.MessageIndex, force bool, messages []arbstate.MessageWithMetadata) error {
	for i, message := range messages {
		ts.messageReceiver <- broadcaster.BroadcastFeedMessage{
			SequenceNumber: pos + arbutil.MessageIndex(i),
			Message:        message,
		}
	}
	return nil
}

func startMakeBroadcastClient(ctx context.Context, t *testing.T, index int, expectedCount int, wg *sync.WaitGroup) {
	ts := NewDummyTransactionStreamer()
	broadcastClient := NewBroadcastClient("ws://127.0.0.1:9742/", nil, 20*time.Second, ts)
	broadcastClient.Start(ctx)
	messageCount := 0

	go func() {
		defer wg.Done()
		defer broadcastClient.Close()
		for {
			select {
			case <-ts.messageReceiver:
				messageCount++

				if messageCount == expectedCount {
					return
				}
			case <-time.After(60 * time.Second):
				t.Errorf("Client %d expected %d meesages, only got %d messages\n", index, expectedCount, messageCount)
				return
			}
		}
	}()

}

func TestServerClientDisconnect(t *testing.T) {
	ctx := context.Background()

	settings := wsbroadcastserver.BroadcasterConfig{
		Addr:          "0.0.0.0",
		IOTimeout:     2 * time.Second,
		Port:          "9743",
		Ping:          1 * time.Second,
		ClientTimeout: 2 * time.Second,
		Queue:         1,
		Workers:       128,
	}

	b := broadcaster.NewBroadcaster(settings)

	err := b.Start(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Stop()

	ts := NewDummyTransactionStreamer()
	broadcastClient := NewBroadcastClient("ws://127.0.0.1:9743/", nil, 20*time.Second, ts)
	broadcastClient.Start(ctx)

	b.BroadcastSingle(arbstate.MessageWithMetadata{}, 0)

	// Wait for client to receive batch to ensure it is connected
	select {
	case receivedMsg := <-ts.messageReceiver:
		t.Logf("Received Message, Sequence Message: %v\n", receivedMsg)
	case <-time.After(5 * time.Second):
		t.Fatal("Client did not receive batch item")
	}

	broadcastClient.Close()

	disconnectTimeout := time.After(5 * time.Second)
	for {
		if b.ClientCount() == 0 {
			break
		}

		select {
		case <-disconnectTimeout:
			t.Fatal("Client was not disconnected")
		case <-time.After(100 * time.Millisecond):
		}
	}
}

func TestBroadcastClientReconnectsOnServerDisconnect(t *testing.T) {
	ctx := context.Background()

	settings := wsbroadcastserver.BroadcasterConfig{
		Addr:          "0.0.0.0",
		IOTimeout:     2 * time.Second,
		Port:          "9743",
		Ping:          50 * time.Second,
		ClientTimeout: 150 * time.Second,
		Queue:         1,
		Workers:       128,
	}

	b1 := broadcaster.NewBroadcaster(settings)

	err := b1.Start(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer b1.Stop()

	broadcastClient := NewBroadcastClient("ws://127.0.0.1:9743/", nil, 2*time.Second, nil)

	broadcastClient.Start(ctx)

	// Client set to timeout connection at 2 seconds, and server set to send ping every 50 seconds,
	// so at least one timeout/reconnect should happen after 4 seconds
	time.Sleep(4 * time.Second)

	if broadcastClient.GetRetryCount() <= 0 {
		t.Error("Should have had some retry counts")
	}
}

func TestBroadcasterSendsCachedMessagesOnClientConnect(t *testing.T) {
	ctx := context.Background()

	settings := wsbroadcastserver.BroadcasterConfig{
		Addr:          "0.0.0.0",
		IOTimeout:     2 * time.Second,
		Port:          "9842",
		Ping:          5 * time.Second,
		ClientTimeout: 15 * time.Second,
		Queue:         1,
		Workers:       128,
	}

	b := broadcaster.NewBroadcaster(settings)

	err := b.Start(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Stop()

	b.BroadcastSingle(arbstate.MessageWithMetadata{}, 0)
	b.BroadcastSingle(arbstate.MessageWithMetadata{}, 1)

	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		connectAndGetCachedMessages(ctx, t, i, &wg)
	}

	wg.Wait()

	// give the above connections time to reconnect
	time.Sleep(4 * time.Second)

	// Confirmed Accumulator will also broadcast to the clients.
	b.Confirm(0) // remove the first message we generated

	updateTimeout := time.After(2 * time.Second)
	for {
		if b.GetCachedMessageCount() == 1 { // should have left the second message
			break
		}

		select {
		case <-updateTimeout:
			t.Fatal("confirmed accumulator did not get updated")
		case <-time.After(10 * time.Millisecond):
		}
	}

	// Send second accumulator again so that the previously added accumulator is sent
	b.Confirm(1)

	updateTimeout = time.After(2 * time.Second)
	for {
		if b.GetCachedMessageCount() == 0 { // should have left the second message
			break
		}

		select {
		case <-updateTimeout:
			t.Fatal("cache did not get cleared")
		case <-time.After(10 * time.Millisecond):
		}
	}
}

func connectAndGetCachedMessages(ctx context.Context, t *testing.T, clientIndex int, wg *sync.WaitGroup) {
	ts := NewDummyTransactionStreamer()
	broadcastClient := NewBroadcastClient("ws://127.0.0.1:9842/", nil, 60*time.Second, ts)
	broadcastClient.Start(ctx)

	go func() {
		defer wg.Done()
		defer broadcastClient.Close()

		// Wait for client to receive first item
		select {
		case receivedMsg := <-ts.messageReceiver:
			t.Logf("client %d received first message: %v\n", clientIndex, receivedMsg)
		case <-time.After(10 * time.Second):
			t.Errorf("client %d did not receive first batch item\n", clientIndex)
			return
		}

		// Wait for client to receive second item
		select {
		case receivedMsg := <-ts.messageReceiver:
			t.Logf("client %d received second message: %v\n", clientIndex, receivedMsg)
		case <-time.After(10 * time.Second):
			t.Errorf("client %d did not receive second batch item\n", clientIndex)
			return
		}
	}()
}
