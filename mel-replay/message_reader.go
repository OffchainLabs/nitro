// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package melreplay

import (
	"bytes"
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/offchainlabs/nitro/arbnode/mel"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
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

func PeekFromAccumulator[T any](
	ctx context.Context,
	preimageResolver PreimageResolver,
	outBox common.Hash,
	lookbacks uint64,
) (*T, error) {
	var msgHash common.Hash
	curr := outBox
	lookbacksForLogging := lookbacks
	for lookbacks > 0 {
		result, err := preimageResolver.ResolveTypedPreimage(arbutil.Keccak256PreimageType, curr)
		if err != nil {
			return nil, err
		}
		if len(result) != 2*common.HashLength {
			return nil, fmt.Errorf("invalid preimage result length: %d, wanted %d", len(result), 2*common.HashLength)
		}
		// Split result into left and right halves.
		// TODO: Make a helper function.
		mid := len(result) / 2
		left := result[:mid]
		msgHash = common.BytesToHash(result[mid:])
		curr = common.BytesToHash(left)
		lookbacks--
	}
	objectBytes, err := preimageResolver.ResolveTypedPreimage(arbutil.Keccak256PreimageType, msgHash)
	if err != nil {
		return nil, err
	}
	object := new(T)
	if err = rlp.Decode(bytes.NewBuffer(objectBytes), &object); err != nil {
		return nil, fmt.Errorf("failed to decode accumulator object at lookback position %d: %w", lookbacksForLogging, err)
	}
	return object, nil
}
