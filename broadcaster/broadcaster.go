// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package broadcaster

import (
	"context"
	"net"

	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/wsbroadcastserver"
)

type Broadcaster struct {
	server        *wsbroadcastserver.WSBroadcastServer
	catchupBuffer *SequenceNumberCatchupBuffer
}

/*
 * BroadcastMessage is the base message type for messages to send over the network.
 *
 * Acts as a variant holding the message types. The type of the message is
 * indicated by whichever of the fields is non-empty. The fields holding the message
 * types are annotated with omitempty so only the populated message is sent as
 * json. The message fields should be pointers or slices and end with
 * "Messages" or "Message".
 *
 * The format is forwards compatible, ie if a json BroadcastMessage is received that
 * has fields that are not in the Go struct then deserialization will succeed
 * skip the unknown field [1]
 *
 * References:
 * [1] https://pkg.go.dev/encoding/json#Unmarshal
 */
type BroadcastMessage struct {
	Version int `json:"version"`
	// TODO better name than messages since there are different types of messages
	Messages                       []*BroadcastFeedMessage         `json:"messages,omitempty"`
	ConfirmedSequenceNumberMessage *ConfirmedSequenceNumberMessage `json:"confirmedSequenceNumberMessage,omitempty"`
}

type BroadcastFeedMessage struct {
	SequenceNumber arbutil.MessageIndex         `json:"sequenceNumber"`
	Message        arbstate.MessageWithMetadata `json:"message"`
}

type ConfirmedSequenceNumberMessage struct {
	SequenceNumber arbutil.MessageIndex `json:"sequenceNumber"`
}

func NewBroadcaster(settings wsbroadcastserver.BroadcasterConfig, chainId uint64, feedErrChan chan error) *Broadcaster {
	catchupBuffer := NewSequenceNumberCatchupBuffer()
	return &Broadcaster{
		server:        wsbroadcastserver.NewWSBroadcastServer(settings, catchupBuffer, chainId, feedErrChan),
		catchupBuffer: catchupBuffer,
	}
}

func (b *Broadcaster) BroadcastSingle(msg arbstate.MessageWithMetadata, seq arbutil.MessageIndex) {
	var broadcastMessages []*BroadcastFeedMessage

	bfm := BroadcastFeedMessage{SequenceNumber: seq, Message: msg}
	broadcastMessages = append(broadcastMessages, &bfm)

	bm := BroadcastMessage{
		Version:  1,
		Messages: broadcastMessages,
	}

	b.server.Broadcast(bm)
}

func (b *Broadcaster) Broadcast(msg BroadcastMessage) {
	b.server.Broadcast(msg)
}

func (b *Broadcaster) Confirm(seq arbutil.MessageIndex) {
	log.Debug("confirming sequence number", "sequenceNumber", seq)
	b.server.Broadcast(BroadcastMessage{
		Version:                        1,
		ConfirmedSequenceNumberMessage: &ConfirmedSequenceNumberMessage{seq}})
}

func (b *Broadcaster) ClientCount() int32 {
	return b.server.ClientCount()
}

func (b *Broadcaster) ListenerAddr() net.Addr {
	return b.server.ListenerAddr()
}

func (b *Broadcaster) GetCachedMessageCount() int {
	return b.catchupBuffer.GetMessageCount()
}

func (b *Broadcaster) Initialize() error {
	return b.server.Initialize()
}

func (b *Broadcaster) Start(ctx context.Context) error {
	return b.server.Start(ctx)
}

func (b *Broadcaster) StopAndWait() {
	b.server.StopAndWait()
}
