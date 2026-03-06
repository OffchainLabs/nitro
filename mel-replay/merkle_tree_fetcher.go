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
	"github.com/offchainlabs/nitro/arbutil"
)

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
		curr, msgHash, err = mel.SplitPreimage(result)
		if err != nil {
			return nil, fmt.Errorf("accumulator preimage at lookback %d: %w", lookbacks, err)
		}
		lookbacks--
	}
	objectBytes, err := preimageResolver.ResolveTypedPreimage(arbutil.Keccak256PreimageType, msgHash)
	if err != nil {
		return nil, err
	}
	object := new(T)
	if err = rlp.Decode(bytes.NewBuffer(objectBytes), &object); err != nil {
		return nil, fmt.Errorf("failed to decode merkle object at lookback position %d: %w", lookbacksForLogging, err)
	}
	return object, nil
}
