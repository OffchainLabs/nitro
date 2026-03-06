// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package melreplay

import (
	"context"
	"fmt"

	"github.com/offchainlabs/nitro/arbnode/mel"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
)

type MessageReader struct {
	preimageResolver PreimageResolver
}

func NewMessageReader(preimageResolver PreimageResolver) *MessageReader {
	return &MessageReader{preimageResolver}
}

func (m *MessageReader) Read(
	ctx context.Context,
	state *mel.State,
	msgIndex uint64,
) (*arbostypes.MessageWithMetadata, error) {
	if msgIndex >= state.MsgCount {
		return nil, fmt.Errorf("index %d out of range, total messages: %d", msgIndex, state.MsgCount)
	}
	return PeekFromAccumulator[arbostypes.MessageWithMetadata](ctx, m.preimageResolver, state.LocalMsgAccumulator, state.MsgCount-msgIndex)
}
