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
	return fetchReceiptFromBlock(len(txes), rf.block.ReceiptHash(), txIndex, rf.preimageResolver)
}

func fetchReceiptFromBlock(
	numReceipts int,
	receiptsRoot common.Hash,
	receiptIndex uint,
	preimageResolver preimageResolver,
) (*types.Receipt, error) {
	// Special case: if there's only one receipt, the trie structure is optimized
	// and we need to handle it differently
	if numReceipts == 1 && receiptIndex == 0 {
		return fetchSingleReceiptFromBlock(receiptsRoot, preimageResolver)
	}

	// Normal case: multiple receipts with branch/extension node structure
	return fetchReceiptFromMultiReceiptBlock(numReceipts, receiptsRoot, receiptIndex, preimageResolver)
}

func fetchSingleReceiptFromBlock(
	receiptsRoot common.Hash,
	preimageResolver preimageResolver,
) (*types.Receipt, error) {
	// For a single receipt, the root is directly the leaf containing the receipt
	nodeData, err := preimageResolver.ResolveTypedPreimage(arbutil.Keccak256PreimageType, receiptsRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to get preimage for root hash %x: %w", receiptsRoot, err)
	}
	var node []any
	if err := rlp.DecodeBytes(nodeData, &node); err != nil {
		return nil, fmt.Errorf("failed to decode root node: %w", err)
	}
	if len(node) != 2 {
		return nil, fmt.Errorf("expected leaf node with 2 elements, got %d", len(node))
	}
	// For a single receipt, the root node should be a leaf containing the receipt
	receipt := new(types.Receipt)
	receiptData := bytes.NewBuffer(node[1].([]byte))
	// We skip the first byte which is the receipt type, as we will only care
	// about non-legacy transaction receipts for MEL.
	if err := rlp.Decode(receiptData, &receipt); err != nil {
		return nil, fmt.Errorf("failed to decode receipt from root node: %w", err)
	}
	return receipt, nil
}

