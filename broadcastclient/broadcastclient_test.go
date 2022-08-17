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
	"github.com/offchainlabs/nitro/util/testhelpers"
	"github.com/offchainlabs/nitro/wsbroadcastserver"
)

func TestReceiveMessages(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	settings := wsbroadcastserver.DefaultTestBroadcasterConfig

	messageCount := 1000
	clientCount := 2
	chainId := uint64(9742)

	feedErrChan := make(chan error, 10)
	b := broadcaster.NewBroadcaster(settings, chainId, feedErrChan)

	Require(t, b.Initialize())
	Require(t, b.Start(ctx))
	defer b.StopAndWait()

	var wg sync.WaitGroup
	for i := 0; i < clientCount; i++ {
		startMakeBroadcastClient(ctx, t, b.ListenerAddr(), i, messageCount, chainId, &wg)
	}

	go func() {
		for i := 0; i < messageCount; i++ {
			b.BroadcastSingle(arbstate.EmptyTestMessageWithMetadata, arbutil.MessageIndex(i))
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

func (ts *dummyTransactionStreamer) AddBroadcastMessages(feedMessages []*broadcaster.BroadcastFeedMessage) error {
	for _, feedMessage := range feedMessages {
		ts.messageReceiver <- *feedMessage
	}
	return nil
}

func newTestBroadcastClient(listenerAddress net.Addr, chainId uint64, currentMessageCount arbutil.MessageIndex, idleTimeout time.Duration, txStreamer TransactionStreamerInterface, feedErrChan chan error) *BroadcastClient {
	port := listenerAddress.(*net.TCPAddr).Port
	return NewBroadcastClient(fmt.Sprintf("ws://127.0.0.1:%d/", port), chainId, currentMessageCount, idleTimeout, txStreamer, feedErrChan)
}

func startMakeBroadcastClient(ctx context.Context, t *testing.T, addr net.Addr, index int, expectedCount int, chainId uint64, wg *sync.WaitGroup) {
	ts := NewDummyTransactionStreamer()
	feedErrChan := make(chan error, 10)
	broadcastClient := newTestBroadcastClient(addr, chainId, 0, 200*time.Millisecond, ts, feedErrChan)
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
			case err := <-feedErrChan:
				t.Error(err)
				return
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
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	settings := wsbroadcastserver.DefaultTestBroadcasterConfig
	settings.Ping = 1 * time.Second

	chainId := uint64(8742)
	feedErrChan := make(chan error, 10)
	b := broadcaster.NewBroadcaster(settings, chainId, feedErrChan)

	Require(t, b.Initialize())
	Require(t, b.Start(ctx))
	defer b.StopAndWait()

	ts := NewDummyTransactionStreamer()
	broadcastClient := newTestBroadcastClient(b.ListenerAddr(), chainId, 0, 200*time.Millisecond, ts, feedErrChan)
	broadcastClient.Start(ctx)

	t.Log("broadcasting seq 0 message")
	b.BroadcastSingle(arbstate.EmptyTestMessageWithMetadata, 0)

	// Wait for client to receive batch to ensure it is connected
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	select {
	case err := <-feedErrChan:
		t.Errorf("Broadcaster error: %s\n", err.Error())
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

func TestServerClientIncorrectChainId(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	settings := wsbroadcastserver.DefaultTestBroadcasterConfig
	settings.Ping = 1 * time.Second

	chainId := uint64(8742)
	feedErrChan := make(chan error, 10)
	b := broadcaster.NewBroadcaster(settings, chainId, feedErrChan)

	Require(t, b.Initialize())
	Require(t, b.Start(ctx))
	defer b.StopAndWait()

	ts := NewDummyTransactionStreamer()
	badFeedErrChan := make(chan error, 10)
	badBroadcastClient := newTestBroadcastClient(b.ListenerAddr(), chainId+1, 0, 200*time.Millisecond, ts, badFeedErrChan)
	badBroadcastClient.Start(ctx)
	badTimer := time.NewTimer(5 * time.Second)
	select {
	case <-badFeedErrChan:
		// Got expected error
		badTimer.Stop()
	case <-badTimer.C:
		t.Fatal("Client channel did not send error as expected")
	}
}

func TestBroadcastClientReconnectsOnServerDisconnect(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	settings := wsbroadcastserver.DefaultTestBroadcasterConfig
	settings.Ping = 50 * time.Second
	settings.ClientTimeout = 150 * time.Second

	feedErrChan := make(chan error, 10)
	chainId := uint64(8742)
	b1 := broadcaster.NewBroadcaster(settings, chainId, feedErrChan)

	Require(t, b1.Initialize())
	Require(t, b1.Start(ctx))
	defer b1.StopAndWait()

	broadcastClient := newTestBroadcastClient(b1.ListenerAddr(), chainId, 0, 200*time.Millisecond, nil, feedErrChan)

	broadcastClient.Start(ctx)
	defer broadcastClient.StopAndWait()

	// Client set to timeout connection at 200 milliseconds, and server set to send ping every 50 seconds,
	// so at least one timeout/reconnect should happen after 1 seconds
	time.Sleep(1 * time.Second)

	if broadcastClient.GetRetryCount() <= 0 {
		t.Error("Should have had some retry counts")
	}
}

func TestBroadcasterSendsCachedMessagesOnClientConnect(t *testing.T) {
	t.Parallel()
	/* Uncomment to enable logging
	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(false)))
	glogger.Verbosity(log.LvlTrace)
	log.Root().SetHandler(glogger)
	*/
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	settings := wsbroadcastserver.DefaultTestBroadcasterConfig

	feedErrChan := make(chan error, 10)
	chainId := uint64(8744)
	b := broadcaster.NewBroadcaster(settings, chainId, feedErrChan)

	Require(t, b.Initialize())
	Require(t, b.Start(ctx))
	defer b.StopAndWait()

	b.BroadcastSingle(arbstate.EmptyTestMessageWithMetadata, 0)
	b.BroadcastSingle(arbstate.EmptyTestMessageWithMetadata, 1)

	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		connectAndGetCachedMessages(ctx, b.ListenerAddr(), chainId, t, i, &wg)
	}

	wg.Wait()

	// give the above connections time to reconnect
	time.Sleep(1 * time.Second)

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

func connectAndGetCachedMessages(ctx context.Context, addr net.Addr, chainId uint64, t *testing.T, clientIndex int, wg *sync.WaitGroup) {
	ts := NewDummyTransactionStreamer()
	feedErrChan := make(chan error, 10)
	broadcastClient := newTestBroadcastClient(addr, chainId, 0, 200*time.Millisecond, ts, feedErrChan)
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

func Require(t *testing.T, err error, printables ...interface{}) {
	t.Helper()
	testhelpers.RequireImpl(t, err, printables...)
}
