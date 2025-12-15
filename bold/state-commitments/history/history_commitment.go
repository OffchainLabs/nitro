// Copyright 2023-2024, Offchain Labs, Inc.
// For license information, see:
// https://github.com/offchainlabs/nitro/blob/master/LICENSE.md

// Package history provides functions for computing merkle tree roots
// and proofs needed for the BoLD protocol's history commitments.
//
// Throughout this package, the following terms are used:
//
//   - leaf: a leaf node in a merkle tree, which is a hash of some data.
//   - virtual: the length of the desired number of leaf nodes. In the BoLD
//     protocol, it is important that all history commitments which for a given
//     challenge edge have the same length, even if the participants disagree
//     about the number of blocks or steps to which they are committing. To
//     solve this, history commitments must have fixed lengths at different
//     challenge levels. Callers only need to provide the leaves they to which
//     they commit, and the virtual length. The last leaf in the list is used
//     to pad the tree to the virtual length.
//   - limit: the length of the leaves that would be in a complete subtree
//     of the depth required to hold the virtual leaves in a tree (or subtree)
//   - pure tree: a tree where len(leaves) == virtual
//   - complete tree: a tree where the number of leaves is a power of 2
//   - complete virtual tree: a tree where the number of leaves including the
//     virtual padding is a power of 2
//   - partial tree: a tree where the number of leaves is not a power of 2
//   - partial virtual tree: a tree where the number of leaves including the
//     virtual padding is not a power of 2
//   - empty hash: common.Hash{}
//     Any time the root of a partial tree (either virtual or pure) is computed,
//     the sibling node of the last node in a layer may be missing. In this case
//     an empty hash (common.Hash{}) is used as the sibling node.
//     Note: This is not the same as padding the leaves of the tree with
//     common.Hash{} values. If that approach were taken, then the higher-level
//     layers would contain the hash of the empty hash, or the hash of multiple
//     empty hashes. This would be less efficient to calculate, and would not
//     change expressiveness or security of the data structure, but it would
//     produce a different root hash.
//   - virtual node: a node in a virtual tree which is not one of the real
//     leaves and not computed from the data in the real leaves.
package history

import (
	"errors"
	"fmt"

	"github.com/ccoveille/go-safecast"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/offchainlabs/nitro/bold/math"
)

var (
	emptyHash    = common.Hash{}
	emptyHistory = History{}
)

// History represents a history commitment in the protocol.
type History struct {
	Height        uint64
	Merkle        common.Hash
	FirstLeaf     common.Hash
	LastLeaf      common.Hash
	LastLeafProof []common.Hash
}

// treePosition tracks the current position in the merkle tree.
type treePosition struct {
	// layer is the layer of the tree.
	layer uint64
	// index is the index of the leaf in this layer of the tree.
	index uint64
}

type historyCommitter struct {
	fillers        []common.Hash
	cursor         treePosition
	lastLeafProver *lastLeafProver
}

func newCommitter() *historyCommitter {
	return &historyCommitter{
		fillers: make([]common.Hash, 0),
	}
}

// soughtHash holds a pointer to the hash and whether it has been found.
//
// Without this type, it would be impossible to distinguish between a hash which
// has not been found and a hash which is the value of common.Hash{}.
// That's because the lastLeafProver's positions map is initialized with pointers
// to common.Hash{} values in a pre-allocated slice.
type soughtHash struct {
	found bool
	hash  *common.Hash
}

// lastLeafProver finds the siblings needed to produce a merkle inclusion
// proof for the last leaf in a virtual merkle tree.
//
// The prover maintains a map of treePositions where sibling nodes live
// and fills them in as the historyCommitter calculates them.
type lastLeafProver struct {
	positions map[treePosition]*soughtHash
	proof     []common.Hash
}

func newLastLeafProver(virtual uint64) (*lastLeafProver, error) {
	positions, err := lastLeafProofPositions(virtual)
	if err != nil {
		return nil, err
	}
	posMap := make(map[treePosition]*soughtHash, len(positions))
	proof := make([]common.Hash, len(positions))
	for i, pos := range positions {
		posMap[pos] = &soughtHash{false, &proof[i]}
	}
	return &lastLeafProver{
		positions: posMap,
		proof:     proof,
	}, nil
}

