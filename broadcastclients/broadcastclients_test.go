// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package broadcastclients

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/broadcastclient"
	"github.com/offchainlabs/nitro/broadcaster"
	"github.com/offchainlabs/nitro/broadcaster/message"
	"github.com/offchainlabs/nitro/util/contracts"
	"github.com/offchainlabs/nitro/util/signature"
	"github.com/offchainlabs/nitro/util/testhelpers"
	"github.com/offchainlabs/nitro/wsbroadcastserver"
)

type MockTransactionStreamer struct {
	messageReceiver chan message.BroadcastFeedMessage
	chainId         uint64
	sequencerAddr   *common.Address
}

func NewMockTransactionStreamer(chainId uint64, sequencerAddr *common.Address) *MockTransactionStreamer {
	return &MockTransactionStreamer{
		messageReceiver: make(chan message.BroadcastFeedMessage, 100),
		chainId:         chainId,
		sequencerAddr:   sequencerAddr,
	}
}

func (ts *MockTransactionStreamer) AddBroadcastMessages(feedMessages []*message.BroadcastFeedMessage) error {
	for _, feedMessage := range feedMessages {
		ts.messageReceiver <- *feedMessage
	}
	return nil
}

func feedMessage(t *testing.T, b *broadcaster.Broadcaster, seqNum arbutil.MessageIndex) []*message.BroadcastFeedMessage {
	msg := arbostypes.MessageWithMetadataAndBlockInfo{
		MessageWithMeta: arbostypes.EmptyTestMessageWithMetadata,
		BlockHash:       nil,
		BlockMetadata:   nil,
	}
	broadcastMsg, err := b.NewBroadcastFeedMessage(msg, seqNum)
	Require(t, err)
	return []*message.BroadcastFeedMessage{broadcastMsg}
}

// Test that a basic setup of broadcaster and BroadcastClients works
func TestBasicBroadcastClientSetup(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	chainId := uint64(1234)
	broadcasterConfig := wsbroadcastserver.DefaultTestBroadcasterConfig
	privateKey, err := crypto.GenerateKey()
	Require(t, err)
	sequencerAddr := crypto.PubkeyToAddress(privateKey.PublicKey)
	dataSigner := signature.DataSignerFromPrivateKey(privateKey)

	feedErrChan := make(chan error, 10)
	b := broadcaster.NewBroadcaster(
		func() *wsbroadcastserver.BroadcasterConfig { return &broadcasterConfig },
		chainId,
		feedErrChan,
		dataSigner,
	)

	Require(t, b.Initialize())
	Require(t, b.Start(ctx))
	defer b.StopAndWait()

	mockTxStreamer := NewMockTransactionStreamer(chainId, &sequencerAddr)

	getPort := func(addr net.Addr) string {
		_, portStr, err := net.SplitHostPort(addr.String())
		if err != nil {
			t.Fatalf("Failed to split host and port: %v", err)
		}
		return portStr
	}

	clientConfig := broadcastclient.DefaultTestConfig
	configFetcher := func() *broadcastclient.Config {
		config := clientConfig
		config.URL = []string{"ws://127.0.0.1:" + getPort(b.ListenerAddr())}
		config.SecondaryURL = []string{}
		config.Verify.AcceptSequencer = true
		return &config
	}

	addressVerifier := contracts.NewMockAddressVerifier(sequencerAddr)
	broadcastClients, err := NewBroadcastClients(
		configFetcher,
		chainId,
		0, // Start from sequence number 0
		mockTxStreamer,
		nil, // No confirmation listener
		feedErrChan,
		addressVerifier,
	)
	Require(t, err)
	if broadcastClients == nil {
		t.Fatal("BroadcastClients is nil")
	}

	broadcastClients.Start(ctx)
	defer broadcastClients.StopAndWait()

	const messageCount = 5
	var wg sync.WaitGroup
	wg.Add(messageCount)
	go func() {
		// Listen for messages
		receivedMessages := 0
		for i := 0; i < messageCount; i++ {
			select {
			case msg := <-mockTxStreamer.messageReceiver:
				t.Logf("Received message with sequence number: %d", msg.SequenceNumber)
				receivedMessages++
				wg.Done()
			case <-time.After(5 * time.Second):
				t.Errorf("Timed out waiting for message %d/%d", i+1, messageCount)
				for j := i; j < messageCount; j++ {
					wg.Done()
				}
				return
			}
		}
		t.Logf("Successfully received %d messages", receivedMessages)
	}()

	// Send messages with sequential sequence numbers
	for i := 0; i < messageCount; i++ {
		// #nosec G115
		err = b.BroadcastFeedMessages(feedMessage(t, b, arbutil.MessageIndex(i)))
		Require(t, err)
	}

	wg.Wait()

	if broadcastClients.connected.Load() != 1 {
		t.Errorf("Expected 1 connected feed, got %d", broadcastClients.connected.Load())
	}

	latestSeq := broadcastClients.latestSequenceNum.Load()
	if latestSeq != messageCount-1 {
		t.Errorf("Expected latest sequence number to be %d, got %d", messageCount-1, latestSeq)
	}
}