// Use the receipts root, num receipts, and receipt index to extract
// a single receipt from a block using preimage reads. This navigates the receipts
// Merkle Patricia Trie (MPT) structure to find the specific receipt hash and then
// retrieves that specific receipt using the preimage read function.
func fetchReceiptFromMultiReceiptBlock(
	numReceipts int,
	receiptsRoot common.Hash,
	receiptIndex uint,
	preimageResolver preimageResolver,
) (*types.Receipt, error) {
	// Encode the transaction index exactly like geth does in DeriveSha
	var keyBuf []byte
	keyBuf = rlp.AppendUint64(keyBuf, uint64(receiptIndex))

	// Convert to nibbles for MPT traversal.
	keyNibbles := keybytesToHex(keyBuf)

	currentHash := receiptsRoot
	keyPos := 0

	for {
		if currentHash == (common.Hash{}) {
			return nil, fmt.Errorf("receipt not found")
		}

		// Get node data from preimage
		nodeData, err := preimageResolver.ResolveTypedPreimage(arbutil.Keccak256PreimageType, currentHash)
		if err != nil {
			return nil, fmt.Errorf("failed to get preimage for hash %x: %w", currentHash, err)
		}

		// Decode the node
		var node []any
		if err := rlp.DecodeBytes(nodeData, &node); err != nil {
			return nil, fmt.Errorf("failed to decode node: %w", err)
		}

		switch len(node) {
		case 17: // Branch node
			if keyPos >= len(keyNibbles) {
				// We've consumed all key nibbles, check the value slot (index 16)
				if node[16] != nil {
					receipt := new(types.Receipt)
					if err := rlp.DecodeBytes(node[16].([]byte), &receipt); err != nil {
						return nil, fmt.Errorf("failed to decode receipt from root node: %w", err)
					}
					return receipt, nil
				}
				return nil, fmt.Errorf("receipt not found at branch value")
			}

			// Follow the path for current nibble
			nibble := keyNibbles[keyPos]
			if nibble >= 16 {
				return nil, fmt.Errorf("invalid nibble: %d", nibble)
			}

			nextNode := node[nibble]
			if nextNode == nil {
				return nil, fmt.Errorf("receipt not found - nil branch at nibble %d", nibble)
			}

			// Extract hash from the next node reference
			nextHashBytes, err := extractBytes(nextNode)
			if err != nil {
				return nil, fmt.Errorf("failed to extract hash bytes from branch node: %w", err)
			}

			// Convert bytes to common.Hash
			if len(nextHashBytes) != 32 {
				return nil, fmt.Errorf("invalid hash length: expected 32 bytes, got %d", len(nextHashBytes))
			}
			currentHash = common.BytesToHash(nextHashBytes)
			keyPos++

		case 2: // Extension or Leaf node
			encodedPath := node[0]
			valueOrHash := node[1]

			// Decode the compact-encoded path
			pathBytes, err := extractBytes(encodedPath)
			if err != nil {
				return nil, fmt.Errorf("failed to extract path bytes: %w", err)
			}

			pathNibbles, isLeaf := decodeCompact(pathBytes)

			// Check if remaining key matches this path
			remainingKey := keyNibbles[keyPos:]

			// FIX: Check bounds before slicing and handle path matching correctly
			if len(remainingKey) < len(pathNibbles) {
				return nil, fmt.Errorf("receipt not found - remaining key too short: remaining=%d, path=%d", len(remainingKey), len(pathNibbles))
			}

			// Compare the path nibbles with the corresponding part of remaining key
			if !nibbleSliceEqual(remainingKey[:len(pathNibbles)], pathNibbles) {
				return nil, fmt.Errorf("receipt not found - path mismatch: expected=%v, got=%v", pathNibbles, remainingKey[:len(pathNibbles)])
			}

			if isLeaf {
				// Found the leaf - check if we've consumed exactly the right amount of key
				if len(remainingKey) == len(pathNibbles) {
					receipt := new(types.Receipt)
					if err := rlp.DecodeBytes(valueOrHash.([]byte), &receipt); err != nil {
						return nil, fmt.Errorf("failed to decode receipt from root node: %w", err)
					}
					return receipt, nil
				}
				return nil, fmt.Errorf("receipt not found - leaf path length mismatch: remaining=%d, path=%d", len(remainingKey), len(pathNibbles))
			} else {
				// Extension node - continue traversal
				nextHashBytes, err := extractBytes(valueOrHash)
				if err != nil {
					return nil, fmt.Errorf("failed to extract hash bytes from extension node: %w", err)
				}

				// Convert bytes to common.Hash
				if len(nextHashBytes) != 32 {
					return nil, fmt.Errorf("invalid hash length: expected 32 bytes, got %d", len(nextHashBytes))
				}
				currentHash = common.BytesToHash(nextHashBytes)
				keyPos += len(pathNibbles)
			}

		default:
			return nil, fmt.Errorf("invalid node length: %d", len(node))
		}
	}
}

// Helper function to debug the decodeCompact function
func decodeCompact(buf []byte) ([]byte, bool) {
	if len(buf) == 0 {
		return nil, false
	}

	// The first byte contains the flags and possibly the first nibble
	firstByte := buf[0]
	// Extract flags from the first nibble (high 4 bits)
	isLeaf := (firstByte & 0x20) != 0    // bit 5 (0x20 = 00100000)
	oddLength := (firstByte & 0x10) != 0 // bit 4 (0x10 = 00010000)

	var nibbles []byte
	if oddLength {
		// Odd length: second nibble of first byte is part of the path
		nibbles = append(nibbles, firstByte&0x0F)
		// Convert remaining bytes to nibbles
		for _, b := range buf[1:] {
			nibbles = append(nibbles, b>>4, b&0x0F)
		}
	} else {
		// Even length: convert all bytes to nibbles, then remove first nibble (padding)
		for _, b := range buf {
			nibbles = append(nibbles, b>>4, b&0x0F)
		}
		// Remove the first nibble (which was padding)
		if len(nibbles) > 0 {
			nibbles = nibbles[1:]
		}
	}
	return nibbles, isLeaf
}

func keybytesToHex(str []byte) []byte {
	l := len(str) * 2
	var nibbles = make([]byte, l)
	for i, b := range str {
		nibbles[i*2] = b / 16
		nibbles[i*2+1] = b % 16
	}
	return nibbles
}

func extractBytes(value interface{}) ([]byte, error) {
	switch v := value.(type) {
	case []byte:
		return v, nil
	case string:
		return []byte(v), nil
	default:
		return nil, fmt.Errorf("cannot extract bytes from type %T", value)
	}
}

func nibbleSliceEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
