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
func getChildrenHashes(preimage []byte) (leftChild, rightChild common.Hash, err error) {
	leftChild = common.BytesToHash(preimage[:32])
	rightChild = common.BytesToHash(preimage[32:])
	return leftChild, rightChild, nil
}

// MerkleTreeContent recursively walks down the Merkle tree and collects leaf node data.
func MerkleTreeContent(oracle func(bytes32) ([]byte, error), rootHash common.Hash) ([][]byte, error) {
	stack := []common.Hash{rootHash}
	var data [][]byte

	for len(stack) > 0 {
		currentHash := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		preimage, err := oracle(currentHash)
		if err != nil {
			return nil, err
		}

		if preimage[0] == leafPrefix[0] {
			data = append(data, preimage[1:])
		} else {
			leftChildHash, rightChildHash, err := getChildrenHashes(preimage[1:])
			if err != nil {
				return nil, err
			}
			stack = append(stack, rightChildHash)
			stack = append(stack, leftChildHash)
		}
	}

	return data, nil
}