// handle filters the hashes found while computing the merkle root looking for
// the sibling nodes needed to produce the merkle inclusion proof, and fills
// them in the proof slice.
func (p *lastLeafProver) handle(hash common.Hash, pos treePosition) {
	if sibling, ok := p.positions[pos]; ok {
		sibling.found = true
		*sibling.hash = hash
	}
}

// handle is called each time a hash is computed in the merkle tree.
//
// The cursor is kept in sync with tree traversal. The implementation of
// handle can therefore assume that the cursor is pointing to the node which
// has the value of the hash.
func (h *historyCommitter) handle(hash common.Hash) {
	if h.lastLeafProver != nil {
		h.lastLeafProver.handle(hash, h.cursor)
	}
}

// proof returns the merkle inclusion proof for the last leaf in a virtual tree.
//
// If the proof is not complete (i.e. some sibling nodes are missing), the
// sibling nodes are filled in with the fillers.
//
// The reason this works, is that the only nodes which are not visited when
// computing the merkle root are those which are in some complete virtual
// subtree.
func (h *historyCommitter) lastLeafProof() []common.Hash {
	for pos, sibling := range h.lastLeafProver.positions {
		if !sibling.found {
			*h.lastLeafProver.positions[pos].hash = h.fillers[pos.layer]
		}
	}
	if len(h.lastLeafProver.proof) == 0 {
		return nil
	}
	return h.lastLeafProver.proof
}

// NewCommitment produces a history commitment from a list of real leaves that
// are virtually padded using the last leaf in the list to some virtual length.
//
// Virtual must be >= len(leaves).
func NewCommitment(leaves []common.Hash, virtual uint64) (History, error) {
	if len(leaves) == 0 {
		return emptyHistory, errors.New("must commit to at least one leaf")
	}
	if virtual < uint64(len(leaves)) {
		return emptyHistory, errors.New("virtual size must be >= len(leaves)")
	}
	comm := newCommitter()
	firstLeaf := leaves[0]
	lastLeaf := leaves[len(leaves)-1]
	prover, err := newLastLeafProver(virtual)
	if err != nil {
		return emptyHistory, err
	}
	comm.lastLeafProver = prover
	root, err := comm.computeRoot(leaves, virtual)
	if err != nil {
		return emptyHistory, err
	}
	lastLeafProof := comm.lastLeafProof()
	return History{
		// Height is the relative height of the history commitment.
		// It's the index of the last leaf in the tree.
		Height:        virtual - 1,
		Merkle:        root,
		FirstLeaf:     firstLeaf,
		LastLeaf:      lastLeaf,
		LastLeafProof: lastLeafProof,
	}, nil
}

// ComputeRoot computes the merkle root of a virtual merkle tree.
func ComputeRoot(leaves []common.Hash, virtual uint64) (common.Hash, error) {
	comm := newCommitter()
	return comm.computeRoot(leaves, virtual)
}

// GeneratePrefixProof generates a prefix proof for a given prefix index.
func GeneratePrefixProof(prefixIndex uint64, leaves []common.Hash, virtual uint64) ([]common.Hash, []common.Hash, error) {
	comm := newCommitter()
	return comm.generatePrefixProof(prefixIndex, leaves, virtual)
}

// computeRoot computes the merkle root of a virtual merkle tree.
func (h *historyCommitter) computeRoot(leaves []common.Hash, virtual uint64) (common.Hash, error) {
	lvLen := uint64(len(leaves))
	if lvLen == 0 {
		return emptyHash, nil
	}
	hashed := h.hashLeaves(leaves)
	limit := nextPowerOf2(virtual)
	depth, err := safecast.ToUint(math.Log2Floor(limit))
	if err != nil {
		return emptyHash, err
	}
	n, err := safecast.ToUint(math.Log2Ceil(virtual))
	if err != nil {
		return emptyHash, err
	}
	n = max(n, 1)
	if err := h.populateFillers(&hashed[lvLen-1], n); err != nil {
		return emptyHash, err
	}
	h.cursor = treePosition{layer: uint64(depth), index: 0}
	return h.partialRoot(hashed, virtual, limit)
}

