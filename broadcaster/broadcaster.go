//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package broadcaster

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/arbitrum/packages/arb-util/configuration"
	"github.com/offchainlabs/arbitrum/packages/arb-util/wsbroadcastserver"
	"github.com/offchainlabs/arbstate/arbstate"
)

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
			log.Error("error sending client cached messages", err, "client", clientConnection.Name, "elapsed", time.Since(start))
			return err
		}
	}

	log.Info("client registered", "client", clientConnection.Name, "elapsed", time.Since(start))

	return nil
}

func (b *SequenceNumberCatchupBuffer) OnDoBroadcast(bmi interface{}) error {
	broadcastMessage, ok := bmi.(BroadcastMessage)
	if !ok {
		log.Crit("Requested to broadcast messasge of unknown type")
	}
	defer func() { atomic.StoreInt32(&b.messageCount, int32(len(b.messages))) }()

	if confirmMsg := broadcastMessage.ConfirmedSequenceNumberMessage; confirmMsg != nil {
		if len(b.messages) == 0 {
			return nil
		}

		// If new sequence number is less than the earliest in the buffer,
		// then do nothing, as this message was probably already confirmed.
		if confirmMsg.SequenceNumber < b.messages[0].SequenceNumber {
			return nil
		}
		confirmedIndex := confirmMsg.SequenceNumber - b.messages[0].SequenceNumber

		if uint64(len(b.messages)) <= confirmedIndex {
			log.Error("ConfirmedSequenceNumber message ", confirmMsg.SequenceNumber, " is past the end of stored messages. Clearing buffer. Final stored sequence number was ", b.messages[len(b.messages)-1])
			b.messages = nil
			return nil
		}

		if b.messages[confirmedIndex].SequenceNumber != confirmMsg.SequenceNumber {
			// Log instead of returning error here so that the message will be sent to downstream
			// relays to also cause them to be cleared.
			log.Error("Invariant violation: Non-sequential messages stored in SequenceNumberCatchupBuffer. Found ", b.messages[confirmedIndex].SequenceNumber, " expected ", confirmMsg.SequenceNumber, ". Clearing buffer.")
			b.messages = nil
			return nil
		}

		b.messages = b.messages[confirmedIndex+1:]
		return nil
	}

	for _, newMsg := range broadcastMessage.Messages {
		if len(b.messages) == 0 {
			b.messages = append(b.messages, newMsg)
		} else if expectedSequenceNumber := b.messages[len(b.messages)-1].SequenceNumber + 1; newMsg.SequenceNumber == expectedSequenceNumber {
			b.messages = append(b.messages, newMsg)
		} else if newMsg.SequenceNumber > expectedSequenceNumber {
			log.Warn("Message with sequence number ",
				newMsg.SequenceNumber, " requested to be broadcast, ",
				"expected ", expectedSequenceNumber,
				", discarding up to ", newMsg.SequenceNumber,
				" from catchup buffer")
			b.messages = nil
			b.messages = append(b.messages, newMsg)
		} else {
			log.Info("Skipping already seen message with sequence number: ", newMsg.SequenceNumber)
		}
	}

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
