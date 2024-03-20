package message

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
)

const (
	V1 = 1
)

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

	CumulativeSumMsgSize uint64 `json:"-"`
}

func (m *BroadcastFeedMessage) Size() uint64 {
	return uint64(len(m.Signature) + len(m.Message.Message.L2msg) + 160)
}

func (m *BroadcastFeedMessage) UpdateCumulativeSumMsgSize(val uint64) {
	m.CumulativeSumMsgSize += val + m.Size()
}

func (m *BroadcastFeedMessage) Hash(chainId uint64) (common.Hash, error) {
	return m.Message.Hash(m.SequenceNumber, chainId)
}

type ConfirmedSequenceNumberMessage struct {
	SequenceNumber arbutil.MessageIndex `json:"sequenceNumber"`
}
