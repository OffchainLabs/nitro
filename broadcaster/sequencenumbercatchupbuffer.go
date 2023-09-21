// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package broadcaster

import (
	"errors"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"

	"github.com/offchainlabs/nitro/arbutil"
	m "github.com/offchainlabs/nitro/broadcaster/message"
	"github.com/offchainlabs/nitro/wsbroadcastserver"
)

const (
	// Do not send cache if requested seqnum is older than last cached minus maxRequestedSeqNumOffset
	maxRequestedSeqNumOffset = arbutil.MessageIndex(10_000)
)

var (
	confirmedSequenceNumberGauge = metrics.NewRegisteredGauge("arb/sequencenumber/confirmed", nil)
	cachedMessagesSentHistogram  = metrics.NewRegisteredHistogram("arb/feed/clients/cache/sent", nil, metrics.NewBoundedHistogramSample())
)

type SequenceNumberCatchupBuffer struct {
	messages     []*m.BroadcastFeedMessage
	messageCount int32
	limitCatchup func() bool
	maxCatchup   func() int
}

func NewSequenceNumberCatchupBuffer(limitCatchup func() bool, maxCatchup func() int) *SequenceNumberCatchupBuffer {
	return &SequenceNumberCatchupBuffer{
		limitCatchup: limitCatchup,
		maxCatchup:   maxCatchup,
	}
}

func (b *SequenceNumberCatchupBuffer) getCacheMessages(requestedSeqNum arbutil.MessageIndex) *m.BroadcastMessage {
	if len(b.messages) == 0 {
		return nil
	}
	var startingIndex int32
	// Ignore messages older than requested sequence number
	firstCachedSeqNum := b.messages[0].SequenceNumber
	if firstCachedSeqNum < requestedSeqNum {
		lastCachedSeqNum := firstCachedSeqNum + arbutil.MessageIndex(len(b.messages)-1)
		if lastCachedSeqNum < requestedSeqNum {
			// Past end, nothing to return
			return nil
		}
		startingIndex = int32(requestedSeqNum - firstCachedSeqNum)
		if startingIndex >= int32(len(b.messages)) {
			log.Error("unexpected startingIndex", "requestedSeqNum", requestedSeqNum, "firstCachedSeqNum", firstCachedSeqNum, "startingIndex", startingIndex, "lastCachedSeqNum", lastCachedSeqNum, "cacheLength", len(b.messages))
			return nil
		}
		if b.messages[startingIndex].SequenceNumber != requestedSeqNum {
			log.Error("requestedSeqNum not found where expected", "requestedSeqNum", requestedSeqNum, "firstCachedSeqNum", firstCachedSeqNum, "startingIndex", startingIndex, "foundSeqNum", b.messages[startingIndex].SequenceNumber)
			return nil
		}
	} else if b.limitCatchup() && firstCachedSeqNum > maxRequestedSeqNumOffset && requestedSeqNum < (firstCachedSeqNum-maxRequestedSeqNumOffset) {
		// Requested seqnum is too old, don't send any cache
		return nil
	}

	messagesToSend := b.messages[startingIndex:]
	if len(messagesToSend) > 0 {
		bm := m.BroadcastMessage{
			Version:  1,
			Messages: messagesToSend,
		}

		return &bm
	}

	return nil
}

func (b *SequenceNumberCatchupBuffer) getCacheMessagesBefore(requestedSeqNum arbutil.MessageIndex) (*m.BroadcastMessage, error) {
	bm := m.BroadcastMessage{Version: 1}
	if len(b.messages) == 0 {
		return &bm, nil
	}

	firstCachedSeqNum := b.messages[0].SequenceNumber
	requestedIndex := -1

	if firstCachedSeqNum <= requestedSeqNum {
		requestedIndex = int(requestedSeqNum - firstCachedSeqNum)
		cacheLength := len(b.messages)
		if requestedIndex < 0 || requestedIndex >= cacheLength {
			msg := fmt.Sprintf("unexpected requestedIndex requestedSeqNum=%d firstCachedSeqNum=%d requestedIndex=%d cacheLength=%d", requestedSeqNum, firstCachedSeqNum, requestedIndex, cacheLength)
			log.Error(msg)
			return nil, errors.New(msg)
		}

		if b.messages[requestedIndex].SequenceNumber != requestedSeqNum {
			msg := fmt.Sprintf("requestedSeqNum not found where expected requestedSeqNum=%d firstCachedSeqNum=%d requestedIndex=%d cacheLength=%d foundSeqNum=%d", requestedSeqNum, firstCachedSeqNum, requestedIndex, cacheLength, b.messages[requestedIndex].SequenceNumber)
			log.Error(msg)
			return nil, errors.New(msg)
		}
	}

	bm.Messages = b.messages[:requestedIndex+1]
	return &bm, nil
}

func (b *SequenceNumberCatchupBuffer) OnRegisterClient(clientConnection *wsbroadcastserver.ClientConnection) (error, int, time.Duration) {
	start := time.Now()
	bm := b.getCacheMessages(clientConnection.RequestedSeqNum())
	var bmCount int
	if bm != nil {
		bmCount = len(bm.Messages)
	}
	if bm != nil {
		// send the newly connected client the requested messages
		err := clientConnection.Write(bm)
		if err != nil {
			log.Error("error sending client cached messages", "error", err, "client", clientConnection.Name, "elapsed", time.Since(start))
			return err, 0, 0
		}
	}

	cachedMessagesSentHistogram.Update(int64(bmCount))

	return nil, bmCount, time.Since(start)
}

// Takes as input an index into the messages array, not a message index
func (b *SequenceNumberCatchupBuffer) pruneBufferToIndex(idx int) {
	b.messages = b.messages[idx:]
	if len(b.messages) > 10 && cap(b.messages) > len(b.messages)*10 {
		// Too much spare capacity, copy to fresh slice to reset memory usage
		b.messages = append([]*BroadcastFeedMessage(nil), b.messages[:len(b.messages)]...)
	}
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

	b.pruneBufferToIndex(int(confirmedIndex) + 1)
}

func (b *SequenceNumberCatchupBuffer) OnDoBroadcast(bmi interface{}) error {
	broadcastMessage, ok := bmi.(m.BroadcastMessage)
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

	maxCatchup := b.maxCatchup()
	if maxCatchup == 0 {
		b.messages = nil
		return nil
	}

	for _, newMsg := range broadcastMessage.Messages {
		if len(b.messages) == 0 {
			// Add to empty list
			b.messages = append(b.messages, newMsg)
		} else if expectedSequenceNumber := b.messages[len(b.messages)-1].SequenceNumber + 1; newMsg.SequenceNumber == expectedSequenceNumber {
			// Next sequence number to add to end of list
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

	if maxCatchup >= 0 && len(b.messages) > maxCatchup {
		b.pruneBufferToIndex(len(b.messages) - maxCatchup)
	}

	return nil

}

// This feels very weird, follows same pattern of OnRegisterClient but it feels like we are splitting a lot of the HTTP logic away from the HTTP server. It might be better to move objects like m.BroadcastMessage to a different lib. Or changing the object that is returned by getCacheMessagesBefore.
func (b *SequenceNumberCatchupBuffer) OnHTTPRequest(w http.ResponseWriter, requestedSeqNum arbutil.MessageIndex) {
}

func (b *SequenceNumberCatchupBuffer) GetMessageCount() int {
	return int(atomic.LoadInt32(&b.messageCount))
}
