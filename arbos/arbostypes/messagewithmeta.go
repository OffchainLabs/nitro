package arbostypes

import (
	"context"
	"encoding/binary"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/offchainlabs/nitro/arbutil"
)

var uniquifyingPrefix = []byte("Arbitrum Nitro Feed:")

type MessageWithMetadata struct {
	Message             *L1IncomingMessage `json:"message"`
	DelayedMessagesRead uint64             `json:"delayedMessagesRead"`
}

type BlockMetadata []byte

type MessageWithMetadataAndBlockInfo struct {
	MessageWithMeta MessageWithMetadata
	BlockHash       *common.Hash
	BlockMetadata   BlockMetadata
}

var EmptyTestMessageWithMetadata = MessageWithMetadata{
	Message: &EmptyTestIncomingMessage,
}

// TestMessageWithMetadataAndRequestId message signature is only verified if requestId defined
var TestMessageWithMetadataAndRequestId = MessageWithMetadata{
	Message: &TestIncomingMessageWithRequestId,
}

// IsTxTimeboosted given a tx's index in the block returns whether the tx was timeboosted or not
func (b BlockMetadata) IsTxTimeboosted(txIndex int) bool {
	maxTxCount := (len(b) - 1) * 8
	if txIndex >= maxTxCount {
		return false
	}
	return b[1+(txIndex/8)]&(1<<(txIndex%8)) != 0
}

func (m *MessageWithMetadata) Hash(sequenceNumber arbutil.MessageIndex, chainId uint64) (common.Hash, error) {
	serializedExtraData := make([]byte, 24)
	binary.BigEndian.PutUint64(serializedExtraData[:8], uint64(sequenceNumber))
	binary.BigEndian.PutUint64(serializedExtraData[8:16], chainId)
	binary.BigEndian.PutUint64(serializedExtraData[16:], m.DelayedMessagesRead)

	serializedMessage, err := rlp.EncodeToBytes(m.Message)
	if err != nil {
		return common.Hash{}, fmt.Errorf("unable to serialize message %v: %w", sequenceNumber, err)
	}

	return crypto.Keccak256Hash(uniquifyingPrefix, serializedExtraData, serializedMessage), nil
}

type InboxMultiplexer interface {
	Pop(context.Context) (*MessageWithMetadata, error)
	DelayedMessagesRead() uint64
}
