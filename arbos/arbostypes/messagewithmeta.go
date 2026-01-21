// Copyright 2023-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
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
	encoded, err := rlp.EncodeToBytes(m.WithMELRelevantFields())
	if err != nil {
		panic(err)
	}
	return crypto.Keccak256Hash(encoded)
}

func (m *MessageWithMetadata) WithMELRelevantFields() *MessageWithMetadata {
	return &MessageWithMetadata{
		Message: &L1IncomingMessage{
			Header: m.Message.Header,
			L2msg:  m.Message.L2msg,
		},
		DelayedMessagesRead: m.DelayedMessagesRead,
	}
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
