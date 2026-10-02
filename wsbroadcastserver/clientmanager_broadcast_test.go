// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package wsbroadcastserver

import (
	"context"
	"encoding/json"
	"net"
	"testing"
	"time"

	"github.com/gobwas/ws/wsutil"
	"github.com/mailru/easygo/netpoll"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/broadcaster/backlog"
	m "github.com/offchainlabs/nitro/broadcaster/message"
)

func waitForClientCount(t *testing.T, cm *ClientManager, expected int32) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for cm.ClientCount() != expected {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %d registered clients, have %d", expected, cm.ClientCount())
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// TestBroadcastRemovesClientFlaggedByEarlierMessage checks that a client that
// has to be disconnected because of one message of a multi-message broadcast
// is disconnected even if the later messages of the same broadcast could be
// delivered to it. Otherwise the client stays connected and silently misses
// the earlier message.
func TestBroadcastRemovesClientFlaggedByEarlierMessage(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config := DefaultTestBroadcasterConfig
	bklg := backlog.NewBacklog(func() *backlog.Config { return &config.Backlog })
	// doBroadcast appends each message to the backlog before fetching the
	// config, so requiring compression only while the backlog holds exactly
	// two messages makes message 2 flag the (uncompressed) client for
	// disconnection while messages 3 and 4 can still be delivered to it.
	configFetcher := func() *BroadcasterConfig {
		cfg := config
		cfg.RequireCompression = bklg.Count() == 2
		return &cfg
	}
	poller, err := netpoll.New(nil)
	Require(t, err)
	cm := NewClientManager(poller, configFetcher, bklg)
	cm.Start(ctx)
	defer cm.StopAndWait()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	Require(t, err)
	defer ln.Close()
	peerConn, err := net.Dial("tcp", ln.Addr().String())
	Require(t, err)
	defer peerConn.Close()
	serverConn, err := ln.Accept()
	Require(t, err)
	defer serverConn.Close()
	desc, err := netpoll.HandleRead(serverConn)
	Require(t, err)
	Require(t, poller.Start(desc, func(netpoll.Event) {}))

	cc := NewClientConnection(serverConn, desc, cm.clientAction, 0, net.IPv4(127, 0, 0, 1), false, config.MaxSendQueue, 0, bklg)
	cc.Start(ctx)
	waitForClientCount(t, cm, 1)

	// Message 1 is delivered normally, so the client is past its initial
	// backlog catch-up.
	cm.Broadcast(&m.BroadcastMessage{Version: 1, Messages: []*m.BroadcastFeedMessage{testFeedMessage(1)}})
	expectFeedMessage(t, peerConn, []arbutil.MessageIndex{1}, false)

	cm.Broadcast(&m.BroadcastMessage{
		Version:  1,
		Messages: []*m.BroadcastFeedMessage{testFeedMessage(2), testFeedMessage(3), testFeedMessage(4)},
	})

	// The client must be removed, and must never have received message 2.
	waitForClientCount(t, cm, 0)
	Require(t, peerConn.SetReadDeadline(time.Now().Add(500*time.Millisecond)))
	for {
		data, err := wsutil.ReadServerText(peerConn)
		if err != nil {
			break
		}
		var bm m.BroadcastMessage
		Require(t, json.Unmarshal(data, &bm))
		for _, msg := range bm.Messages {
			if msg.SequenceNumber == 2 {
				t.Fatal("client received the message it was supposed to be disconnected for")
			}
		}
	}
}
