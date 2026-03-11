// Copyright 2024-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package gethexec

import (
	"encoding/binary"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
)

// BuildClassicMerkleTree populates db with a binary merkle tree over the given leaf data.
// Returns the root hash and the number of leaves (merkleSize).
// Exported for use in tests across packages.
func BuildClassicMerkleTree(db ethdb.KeyValueWriter, leaves [][]byte) (common.Hash, uint64, error) {
	if len(leaves) == 0 {
		return common.Hash{}, 0, fmt.Errorf("BuildClassicMerkleTree requires at least one leaf")
	}
	hashes := make([]common.Hash, len(leaves))
	for i, leaf := range leaves {
		h := crypto.Keccak256Hash(leaf)
		hashes[i] = h
		if err := db.Put(h.Bytes(), leaf); err != nil {
			return common.Hash{}, 0, fmt.Errorf("failed to store leaf %d: %w", i, err)
		}
	}
	for len(hashes) > 1 {
		var next []common.Hash
		for i := 0; i < len(hashes); i += 2 {
			if i+1 < len(hashes) {
				var nodeData [64]byte
				copy(nodeData[0:32], hashes[i].Bytes())
				copy(nodeData[32:64], hashes[i+1].Bytes())
				parentHash := crypto.Keccak256Hash(nodeData[:])
				if err := db.Put(parentHash.Bytes(), nodeData[:]); err != nil {
					return common.Hash{}, 0, fmt.Errorf("failed to store internal node: %w", err)
				}
				next = append(next, parentHash)
			} else {
				next = append(next, hashes[i])
			}
		}
		hashes = next
	}
	return hashes[0], uint64(len(leaves)), nil
}

// WriteClassicBatchHeader writes a classic-msg batch header (8-byte merkleSize + 32-byte root)
// keyed by keccak256("msgBatch" || batchNum.Bytes()).
// Exported for use in tests across packages.
func WriteClassicBatchHeader(db ethdb.KeyValueWriter, batchNum *big.Int, root common.Hash, merkleSize uint64) error {
	key := msgBatchKey(batchNum)
	header := make([]byte, 40)
	binary.BigEndian.PutUint64(header[0:8], merkleSize)
	copy(header[8:40], root.Bytes())
	if err := db.Put(key, header); err != nil {
		return fmt.Errorf("failed to write batch header: %w", err)
	}
	return nil
}
