// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package wsbroadcastserver

import (
	"context"
	"errors"
	"net"
	"os"
	"testing"
	"time"

	"github.com/mailru/easygo/netpoll"

	"github.com/offchainlabs/nitro/broadcaster/backlog"
)

// TestRemoveUnregisteredClientClosesConnection reproduces a client hanging up
// before the ClientConnection has registered with the ClientManager (for
// example while the backlog is still being written, or during the configured
// client delay). The netpoll Hup handler only calls Remove(), so the manager
// must close the connection itself even though it was never registered,
// otherwise the socket, its poller descriptor and the ClientConnection leak.
func TestRemoveUnregisteredClientClosesConnection(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config := DefaultTestBroadcasterConfig
	configFetcher := func() *BroadcasterConfig { return &config }
	bklg := backlog.NewBacklog(func() *backlog.Config { return &config.Backlog })
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

	// A long client delay keeps the connection unregistered for the duration of the test.
	cc := NewClientConnection(serverConn, desc, cm.clientAction, 0, net.IPv4(127, 0, 0, 1), false, config.MaxSendQueue, time.Hour, bklg)
	cc.Start(ctx)

	// The peer hangs up: this is what the poller's Hup handler does.
	cc.Remove()

	Require(t, peerConn.SetReadDeadline(time.Now().Add(5*time.Second)))
	_, err = peerConn.Read(make([]byte, 1))
	if err == nil || errors.Is(err, os.ErrDeadlineExceeded) {
		t.Fatal("server side of the connection was not closed after removing an unregistered client")
	}
	if !cc.Stopped() {
		t.Fatal("ClientConnection was not stopped after being removed")
	}
	if count := cm.ClientCount(); count != 0 {
		t.Fatalf("expected 0 registered clients, got %d", count)
	}
}

// TestConnectionLimitedClientDoesNotAffectClientCount checks that a client
// rejected by the connection limiter is closed without decrementing the
// registered client accounting, which was never incremented for it.
func TestConnectionLimitedClientDoesNotAffectClientCount(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config := DefaultTestBroadcasterConfig
	config.ConnectionLimits.Enable = true
	config.ConnectionLimits.PerIpLimit = 1
	configFetcher := func() *BroadcasterConfig { return &config }
	bklg := backlog.NewBacklog(func() *backlog.Config { return &config.Backlog })
	poller, err := netpoll.New(nil)
	Require(t, err)
	cm := NewClientManager(poller, configFetcher, bklg)
	cm.Start(ctx)
	defer cm.StopAndWait()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	Require(t, err)
	defer ln.Close()

	// The connection limiter ignores loopback addresses, so pretend the
	// clients connect from a public IP (as reported by the proxy header).
	ip := net.ParseIP("203.0.113.7")
	newClient := func() (net.Conn, *ClientConnection) {
		peerConn, err := net.Dial("tcp", ln.Addr().String())
		Require(t, err)
		serverConn, err := ln.Accept()
		Require(t, err)
		desc, err := netpoll.HandleRead(serverConn)
		Require(t, err)
		Require(t, poller.Start(desc, func(netpoll.Event) {}))
		cc := NewClientConnection(serverConn, desc, cm.clientAction, 0, ip, false, config.MaxSendQueue, 0, bklg)
		cc.Start(ctx)
		return peerConn, cc
	}
	expectClosed := func(peerConn net.Conn, what string) {
		t.Helper()
		Require(t, peerConn.SetReadDeadline(time.Now().Add(5*time.Second)))
		_, err := peerConn.Read(make([]byte, 1))
		if err == nil || errors.Is(err, os.ErrDeadlineExceeded) {
			t.Fatalf("%s was not closed", what)
		}
	}

	firstPeer, first := newClient()
	defer firstPeer.Close()
	waitFor(t, func() bool { return cm.ClientCount() == 1 })

	// Same IP, over the limit: must be closed without being counted as a disconnect.
	secondPeer, _ := newClient()
	defer secondPeer.Close()
	expectClosed(secondPeer, "connection limited client")
	if count := cm.ClientCount(); count != 1 {
		t.Fatalf("expected 1 registered client after limiting a second one, got %d", count)
	}

	// Once the registered client leaves, its slot is released for the IP again.
	first.Remove()
	expectClosed(firstPeer, "removed client")
	waitFor(t, func() bool { return cm.ClientCount() == 0 })

	thirdPeer, _ := newClient()
	defer thirdPeer.Close()
	waitFor(t, func() bool { return cm.ClientCount() == 1 })
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for condition")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// TestRemoveClientBeforeStart covers a client hanging up after it was
// registered with the poller but before its thread was started: the thread
// must then exit immediately instead of waiting for messages that will never
// be delivered to an unregistered client.
func TestRemoveClientBeforeStart(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config := DefaultTestBroadcasterConfig
	configFetcher := func() *BroadcasterConfig { return &config }
	bklg := backlog.NewBacklog(func() *backlog.Config { return &config.Backlog })
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
	cc.Remove()
	waitFor(t, cc.closed.Load)

	// Starting a client that has already been closed must leave it stopped, so
	// that its thread exits right away instead of blocking on the out channel.
	cc.Start(ctx)
	if !cc.Stopped() {
		t.Fatal("client started after being removed is not stopped")
	}
	if count := cm.ClientCount(); count != 0 {
		t.Fatalf("expected 0 registered clients, got %d", count)
	}
}
