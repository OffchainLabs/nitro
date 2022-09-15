// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package broadcaster

import (
	"context"
	"errors"
	"github.com/ethereum/go-ethereum/metrics"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/wsbroadcastserver"
)

var (
	confirmedSequenceNumberGauge = metrics.NewRegisteredGauge("arb/feed/sequence-number/confirmed", nil)
	latestSequenceNumberGauge    = metrics.NewRegisteredGauge("arb/feed/sequence-number/latest", nil)
)

type SequenceNumberCatchupBuffer struct {
	messages     []*BroadcastFeedMessage
	messageCount int32
}

func NewSequenceNumberCatchupBuffer() *SequenceNumberCatchupBuffer {
	return &SequenceNumberCatchupBuffer{}
}

func (b *SequenceNumberCatchupBuffer) getCacheMessages(requestedSeqNum arbutil.MessageIndex) *BroadcastMessage {
	if b.messageCount == 0 {
		return nil
	}
	var startingIndex int32
	// Ignore messages older than requested sequence number
	if b.messages[0].SequenceNumber < requestedSeqNum {
		startingIndex = int32(requestedSeqNum - b.messages[0].SequenceNumber)
		if startingIndex >= b.messageCount {
			// Past end, nothing to return
			return nil
		}
		if b.messages[startingIndex].SequenceNumber != requestedSeqNum {
			log.Error("requestedSeqNum not found where expected", "requestedSeqNum", requestedSeqNum, "seqNumZero", b.messages[0].SequenceNumber, "startingIndex", startingIndex, "foundSeqNum", b.messages[startingIndex].SequenceNumber)
			return nil
		}
	}

	messagesToSend := b.messages[startingIndex:]
	if len(messagesToSend) > 0 {
		bm := BroadcastMessage{
			Version:  1,
			Messages: messagesToSend,
		}

		return &bm
	}

	return nil
}

func (b *SequenceNumberCatchupBuffer) OnRegisterClient(ctx context.Context, clientConnection *wsbroadcastserver.ClientConnection) error {
	start := time.Now()
	bm := b.getCacheMessages(clientConnection.RequestedSeqNum())
	if bm != nil {
		// send the newly connected client the requested messages
		err := clientConnection.Write(bm)
		if err != nil {
			log.Error("error sending client cached messages", err, "client", clientConnection.Name, "elapsed", time.Since(start))
			return err
		}
	}

	log.Info("client registered", "client", clientConnection.Name, "elapsed", time.Since(start))

	return nil
}

func (b *SequenceNumberCatchupBuffer) deleteConfirmed(confirmedSequenceNumber arbutil.MessageIndex) {
	if len(b.messages) == 0 {
		return
	}

	firstSequenceNumber := b.messages[0].SequenceNumber

	if confirmedSequenceNumber < firstSequenceNumber {
		// Confirmed sequence number is older than cache, so nothing to do
		return
	}

	confirmedIndex := uint64(confirmedSequenceNumber - firstSequenceNumber)

	if confirmedIndex >= uint64(len(b.messages)) {
		log.Error("ConfirmedSequenceNumber is past the end of stored messages", "confirmedSequenceNumber", confirmedSequenceNumber, "firstSequenceNumber", firstSequenceNumber, "cacheLength", len(b.messages))
		b.messages = nil
		return
	}

	if b.messages[confirmedIndex].SequenceNumber != confirmedSequenceNumber {
		// Log instead of returning error here so that the message will be sent to downstream
		// relays to also cause them to be cleared.
		log.Error("Invariant violation: confirmedSequenceNumber is not where expected, clearing buffer", "confirmedSequenceNumber", confirmedSequenceNumber, "firstSequenceNumber", firstSequenceNumber, "cacheLength", len(b.messages), "foundSequenceNumber", b.messages[confirmedIndex].SequenceNumber)
		b.messages = nil
		return
	}

	b.messages = b.messages[confirmedIndex+1:]
	if len(b.messages) > 10 && cap(b.messages) > len(b.messages)*10 {
		// Too much spare capacity, copy to fresh slice to reset memory usage
		b.messages = append([]*BroadcastFeedMessage(nil), b.messages[:len(b.messages)]...)
	}
}

func (b *SequenceNumberCatchupBuffer) OnDoBroadcast(bmi interface{}) error {
	broadcastMessage, ok := bmi.(BroadcastMessage)
	if !ok {
		msg := "requested to broadcast message of unknown type"
		log.Error(msg)
		return errors.New(msg)
	}
	defer func() { atomic.StoreInt32(&b.messageCount, int32(len(b.messages))) }()

	if confirmMsg := broadcastMessage.ConfirmedSequenceNumberMessage; confirmMsg != nil {
		b.deleteConfirmed(confirmMsg.SequenceNumber)
		confirmedSequenceNumberGauge.Update(int64(confirmMsg.SequenceNumber))
	}

	for _, newMsg := range broadcastMessage.Messages {
		if len(b.messages) == 0 {
			// Add to empty list
			b.messages = append(b.messages, newMsg)
			latestSequenceNumberGauge.Update(int64(newMsg.SequenceNumber))
		} else if expectedSequenceNumber := b.messages[len(b.messages)-1].SequenceNumber + 1; newMsg.SequenceNumber == expectedSequenceNumber {
			// Next sequence number to add to end of list
			b.messages = append(b.messages, newMsg)
			latestSequenceNumberGauge.Update(int64(newMsg.SequenceNumber))
		} else if newMsg.SequenceNumber > expectedSequenceNumber {
			log.Warn(
				"Message requested to be broadcast has unexpected sequence number; discarding to seqNum from catchup buffer",
				"seqNum", newMsg.SequenceNumber,
				"expectedSeqNum", expectedSequenceNumber,
			)
			b.messages = nil
			b.messages = append(b.messages, newMsg)
			latestSequenceNumberGauge.Update(int64(newMsg.SequenceNumber))
		} else {
			log.Info("Skipping already seen message", "seqNum", newMsg.SequenceNumber)
		}
	}

	return nil

}

func (b *SequenceNumberCatchupBuffer) GetMessageCount() int {
	return int(atomic.LoadInt32(&b.messageCount))
}