// generatePrefixProof generates a prefix proof for a given prefix index.
//
// A prefix proof consists of the data needed to prove that a merkle root
// created from the leaves upto the prefix index represents a merkle tree which
// spans a specific prefix of the virtual merkle tree.
func (h *historyCommitter) generatePrefixProof(prefixIndex uint64, leaves []common.Hash, virtual uint64) ([]common.Hash, []common.Hash, error) {
	hashed := h.hashLeaves(leaves)
	prefixExpansion, proof, err := h.prefixAndProof(prefixIndex, hashed, virtual)
	if err != nil {
		return nil, nil, err
	}
	prefixExpansion = trimTrailingEmptyHashes(prefixExpansion)
	proof = filterEmptyHashes(proof)
	return prefixExpansion, proof, nil
}

// hashLeaves returns a slice of hashes of the leaves
func (h *historyCommitter) hashLeaves(leaves []common.Hash) []common.Hash {
	hashedLeaves := make([]common.Hash, len(leaves))
	for i := range leaves {
		hashedLeaves[i] = crypto.Keccak256Hash(leaves[i][:])
	}
	return hashedLeaves
}

// partialRoot returns the merkle root of a possibly partial hashtree where the
// first layer is passed as leaves, then padded by repeating the last leaf
// until it reaches virtual and terminated with a single common.Hash{}.
//
// limit is a power of 2 which is greater or equal to virtual, and defines how
// deep the complete tree analogous to this partial one would be.
//
// Implementation note: The historyCommitter's fillers member must be populated
// correctly before calling this method. There must be at least
// Log2FCeil(virtual) filler nodes to properly pad each layer of the tree if it
// is a partial virtual tree.
//
// The algorithm is split in three different logical cases:
//
//  1. If the virtual length is less than or equal to half the limit (this can
//     never happen in the first iteration of the algorithm), the left half of
//     the tree is computed by recursion and the right half is an empty hash.
//  2. If the leaves all fit in the left half, then both halves of the tree are
//     computed by recursion. This is the most common starting scenario.
//     There is a special case when the virtual length is equal to the limit,
//     and the right half is a complete virtual tree. In this case, the right
//     subtree is just a lookup in the precomputed fillers.
//  3. If the leaves do not fit in the left half, then both halves are computed
//     by recursion.
func (h *historyCommitter) partialRoot(leaves []common.Hash, virtual, limit uint64) (common.Hash, error) {
	if len(leaves) == 0 {
		return emptyHash, errors.New("nil leaves")
	}
	lvLen := uint64(len(leaves))
	if virtual < lvLen {
		return emptyHash, fmt.Errorf("virtual %d should be >= num leaves %d", virtual, lvLen)
	}
	if limit < virtual {
		return emptyHash, fmt.Errorf("limit %d should be >= virtual %d", limit, virtual)
	}
	minFillers := math.Log2Ceil(virtual)
	if len(h.fillers) < minFillers {
		return emptyHash, fmt.Errorf("insufficient fillers, want %d, got %d", minFillers, len(h.fillers))
	}
	if limit == 1 {
		h.handle(leaves[0])
		return leaves[0], nil
	}

	h.cursor.layer--
	var left, right common.Hash
	var err error
	mid := limit / 2

	// Deal with the left child first
	h.cursor.index *= 2
	var lLeaves []common.Hash
	var lVirtual uint64
	if virtual > mid {
		// Case 2 or 3: A complete subtree can be computed
		lVirtual = mid
		if lvLen > mid {
			// Case 3: A complete pure subtree can be computed
			lLeaves = leaves[:mid]
		} else {
			// Case 2: A complete virtual subtree can be computed
			lLeaves = leaves
		}
	} else {
		// Case 1: A partial virtual tree can be computed
		lLeaves = leaves
		lVirtual = virtual
	}
	left, err = h.partialRoot(lLeaves, lVirtual, mid)
	if err != nil {
		return emptyHash, err
	}

	// Deal with the right child
	h.cursor.index++
	if virtual > mid {
		// Case 2 or 3: The virtual size is greater than half the limit
		if lvLen <= mid && virtual == limit {
			// This is a special case of 2 where the entire right subtree is
			// made purely of virtual nodes, and it is a complete tree.
			// So, the root of the subtree will be the precomputed filler
			// at the current layer.
			right = h.fillers[math.Log2Floor(mid)]
			h.handle(right)
		} else {
			var rLeaves []common.Hash
			if lvLen > mid {
				// Case 3: The leaves do not fit in the first half
				rLeaves = leaves[mid:]
			} else {
				// Case 2: The leaves fit in the first half
				rLeaves = []common.Hash{h.fillers[0]}
			}
			right, err = h.partialRoot(rLeaves, virtual-mid, mid)
			if err != nil {
				return emptyHash, err
			}
		}
	} else {
		// Case 1: The virtual size is less than half the limit
		right = emptyHash
		h.handle(right)
	}

	leaves[0] = crypto.Keccak256Hash(left[:], right[:])

	// Restore the cursor layer to the state for this level of recursion
	h.cursor.index /= 2
	h.cursor.layer++
	h.handle(leaves[0])

	return leaves[0], nil
}

