// Copyright 2024-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package gethexec

import (
	"encoding/binary"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/node"

	"github.com/offchainlabs/nitro/util/dbutil"
)

// buildMerkleTree populates db with a binary merkle tree over the given leaf data.
// Returns the root hash and the number of leaves (merkleSize).
func buildMerkleTree(t *testing.T, db ethdb.KeyValueWriter, leaves [][]byte) (common.Hash, uint64) {
	t.Helper()
	if len(leaves) == 0 {
		t.Fatal("buildMerkleTree requires at least one leaf")
	}
	// Store leaves keyed by their hash
	hashes := make([]common.Hash, len(leaves))
	for i, leaf := range leaves {
		h := crypto.Keccak256Hash(leaf)
		hashes[i] = h
		if err := db.Put(h.Bytes(), leaf); err != nil {
			t.Fatalf("failed to store leaf %d: %v", i, err)
		}
	}
	// Build tree bottom-up. Each internal node is stored as leftHash || rightHash (64 bytes),
	// keyed by the keccak256 of that concatenation — matching what GetMsg expects.
	for len(hashes) > 1 {
		var next []common.Hash
		for i := 0; i < len(hashes); i += 2 {
			if i+1 < len(hashes) {
				var nodeData [64]byte
				copy(nodeData[0:32], hashes[i].Bytes())
				copy(nodeData[32:64], hashes[i+1].Bytes())
				parentHash := crypto.Keccak256Hash(nodeData[:])
				if err := db.Put(parentHash.Bytes(), nodeData[:]); err != nil {
					t.Fatalf("failed to store internal node: %v", err)
				}
				next = append(next, parentHash)
			} else {
				next = append(next, hashes[i])
			}
		}
		hashes = next
	}
	return hashes[0], uint64(len(leaves))
}

// writeBatchHeader writes a classic-msg batch header (8-byte merkleSize + 32-byte root)
// keyed by keccak256("msgBatch" || batchNum.Bytes()).
func writeBatchHeader(t *testing.T, db ethdb.KeyValueWriter, batchNum *big.Int, root common.Hash, merkleSize uint64) {
	t.Helper()
	key := crypto.Keccak256(append([]byte("msgBatch"), batchNum.Bytes()...))
	header := make([]byte, 40)
	binary.BigEndian.PutUint64(header[0:8], merkleSize)
	copy(header[8:40], root.Bytes())
	if err := db.Put(key, header); err != nil {
		t.Fatalf("failed to write batch header: %v", err)
	}
}

func TestClassicOutboxRetrieverGetMsg(t *testing.T) {
	t.Parallel()
	db := rawdb.NewMemoryDatabase()

	leaves := [][]byte{
		[]byte("message-0"),
		[]byte("message-1"),
		[]byte("message-2"),
		[]byte("message-3"),
	}
	root, merkleSize := buildMerkleTree(t, db, leaves)
	writeBatchHeader(t, db, big.NewInt(0), root, merkleSize)

	retriever := NewClassicOutboxRetriever(db)

	for i, expected := range leaves {
		msg, err := retriever.GetMsg(big.NewInt(0), uint64(i))
		if err != nil {
			t.Fatalf("GetMsg(batch=0, index=%d) error: %v", i, err)
		}
		if string(msg.Data) != string(expected) {
			t.Errorf("GetMsg(batch=0, index=%d) data = %q, want %q", i, msg.Data, expected)
		}
		if msg.PathInt == nil {
			t.Errorf("GetMsg(batch=0, index=%d) PathInt is nil", i)
		}
		// 4-leaf tree has depth 2, so proof should have 2 sibling nodes
		if len(msg.ProofNodes) != 2 {
			t.Errorf("GetMsg(batch=0, index=%d) proof length = %d, want 2", i, len(msg.ProofNodes))
		}
	}
}

func TestClassicOutboxRetrieverSingleLeaf(t *testing.T) {
	t.Parallel()
	db := rawdb.NewMemoryDatabase()

	leaves := [][]byte{[]byte("only-message")}
	root, merkleSize := buildMerkleTree(t, db, leaves)
	writeBatchHeader(t, db, big.NewInt(1), root, merkleSize)

	retriever := NewClassicOutboxRetriever(db)
	msg, err := retriever.GetMsg(big.NewInt(1), 0)
	if err != nil {
		t.Fatalf("GetMsg error: %v", err)
	}
	if string(msg.Data) != "only-message" {
		t.Errorf("data = %q, want %q", msg.Data, "only-message")
	}
	// Single leaf: no merkle traversal needed, so no proof nodes
	if len(msg.ProofNodes) != 0 {
		t.Errorf("proof length = %d, want 0", len(msg.ProofNodes))
	}
}

func TestClassicOutboxRetrieverNonPowerOfTwoLeaves(t *testing.T) {
	t.Parallel()
	db := rawdb.NewMemoryDatabase()

	// 3 leaves exercises the non-power-of-two branch (bits.OnesCount64 != 1)
	leaves := [][]byte{
		[]byte("leaf-0"),
		[]byte("leaf-1"),
		[]byte("leaf-2"),
	}
	root, merkleSize := buildMerkleTree(t, db, leaves)
	writeBatchHeader(t, db, big.NewInt(0), root, merkleSize)

	retriever := NewClassicOutboxRetriever(db)
	for i, expected := range leaves {
		msg, err := retriever.GetMsg(big.NewInt(0), uint64(i))
		if err != nil {
			t.Fatalf("GetMsg(index=%d) error: %v", i, err)
		}
		if string(msg.Data) != string(expected) {
			t.Errorf("GetMsg(index=%d) data = %q, want %q", i, msg.Data, expected)
		}
	}
}

