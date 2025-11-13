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

// SignaturePreimage creates a preimage for the feed signature that include all fields that need to
// be signed. Be aware that changing this function can break compatibility with older clients.
func (m *BroadcastFeedMessage) SignaturePreimage(chainId uint64) common.Hash {
	preimage := []byte{}
	preimage = append(preimage, uniquifyingPrefix...)

	preimage = binary.BigEndian.AppendUint64(preimage, chainId)
	preimage = binary.BigEndian.AppendUint64(preimage, uint64(m.SequenceNumber))
	if m.BlockHash != nil {
		preimage = append(preimage, m.BlockHash.Bytes()...)
	}
	preimage = append(preimage, m.BlockMetadata...)
	preimage = binary.BigEndian.AppendUint64(preimage, m.Message.DelayedMessagesRead)

	l1IncomingMessage := m.Message.Message
	preimage = append(preimage, l1IncomingMessage.Header.Poster.Bytes()...)
	preimage = binary.BigEndian.AppendUint64(preimage, l1IncomingMessage.Header.BlockNumber)
	preimage = binary.BigEndian.AppendUint64(preimage, l1IncomingMessage.Header.Timestamp)
	if l1IncomingMessage.Header.RequestId != nil {
		preimage = append(preimage, l1IncomingMessage.Header.RequestId.Bytes()...)
	}
	if l1IncomingMessage.Header.L1BaseFee != nil {
		preimage = append(preimage, l1IncomingMessage.Header.L1BaseFee.Bytes()...)
	}
	preimage = append(preimage, l1IncomingMessage.L2msg...)

	return crypto.Keccak256Hash(preimage)
}

type ConfirmedSequenceNumberMessage struct {
	SequenceNumber arbutil.MessageIndex `json:"sequenceNumber"`
}
