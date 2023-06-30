// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/nitro/blob/master/LICENSE

package broadcaster

import (
	"context"
	"net"

	"github.com/gobwas/ws"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/util/signature"
	"github.com/offchainlabs/nitro/wsbroadcastserver"
)

type Broadcaster struct {
	server        *wsbroadcastserver.WSBroadcastServer
	catchupBuffer *SequenceNumberCatchupBuffer
	chainId       uint64
	dataSigner    signature.DataSignerFunc
}

// BroadcastMessage is the base message type for messages to send over the network.
//
// Acts as a variant holding the message types. The type of the message is
// indicated by whichever of the fields is non-empty. The fields holding the message
// types are annotated with omitempty so only the populated message is sent as
// json. The message fields should be pointers or slices and end with
// "Messages" or "Message".
//
// The format is forwards compatible, ie if a json BroadcastMessage is received that
// has fields that are not in the Go struct then deserialization will succeed
// skip the unknown field [1]
//
// References:
// [1] https://pkg.go.dev/encoding/json#Unmarshal
type BroadcastMessage struct {
	Version int `json:"version"`
	// TODO better name than messages since there are different types of messages
	Messages                       []*BroadcastFeedMessage         `json:"messages,omitempty"`
	ConfirmedSequenceNumberMessage *ConfirmedSequenceNumberMessage `json:"confirmedSequenceNumberMessage,omitempty"`
}

type BroadcastFeedMessage struct {
	SequenceNumber arbutil.MessageIndex           `json:"sequenceNumber"`
	Message        arbostypes.MessageWithMetadata `json:"message"`
	Signature      []byte                         `json:"signature"`
}

func (m *BroadcastFeedMessage) Hash(chainId uint64) (common.Hash, error) {
	return m.Message.Hash(m.SequenceNumber, chainId)
}

type ConfirmedSequenceNumberMessage struct {
	SequenceNumber arbutil.MessageIndex `json:"sequenceNumber"`
}

func NewBroadcaster(config wsbroadcastserver.BroadcasterConfigFetcher, chainId uint64, feedErrChan chan error, dataSigner signature.DataSignerFunc) *Broadcaster {
	catchupBuffer := NewSequenceNumberCatchupBuffer(func() bool { return config().LimitCatchup })
	return &Broadcaster{
		server:        wsbroadcastserver.NewWSBroadcastServer(config, catchupBuffer, chainId, feedErrChan),
		catchupBuffer: catchupBuffer,
		chainId:       chainId,
		dataSigner:    dataSigner,
	}
}

func (b *Broadcaster) NewBroadcastFeedMessage(message arbostypes.MessageWithMetadata, sequenceNumber arbutil.MessageIndex) (*BroadcastFeedMessage, error) {
	var messageSignature []byte
	if b.dataSigner != nil {
		hash, err := message.Hash(sequenceNumber, b.chainId)
		if err != nil {
			return nil, err
		}
		messageSignature, err = b.dataSigner(hash.Bytes())
		if err != nil {
			return nil, err
		}
	}

	return &BroadcastFeedMessage{
		SequenceNumber: sequenceNumber,
		Message:        message,
		Signature:      messageSignature,
	}, nil
}

func (b *Broadcaster) BroadcastSingle(msg arbostypes.MessageWithMetadata, seq arbutil.MessageIndex) error {
	defer func() {
		if r := recover(); r != nil {
			log.Error("recovered error in BroadcastSingle", "recover", r)
		}
	}()
	bfm, err := b.NewBroadcastFeedMessage(msg, seq)
	if err != nil {
		return err
	}

	b.BroadcastSingleFeedMessage(bfm)
	return nil
}

func (b *Broadcaster) BroadcastSingleFeedMessage(bfm *BroadcastFeedMessage) {
	broadcastFeedMessages := make([]*BroadcastFeedMessage, 0, 1)

	broadcastFeedMessages = append(broadcastFeedMessages, bfm)

	b.BroadcastFeedMessages(broadcastFeedMessages)
}

func (b *Broadcaster) BroadcastFeedMessages(messages []*BroadcastFeedMessage) {

	bm := BroadcastMessage{
		Version:  1,
		Messages: messages,
	}

	b.server.Broadcast(bm)
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

func (b *Broadcaster) StartWithHeader(ctx context.Context, header ws.HandshakeHeader) error {
	return b.server.StartWithHeader(ctx, header)
}

func (b *Broadcaster) StopAndWait() {
	b.server.StopAndWait()
}

func (b *Broadcaster) Started() bool {
	return b.server.Started()
}
