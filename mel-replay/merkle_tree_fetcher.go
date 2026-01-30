package melreplay

import (
	"bytes"
	"fmt"
	"math/bits"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/offchainlabs/nitro/arbutil"
)

func fetchObjectFromMerkleTree[T any](merkleRoot common.Hash, merkleDepth int, msgIndex uint64, preimageResolver PreimageResolver) (*T, error) {
	originalMsgIndex := msgIndex
	// Start traversal from the merkle root.
	currentHash := merkleRoot
	currentDepth := merkleDepth
	// Traverse down the Merkle tree to find the leaf at the given index.
	for currentDepth > 0 {
		// Resolve the preimage to get left and right children.
		result, err := preimageResolver.ResolveTypedPreimage(arbutil.Keccak256PreimageType, currentHash)
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
	// At this point, currentHash should be the hash of the object.
	objectHashBytes, err := preimageResolver.ResolveTypedPreimage(arbutil.Keccak256PreimageType, currentHash)
	if err != nil {
		return nil, err
	}
	objectBytes, err := preimageResolver.ResolveTypedPreimage(arbutil.Keccak256PreimageType, common.BytesToHash(objectHashBytes))
	if err != nil {
		return nil, err
	}
	object := new(T)
	if err = rlp.Decode(bytes.NewBuffer(objectBytes), &object); err != nil {
		return nil, fmt.Errorf("failed to decode merkle object at index %d: %w", originalMsgIndex, err)
	}
	return object, nil
}

func NextPowerOfTwo(n uint64) uint64 {
	if n == 0 {
		return 1
	}
	if n&(n-1) == 0 {
		return n
	}
	return 1 << bits.Len64(n)
}
