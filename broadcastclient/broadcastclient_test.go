// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package broadcastclient

import (
	"context"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/broadcaster"
	"github.com/offchainlabs/nitro/wsbroadcastserver"
)

func TestReceiveMessages(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	settings := wsbroadcastserver.BroadcasterConfig{
		Addr:          "0.0.0.0",
		IOTimeout:     2 * time.Second,
		Port:          "0",
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
	defer b.StopAndWait()

	var wg sync.WaitGroup
	for i := 0; i < clientCount; i++ {
		startMakeBroadcastClient(ctx, t, b.ListenerAddr(), i, messageCount, &wg)
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

func newTestBroadcastClient(listenerAddress net.Addr, idleTimeout time.Duration, txStreamer TransactionStreamerInterface) *BroadcastClient {
	port := listenerAddress.(*net.TCPAddr).Port
	return NewBroadcastClient(fmt.Sprintf("ws://127.0.0.1:%d/", port), nil, idleTimeout, txStreamer)
}

func startMakeBroadcastClient(ctx context.Context, t *testing.T, addr net.Addr, index int, expectedCount int, wg *sync.WaitGroup) {
	ts := NewDummyTransactionStreamer()
	broadcastClient := newTestBroadcastClient(addr, 20*time.Second, ts)
	broadcastClient.Start(ctx)
	messageCount := 0

	wg.Add(1)

	go func() {
		defer wg.Done()
		defer broadcastClient.StopAndWait()
		for {
			gotMsg := false
			timer := time.NewTimer(60 * time.Second)
			select {
			case <-ts.messageReceiver:
				messageCount++
				gotMsg = true
			case <-timer.C:
			case <-ctx.Done():
			}
			timer.Stop()
			if !gotMsg {
				t.Errorf("Client %d expected %d meesages, only got %d messages\n", index, expectedCount, messageCount)
				return
			}
			if messageCount == expectedCount {
				return
			}
		}
	}()

}

func TestServerClientDisconnect(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	settings := wsbroadcastserver.BroadcasterConfig{
		Addr:          "0.0.0.0",
		IOTimeout:     2 * time.Second,
		Port:          "0",
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
	defer b.StopAndWait()

	ts := NewDummyTransactionStreamer()
	broadcastClient := newTestBroadcastClient(b.ListenerAddr(), 20*time.Second, ts)
	broadcastClient.Start(ctx)

	b.BroadcastSingle(arbstate.MessageWithMetadata{}, 0)

	// Wait for client to receive batch to ensure it is connected
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	select {
	case receivedMsg := <-ts.messageReceiver:
		t.Logf("Received Message, Sequence Message: %v\n", receivedMsg)
	case <-timer.C:
		t.Fatal("Client did not receive batch item")
	}

	broadcastClient.StopAndWait()

	disconnectTimer := time.NewTimer(5 * time.Second)
	defer disconnectTimer.Stop()
	for {
		if b.ClientCount() == 0 {
			break
		}

		select {
		case <-disconnectTimer.C:
			t.Fatal("Client was not disconnected")
		default:
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func TestBroadcastClientReconnectsOnServerDisconnect(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	settings := wsbroadcastserver.BroadcasterConfig{
		Addr:          "0.0.0.0",
		IOTimeout:     2 * time.Second,
		Port:          "0",
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
	defer b1.StopAndWait()

	broadcastClient := newTestBroadcastClient(b1.ListenerAddr(), 2*time.Second, nil)

	broadcastClient.Start(ctx)

	// Client set to timeout connection at 2 seconds, and server set to send ping every 50 seconds,
	// so at least one timeout/reconnect should happen after 4 seconds
	time.Sleep(4 * time.Second)

	if broadcastClient.GetRetryCount() <= 0 {
		t.Error("Should have had some retry counts")
	}
}

func TestBroadcasterSendsCachedMessagesOnClientConnect(t *testing.T) {
	/* Uncomment to enable logging
	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(false)))
	glogger.Verbosity(log.LvlTrace)
	log.Root().SetHandler(glogger)
	*/
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	settings := wsbroadcastserver.BroadcasterConfig{
		Addr:          "0.0.0.0",
		IOTimeout:     2 * time.Second,
		Port:          "0",
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
	defer b.StopAndWait()

	b.BroadcastSingle(arbstate.MessageWithMetadata{}, 0)
	b.BroadcastSingle(arbstate.MessageWithMetadata{}, 1)

	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		connectAndGetCachedMessages(ctx, b.ListenerAddr(), t, i, &wg)
	}

	wg.Wait()

	// give the above connections time to reconnect
	time.Sleep(4 * time.Second)

	// Confirmed Accumulator will also broadcast to the clients.
	b.Confirm(0) // remove the first message we generated

	updateTimer := time.NewTimer(2 * time.Second)
	defer updateTimer.Stop()
	for {
		if b.GetCachedMessageCount() == 1 { // should have left the second message
			break
		}

		select {
		case <-updateTimer.C:
			t.Fatal("confirmed accumulator did not get updated")
		default:
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Send second accumulator again so that the previously added accumulator is sent
	b.Confirm(1)

	updateTimer = time.NewTimer(2 * time.Second)
	defer updateTimer.Stop()
	for {
		if b.GetCachedMessageCount() == 0 { // should have left the second message
			break
		}

		select {
		case <-updateTimer.C:
			t.Fatal("cache did not get cleared")
		default:
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func connectAndGetCachedMessages(ctx context.Context, addr net.Addr, t *testing.T, clientIndex int, wg *sync.WaitGroup) {
	ts := NewDummyTransactionStreamer()
	broadcastClient := newTestBroadcastClient(addr, 60*time.Second, ts)
	broadcastClient.Start(ctx)

	go func() {
		defer wg.Done()
		defer broadcastClient.StopAndWait()

		gotMsg := false
		// Wait for client to receive first item
		timer := time.NewTimer(10 * time.Second)
		defer timer.Stop()
		select {
		case receivedMsg := <-ts.messageReceiver:
			t.Logf("client %d received first message: %v\n", clientIndex, receivedMsg)
			gotMsg = true
		case <-timer.C:
		case <-ctx.Done():
		}
		if !gotMsg {
			t.Errorf("client %d did not receive first batch item\n", clientIndex)
			return
		}

		gotMsg = false
		// Wait for client to receive second item
		timer = time.NewTimer(10 * time.Second)
		defer timer.Stop()
		select {
		case receivedMsg := <-ts.messageReceiver:
			t.Logf("client %d received second message: %v\n", clientIndex, receivedMsg)
			gotMsg = true
		case <-timer.C:
		case <-ctx.Done():
		}
		if !gotMsg {
			t.Errorf("client %d did not receive second batch item\n", clientIndex)
			return
		}

	}()
}
