package tree

import (
	"math/bits"

	"github.com/ethereum/go-ethereum/common"
)

type bytes32 = common.Hash

// HashFromByteSlices computes a Merkle tree where the leaves are the byte slice,
// in the provided order. It follows RFC-6962.
func HashFromByteSlices(record func(bytes32, []byte), items [][]byte) []byte {
	switch len(items) {
	case 0:
		emptyHash := emptyHash()
		record(common.BytesToHash(emptyHash), []byte{})
		return emptyHash
	case 1:
		return leafHash(record, items[0])
	default:
		k := getSplitPoint(int64(len(items)))
		left := HashFromByteSlices(record, items[:k])
		right := HashFromByteSlices(record, items[k:])
		return innerHash(record, left, right)
	}
}

// getSplitPoint returns the largest power of 2 less than length
func getSplitPoint(length int64) int64 {
	if length < 1 {
		panic("Trying to split a tree with size < 1")
	}
	uLength := uint(length)
	bitlen := bits.Len(uLength)
	k := int64(1 << uint(bitlen-1))
	if k == length {
		k >>= 1
	}
	return k
}

// getChildrenHashes splits the preimage into the hashes of the left and right children.
func getChildrenHashes(oracle func(bytes32) ([]byte, error), preimage []byte) (leftChild, rightChild common.Hash, err error) {
	leftChild = common.BytesToHash(preimage[:32])
	rightChild = common.BytesToHash(preimage[32:])
	return leftChild, rightChild, nil
}

// walkMerkleTree recursively walks down the Merkle tree and collects leaf node data.
func MerkleTreeContent(oracle func(bytes32) ([]byte, error), rootHash common.Hash) ([][]byte, error) {
	preimage, err := oracle(rootHash)
	if err != nil {
		return nil, err
	}

	if preimage[0] == leafPrefix[0] {
		return [][]byte{preimage[1:]}, nil
	}

	leftChildHash, rightChildHash, err := getChildrenHashes(oracle, preimage[1:])
	if err != nil {
		return nil, err
	}
	leftData, err := MerkleTreeContent(oracle, leftChildHash)
	if err != nil {
		return nil, err
	}
	rightData, err := MerkleTreeContent(oracle, rightChildHash)
	if err != nil {
		return nil, err
	}

	// Combine the data from the left and right subtrees.
	return append(leftData, rightData...), nil
}
