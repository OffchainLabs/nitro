package inclusionproofs

import (
	"math"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pkg/errors"
)

var (
	ErrProofTooLong  = errors.New("merkle proof too long")
	ErrInvalidTree   = errors.New("invalid merkle tree")
	ErrInvalidLeaves = errors.New("invalid number of leaves for merkle tree")
)

// CalculateRootFromProof calculates a Merkle root from a Merkle proof, index, and leaf.
func CalculateRootFromProof(proof []common.Hash, index uint64, leaf common.Hash) (common.Hash, error) {
	if len(proof) > 256 {
		return common.Hash{}, ErrProofTooLong
	}
	h := crypto.Keccak256Hash(leaf[:])
	for i := 0; i < len(proof); i++ {
		node := proof[i]
		if index&(1<<i) == 0 {
			h = crypto.Keccak256Hash(h[:], node[:])
		} else {
			h = crypto.Keccak256Hash(node[:], h[:])
		}
	}
	return h, nil
}

// MerkleRoot from a tree.
func MerkleRoot(tree [][]common.Hash) (common.Hash, error) {
	if len(tree) == 0 || len(tree[len(tree)-1]) == 0 {
		return common.Hash{}, ErrInvalidTree
	}
	return tree[len(tree)-1][0], nil
}

// ComputeMerkleTree from a list of hashes. If not a power of two,
// pads with empty [32]byte{} until the length is a power of two.
// Creates a tree where the last level is the root.
// This is inspired by the Sparse Merkle Tree data structure from
// https://github.com/prysmaticlabs/prysm/tree/develop/container/trie/sparse_merkle.go
func ComputeMerkleTree(items []common.Hash) [][]common.Hash {
	leaves := make([]common.Hash, len(items))
	for i, r := range items {
		// Rehash to match the Merkle expansion functions.
		leaves[i] = crypto.Keccak256Hash(r[:])
	}
	for !isPowerOfTwo(uint64(len(leaves))) {
		leaves = append(leaves, common.Hash{})
	}
	height := uint64(math.Log2(float64(len(leaves))))
	layers := make([][]common.Hash, height+1)
	layers[0] = leaves
	for i := uint64(0); i < height; i++ {
		updatedValues := make([]common.Hash, 0)
		for j := 0; j < len(layers[i]); j += 2 {
			hashed := crypto.Keccak256Hash(layers[i][j][:], layers[i][j+1][:])
			updatedValues = append(updatedValues, hashed)
		}
		layers[i+1] = updatedValues
	}
	return layers
}

// GenerateMerkleProof for an index in a Merkle tree.
func GenerateMerkleProof(index uint64, tree [][]common.Hash) ([]common.Hash, error) {
	if len(tree) == 0 {
		return nil, ErrInvalidTree
	}
	proof := make([]common.Hash, len(tree)-1)
	leaves := tree[0]
	if index >= uint64(len(leaves)) {
		return nil, ErrInvalidLeaves
	}
	for i := 0; i < len(tree)-1; i++ {
		subIndex := (index / (1 << i)) ^ 1
		proof[i] = tree[i][subIndex]
	}
	return proof, nil
}

func isPowerOfTwo(x uint64) bool {
	if x == 0 {
		return false
	}
	return x&(x-1) == 0
}
