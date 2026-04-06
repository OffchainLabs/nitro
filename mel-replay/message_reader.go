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

// Read will be only able to fetch L2 message extracted in the given mel state and is part of the LocalMsgAccumulator, if the msgIndex
// corresponds to an L2 message that was extracted in a previous state- then it will fail to fetch the preimage and return an appropriate error
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
	if lookbacks == 0 {
		return nil, fmt.Errorf("lookbacks must be >= 1, got 0")
	}
	var msgHash common.Hash
	curr := outBox
	totalLookbacks := lookbacks
	for lookbacks > 0 {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		result, err := preimageResolver.ResolveTypedPreimage(arbutil.Keccak256PreimageType, curr)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve preimage at lookback %d/%d: %w", lookbacks, totalLookbacks, err)
		}
		curr, msgHash, err = mel.SplitPreimage(result)
		if err != nil {
			return nil, fmt.Errorf("accumulator preimage at lookback %d/%d: %w", lookbacks, totalLookbacks, err)
		}
		lookbacks--
	}
	objectBytes, err := preimageResolver.ResolveTypedPreimage(arbutil.Keccak256PreimageType, msgHash)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve message content preimage (after %d lookbacks): %w", totalLookbacks, err)
	}
	object := new(T)
	if err = rlp.Decode(bytes.NewBuffer(objectBytes), &object); err != nil {
		return nil, fmt.Errorf("failed to decode accumulator object (after %d lookbacks): %w", totalLookbacks, err)
	}
	return object, nil
}