func (h *historyCommitter) subtreeExpansion(leaves []common.Hash, virtual, limit uint64, stripped bool) (proof []common.Hash, err error) {
	lvLen := uint64(len(leaves))
	if lvLen == 0 {
		return make([]common.Hash, 0), nil
	}
	if virtual == 0 {
		for i := limit; i > 1; i /= 2 {
			proof = append(proof, emptyHash)
		}
		return
	}
	if limit == 0 {
		limit = nextPowerOf2(virtual)
	}
	if limit == virtual {
		left, err2 := h.partialRoot(leaves, limit, limit)
		if err2 != nil {
			return nil, err2
		}
		if !stripped {
			for i := limit; i > 1; i /= 2 {
				proof = append(proof, emptyHash)
			}
		}
		return append(proof, left), nil
	}
	mid := limit / 2
	if lvLen > mid {
		left, err2 := h.partialRoot(leaves[:mid], mid, mid)
		if err2 != nil {
			return nil, err2
		}
		proof, err = h.subtreeExpansion(leaves[mid:], virtual-mid, mid, stripped)
		if err != nil {
			return nil, err
		}
		return append(proof, left), nil
	}
	if virtual >= mid {
		left, err2 := h.partialRoot(leaves, mid, mid)
		if err2 != nil {
			return nil, err2
		}
		if len(h.fillers) == 0 {
			return nil, errors.New("fillers is empty")
		}
		proof, err = h.subtreeExpansion([]common.Hash{h.fillers[0]}, virtual-mid, mid, stripped)
		if err != nil {
			return nil, err
		}
		return append(proof, left), nil
	}
	if stripped {
		return h.subtreeExpansion(leaves, virtual, mid, stripped)
	}
	expac, err := h.subtreeExpansion(leaves, virtual, mid, stripped)
	if err != nil {
		return nil, err
	}
	return append(expac, emptyHash), nil
}

func (h *historyCommitter) proof(index uint64, leaves []common.Hash, virtual, limit uint64) (tail []common.Hash, err error) {
	lvLen := uint64(len(leaves))
	if lvLen == 0 {
		return nil, errors.New("empty leaves slice")
	}
	if limit == 0 {
		limit = nextPowerOf2(virtual)
	}
	if limit == 1 {
		// Can only reach this with index == 0
		return
	}
	mid := limit / 2
	if index >= mid {
		if lvLen > mid {
			return h.proof(index-mid, leaves[mid:], virtual-mid, mid)
		}
		if len(h.fillers) == 0 {
			return nil, errors.New("fillers is empty")
		}
		return h.proof(index-mid, []common.Hash{h.fillers[0]}, virtual-mid, mid)
	}
	if lvLen > mid {
		tail, err = h.proof(index, leaves[:mid], mid, mid)
		if err != nil {
			return nil, err
		}
		right, err2 := h.subtreeExpansion(leaves[mid:], virtual-mid, mid, true)
		if err2 != nil {
			return nil, err2
		}
		for i := len(right) - 1; i >= 0; i-- {
			tail = append(tail, right[i])
		}
		return tail, nil
	}
	if virtual > mid {
		tail, err = h.proof(index, leaves, mid, mid)
		if err != nil {
			return nil, err
		}
		if len(h.fillers) == 0 {
			return nil, errors.New("fillers is empty")
		}
		right, err := h.subtreeExpansion([]common.Hash{h.fillers[0]}, virtual-mid, mid, true)
		if err != nil {
			return nil, err
		}
		for i := len(right) - 1; i >= 0; i-- {
			tail = append(tail, right[i])
		}
		return tail, nil
	}
	return h.proof(index, leaves, virtual, mid)
}

