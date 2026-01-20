package arbostypes

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
)

type MessageWithMetadata struct {
	Message             *L1IncomingMessage `json:"message"`
	DelayedMessagesRead uint64             `json:"delayedMessagesRead"`
}

func (m *MessageWithMetadata) Hash() common.Hash {
	encoded, err := rlp.EncodeToBytes(m)
	if err != nil {
		panic(err)
	}
	return crypto.Keccak256Hash(encoded)
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
