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

	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/broadcaster/backlog"
	m "github.com/offchainlabs/nitro/broadcaster/message"
)

func testFeedMessage(seqNum arbutil.MessageIndex) *m.BroadcastFeedMessage {
	return &m.BroadcastFeedMessage{
		SequenceNumber: seqNum,
		Message:        arbostypes.EmptyTestMessageWithMetadata,
		Signature:      []byte{},
	}
}

func appendToBacklog(t *testing.T, bklg backlog.Backlog, seqNums ...arbutil.MessageIndex) {
	t.Helper()
	for _, seqNum := range seqNums {
		Require(t, bklg.Append(&m.BroadcastMessage{Version: 1, Messages: []*m.BroadcastFeedMessage{testFeedMessage(seqNum)}}))
	}
}

// serializeForClient serializes bm the same way ClientManager.doBroadcast does
// for a client without compression.
func serializeForClient(t *testing.T, bm *m.BroadcastMessage) []byte {
	t.Helper()
	notCompressed, _, err := serializeMessage(bm, true, false)
	Require(t, err)
	return notCompressed.Bytes()
}

// readFeedMessage reads the next broadcast message the server wrote to the
// client and returns the sequence numbers it carries and whether it carried a
// confirmed sequence number.
func readFeedMessage(t *testing.T, peerConn net.Conn) ([]arbutil.MessageIndex, bool) {
	t.Helper()
	Require(t, peerConn.SetReadDeadline(time.Now().Add(5*time.Second)))
	data, err := wsutil.ReadServerText(peerConn)
	Require(t, err, "reading feed message")
	var bm m.BroadcastMessage
	Require(t, json.Unmarshal(data, &bm))
	seqNums := make([]arbutil.MessageIndex, 0, len(bm.Messages))
	for _, msg := range bm.Messages {
		seqNums = append(seqNums, msg.SequenceNumber)
	}
	return seqNums, bm.ConfirmedSequenceNumberMessage != nil
}

func expectFeedMessage(t *testing.T, peerConn net.Conn, expectedSeqNums []arbutil.MessageIndex, expectConfirmed bool) {
	t.Helper()
	seqNums, confirmed := readFeedMessage(t, peerConn)
	if confirmed != expectConfirmed {
		t.Fatalf("expected confirmed sequence number message: %v, got %v (sequence numbers %v)", expectConfirmed, confirmed, seqNums)
	}
	if len(seqNums) != len(expectedSeqNums) {
		t.Fatalf("expected sequence numbers %v, got %v", expectedSeqNums, seqNums)
	}
	for i := range seqNums {
		if seqNums[i] != expectedSeqNums[i] {
			t.Fatalf("expected sequence numbers %v, got %v", expectedSeqNums, seqNums)
		}
	}
}

// TestClientConnectionCatchesUpAfterConfirmedSequenceNumber checks that
// messages broadcast between a client's backlog being written and the client
// being registered are still delivered when the first message the registered
// client receives is a confirmed sequence number message rather than a feed
// message.
func TestClientConnectionCatchesUpAfterConfirmedSequenceNumber(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config := DefaultTestBroadcasterConfig
	bklg := backlog.NewBacklog(func() *backlog.Config { return &config.Backlog })
	appendToBacklog(t, bklg, 1, 2, 3)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	Require(t, err)
	defer ln.Close()
	peerConn, err := net.Dial("tcp", ln.Addr().String())
	Require(t, err)
	defer peerConn.Close()
	serverConn, err := ln.Accept()
	Require(t, err)
	defer serverConn.Close()

	clientAction := make(chan ClientConnectionAction, 1)
	cc := NewClientConnection(serverConn, nil, clientAction, 1, net.IPv4(127, 0, 0, 1), false, config.MaxSendQueue, 0, bklg)
	cc.Start(ctx)
	defer cc.StopAndWait()

	// The backlog is sent before the client registers with the ClientManager.
	expectFeedMessage(t, peerConn, []arbutil.MessageIndex{1, 2, 3}, false)
	select {
	case action := <-clientAction:
		if !action.create {
			t.Fatal("client asked to be removed instead of registered")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for the client to register")
	}

	// Messages 4 and 5 are broadcast while the client is not yet registered,
	// so it never sees them on its out channel.
	appendToBacklog(t, bklg, 4, 5)
	cc.Registered()

	// The first thing the registered client receives is a confirmation.
	confirmation := &m.BroadcastMessage{
		Version:                        1,
		ConfirmedSequenceNumberMessage: &m.ConfirmedSequenceNumberMessage{SequenceNumber: 2},
	}
	Require(t, bklg.Append(confirmation))
	cc.out <- message{sequenceNumber: nil, data: serializeForClient(t, confirmation)}
	expectFeedMessage(t, peerConn, nil, true)

	// Then message 6 is broadcast: 4 and 5 must be caught up from the backlog first.
	feedMessage := &m.BroadcastMessage{Version: 1, Messages: []*m.BroadcastFeedMessage{testFeedMessage(6)}}
	Require(t, bklg.Append(feedMessage))
	seqNum := arbutil.MessageIndex(6)
	cc.out <- message{sequenceNumber: &seqNum, data: serializeForClient(t, feedMessage)}
	expectFeedMessage(t, peerConn, []arbutil.MessageIndex{4, 5}, false)
	expectFeedMessage(t, peerConn, []arbutil.MessageIndex{6}, false)
}
