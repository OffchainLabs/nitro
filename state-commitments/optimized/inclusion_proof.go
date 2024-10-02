package optimized

import (
	"errors"
	"math"

	"github.com/ethereum/go-ethereum/common"
)

// Computes the Merkle proof for a leaf at a given index.
// It uses the last leaf to pad the tree up to the 'virtual' size if needed.
func (h *HistoryCommitter) computeMerkleProof(leafIndex uint64, leaves []common.Hash, virtual uint64) ([]common.Hash, error) {
	if len(leaves) == 0 {
		return nil, nil
	}
	ok, err := isPowTwo(virtual)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("virtual size must be a power of 2")
	}
	if leafIndex >= uint64(len(leaves)) {
		return nil, errors.New("leaf index out of bounds")
	}
	if virtual < uint64(len(leaves)) {
		return nil, errors.New("virtual size must be greater than or equal to the number of leaves")
	}
	numRealLeaves := uint64(len(leaves))
	lastLeaf, err := h.hash(leaves[numRealLeaves-1][:])
	if err != nil {
		return nil, err
	}
	depth := int(math.Ceil(math.Log2(float64(virtual))))

	// Precompute virtual hashes
	virtualHashes, err := h.precomputeRepeatedHashes(&lastLeaf, depth)
	if err != nil {
		return nil, err
	}
	var proof []common.Hash
	for level := 0; level < depth; level++ {
		nodeIndex := leafIndex >> level
		siblingHash, exists, err := h.computeSiblingHash(nodeIndex, uint64(level), numRealLeaves, virtual, leaves, virtualHashes)
		if err != nil {
			return nil, err
		}
		if exists {
			proof = append(proof, siblingHash)
		}
	}
	return proof, nil
}

// Computes the hash of a node's sibling at a given index and level.
func (h *HistoryCommitter) computeSiblingHash(
	nodeIndex uint64,
	level uint64,
	N uint64,
	virtual uint64,
	hLeaves []common.Hash,
	hNHashes []common.Hash,
) (common.Hash, bool, error) {
	siblingIndex := nodeIndex ^ 1
	// Essentially ceil(virtual / (2 ** level))
	numNodes := (virtual + (1 << level) - 1) / (1 << level)
	if siblingIndex >= numNodes {
		// No sibling exists, so use a zero hash.
		return common.Hash{}, false, nil
	} else if siblingIndex >= paddingStartIndexAtLevel(N, level) {
		return hNHashes[level], true, nil
	} else {
		siblingHash, err := h.computeNodeHash(siblingIndex, level, N, hLeaves, hNHashes)
		if err != nil {
			return emptyHash, false, err
		}
		return siblingHash, true, nil
	}
}

// Recursively computes the hash of a node at a given index and level.
func (h *HistoryCommitter) computeNodeHash(
	nodeIndex uint64, level uint64, numRealLeaves uint64, leaves []common.Hash, virtualHashes []common.Hash,
) (common.Hash, error) {
	if level == 0 {
		if nodeIndex >= numRealLeaves {
			// Node is in padding (the virtual segment of the tree).
			return virtualHashes[0], nil
		} else {
			return h.hash(leaves[nodeIndex][:])
		}
	} else {
		if nodeIndex >= paddingStartIndexAtLevel(numRealLeaves, level) {
			return virtualHashes[level], nil
		} else {
			leftChild, err := h.computeNodeHash(2*nodeIndex, level-1, numRealLeaves, leaves, virtualHashes)
			if err != nil {
				return emptyHash, err
			}
			rightChild, err := h.computeNodeHash(2*nodeIndex+1, level-1, numRealLeaves, leaves, virtualHashes)
			if err != nil {
				return emptyHash, err
			}
			data := append(leftChild.Bytes(), rightChild.Bytes()...)
			return h.hash(data)
		}
	}
}

// Calculates the index at which padding starts at a given tree level.
func paddingStartIndexAtLevel(N uint64, level uint64) uint64 {
	return N / (1 << level)
}
