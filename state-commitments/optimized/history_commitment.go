package optimized

import (
	"errors"
	"fmt"
	"math"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

var emptyHash = common.Hash{}

type Commitment struct {
	Height         uint64
	Merkle         common.Hash
	FirstLeaf      common.Hash
	LastLeafProof  []common.Hash
	FirstLeafProof []common.Hash
	LastLeaf       common.Hash
}

// NewCommitment produces a history commitment from a list of leaves that are virtually padded using
// the last leaf in the list to some virtual length, without making those extra allocations needed to do so.
// Virtual must be >= len(leaves).
func NewCommitment(leaves []common.Hash, virtual uint64) (*Commitment, error) {
	if len(leaves) == 0 {
		return nil, errors.New("must commit to at least one leaf")
	}
	if virtual == 0 {
		return nil, errors.New("virtual size cannot be zero")
	}
	if virtual < uint64(len(leaves)) {
		return nil, errors.New("virtual size must be greater than or equal to the number of leaves")
	}
	comm := NewCommitter()
	firstLeaf := leaves[0]
	lastLeaf := leaves[len(leaves)-1]
	var firstLeafProof, lastLeafProof []common.Hash
	ok, err := isPowTwo(virtual)
	if err != nil {
		return nil, err
	}
	if ok {
		firstLeafProof, err = comm.computeMerkleProof(0, leaves, virtual)
		if err != nil {
			return nil, err
		}
		lastLeafProof, err = comm.computeMerkleProof(virtual-1, leaves, virtual)
		if err != nil {
			return nil, err
		}
	}
	root, err := comm.ComputeRoot(leaves, virtual)
	if err != nil {
		return nil, err
	}
	return &Commitment{
		Height:         virtual - 1,
		Merkle:         root,
		FirstLeaf:      firstLeaf,
		LastLeaf:       lastLeaf,
		FirstLeafProof: firstLeafProof,
		LastLeafProof:  lastLeafProof,
	}, nil
}

type HistoryCommitter struct {
	lastLeafFillers []common.Hash
	keccak          crypto.KeccakState
}

func NewCommitter() *HistoryCommitter {
	return &HistoryCommitter{
		lastLeafFillers: make([]common.Hash, 0),
		keccak:          crypto.NewKeccakState(),
	}
}

func (h *HistoryCommitter) hash(item []byte) (common.Hash, error) {
	defer h.keccak.Reset()
	if _, err := h.keccak.Write(item); err != nil {
		return emptyHash, err
	}
	var result common.Hash
	if _, err := h.keccak.Read(result[:]); err != nil {
		return emptyHash, err
	}
	return result, nil
}

func (h *HistoryCommitter) ComputeRoot(leaves []common.Hash, virtual uint64) (common.Hash, error) {
	if len(leaves) == 0 {
		return emptyHash, nil
	}
	rehashedLeaves := make([]common.Hash, len(leaves))
	for i, leaf := range leaves {
		result, err := h.hash(leaf[:])
		if err != nil {
			return emptyHash, err
		}
		rehashedLeaves[i] = result
	}
	return h.computeVirtualSparseTree(rehashedLeaves, virtual, 0)
}

func (h *HistoryCommitter) GeneratePrefixProof(prefixIndex uint64, leaves []common.Hash, virtual uint64) ([]common.Hash, []common.Hash, error) {
	rehashedLeaves := make([]common.Hash, len(leaves))
	for i, leaf := range leaves {
		result, err := h.hash(leaf[:])
		if err != nil {
			return nil, nil, err
		}
		rehashedLeaves[i] = result
	}
	prefixExpansion, proof, err := h.prefixAndProof(prefixIndex, rehashedLeaves, virtual)
	if err != nil {
		return nil, nil, err
	}
	prefixExpansion = trimTrailingZeroHashes(prefixExpansion)
	proof = trimZeroes(proof)
	return prefixExpansion, proof, nil
}

// computeSparseTree returns the htr of a hashtree with the given leaves and
// limit. Any non-allocated leaf is filled with the passed zeroHash of depth 0.
// Recursively, any non allocated intermediate layer at depth i is filled with
// the passed zeroHash.
// limit is assumed to be a power of two which is higher or equal than the
// length of the leaves.
// fillers is assumed to be precomputed to the necessary limit.
// It is a programming error to call this function with a limit of 0.
//
// Zero allocations
// Computes O(len(leaves)) hashes.
func (h *HistoryCommitter) computeSparseTree(leaves []common.Hash, limit uint64, fillers []common.Hash) (common.Hash, error) {
	if limit == 0 {
		panic("limit must be greater than 0")
	}
	m := len(leaves)
	if m == 0 {
		return emptyHash, nil
	}
	if limit < 2 {
		return leaves[0], nil
	}
	depth := int(math.Log2(float64(limit)))
	for j := 0; j < depth; j++ {
		// Check to ensure we don't access out of bounds.
		for i := 0; i < m/2; i++ {
			if _, err := h.keccak.Write(leaves[2*i][:]); err != nil {
				return emptyHash, err
			}
			if _, err := h.keccak.Write(leaves[2*i+1][:]); err != nil {
				return emptyHash, err
			}
			if _, err := h.keccak.Read(leaves[i][:]); err != nil {
				return emptyHash, err
			}
			h.keccak.Reset()
		}
		if m&1 == 1 {
			// Check to ensure m-1 is a valid index.
			if _, err := h.keccak.Write(leaves[m-1][:]); err != nil {
				return emptyHash, err
			}
			if j < len(fillers) { // Check to prevent index out of range for fillers.
				if _, err := h.keccak.Write(fillers[j][:]); err != nil {
					return emptyHash, err
				}
			} else {
				// Handle the case where j is out of range for fillers.
				return emptyHash, errors.New("insufficient fillers")
			}
			if _, err := h.keccak.Read(leaves[(m-1)/2][:]); err != nil {
				return emptyHash, err
			}
			h.keccak.Reset()
		}
		m = (m + 1) / 2
	}
	return leaves[0], nil
}

// computeVirtualSparseTree returns the htr of a hashtree where the first layer
// is passed as leaves, the completed with the last leaf until it reaches
// virtual and finally completed with zero hashes until it reaches limit.
// limit is assumed to be either 0 or a power of 2 which is greater or equal to
// virtual. If limit is zero it behaves as if it were the smallest power of two
// that is greater or equal than virtual.
//
// The algorithm is split in three different logic parts:
//
//  1. If the virtual length is less than half the limit (this can never happen
//     in the first iteration of the algorithm), then the first half of the tree
//     is computed by recursion and the second half is a zero hash of a given
//     depth.
//  2. If the leaves all fit in the first half, then we can optimize the first
//     half to being a simple sparse tree, just that instead of filling with zero
//     hashes we fill with the precomputed virtual hashes. This is the most common
//     starting scenario. The second part is computed by recursion.
//  3. If the leaves do not fit in the first half, then we can compute the first half of
//     the tree as a normal full hashtree. The second part is computed by recursion.
func (h *HistoryCommitter) computeVirtualSparseTree(leaves []common.Hash, virtual, limit uint64) (common.Hash, error) {
	m := uint64(len(leaves))
	if m == 0 {
		return emptyHash, errors.New("nil leaves")
	}
	if virtual < m {
		return emptyHash, fmt.Errorf("virtual %d should be >= num leaves %d", virtual, m)
	}
	var err error
	if limit == 0 {
		limit = nextPowerOf2(virtual)
		n := 1
		if virtual > m {
			logValue := math.Log2(float64(limit))
			n = int(logValue) + 1
		}
		h.lastLeafFillers, err = h.precomputeRepeatedHashes(&leaves[m-1], n)
		if err != nil {
			return emptyHash, err
		}
	}
	if limit < virtual {
		return emptyHash, fmt.Errorf("limit %d should be >= virtual %d", limit, virtual)
	}
	if limit == 1 {
		return leaves[0], nil
	}
	var left, right common.Hash
	if virtual > limit/2 {
		if m > limit/2 {
			left, err = h.computeSparseTree(leaves[:limit/2], limit/2, nil)
			if err != nil {
				return emptyHash, err
			}
			right, err = h.computeVirtualSparseTree(leaves[limit/2:], virtual-limit/2, limit/2)
			if err != nil {
				return emptyHash, err
			}
		} else {
			left, err = h.computeSparseTree(leaves, limit/2, h.lastLeafFillers)
			if err != nil {
				return emptyHash, err
			}
			if virtual == limit {
				if len(h.lastLeafFillers) > int(math.Log2(float64(limit/2))) {
					right = h.lastLeafFillers[int(math.Log2(float64(limit/2)))]
				} else {
					return emptyHash, errors.New("insufficient lastLeafFillers")
				}
			} else {
				if len(h.lastLeafFillers) > 0 {
					right, err = h.computeVirtualSparseTree([]common.Hash{h.lastLeafFillers[0]}, virtual-limit/2, limit/2)
					if err != nil {
						return emptyHash, err
					}
				} else {
					return emptyHash, errors.New("empty lastLeafFillers")
				}
			}
		}
	} else {
		left, err = h.computeVirtualSparseTree(leaves, virtual, limit/2)
		if err != nil {
			return emptyHash, err
		}
		right = emptyHash
	}
	if _, err = h.keccak.Write(left[:]); err != nil {
		return emptyHash, err
	}
	if _, err = h.keccak.Write(right[:]); err != nil {
		return emptyHash, err
	}
	if _, err = h.keccak.Read(leaves[0][:]); err != nil {
		return emptyHash, err
	}
	h.keccak.Reset()
	return leaves[0], nil
}

func (h *HistoryCommitter) subtreeExpansion(leaves []common.Hash, virtual, limit uint64, stripped bool) (proof []common.Hash, err error) {
	m := uint64(len(leaves))
	if m == 0 {
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
		left, err2 := h.computeSparseTree(leaves, limit, h.lastLeafFillers)
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
	if m > limit/2 {
		left, err2 := h.computeSparseTree(leaves[:limit/2], limit/2, nil)
		if err2 != nil {
			return nil, err2
		}
		proof, err = h.subtreeExpansion(leaves[limit/2:], virtual-limit/2, limit/2, stripped)
		if err != nil {
			return nil, err
		}
		return append(proof, left), nil
	}
	if virtual >= limit/2 {
		left, err2 := h.computeSparseTree(leaves, limit/2, h.lastLeafFillers)
		if err2 != nil {
			return nil, err2
		}
		// Check if h.lastLeafFillers is not empty before accessing its first element
		if len(h.lastLeafFillers) > 0 {
			proof, err = h.subtreeExpansion([]common.Hash{h.lastLeafFillers[0]}, virtual-limit/2, limit/2, stripped)
			if err != nil {
				return nil, err
			}
			return append(proof, left), nil
		} else {
			return nil, errors.New("lastLeafFillers is empty")
		}
	}
	if stripped {
		return h.subtreeExpansion(leaves, virtual, limit/2, stripped)
	}
	expac, err := h.subtreeExpansion(leaves, virtual, limit/2, stripped)
	if err != nil {
		return nil, err
	}
	return append(expac, emptyHash), nil
}

func (h *HistoryCommitter) proof(index uint64, leaves []common.Hash, virtual, limit uint64) (tail []common.Hash, err error) {
	m := uint64(len(leaves))
	if m == 0 {
		return nil, errors.New("empty leaves slice")
	}
	if limit == 0 {
		limit = nextPowerOf2(virtual)
	}
	if limit == 1 {
		// Can only reach this with index == 0
		return
	}
	if index >= limit/2 {
		if m > limit/2 {
			return h.proof(index-limit/2, leaves[limit/2:], virtual-limit/2, limit/2)
		}
		if len(h.lastLeafFillers) > 0 {
			return h.proof(index-limit/2, []common.Hash{h.lastLeafFillers[0]}, virtual-limit/2, limit/2)
		} else {
			return nil, errors.New("lastLeafFillers is empty")
		}
	}
	if m > limit/2 {
		tail, err = h.proof(index, leaves[:limit/2], limit/2, limit/2)
		if err != nil {
			return nil, err
		}
		right, err2 := h.subtreeExpansion(leaves[limit/2:], virtual-limit/2, limit/2, true)
		if err2 != nil {
			return nil, err2
		}
		for i := len(right) - 1; i >= 0; i-- {
			tail = append(tail, right[i])
		}
		return tail, nil
	}
	if virtual > limit/2 {
		tail, err = h.proof(index, leaves, limit/2, limit/2)
		if err != nil {
			return nil, err
		}
		if len(h.lastLeafFillers) > 0 {
			right, err := h.subtreeExpansion([]common.Hash{h.lastLeafFillers[0]}, virtual-limit/2, limit/2, true)
			if err != nil {
				return nil, err
			}
			for i := len(right) - 1; i >= 0; i-- {
				tail = append(tail, right[i])
			}
		} else {
			return nil, errors.New("lastLeafFillers is empty")
		}
		return tail, nil
	}
	return h.proof(index, leaves, virtual, limit/2)
}

func (h *HistoryCommitter) prefixAndProof(index uint64, leaves []common.Hash, virtual uint64) (prefix []common.Hash, tail []common.Hash, err error) {
	m := uint64(len(leaves))
	if m == 0 {
		return nil, nil, errors.New("nil leaves")
	}
	if virtual == 0 {
		return nil, nil, errors.New("virtual size cannot be zero")
	}
	if m > virtual {
		return nil, nil, fmt.Errorf("num leaves %d should be <= virtual %d", m, virtual)
	}
	if index+1 > virtual {
		return nil, nil, fmt.Errorf("index %d + 1 should be <= virtual %d", index, virtual)
	}
	logVirtual := int(math.Log2(float64(virtual)) + 1)
	h.lastLeafFillers, err = h.precomputeRepeatedHashes(&leaves[m-1], logVirtual)
	if err != nil {
		return nil, nil, err
	}

	if index+1 > m {
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

// precomputeRepeatedHashes returns a slice where built recursively as
// ret[0] = the passed in leaf
// ret[i+1] = Hash(ret[i] + ret[i])
// Allocates n hashes
// Computes n-1 hashes
// Copies 1 hash
func (h *HistoryCommitter) precomputeRepeatedHashes(leaf *common.Hash, n int) ([]common.Hash, error) {
	if leaf == nil {
		return nil, errors.New("nil leaf pointer")
	}
	if len(h.lastLeafFillers) > 0 && h.lastLeafFillers[0] == *leaf && len(h.lastLeafFillers) >= n {
		return h.lastLeafFillers, nil
	}
	if n < 0 {
		return nil, fmt.Errorf("invalid n: %d, must be non-negative", n)
	}
	ret := make([]common.Hash, n)
	copy(ret[0][:], (*leaf)[:])
	for i := 1; i < n; i++ {
		if _, err := h.keccak.Write(ret[i-1][:]); err != nil {
			return nil, fmt.Errorf("keccak write error: %w", err)
		}
		if _, err := h.keccak.Write(ret[i-1][:]); err != nil {
			return nil, fmt.Errorf("keccak write error: %w", err)
		}
		if _, err := h.keccak.Read(ret[i][:]); err != nil {
			return nil, fmt.Errorf("keccak read error: %w", err)
		}
		h.keccak.Reset()
	}
	return ret, nil
}

func nextPowerOf2(n uint64) uint64 {
	if n == 0 {
		return 1
	}
	n--         // Decrement n to handle the case where n is already a power of 2
	n |= n >> 1 // Propagate the highest bit set
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	n |= n >> 32
	return n + 1 // Increment n to get the next power of 2
}

func trimTrailingZeroHashes(hashes []common.Hash) []common.Hash {
	// Start from the end of the slice
	for i := len(hashes) - 1; i >= 0; i-- {
		// If we find a non-zero hash, return the slice up to and including this element
		if hashes[i] != (common.Hash{}) {
			return hashes[:i+1]
		}
	}
	// If all elements are zero, return an empty slice
	return []common.Hash{}
}

func trimZeroes(hashes []common.Hash) []common.Hash {
	newHashes := make([]common.Hash, 0, len(hashes))
	for _, h := range hashes {
		if h == (common.Hash{}) {
			continue
		}
		newHashes = append(newHashes, h)
	}
	return newHashes
}

func isPowTwo(n uint64) (bool, error) {
	if n == 0 {
		return false, errors.New("n must be non-zero")
	}
	return (n & (n - 1)) == 0, nil
}
