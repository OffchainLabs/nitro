// Copyright 2024-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package gethexec

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/core/rawdb"
)

func TestClassicOutboxRetrieverGetMsg(t *testing.T) {
	t.Parallel()
	db := rawdb.NewMemoryDatabase()

	leaves := [][]byte{
		[]byte("message-0"),
		[]byte("message-1"),
		[]byte("message-2"),
		[]byte("message-3"),
	}
	root, merkleSize, err := BuildClassicMerkleTree(db, leaves)
	if err != nil {
		t.Fatal(err)
	}
	if err := WriteClassicBatchHeader(db, big.NewInt(0), root, merkleSize); err != nil {
		t.Fatal(err)
	}

	retriever := NewClassicOutboxRetriever(db)

	for i, expected := range leaves {
		msg, err := retriever.GetMsg(big.NewInt(0), uint64(i)) //#nosec G115
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
	root, merkleSize, err := BuildClassicMerkleTree(db, leaves)
	if err != nil {
		t.Fatal(err)
	}
	if err := WriteClassicBatchHeader(db, big.NewInt(1), root, merkleSize); err != nil {
		t.Fatal(err)
	}

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
	root, merkleSize, err := BuildClassicMerkleTree(db, leaves)
	if err != nil {
		t.Fatal(err)
	}
	if err := WriteClassicBatchHeader(db, big.NewInt(0), root, merkleSize); err != nil {
		t.Fatal(err)
	}

	retriever := NewClassicOutboxRetriever(db)
	for i, expected := range leaves {
		msg, err := retriever.GetMsg(big.NewInt(0), uint64(i)) //#nosec G115
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
	root, merkleSize, err := BuildClassicMerkleTree(db, leaves)
	if err != nil {
		t.Fatal(err)
	}
	if err := WriteClassicBatchHeader(db, big.NewInt(0), root, merkleSize); err != nil {
		t.Fatal(err)
	}

	retriever := NewClassicOutboxRetriever(db)

	// Non-existent batch
	_, err = retriever.GetMsg(big.NewInt(99), 0)
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
			root, merkleSize, err := BuildClassicMerkleTree(db, leaves)
			if err != nil {
				t.Fatal(err)
			}
			if err := WriteClassicBatchHeader(db, big.NewInt(0), root, merkleSize); err != nil {
				t.Fatal(err)
			}
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