func TestClassicOutboxRetrieverErrors(t *testing.T) {
	t.Parallel()
	db := rawdb.NewMemoryDatabase()

	leaves := [][]byte{[]byte("msg-0"), []byte("msg-1")}
	root, merkleSize := buildMerkleTree(t, db, leaves)
	writeBatchHeader(t, db, big.NewInt(0), root, merkleSize)

	retriever := NewClassicOutboxRetriever(db)

	// Non-existent batch
	_, err := retriever.GetMsg(big.NewInt(99), 0)
	if err == nil {
		t.Error("expected error for non-existent batch, got nil")
	}

	// Index out of range
	_, err = retriever.GetMsg(big.NewInt(0), 999)
	if err == nil {
		t.Error("expected error for out-of-range index, got nil")
	}

	// Index exactly equal to merkleSize (one past last valid index).
	// Valid indices for merkleSize=2 are 0 and 1; index 2 should be rejected.
	_, err = retriever.GetMsg(big.NewInt(0), merkleSize)
	if err == nil {
		t.Errorf("expected error for index == merkleSize (%d), got nil", merkleSize)
	}
}

func TestClassicOutboxRetrieverBoundaryIndex(t *testing.T) {
	t.Parallel()
	// Test the boundary between valid and invalid indices across different tree sizes.
	// The last valid index is merkleSize-1; merkleSize itself must be rejected.
	treeSizes := []int{1, 2, 3, 4, 5, 7, 8}
	for _, size := range treeSizes {
		size := size
		t.Run(fmt.Sprintf("size-%d", size), func(t *testing.T) {
			t.Parallel()
			db := rawdb.NewMemoryDatabase()
			leaves := make([][]byte, size)
			for i := range leaves {
				leaves[i] = []byte(fmt.Sprintf("leaf-%d", i))
			}
			root, merkleSize := buildMerkleTree(t, db, leaves)
			writeBatchHeader(t, db, big.NewInt(0), root, merkleSize)
			retriever := NewClassicOutboxRetriever(db)

			// Last valid index should succeed
			lastValid := merkleSize - 1
			msg, err := retriever.GetMsg(big.NewInt(0), lastValid)
			if err != nil {
				t.Fatalf("GetMsg(index=%d) should succeed for merkleSize=%d, got: %v", lastValid, merkleSize, err)
			}
			expected := fmt.Sprintf("leaf-%d", lastValid)
			if string(msg.Data) != expected {
				t.Errorf("GetMsg(index=%d) data = %q, want %q", lastValid, msg.Data, expected)
			}

			// First invalid index (== merkleSize) should fail
			_, err = retriever.GetMsg(big.NewInt(0), merkleSize)
			if err == nil {
				t.Errorf("GetMsg(index=%d) should fail for merkleSize=%d", merkleSize, merkleSize)
			}

			// One beyond that should also fail
			_, err = retriever.GetMsg(big.NewInt(0), merkleSize+1)
			if err == nil {
				t.Errorf("GetMsg(index=%d) should fail for merkleSize=%d", merkleSize+1, merkleSize)
			}
		})
	}
}

// TestClassicMsgDatabaseReopen verifies that the classic-msg database
// can be created, populated, closed, and reopened read-only with the
// production options used by CreateExecutionNode.
func TestClassicMsgDatabaseReopen(t *testing.T) {
	t.Parallel()
	stackConf := node.DefaultConfig
	stackConf.DataDir = t.TempDir()
	stack, err := node.New(&stackConf)
	if err != nil {
		t.Fatalf("Failed to create stack: %v", err)
	}
	defer stack.Close()

	// Create and populate the classic-msg database
	db, err := stack.OpenDatabaseWithOptions("classic-msg", node.DatabaseOptions{
		Cache:     16,
		Handles:   16,
		NoFreezer: true,
	})
	if err != nil {
		t.Fatalf("Failed to create classic-msg database: %v", err)
	}

	leaves := [][]byte{[]byte("test-outbox-msg")}
	root, merkleSize := buildMerkleTree(t, db, leaves)
	writeBatchHeader(t, db, big.NewInt(0), root, merkleSize)
	db.Close()
	stack.Close()

	// Reopen with the exact options matching production (see OpenDatabaseWithOptions("classic-msg", ...) in node.go)
	stack2Conf := node.DefaultConfig
	stack2Conf.DataDir = stackConf.DataDir
	stack2, err := node.New(&stack2Conf)
	if err != nil {
		t.Fatalf("Failed to create stack: %v", err)
	}
	defer stack2.Close()

	db2, err := stack2.OpenDatabaseWithOptions("classic-msg", node.DatabaseOptions{
		MetricsNamespace: "classicmsg/",
		Cache:            0,
		Handles:          0,
		ReadOnly:         true,
		NoFreezer:        true,
	})
	if err != nil {
		if dbutil.IsNotExistError(err) {
			t.Fatalf("Database should exist but got not-exist error: %v", err)
		}
		t.Fatalf("Failed to open classic-msg with NoFreezer: %v", err)
	}
	defer db2.Close()

	// Verify data survives the close/reopen cycle via ClassicOutboxRetriever
	retriever := NewClassicOutboxRetriever(db2)
	msg, err := retriever.GetMsg(big.NewInt(0), 0)
	if err != nil {
		t.Fatalf("GetMsg failed after reopening database: %v", err)
	}
	if string(msg.Data) != "test-outbox-msg" {
		t.Errorf("data = %q, want %q", msg.Data, "test-outbox-msg")
	}
}
