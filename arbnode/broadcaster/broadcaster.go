//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package broadcaster

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/offchainlabs/arbitrum/packages/arb-util/configuration"
	"github.com/offchainlabs/arbitrum/packages/arb-util/wsbroadcastserver"
	"github.com/offchainlabs/arbstate/arbstate"

	"github.com/rs/zerolog/log"
)

var logger = log.With().Caller().Str("component", "broadcaster").Logger()

type Broadcaster struct {
	server        *wsbroadcastserver.WSBroadcastServer
	catchupBuffer *SequenceNumberCatchupBuffer
}

/*
 * The base message type for messages to send over the network.
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
	SequenceNumber uint64                       `json:"sequenceNumber"`
	Message        arbstate.MessageWithMetadata `json:"message"`
}

type ConfirmedSequenceNumberMessage struct {
	SequenceNumber uint64 `json:"sequenceNumber"`
}

type SequenceNumberCatchupBuffer struct {
	messages     []*BroadcastFeedMessage
	messageCount int32
}

func NewSequenceNumberCatchupBuffer() *SequenceNumberCatchupBuffer {
	return &SequenceNumberCatchupBuffer{}
}

func (b *SequenceNumberCatchupBuffer) OnRegisterClient(ctx context.Context, clientConnection *wsbroadcastserver.ClientConnection) error {
	start := time.Now()
	if len(b.messages) > 0 {
		// send the newly connected client all the messages we've got...
		bm := BroadcastMessage{
			Version:  1,
			Messages: b.messages,
		}

		err := clientConnection.Write(bm)
		if err != nil {
			logger.Error().Err(err).Str("client", clientConnection.Name).Str("elapsed", time.Since(start).String()).Msg("error sending client cached messages")
			return err
		}
	}

	logger.Info().Str("client", clientConnection.Name).Str("elapsed", time.Since(start).String()).Msg("client registered")

	return nil
}

func (b *SequenceNumberCatchupBuffer) OnDoBroadcast(bmi interface{}) error {
	bm := bmi.(BroadcastMessage)
	if confirmMsg := bm.ConfirmedSequenceNumberMessage; confirmMsg != nil {
		latestConfirmedIndex := -1
		for i, storedMsg := range b.messages {
			if storedMsg.SequenceNumber <= confirmMsg.SequenceNumber {
				latestConfirmedIndex = i
			} else {
				break
			}
		}
		b.messages = b.messages[latestConfirmedIndex+1:]
	} else if len(bm.Messages) > 0 {
		// Add to cache to send to new clients
		if len(b.messages) == 0 {
			// Current list is empty
			b.messages = append(b.messages, bm.Messages...)
		} else if b.messages[len(b.messages)-1].SequenceNumber < bm.Messages[0].SequenceNumber {
			// TODO what about skipped sequence numbers?
			b.messages = append(b.messages, bm.Messages...)
		} else {
			// TODO Should this be fatal?
			logger.Fatal().Uint64("last SequenceNumber", b.messages[len(b.messages)-1].SequenceNumber).Uint64("current SequenceNumber", bm.Messages[0].SequenceNumber).Msg("broadcaster reorg not supported")
		}
	}

	atomic.StoreInt32(&b.messageCount, int32(len(b.messages)))

	return nil

}

func (b *SequenceNumberCatchupBuffer) GetMessageCount() int {
	return int(atomic.LoadInt32(&b.messageCount))
}

func NewBroadcaster(settings configuration.FeedOutput) *Broadcaster {
	catchupBuffer := NewSequenceNumberCatchupBuffer()
	return &Broadcaster{
		server:        wsbroadcastserver.NewWSBroadcastServer(settings, catchupBuffer),
		catchupBuffer: catchupBuffer,
	}
}

func (b *Broadcaster) BroadcastSingle(msg arbstate.MessageWithMetadata, seq uint64) {
	var broadcastMessages []*BroadcastFeedMessage

	bfm := BroadcastFeedMessage{SequenceNumber: seq, Message: msg}
	broadcastMessages = append(broadcastMessages, &bfm)

	bm := BroadcastMessage{
		Version:  1,
		Messages: broadcastMessages,
	}

	b.server.Broadcast(bm)
}

func (b *Broadcaster) Confirm(seq uint64) {
	b.server.Broadcast(BroadcastMessage{
		Version:                        1,
		ConfirmedSequenceNumberMessage: &ConfirmedSequenceNumberMessage{seq}})
}

func (b *Broadcaster) ClientCount() int32 {
	return b.server.ClientCount()
}

func (b *Broadcaster) Start(ctx context.Context) error {
	return b.server.Start(ctx)
}

func (b *Broadcaster) Stop() {
	b.server.Stop()
}
