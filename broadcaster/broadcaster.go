// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package broadcaster

import (
	"context"
	"net"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/cmd/util"
	"github.com/offchainlabs/nitro/wsbroadcastserver"
)

type Broadcaster struct {
	server        *wsbroadcastserver.WSBroadcastServer
	catchupBuffer *SequenceNumberCatchupBuffer
	chainId       uint64
	dataSigner    util.DataSignerFunc
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
	Signature      []byte                       `json:"signature"`
}

func NewBroadcastFeedMessage(message arbstate.MessageWithMetadata, sequenceNumber arbutil.MessageIndex, chainId uint64, dataSigner util.DataSignerFunc) (*BroadcastFeedMessage, error) {
	var signature []byte
	// Don't need signature if request id is not present
	if message.Message.Header.RequestId != nil {
		hash, err := message.Hash(sequenceNumber, chainId)
		if err != nil {
			return nil, err
		}

		if dataSigner != nil {
			signature, err = dataSigner(hash.Bytes())
			if err != nil {
				return nil, err
			}
		}
	}

	return &BroadcastFeedMessage{
		SequenceNumber: sequenceNumber,
		Message:        message,
		Signature:      signature,
	}, nil
}

func (m *BroadcastFeedMessage) Hash(chainId uint64) (common.Hash, error) {
	return m.Message.Hash(m.SequenceNumber, chainId)
}

type ConfirmedSequenceNumberMessage struct {
	SequenceNumber arbutil.MessageIndex `json:"sequenceNumber"`
}

func NewBroadcaster(settings wsbroadcastserver.BroadcasterConfig, chainId uint64, feedErrChan chan error, dataSigner util.DataSignerFunc) *Broadcaster {
	catchupBuffer := NewSequenceNumberCatchupBuffer()
	return &Broadcaster{
		server:        wsbroadcastserver.NewWSBroadcastServer(settings, catchupBuffer, chainId, feedErrChan),
		catchupBuffer: catchupBuffer,
		chainId:       chainId,
		dataSigner:    dataSigner,
	}
}

func (b *Broadcaster) BroadcastSingle(msg arbstate.MessageWithMetadata, seq arbutil.MessageIndex) error {
	bfm, err := NewBroadcastFeedMessage(msg, seq, b.chainId, b.dataSigner)
	if err != nil {
		return err
	}

	b.BroadcastSingleFeedMessage(bfm)
	return nil
}

func (b *Broadcaster) BroadcastSingleFeedMessage(bfm *BroadcastFeedMessage) {
	broadcastFeedMessages := make([]*BroadcastFeedMessage, 0, 1)

	broadcastFeedMessages = append(broadcastFeedMessages, bfm)

	bm := BroadcastMessage{
		Version:  1,
		Messages: broadcastFeedMessages,
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
