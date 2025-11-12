package arbostypes

import (
	"context"
	"encoding/binary"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/offchainlabs/nitro/arbutil"
)

var uniquifyingPrefix = []byte("Arbitrum Nitro Feed:")

type MessageWithMetadata struct {
	Message             *L1IncomingMessage `json:"message"`
	DelayedMessagesRead uint64             `json:"delayedMessagesRead"`
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

func (m *MessageWithMetadata) Hash(sequenceNumber arbutil.MessageIndex, chainId uint64) common.Hash {
	serializedExtraData := make([]byte, 24)
	binary.BigEndian.PutUint64(serializedExtraData[:8], uint64(sequenceNumber))
	binary.BigEndian.PutUint64(serializedExtraData[8:16], chainId)
	binary.BigEndian.PutUint64(serializedExtraData[16:], m.DelayedMessagesRead)
	messageHash := m.Message.Hash().Bytes()
	return crypto.Keccak256Hash(uniquifyingPrefix, serializedExtraData, messageHash)
}

type InboxMultiplexer interface {
	Pop(context.Context) (*MessageWithMetadata, error)
	DelayedMessagesRead() uint64
}