func (h *historyCommitter) prefixAndProof(index uint64, leaves []common.Hash, virtual uint64) (prefix []common.Hash, tail []common.Hash, err error) {
	lvLen := uint64(len(leaves))
	if lvLen == 0 {
		return nil, nil, errors.New("nil leaves")
	}
	if virtual == 0 {
		return nil, nil, errors.New("virtual size cannot be zero")
	}
	if lvLen > virtual {
		return nil, nil, fmt.Errorf("num leaves %d should be <= virtual %d", lvLen, virtual)
	}
	if index+1 > virtual {
		return nil, nil, fmt.Errorf("index %d + 1 should be <= virtual %d", index, virtual)
	}
	logVirtual, err := safecast.ToUint(math.Log2Floor(virtual) + 1)
	if err != nil {
		return nil, nil, err
	}
	if err = h.populateFillers(&leaves[lvLen-1], logVirtual); err != nil {
		return nil, nil, err
	}

	if index+1 > lvLen {
		prefix, err = h.subtreeExpansion(leaves, index+1, 0, false)
	} else {
		prefix, err = h.subtreeExpansion(leaves[:index+1], index+1, 0, false)
	}
	if err != nil {
		return nil, nil, err
	}
	tail, err = h.proof(index, leaves, virtual, 0)
	return
}

// populateFillers returns a slice built recursively as
// ret[0] = the passed in leaf
// ret[i+1] = Hash(ret[i] + ret[i])
//
// Allocates n hashes
// Computes n-1 hashes
// Copies 1 hash
func (h *historyCommitter) populateFillers(leaf *common.Hash, n uint) error {
	if leaf == nil {
		return errors.New("nil leaf pointer")
	}
	h.fillers = make([]common.Hash, n)
	copy(h.fillers[0][:], (*leaf)[:])
	for i := uint(1); i < n; i++ {
		h.fillers[i] = crypto.Keccak256Hash(h.fillers[i-1][:], h.fillers[i-1][:])
	}
	return nil
}

// lastLeafProofPositions returns the positions in a virtual merkle tree
// of the sibling nodes that need to be hashed with the last leaf at each
// layer to compute the root of the tree.
func lastLeafProofPositions(virtual uint64) ([]treePosition, error) {
	if virtual == 0 {
		return nil, errors.New("virtual size cannot be zero")
	}
	if virtual == 1 {
		return []treePosition{}, nil
	}
	limit := nextPowerOf2(virtual)
	depth := math.Log2Floor(limit)
	positions := make([]treePosition, depth)
	idx := uint64(virtual) - 1
	for l := range positions {
		lU64, err := safecast.ToUint64(l)
		if err != nil {
			return nil, err
		}
		positions[l] = sibling(idx, lU64)
		idx = parent(idx)
	}
	return positions, nil
}

// sibling returns the position of the sibling of the node at the given layer
func sibling(index, layer uint64) treePosition {
	return treePosition{layer: layer, index: index ^ 1}
}

// parent returns the index of the parent of the node in the next higher layer
func parent(index uint64) uint64 {
	return index >> 1
}

func nextPowerOf2(n uint64) uint64 {
	if n == 0 {
		return 1
	}
	n--         // Decrement n to handle the case where n is a power of 2
	n |= n >> 1 // Propagate the highest bit set
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	n |= n >> 32
	return n + 1 // Increment n to get the next power of 2
}

func trimTrailingEmptyHashes(hashes []common.Hash) []common.Hash {
	// Start from the end of the slice
	for i := len(hashes) - 1; i >= 0; i-- {
		if hashes[i] != emptyHash {
			return hashes[:i+1]
		}
	}
	// If all elements are zero, return an empty slice
	return []common.Hash{}
}

func filterEmptyHashes(hashes []common.Hash) []common.Hash {
	newHashes := make([]common.Hash, 0, len(hashes))
	for _, h := range hashes {
		if h == emptyHash {
			continue
		}
		newHashes = append(newHashes, h)
	}
	return newHashes
}
