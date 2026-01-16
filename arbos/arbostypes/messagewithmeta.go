package arbostypes

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/offchainlabs/nitro/util/arbmath"
)

type MessageWithMetadata struct {
	Message             *L1IncomingMessage `json:"message"`
	DelayedMessagesRead uint64             `json:"delayedMessagesRead"`
}

func (m *MessageWithMetadata) Hash() common.Hash {
	hash := crypto.Keccak256(
		[]byte{m.Message.Header.Kind},
		m.Message.Header.Poster.Bytes(),
		arbmath.UintToBytes(m.Message.Header.BlockNumber),
		arbmath.UintToBytes(m.Message.Header.Timestamp),
		m.Message.Header.RequestId.Bytes(),
		arbmath.U256Bytes(m.Message.Header.L1BaseFee),
		crypto.Keccak256(m.Message.L2msg),
	)
	return crypto.Keccak256Hash(hash)
}

// lint:require-exhaustive-initialization
type MessageWithMetadataAndBlockInfo struct {
	MessageWithMeta MessageWithMetadata
	BlockHash       *common.Hash
	BlockMetadata   common.BlockMetadata
}

var EmptyTestMessageWithMetadata = MessageWithMetadata{
	Message: &EmptyTestIncomingMessage,
}

// TestMessageWithMetadataAndRequestId message signature is only verified if requestId defined
var TestMessageWithMetadataAndRequestId = MessageWithMetadata{
	Message: &TestIncomingMessageWithRequestId,
}

type InboxMultiplexer interface {
	Pop(context.Context) (*MessageWithMetadata, error)
	DelayedMessagesRead() uint64
}
