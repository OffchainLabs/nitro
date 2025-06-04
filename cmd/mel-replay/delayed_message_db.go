package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/bits"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"

	meltypes "github.com/offchainlabs/nitro/arbnode/message-extraction/types"
	"github.com/offchainlabs/nitro/arbutil"
)

type preimageResolver interface {
	ResolveTypedPreimage(preimageType arbutil.PreimageType, hash common.Hash) ([]byte, error)
}

type delayedMessageDatabase struct {
	preimageResolver preimageResolver
}

func (d *delayedMessageDatabase) ReadDelayedMessage(
	ctx context.Context,
	state *meltypes.State,
	msgIndex uint64,
) (*meltypes.DelayedInboxMessage, error) {
	totalMsgsSeen := state.DelayedMessagedSeen
	if totalMsgsSeen == 0 {
		return nil, errors.New("no delayed messages available")
	}
	if msgIndex >= totalMsgsSeen {
		return nil, fmt.Errorf("index %d out of range, total delayed messages seen: %d", msgIndex, totalMsgsSeen)
	}
	treeSize := nextPowerOfTwo(totalMsgsSeen)
	merkleDepth := bits.TrailingZeros64(treeSize)

	// Start traversal from root, which is the delayed messages seen root.
	merkleRoot := state.DelayedMessagesSeenRoot
	currentHash := merkleRoot
	currentDepth := merkleDepth

	// Traverse down the Merkle tree to find the leaf at the given index.
	for currentDepth > 0 {
		// Resolve the preimage to get left and right children.
		result, err := d.preimageResolver.ResolveTypedPreimage(arbutil.Keccak256PreimageType, currentHash)
		if err != nil {
			return nil, err
		}
		if len(result) != 64 {
			return nil, fmt.Errorf("invalid preimage result length: %d, wanted 64", len(result))
		}
		// Split result into left and right halves.
		mid := len(result) / 2
		left := result[:mid]
		right := result[mid:]

		// Calculate which subtree contains our index.
		subtreeSize := uint64(1) << (currentDepth - 1)
		if msgIndex < subtreeSize {
			// Go left.
			currentHash = common.BytesToHash(left)
		} else {
			// Go right.
			currentHash = common.BytesToHash(right)
			msgIndex -= subtreeSize
		}
		currentDepth--
	}
	// At this point, currentHash should be the hash of the delayed message.
	delayedMsgBytes, err := d.preimageResolver.ResolveTypedPreimage(arbutil.Keccak256PreimageType, currentHash)
	if err != nil {
		return nil, err
	}
	delayedMessage := new(meltypes.DelayedInboxMessage)
	if err = rlp.Decode(bytes.NewBuffer(delayedMsgBytes), &delayedMessage); err != nil {
		return nil, fmt.Errorf("failed to decode delayed message at index %d: %w", msgIndex, err)
	}
	return delayedMessage, nil
}

func nextPowerOfTwo(n uint64) uint64 {
	if n == 0 {
		return 1
	}
	if n&(n-1) == 0 {
		return n
	}
	return 1 << bits.Len64(n)
}
