package message

import (
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
)

func CreateDummyBroadcastMessages(seqNums []arbutil.MessageIndex) []*BroadcastFeedMessage {
	return CreateDummyBroadcastMessagesImpl(seqNums, len(seqNums))
}

func CreateDummyBroadcastMessagesImpl(seqNums []arbutil.MessageIndex, length int) []*BroadcastFeedMessage {
	broadcastMessages := make([]*BroadcastFeedMessage, 0, length)
	for _, seqNum := range seqNums {
		broadcastMessage := &BroadcastFeedMessage{
			SequenceNumber: seqNum,
			Message:        arbostypes.EmptyTestMessageWithMetadata,
		}
		broadcastMessages = append(broadcastMessages, broadcastMessage)
	}

	return broadcastMessages
}
