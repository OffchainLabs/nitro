package main

import (
	"bytes"
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/offchainlabs/nitro/arbutil"
)

type receiptFetcherForBlock struct {
	block            *types.Block
	preimageResolver preimageResolver
}

func (rf *receiptFetcherForBlock) ReceiptForTransactionIndex(
	ctx context.Context,
	txIndex uint,
) (*types.Receipt, error) {
	txes := rf.block.Transactions()
	if int(txIndex) >= len(txes) {
		return nil, fmt.Errorf("transaction index %d out of bounds for block with %d transactions", txIndex, len(txes))
	}
	return fetchReceiptFromBlock(rf.block.ReceiptHash(), txIndex, rf.preimageResolver)
}

// Fetches a specific receipt index from a block's receipt trie by navigating its
// Merkle Patricia Trie structure. It uses the preimage resolver to fetch preimages
// of trie nodes as needed, and determines how to navigate depending on the structure of the trie nodes.
func fetchReceiptFromBlock(
	receiptsRoot common.Hash,
	receiptIndex uint,
	preimageResolver preimageResolver,
) (*types.Receipt, error) {
	currentNodeHash := receiptsRoot
	currentPath := []byte{} // Track nibbles consumed so far.
	receiptKey, err := rlp.EncodeToBytes(receiptIndex)
	if err != nil {
		return nil, err
	}
	targetNibbles := keyToNibbles(receiptKey)
	for {
		nodeData, err := preimageResolver.ResolveTypedPreimage(arbutil.Keccak256PreimageType, currentNodeHash)
		if err != nil {
			return nil, err
		}
		var node []any
		if err = rlp.DecodeBytes(nodeData, &node); err != nil {
			return nil, fmt.Errorf("failed to decode RLP node: %w", err)
		}
		switch len(node) {
		case 17:
			// We hit a branch node, which has 16 children and a value.
			if len(currentPath) >= len(targetNibbles) {
				// A branch node's 17th item could be the value, so we check if it contains the receipt.
				if valueBytes, ok := node[16].([]byte); ok && len(valueBytes) > 0 {
					// This branch node has the actual value as the last item, so we decode the receipt
					receipt := new(types.Receipt)
					receiptData := bytes.NewBuffer(valueBytes)
					if err = rlp.Decode(receiptData, &receipt); err != nil {
						return nil, fmt.Errorf("failed to decode receipt: %w", err)
					}
					return receipt, nil
				}
				return nil, fmt.Errorf("no receipt found at target key")
			}
			// Get the next nibble to follow.
			targetNibble := targetNibbles[len(currentPath)]
			childData, ok := node[targetNibble].([]byte)
			if !ok || len(childData) == 0 {
				return nil, fmt.Errorf("no child at nibble %d", targetNibble)
			}
			// Move to the child node, which is the next hash we have to navigate.
			currentNodeHash = common.BytesToHash(childData)
			currentPath = append(currentPath, targetNibble)
		case 2:
			keyPath, ok := node[0].([]byte)
			if !ok {
				return nil, fmt.Errorf("invalid key path in node")
			}
			// Check if it is a leaf or extension node.
			if isLeaf(keyPath) {
				// Check that the keyPath matches the target nibbles,
				// otherwise, the receipt does not exist in the trie.
				leafKey := extractKeyNibbles(keyPath)
				expectedPath := append(currentPath, leafKey...)
				if !bytes.Equal(expectedPath, targetNibbles) {
					return nil, fmt.Errorf("leaf key does not match target nibbles")
				}

				receipt := new(types.Receipt)
				receiptData := bytes.NewBuffer(node[1].([]byte))
				if err = rlp.Decode(receiptData, &receipt); err != nil {
					return nil, fmt.Errorf("failed to decode receipt: %w", err)
				}
				return receipt, nil
			}
			// If the node is not a leaf node, it is an extension node.
			// We extract the extension key path and append it to our current path.
			extKey := extractKeyNibbles(keyPath)
			newPath := append(currentPath, extKey...)

			// Check if our target key matches this extension path.
			if len(newPath) > len(targetNibbles) || !bytes.Equal(newPath, targetNibbles[:len(newPath)]) {
				return nil, fmt.Errorf("extension path mismatch")
			}
			nextNodeBytes, ok := node[1].([]byte)
			if !ok {
				return nil, fmt.Errorf("invalid next node in extension")
			}
			// We navigate to the next node in the trie.
			currentNodeHash = common.BytesToHash(nextNodeBytes)
			currentPath = newPath
		default:
			return nil, fmt.Errorf("invalid node structure: unexpected length %d", len(node))
		}
	}
}

// Converts a byte slice key into a slice of nibbles (4-bit values).
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

	firstByte := keyPath[0]
	isOdd := (firstByte & 0x10) != 0

	var nibbles []byte
	if isOdd {
		// Odd length: first nibble is in the first byte.
		nibbles = append(nibbles, firstByte&0x0f)
		keyPath = keyPath[1:]
	} else {
		keyPath = keyPath[1:]
	}
	// Convert remaining bytes to nibbles.
	for _, b := range keyPath {
		nibbles = append(nibbles, b>>4)
		nibbles = append(nibbles, b&0x0f)
	}

	return nibbles
}

func isLeaf(keyPath []byte) bool {
	firstByte := keyPath[0]
	firstNibble := firstByte >> 4
	// 2 or 3 indicates leaf, while 0 or 1 indicates extension nodes in the Ethereum MPT specification.
	return firstNibble >= 2
}
