package util

import (
	"errors"

	"bytes"
	"fmt"
	"github.com/ethereum/go-ethereum/crypto"
)

// SparseMerkleTrie implements a sparse, general purpose Merkle trie.
type SparseMerkleTrie struct {
	depth         uint
	branches      [][][]byte
	originalItems [][]byte // list of provided items before hashing them into leaves.
}

// GenerateTrieFromItems constructs a Merkle trie from a sequence of byte slices.
func GenerateTrieFromItems(items [][]byte, depth uint64) (*SparseMerkleTrie, error) {
	if len(items) == 0 {
		return nil, errors.New("no items provided to generate Merkle trie")
	}
	leaves := items
	layers := make([][][]byte, depth+1)
	transformedLeaves := make([][]byte, len(leaves))
	for i := range leaves {
		arr := toBytes32(leaves[i])
		transformedLeaves[i] = arr[:]
	}
	layers[0] = transformedLeaves
	for i := uint64(0); i < depth; i++ {
		if len(layers[i])%2 == 1 {
			layers[i] = append(layers[i], ZeroHashes[i][:])
		}
		updatedValues := make([][]byte, 0)
		for j := 0; j < len(layers[i]); j += 2 {
			concat := crypto.Keccak256Hash(append(layers[i][j], layers[i][j+1]...))
			updatedValues = append(updatedValues, concat[:])
		}
		layers[i+1] = updatedValues
	}
	return &SparseMerkleTrie{
		branches:      layers,
		originalItems: items,
		depth:         uint(depth),
	}, nil
}

func (m *SparseMerkleTrie) Root() []byte {
	return m.branches[len(m.branches)-1][0]
}

// MerkleProof computes a proof from a trie's branches using a Merkle index.
func (m *SparseMerkleTrie) MerkleProof(index int) ([][]byte, error) {
	if index < 0 {
		return nil, fmt.Errorf("merkle index is negative: %d", index)
	}
	leaves := m.branches[0]
	if index >= len(leaves) {
		return nil, fmt.Errorf("merkle index out of range in trie, max range: %d, received: %d", len(leaves), index)
	}
	merkleIndex := uint(index)
	proof := make([][]byte, m.depth+1)
	for i := uint(0); i < m.depth; i++ {
		subIndex := (merkleIndex / (1 << i)) ^ 1
		if subIndex < uint(len(m.branches[i])) {
			item := toBytes32(m.branches[i][subIndex])
			proof[i] = item[:]
		} else {
			proof[i] = ZeroHashes[i][:]
		}
	}
	return proof, nil
}

// VerifyMerkleProofWithDepth verifies a Merkle branch against a root of a trie.
func VerifyMerkleProofWithDepth(root, item []byte, merkleIndex uint64, proof [][]byte, depth uint64) bool {
	if uint64(len(proof)) != depth+1 {
		return false
	}
	if depth >= 64 {
		return false // PowerOf2 would overflow.
	}
	node := toBytes32(item)
	for i := uint64(0); i <= depth; i++ {
		if (merkleIndex / powerOf2(i) % 2) != 0 {
			node = crypto.Keccak256Hash(append(proof[i], node[:]...))
		} else {
			node = crypto.Keccak256Hash(append(node[:], proof[i]...))
		}
	}

	return bytes.Equal(root, node[:])
}

// VerifyMerkleProof given a trie root, a leaf, the generalized merkle index
// of the leaf in the trie, and the proof itself.
func VerifyMerkleProof(root, item []byte, merkleIndex uint64, proof [][]byte) bool {
	if len(proof) == 0 {
		return false
	}
	return VerifyMerkleProofWithDepth(root, item, merkleIndex, proof, uint64(len(proof)-1))
}

func toBytes32(b []byte) [32]byte {
	var target [32]byte
	copy(target[:], b)
	return target
}

func powerOf2(n uint64) uint64 {
	if n >= 64 {
		panic("integer overflow")
	}
	return 1 << n
}
