package broadcaster

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/wsbroadcastserver"
)

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
		confirmedIndex := uint64(confirmMsg.SequenceNumber - b.messages[0].SequenceNumber)

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
			log.Warn(
				"Message requested to be broadcast has unexpected sequence number; discarding to seqNum from catchup buffer",
				"seqNum", newMsg.SequenceNumber,
				"expectedSeqNum", expectedSequenceNumber,
			)
			b.messages = nil
			b.messages = append(b.messages, newMsg)
		} else {
			log.Info("Skipping already seen message", "seqNum", newMsg.SequenceNumber)
		}
	}

	return nil

}

func (b *SequenceNumberCatchupBuffer) GetMessageCount() int {
	return int(atomic.LoadInt32(&b.messageCount))
}
