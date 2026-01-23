package melreplay

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/offchainlabs/nitro/arbutil"
)

// Fetches a specific object at index from a block's Receipt/Tx trie by navigating its
// Merkle Patricia Trie structure. It uses the preimage resolver to fetch preimages
// of trie nodes as needed, and determines how to navigate depending on the structure of the trie nodes.
func fetchObjectFromTrie[T any](root common.Hash, index uint, preimageResolver PreimageResolver) (*T, error) {
	var empty *T
	currentNodeHash := root
	currentPath := []byte{} // Track nibbles consumed so far.
	receiptKey, err := rlp.EncodeToBytes(index)
	if err != nil {
		return empty, err
	}
	targetNibbles := keyToNibbles(receiptKey)
	for {
		nodeData, err := preimageResolver.ResolveTypedPreimage(arbutil.Keccak256PreimageType, currentNodeHash)
		if err != nil {
			return empty, err
		}
		var node []any
		if err = rlp.DecodeBytes(nodeData, &node); err != nil {
			return empty, fmt.Errorf("failed to decode RLP node: %w", err)
		}
		switch len(node) {
		case 17:
			// We hit a branch node, which has 16 children and a value.
			if len(currentPath) == len(targetNibbles) {
				// A branch node's 17th item could be the value, so we check if it contains the receipt.
				if valueBytes, ok := node[16].([]byte); ok && len(valueBytes) > 0 {
					// This branch node has the actual value as the last item, so we decode the receipt
					return decodeBinary[T](valueBytes)
				}
				return empty, fmt.Errorf("no receipt found at target key")
			}
			// Get the next nibble to follow.
			targetNibble := targetNibbles[len(currentPath)]
			childData, ok := node[targetNibble].([]byte)
			if !ok || len(childData) == 0 {
				return empty, fmt.Errorf("no child at nibble %d", targetNibble)
			}
			// Move to the child node, which is the next hash we have to navigate.
			currentNodeHash = common.BytesToHash(childData)
			currentPath = append(currentPath, targetNibble)
		case 2:
			keyPath, ok := node[0].([]byte)
			if !ok {
				return empty, fmt.Errorf("invalid key path in node")
			}
			key := extractKeyNibbles(keyPath)
			expectedPath := make([]byte, 0)
			expectedPath = append(expectedPath, currentPath...)
			expectedPath = append(expectedPath, key...)

			// Check if it is a leaf or extension node.
			leaf, err := isLeaf(keyPath)
			if err != nil {
				return empty, err
			}
			if leaf {
				// Check that the keyPath matches the target nibbles,
				// otherwise, the receipt does not exist in the trie.
				if !bytes.Equal(expectedPath, targetNibbles) {
					return empty, fmt.Errorf("leaf key does not match target nibbles")
				}
				rawData, ok := node[1].([]byte)
				if !ok {
					return empty, fmt.Errorf("invalid receipt data in leaf node")
				}
				return decodeBinary[T](rawData)
			}
			// If the node is not a leaf node, it is an extension node.
			// Check if our target key matches this extension path.
			if len(expectedPath) > len(targetNibbles) || !bytes.Equal(expectedPath, targetNibbles[:len(expectedPath)]) {
				return empty, fmt.Errorf("extension path mismatch")
			}
			nextNodeBytes, ok := node[1].([]byte)
			if !ok {
				return empty, fmt.Errorf("invalid next node in extension")
			}
			// We navigate to the next node in the trie.
			currentNodeHash = common.BytesToHash(nextNodeBytes)
			currentPath = expectedPath
		default:
			return empty, fmt.Errorf("invalid node structure: unexpected length %d", len(node))
		}
	}
}

// Converts a byte slice key into a slice of nibbles (4-bit values).
// Keys are encoded in big endian format, which is required by Ethereum MPTs.
func keyToNibbles(key []byte) []byte {
	nibbles := make([]byte, len(key)*2)
	for i, b := range key {
		nibbles[i*2] = b >> 4
		nibbles[i*2+1] = b & 0x0f
	}
	return nibbles
}

// Extracts the key nibbles from a key path, handling odd/even length cases.
func extractKeyNibbles(keyPath []byte) []byte {
	if len(keyPath) == 0 {
		return nil
	}
	nibbles := keyToNibbles(keyPath)
	if nibbles[0]&1 != 0 {
		return nibbles[1:]
	}
	return nibbles[2:]
}

func isLeaf(keyPath []byte) (bool, error) {
	firstByte := keyPath[0]
	firstNibble := firstByte >> 4
	// 2 or 3 indicates leaf, while 0 or 1 indicates extension nodes in the Ethereum MPT specification.
	if firstNibble > 3 {
		return false, errors.New("first nibble cannot be greater than 3")
	}
	return firstNibble >= 2, nil
}

func decodeBinary[T any](data []byte) (*T, error) {
	var empty *T
	if len(data) == 0 {
		return empty, errors.New("empty data cannot be decoded")
	}
	v := new(T)
	u, ok := any(v).(interface{ UnmarshalBinary([]byte) error })
	if !ok {
		return empty, errors.New("decodeBinary is called on a type that doesnt implement UnmarshalBinary")
	}
	if err := u.UnmarshalBinary(data); err != nil {
		return empty, err
	}
	return v, nil
}
