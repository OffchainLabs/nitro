// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package broadcastclient

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/gobwas/ws"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/broadcaster"
	"github.com/offchainlabs/nitro/util/contracts"
	"github.com/offchainlabs/nitro/util/signature"
	"github.com/offchainlabs/nitro/util/testhelpers"
	"github.com/offchainlabs/nitro/wsbroadcastserver"
)

func TestReceiveMessagesWithoutCompression(t *testing.T) {
	t.Parallel()
	testReceiveMessages(t, false, false, false, false)
}

func TestReceiveMessagesWithCompression(t *testing.T) {
	t.Parallel()
	testReceiveMessages(t, true, true, false, false)
}

func TestReceiveMessagesWithServerOptionalCompression(t *testing.T) {
	t.Parallel()
	testReceiveMessages(t, true, true, false, false)
}

func TestReceiveMessagesWithServerOnlyCompression(t *testing.T) {
	t.Parallel()
	testReceiveMessages(t, false, true, false, false)
}

func TestReceiveMessagesWithClientOnlyCompression(t *testing.T) {
	t.Parallel()
	testReceiveMessages(t, true, false, false, false)
}

func TestReceiveMessagesWithRequiredCompression(t *testing.T) {
	t.Parallel()
	testReceiveMessages(t, true, true, true, false)
}

func TestReceiveMessagesWithRequiredCompressionButClientDisabled(t *testing.T) {
	t.Parallel()
	testReceiveMessages(t, false, true, true, true)
}

func testReceiveMessages(t *testing.T, clientCompression bool, serverCompression bool, serverRequire bool, expectNoMessagesReceived bool) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	broadcasterConfig := wsbroadcastserver.DefaultTestBroadcasterConfig
	broadcasterConfig.EnableCompression = serverCompression
	broadcasterConfig.RequireCompression = serverRequire

	messageCount := 1000
	clientCount := 2
	chainId := uint64(9742)

	privateKey, err := crypto.GenerateKey()
	Require(t, err)
	sequencerAddr := crypto.PubkeyToAddress(privateKey.PublicKey)
	dataSigner := signature.DataSignerFromPrivateKey(privateKey)

	feedErrChan := make(chan error, 10)
	b := broadcaster.NewBroadcaster(func() *wsbroadcastserver.BroadcasterConfig { return &broadcasterConfig }, chainId, feedErrChan, dataSigner)

	Require(t, b.Initialize())
	Require(t, b.Start(ctx))
	defer b.StopAndWait()

	config := DefaultTestConfig
	config.EnableCompression = clientCompression
	var wg sync.WaitGroup
	var expectedCount int
	if expectNoMessagesReceived {
		expectedCount = 0
	} else {
		expectedCount = messageCount
	}
	for i := 0; i < clientCount; i++ {
		startMakeBroadcastClient(ctx, t, config, b.ListenerAddr(), i, expectedCount, chainId, &wg, &sequencerAddr)
	}

	go func() {
		for i := 0; i < messageCount; i++ {
			Require(t, b.BroadcastSingle(arbostypes.TestMessageWithMetadataAndRequestId, arbutil.MessageIndex(i)))
		}
	}()

	wg.Wait()

}

