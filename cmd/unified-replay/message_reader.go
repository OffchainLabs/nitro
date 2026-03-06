// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package main

import (
	"context"
	"fmt"

	"github.com/offchainlabs/nitro/arbnode/mel"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	melreplay "github.com/offchainlabs/nitro/mel-replay"
)

type messageReader struct {
	preimageResolver melreplay.PreimageResolver
}

func newMessageReader(preimageResolver melreplay.PreimageResolver) *messageReader {
	return &messageReader{preimageResolver}
}

func (m *messageReader) Read(
	ctx context.Context,
	state *mel.State,
	msgIndex uint64,
) (*arbostypes.MessageWithMetadata, error) {
	if msgIndex >= state.MsgCount {
		return nil, fmt.Errorf("index %d out of range, total messages: %d", msgIndex, state.MsgCount)
	}
	return melreplay.PeekFromAccumulator[arbostypes.MessageWithMetadata](ctx, m.preimageResolver, state.LocalMsgAccumulator, state.MsgCount-msgIndex)
}