func createBroadcaster(t *testing.T, name string, chainId uint64, privateKey *ecdsa.PrivateKey) (*broadcaster.Broadcaster, chan error) {
	t.Helper()

	broadcasterConfig := wsbroadcastserver.DefaultTestBroadcasterConfig
	broadcasterConfig.Ping = 100 * time.Millisecond
	broadcasterConfig.ClientTimeout = 30 * time.Second

	dataSigner := signature.DataSignerFromPrivateKey(privateKey)

	feedErrChan := make(chan error, 10)
	b := broadcaster.NewBroadcaster(
		func() *wsbroadcastserver.BroadcasterConfig { return &broadcasterConfig },
		chainId,
		feedErrChan,
		dataSigner,
	)

	err := b.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize %s broadcaster: %v", name, err)
	}

	err = b.Start(context.Background())
	if err != nil {
		t.Fatalf("Failed to start %s broadcaster: %v", name, err)
	}

	t.Logf("%s broadcaster listening on: %s", name, b.ListenerAddr())
	return b, feedErrChan
}

// Test the failover from primary to secondary broadcaster
func TestPrimaryToSecondaryFailover(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	chainId := uint64(1234)
	primaryKey, err := crypto.GenerateKey()
	Require(t, err)

	primaryB, primaryErrChan := createBroadcaster(t, "Primary", chainId, primaryKey)
	secondaryB, secondaryErrChan := createBroadcaster(t, "Secondary", chainId, primaryKey)

	// We'll stop primary explicitly during the test, only defer the secondary
	defer secondaryB.StopAndWait()

	sequencerAddr := crypto.PubkeyToAddress(primaryKey.PublicKey)
	mockTxStreamer := NewMockTransactionStreamer(chainId, &sequencerAddr)

	getPort := func(addr net.Addr) string {
		_, portStr, err := net.SplitHostPort(addr.String())
		if err != nil {
			t.Fatalf("Failed to split host and port: %v", err)
		}
		return portStr
	}

	// Create a configuration function for the broadcast client with both primary and secondary
	clientConfig := broadcastclient.DefaultTestConfig
	clientConfig.ReconnectInitialBackoff = 200 * time.Millisecond

	primaryWsUrl := fmt.Sprintf("ws://127.0.0.1:%s", getPort(primaryB.ListenerAddr()))
	secondaryWsUrl := fmt.Sprintf("ws://127.0.0.1:%s", getPort(secondaryB.ListenerAddr()))

	t.Logf("Primary URL: %s", primaryWsUrl)
	t.Logf("Secondary URL: %s", secondaryWsUrl)

	configFetcher := func() *broadcastclient.Config {
		config := clientConfig
		config.URL = []string{primaryWsUrl}
		config.SecondaryURL = []string{secondaryWsUrl}
		config.Verify.AcceptSequencer = true
		return &config
	}

	addressVerifier := contracts.NewMockAddressVerifier(sequencerAddr)

	// Main error channel that both broadcasters will write to
	feedErrChan := make(chan error, 20)

	// Forward errors from individual broadcasters to the main error channel
	go func() {
		for {
			select {
			case err := <-primaryErrChan:
				feedErrChan <- fmt.Errorf("primary broadcaster error: %w", err)
			case err := <-secondaryErrChan:
				feedErrChan <- fmt.Errorf("secondary broadcaster error: %w", err)
			case <-ctx.Done():
				return
			}
		}
	}()

	// Create BroadcastClients
	broadcastClients, err := NewBroadcastClients(
		configFetcher,
		chainId,
		0, // Start from sequence number 0
		mockTxStreamer,
		nil, // No confirmation listener
		feedErrChan,
		addressVerifier,
	)
	Require(t, err)
	if broadcastClients == nil {
		t.Fatal("BroadcastClients is nil")
	}

	broadcastClients.Start(ctx)
	defer broadcastClients.StopAndWait()

	t.Log("Phase 1: Sending messages from primary broadcaster")
	receivedChan := make(chan struct{}, 100)
	seqNumChan := make(chan arbutil.MessageIndex, 100)

	go func() {
		for {
			select {
			case msg := <-mockTxStreamer.messageReceiver:
				receivedChan <- struct{}{}
				seqNumChan <- msg.SequenceNumber
			case <-ctx.Done():
				return
			}
		}
	}()

	// Send 5 messages from primary
	const initialMessageCount = 5
	for i := 0; i < initialMessageCount; i++ {
		err = primaryB.BroadcastFeedMessages(feedMessage(t, primaryB, arbutil.MessageIndex(i))) // #nosec G115
		Require(t, err)
		time.Sleep(50 * time.Millisecond)
	}

	// Verify we received all messages
	for i := 0; i < initialMessageCount; i++ {
		select {
		case <-receivedChan:
			// Message received
		case <-time.After(5 * time.Second):
			t.Fatalf("Timed out waiting for message %d/%d from primary", i+1, initialMessageCount)
		}
	}

	// Give the client time to process the messages
	time.Sleep(500 * time.Millisecond)

	latestSeq := broadcastClients.latestSequenceNum.Load()
	if latestSeq != initialMessageCount-1 {
		t.Errorf("Expected latest sequence number to be %d, got %d", initialMessageCount-1, latestSeq)
	}

	t.Log("Phase 2: Stopping primary broadcaster, using secondary")
	primaryB.StopAndWait()
	t.Log("Primary broadcaster stopped")

	time.Sleep(time.Second * 2)

	const secondaryMessageCount = 3
	startSeq := initialMessageCount // Continue from where primary left off
	t.Logf("Sending %d messages from secondary starting at sequence %d", secondaryMessageCount, startSeq)

	for i := 0; i < secondaryMessageCount; i++ {
		err = secondaryB.BroadcastFeedMessages(feedMessage(t, secondaryB, arbutil.MessageIndex(startSeq+i))) // #nosec G115
		Require(t, err)
		time.Sleep(50 * time.Millisecond)
	}

	// Wait for messages to be received
	receivedFromSecondary := 0
	for i := 0; i < secondaryMessageCount; i++ {
		select {
		case <-receivedChan:
			receivedFromSecondary++
		case <-time.After(5 * time.Second):
			t.Errorf("Timed out waiting for message %d/%d from secondary", i+1, secondaryMessageCount)
			break
		}
	}

	if receivedFromSecondary != secondaryMessageCount {
		t.Errorf("Only received %d/%d messages from secondary feed",
			receivedFromSecondary, secondaryMessageCount)
	}

	// Verify sequence numbers were updated correctly
	finalLatestSeq := broadcastClients.latestSequenceNum.Load()
	if finalLatestSeq != latestSeq+secondaryMessageCount {
		t.Errorf("Latest sequence number not updated after receiving from secondary. Expected %d, got %d",
			latestSeq+secondaryMessageCount, finalLatestSeq)
	}

	// Check for any errors that occurred during the test
	select {
	case err := <-feedErrChan:
		// Some errors are expected when the primary disconnects
		t.Logf("Feed error received (expected): %v", err)
	default:
	}
}

func Require(t *testing.T, err error, printables ...interface{}) {
	t.Helper()
	testhelpers.RequireImpl(t, err, printables...)
}