func TestInvalidSignature(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	settings := wsbroadcastserver.DefaultTestBroadcasterConfig

	messageCount := 1
	chainId := uint64(9742)

	privateKey, err := crypto.GenerateKey()
	Require(t, err)
	dataSigner := signature.DataSignerFromPrivateKey(privateKey)

	fatalErrChan := make(chan error, 10)
	b := broadcaster.NewBroadcaster(func() *wsbroadcastserver.BroadcasterConfig { return &settings }, chainId, fatalErrChan, dataSigner)

	Require(t, b.Initialize())
	Require(t, b.Start(ctx))
	defer b.StopAndWait()

	badPrivateKey, err := crypto.GenerateKey()
	Require(t, err)
	badPublicKey := badPrivateKey.Public()
	badSequencerAddr := crypto.PubkeyToAddress(*badPublicKey.(*ecdsa.PublicKey))
	config := DefaultTestConfig

	ts := NewDummyTransactionStreamer(chainId, &badSequencerAddr)
	broadcastClient, err := newTestBroadcastClient(
		config,
		b.ListenerAddr(),
		chainId,
		0,
		ts,
		nil,
		fatalErrChan,
		&badSequencerAddr,
	)
	Require(t, err)
	broadcastClient.Start(ctx)

	go func() {
		for i := 0; i < messageCount; i++ {
			Require(t, b.BroadcastSingle(arbostypes.TestMessageWithMetadataAndRequestId, arbutil.MessageIndex(i)))
		}
	}()

	timer := time.NewTimer(2 * time.Second)
	select {
	case err := <-fatalErrChan:
		if errors.Is(err, signature.ErrSignatureNotVerified) {
			t.Log("feed error found as expected")
			return
		}
		t.Errorf("unexpected error occurred: %v", err)
		return
	case <-timer.C:
		t.Error("no feed errors detected")
		return
	case <-ctx.Done():
		timer.Stop()
		return
	}
}

type dummyTransactionStreamer struct {
	messageReceiver chan broadcaster.BroadcastFeedMessage
	chainId         uint64
	sequencerAddr   *common.Address
}

func NewDummyTransactionStreamer(chainId uint64, sequencerAddr *common.Address) *dummyTransactionStreamer {
	return &dummyTransactionStreamer{
		messageReceiver: make(chan broadcaster.BroadcastFeedMessage),
		chainId:         chainId,
		sequencerAddr:   sequencerAddr,
	}
}

func (ts *dummyTransactionStreamer) AddBroadcastMessages(feedMessages []*broadcaster.BroadcastFeedMessage) error {
	for _, feedMessage := range feedMessages {
		ts.messageReceiver <- *feedMessage
	}
	return nil
}

func newTestBroadcastClient(config Config, listenerAddress net.Addr, chainId uint64, currentMessageCount arbutil.MessageIndex, txStreamer TransactionStreamerInterface, confirmedSequenceNumberListener chan arbutil.MessageIndex, feedErrChan chan error, validAddr *common.Address) (*BroadcastClient, error) {
	port := listenerAddress.(*net.TCPAddr).Port
	var av contracts.AddressVerifierInterface
	if validAddr != nil {
		config.Verify.AcceptSequencer = true
		av = contracts.NewMockAddressVerifier(*validAddr)
	} else {
		config.Verify.AcceptSequencer = false
	}
	return NewBroadcastClient(func() *Config { return &config }, fmt.Sprintf("ws://127.0.0.1:%d/", port), chainId, currentMessageCount, txStreamer, confirmedSequenceNumberListener, feedErrChan, av, func(_ int32) {})
}

