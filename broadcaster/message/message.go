package message

import (
	"encoding/binary"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
)

const (
	V1                 = 1
	TimeboostedVersion = byte(0)
)

var uniquifyingPrefix = []byte("Arbitrum Nitro Feed:")

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
	BlockHash      *common.Hash                   `json:"blockHash,omitempty"`
	Signature      []byte                         `json:"signatureV2"`
	BlockMetadata  common.BlockMetadata           `json:"blockMetadata,omitempty"`

	CumulativeSumMsgSize uint64 `json:"-"`
}

func (m *BroadcastFeedMessage) Size() uint64 {
	// #nosec G115
	return uint64(len(m.Signature) + len(m.Message.Message.L2msg) + 160)
}

func (m *BroadcastFeedMessage) UpdateCumulativeSumMsgSize(val uint64) {
	m.CumulativeSumMsgSize += val + m.Size()
}

// SignatureHash creates a hash for the feed message that include all fields that need to
// be signed. Be aware that changing this function can break compatibility with older clients.
func (m *BroadcastFeedMessage) SignatureHash(chainId uint64) common.Hash {
	data := []byte{}
	data = append(data, uniquifyingPrefix...)

	data = binary.BigEndian.AppendUint64(data, chainId)
	data = binary.BigEndian.AppendUint64(data, uint64(m.SequenceNumber))
	if m.BlockHash != nil {
		data = append(data, m.BlockHash.Bytes()...)
	}
	data = append(data, m.BlockMetadata...)
	data = binary.BigEndian.AppendUint64(data, m.Message.DelayedMessagesRead)

	l1IncomingMessage := m.Message.Message
	data = append(data, l1IncomingMessage.Header.Kind)
	data = append(data, l1IncomingMessage.Header.Poster.Bytes()...)
	data = binary.BigEndian.AppendUint64(data, l1IncomingMessage.Header.BlockNumber)
	data = binary.BigEndian.AppendUint64(data, l1IncomingMessage.Header.Timestamp)
	if l1IncomingMessage.Header.RequestId != nil {
		data = append(data, l1IncomingMessage.Header.RequestId.Bytes()...)
	}
	if l1IncomingMessage.Header.L1BaseFee != nil {
		data = append(data, l1IncomingMessage.Header.L1BaseFee.Bytes()...)
	}
	data = append(data, l1IncomingMessage.L2msg...)

	return crypto.Keccak256Hash(data)
}

type ConfirmedSequenceNumberMessage struct {
	SequenceNumber arbutil.MessageIndex `json:"sequenceNumber"`
}
