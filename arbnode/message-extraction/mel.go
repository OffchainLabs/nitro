package mel

import (
	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/util/headerreader"
)

type MessageExtractionLayer struct {
	l1Reader       *headerreader.HeaderReader
	delayedBridge  *arbnode.DelayedBridge
	sequencerInbox *arbnode.SequencerInbox
}

func NewMessageExtractionLayer(
	l1Reader *headerreader.HeaderReader,
	delayedBridge *arbnode.DelayedBridge,
	sequencerInbox *arbnode.SequencerInbox,
) *MessageExtractionLayer {
	return &MessageExtractionLayer{
		l1Reader:       l1Reader,
		delayedBridge:  delayedBridge,
		sequencerInbox: sequencerInbox,
	}
}