func startMakeBroadcastClient(ctx context.Context, t *testing.T, clientConfig Config, addr net.Addr, index int, expectedCount int, chainId uint64, wg *sync.WaitGroup, sequencerAddr *common.Address) {
	ts := NewDummyTransactionStreamer(chainId, sequencerAddr)
	feedErrChan := make(chan error, 10)
	broadcastClient, err := newTestBroadcastClient(
		clientConfig,
		addr,
		chainId,
		0,
		ts,
		nil,
		feedErrChan,
		sequencerAddr,
	)
	Require(t, err)
	broadcastClient.Start(ctx)
	messageCount := 0

	wg.Add(1)

	go func() {
		defer wg.Done()
		defer broadcastClient.StopAndWait()
		var timeout time.Duration
		if expectedCount == 0 {
			timeout = 1 * time.Second
		} else {
			timeout = 60 * time.Second
		}
		for {
			gotMsg := false
			timer := time.NewTimer(timeout)
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
			if (!gotMsg && expectedCount > 0) || (gotMsg && expectedCount == 0) {
				t.Errorf("Client %d expected %d meesages, got %d messages\n", index, expectedCount, messageCount)
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

	config := wsbroadcastserver.DefaultTestBroadcasterConfig
	config.Ping = 1 * time.Second

	privateKey, err := crypto.GenerateKey()
	Require(t, err)
	sequencerAddr := crypto.PubkeyToAddress(privateKey.PublicKey)
	dataSigner := signature.DataSignerFromPrivateKey(privateKey)

	chainId := uint64(8742)
	feedErrChan := make(chan error, 10)
	b := broadcaster.NewBroadcaster(func() *wsbroadcastserver.BroadcasterConfig { return &config }, chainId, feedErrChan, dataSigner)

	Require(t, b.Initialize())
	Require(t, b.Start(ctx))
	defer b.StopAndWait()

	ts := NewDummyTransactionStreamer(chainId, nil)
	broadcastClient, err := newTestBroadcastClient(
		DefaultTestConfig,
		b.ListenerAddr(),
		chainId,
		0,
		ts,
		nil,
		feedErrChan,
		&sequencerAddr,
	)
	Require(t, err)
	broadcastClient.Start(ctx)

	t.Log("broadcasting seq 0 message")
	Require(t, b.BroadcastSingle(arbostypes.EmptyTestMessageWithMetadata, 0))

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
		case err := <-feedErrChan:
			t.Errorf("Broadcaster error: %s\n", err.Error())
		case <-disconnectTimer.C:
			t.Fatal("Client was not disconnected")
		default:
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func TestBroadcastClientConfirmedMessage(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config := wsbroadcastserver.DefaultTestBroadcasterConfig
	config.Ping = 1 * time.Second

	privateKey, err := crypto.GenerateKey()
	Require(t, err)
	sequencerAddr := crypto.PubkeyToAddress(privateKey.PublicKey)
	dataSigner := signature.DataSignerFromPrivateKey(privateKey)

	chainId := uint64(8742)
	feedErrChan := make(chan error, 10)
	b := broadcaster.NewBroadcaster(func() *wsbroadcastserver.BroadcasterConfig { return &config }, chainId, feedErrChan, dataSigner)

	Require(t, b.Initialize())
	Require(t, b.Start(ctx))
	defer b.StopAndWait()

	confirmedSequenceNumberListener := make(chan arbutil.MessageIndex, 10)
	ts := NewDummyTransactionStreamer(chainId, nil)
	broadcastClient, err := newTestBroadcastClient(
		DefaultTestConfig,
		b.ListenerAddr(),
		chainId,
		0,
		ts,
		confirmedSequenceNumberListener,
		feedErrChan,
		&sequencerAddr,
	)
	Require(t, err)
	broadcastClient.Start(ctx)

	t.Log("broadcasting seq 0 message")
	Require(t, b.BroadcastSingle(arbostypes.EmptyTestMessageWithMetadata, 0))

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

	confirmNumber := arbutil.MessageIndex(42)
	b.Confirm(42)

	// Wait for client to receive confirm message
	timer2 := time.NewTimer(5 * time.Second)
	defer timer2.Stop()
	select {
	case err := <-feedErrChan:
		t.Errorf("Broadcaster error: %s", err.Error())
	case confirmed := <-confirmedSequenceNumberListener:
		if confirmed == confirmNumber {
			t.Logf("got confirmed: %v", confirmed)
		} else {
			t.Errorf("Incorrect number confirmed: %v, expected: %v", confirmed, confirmNumber)
		}
	case <-timer2.C:
		t.Fatal("Client did not receive confirm message")
	}

	broadcastClient.StopAndWait()
}
func TestServerIncorrectChainId(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config := wsbroadcastserver.DefaultTestBroadcasterConfig
	config.Ping = 1 * time.Second

	privateKey, err := crypto.GenerateKey()
	Require(t, err)
	sequencerAddr := crypto.PubkeyToAddress(privateKey.PublicKey)
	dataSigner := signature.DataSignerFromPrivateKey(privateKey)

	chainId := uint64(8742)
	feedErrChan := make(chan error, 10)
	b := broadcaster.NewBroadcaster(func() *wsbroadcastserver.BroadcasterConfig { return &config }, chainId, feedErrChan, dataSigner)

	Require(t, b.Initialize())
	Require(t, b.Start(ctx))
	defer b.StopAndWait()

	ts := NewDummyTransactionStreamer(chainId, nil)
	badFeedErrChan := make(chan error, 10)
	badBroadcastClient, err := newTestBroadcastClient(
		DefaultTestConfig,
		b.ListenerAddr(),
		chainId+1,
		0,
		ts,
		nil,
		badFeedErrChan,
		&sequencerAddr,
	)
	Require(t, err)
	badBroadcastClient.Start(ctx)
	badTimer := time.NewTimer(5 * time.Second)
	select {
	case err := <-feedErrChan:
		// Got unexpected error
		t.Errorf("Unexpected error %v", err)
		badTimer.Stop()
	case err := <-badFeedErrChan:
		if !errors.Is(err, ErrIncorrectChainId) {
			// Got unexpected error
			t.Errorf("Unexpected error %v", err)
		}
		badTimer.Stop()
	case <-badTimer.C:
		t.Fatal("Client channel did not send error as expected")
	}
}

func TestServerMissingChainId(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	settings := wsbroadcastserver.DefaultTestBroadcasterConfig
	settings.Ping = 1 * time.Second

	privateKey, err := crypto.GenerateKey()
	Require(t, err)
	sequencerAddr := crypto.PubkeyToAddress(privateKey.PublicKey)
	dataSigner := signature.DataSignerFromPrivateKey(privateKey)

	chainId := uint64(8742)
	feedErrChan := make(chan error, 10)
	b := broadcaster.NewBroadcaster(func() *wsbroadcastserver.BroadcasterConfig { return &settings }, chainId, feedErrChan, dataSigner)

	header := ws.HandshakeHeaderHTTP(http.Header{
		wsbroadcastserver.HTTPHeaderFeedServerVersion: []string{strconv.Itoa(wsbroadcastserver.FeedServerVersion)},
	})

	Require(t, b.Initialize())
	Require(t, b.StartWithHeader(ctx, header))
	defer b.StopAndWait()

	clientConfig := DefaultTestConfig
	clientConfig.RequireChainId = true

	ts := NewDummyTransactionStreamer(chainId, nil)
	badFeedErrChan := make(chan error, 10)
	badBroadcastClient, err := newTestBroadcastClient(
		clientConfig,
		b.ListenerAddr(),
		chainId,
		0,
		ts,
		nil,
		badFeedErrChan,
		&sequencerAddr,
	)
	Require(t, err)
	badBroadcastClient.Start(ctx)
	badTimer := time.NewTimer(5 * time.Second)
	select {
	case err := <-feedErrChan:
		// Got unexpected error
		t.Errorf("Unexpected error %v", err)
		badTimer.Stop()
	case err := <-badFeedErrChan:
		if !errors.Is(err, ErrMissingChainId) {
			// Got unexpected error
			t.Errorf("Unexpected error %v", err)
		}
		badTimer.Stop()
	case <-badTimer.C:
		t.Fatal("Client channel did not send error as expected")
	}
}

func TestServerIncorrectFeedServerVersion(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	settings := wsbroadcastserver.DefaultTestBroadcasterConfig
	settings.Ping = 1 * time.Second

	privateKey, err := crypto.GenerateKey()
	Require(t, err)
	sequencerAddr := crypto.PubkeyToAddress(privateKey.PublicKey)
	dataSigner := signature.DataSignerFromPrivateKey(privateKey)

	chainId := uint64(8742)
	feedErrChan := make(chan error, 10)
	b := broadcaster.NewBroadcaster(func() *wsbroadcastserver.BroadcasterConfig { return &settings }, chainId, feedErrChan, dataSigner)

	header := ws.HandshakeHeaderHTTP(http.Header{
		wsbroadcastserver.HTTPHeaderChainId:           []string{strconv.FormatUint(chainId, 10)},
		wsbroadcastserver.HTTPHeaderFeedServerVersion: []string{strconv.Itoa(wsbroadcastserver.FeedServerVersion + 1)},
	})

	Require(t, b.Initialize())
	Require(t, b.StartWithHeader(ctx, header))
	defer b.StopAndWait()

	ts := NewDummyTransactionStreamer(chainId, nil)
	badFeedErrChan := make(chan error, 10)
	badBroadcastClient, err := newTestBroadcastClient(
		DefaultTestConfig,
		b.ListenerAddr(),
		chainId,
		0,
		ts,
		nil,
		badFeedErrChan,
		&sequencerAddr,
	)
	Require(t, err)
	badBroadcastClient.Start(ctx)
	badTimer := time.NewTimer(5 * time.Second)
	select {
	case err := <-feedErrChan:
		// Got unexpected error
		t.Errorf("Unexpected error %v", err)
		badTimer.Stop()
	case err := <-badFeedErrChan:
		if !errors.Is(err, ErrIncorrectFeedServerVersion) {
			// Got unexpected error
			t.Errorf("Unexpected error %v", err)
		}
		badTimer.Stop()
	case <-badTimer.C:
		t.Fatal("Client channel did not send error as expected")
	}
}

func TestServerMissingFeedServerVersion(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	settings := wsbroadcastserver.DefaultTestBroadcasterConfig
	settings.Ping = 1 * time.Second

	privateKey, err := crypto.GenerateKey()
	Require(t, err)
	sequencerAddr := crypto.PubkeyToAddress(privateKey.PublicKey)
	dataSigner := signature.DataSignerFromPrivateKey(privateKey)

	chainId := uint64(8742)
	feedErrChan := make(chan error, 10)
	b := broadcaster.NewBroadcaster(func() *wsbroadcastserver.BroadcasterConfig { return &settings }, chainId, feedErrChan, dataSigner)

	header := ws.HandshakeHeaderHTTP(http.Header{
		wsbroadcastserver.HTTPHeaderChainId: []string{strconv.FormatUint(chainId, 10)},
	})

	Require(t, b.Initialize())
	Require(t, b.StartWithHeader(ctx, header))
	defer b.StopAndWait()

	clientConfig := DefaultTestConfig
	clientConfig.RequireFeedVersion = true

	ts := NewDummyTransactionStreamer(chainId, nil)
	badFeedErrChan := make(chan error, 10)
	badBroadcastClient, err := newTestBroadcastClient(
		clientConfig,
		b.ListenerAddr(),
		chainId,
		0,
		ts,
		nil,
		badFeedErrChan,
		&sequencerAddr,
	)
	Require(t, err)
	badBroadcastClient.Start(ctx)
	badTimer := time.NewTimer(5 * time.Second)
	select {
	case err := <-feedErrChan:
		// Got unexpected error
		t.Errorf("Unexpected error %v", err)
		badTimer.Stop()
	case err := <-badFeedErrChan:
		if !errors.Is(err, ErrMissingFeedServerVersion) {
			// Got unexpected error
			t.Errorf("Unexpected error %v", err)
		}
		badTimer.Stop()
	case <-badTimer.C:
		t.Fatal("Client channel did not send error as expected")
	}
}

func TestBroadcastClientReconnectsOnServerDisconnect(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config := wsbroadcastserver.DefaultTestBroadcasterConfig
	config.Ping = 50 * time.Second
	config.ClientTimeout = 150 * time.Second

	privateKey, err := crypto.GenerateKey()
	Require(t, err)
	sequencerAddr := crypto.PubkeyToAddress(privateKey.PublicKey)
	dataSigner := signature.DataSignerFromPrivateKey(privateKey)

	feedErrChan := make(chan error, 10)
	chainId := uint64(8742)
	b1 := broadcaster.NewBroadcaster(func() *wsbroadcastserver.BroadcasterConfig { return &config }, chainId, feedErrChan, dataSigner)

	Require(t, b1.Initialize())
	Require(t, b1.Start(ctx))
	defer b1.StopAndWait()

	broadcastClient, err := newTestBroadcastClient(
		DefaultTestConfig,
		b1.ListenerAddr(),
		chainId,
		0,
		nil,
		nil,
		feedErrChan,
		&sequencerAddr,
	)
	Require(t, err)
	broadcastClient.Start(ctx)
	defer broadcastClient.StopAndWait()

	// Client set to timeout connection at 200 milliseconds, and server set to send ping every 50 seconds,
	// so at least one timeout/reconnect should happen after 1 seconds
	time.Sleep(1 * time.Second)

	select {
	case err := <-feedErrChan:
		t.Errorf("Broadcaster error: %s\n", err.Error())
	default:
	}

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

	privateKey, err := crypto.GenerateKey()
	Require(t, err)
	sequencerAddr := crypto.PubkeyToAddress(privateKey.PublicKey)
	dataSigner := signature.DataSignerFromPrivateKey(privateKey)

	feedErrChan := make(chan error, 10)
	chainId := uint64(8744)
	b := broadcaster.NewBroadcaster(func() *wsbroadcastserver.BroadcasterConfig { return &settings }, chainId, feedErrChan, dataSigner)

	Require(t, b.Initialize())
	Require(t, b.Start(ctx))
	defer b.StopAndWait()

	Require(t, b.BroadcastSingle(arbostypes.EmptyTestMessageWithMetadata, 0))
	Require(t, b.BroadcastSingle(arbostypes.EmptyTestMessageWithMetadata, 1))

	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		connectAndGetCachedMessages(ctx, b.ListenerAddr(), chainId, t, i, feedErrChan, &sequencerAddr, &wg)
	}

	select {
	case err := <-feedErrChan:
		t.Fatalf("Broadcaster error: %s\n", err.Error())
	case <-time.After(10 * time.Millisecond):
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
		case err := <-feedErrChan:
			t.Errorf("Broadcaster error: %s\n", err.Error())
		default:
		}
		time.Sleep(10 * time.Millisecond)
	}

	b.Confirm(1)

	updateTimer2 := time.NewTimer(2 * time.Second)
	defer updateTimer2.Stop()
	for {
		if b.GetCachedMessageCount() == 0 { // should have left the second message
			break
		}

		select {
		case <-updateTimer2.C:
			t.Fatal("cache did not get cleared")
		default:
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func connectAndGetCachedMessages(ctx context.Context, addr net.Addr, chainId uint64, t *testing.T, clientIndex int, feedErrChan chan error, sequencerAddr *common.Address, wg *sync.WaitGroup) {
	ts := NewDummyTransactionStreamer(chainId, nil)
	broadcastClient, err := newTestBroadcastClient(
		DefaultTestConfig,
		addr,
		chainId,
		0,
		ts,
		nil,
		feedErrChan,
		sequencerAddr,
	)
	Require(t, err)
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
		case err := <-feedErrChan:
			t.Errorf("client %d feed error: %v\n", clientIndex, err)
		case <-timer.C:
		case <-ctx.Done():
		}
		if !gotMsg {
			t.Errorf("client %d did not receive first batch item\n", clientIndex)
			return
		}

		gotMsg = false
		// Wait for client to receive second item
		timer2 := time.NewTimer(10 * time.Second)
		defer timer2.Stop()
		select {
		case receivedMsg := <-ts.messageReceiver:
			t.Logf("client %d received second message: %v\n", clientIndex, receivedMsg)
			gotMsg = true
		case err := <-feedErrChan:
			t.Errorf("client %d feed error: %v\n", clientIndex, err)
		case <-timer2.C:
		case <-ctx.Done():
		}
		if !gotMsg {
			t.Errorf("client %d did not receive second batch item\n", clientIndex)
		}
	}()
}

func Require(t *testing.T, err error, printables ...interface{}) {
	t.Helper()
	testhelpers.RequireImpl(t, err, printables...)
}
