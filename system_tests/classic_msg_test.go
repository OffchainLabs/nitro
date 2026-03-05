// Copyright 2024-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"encoding/binary"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/node"

	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

func buildTestMerkleTree(t *testing.T, db ethdb.KeyValueWriter, leaves [][]byte) (common.Hash, uint64) {
	t.Helper()
	hashes := make([]common.Hash, len(leaves))
	for i, leaf := range leaves {
		h := crypto.Keccak256Hash(leaf)
		hashes[i] = h
		if err := db.Put(h.Bytes(), leaf); err != nil {
			t.Fatalf("failed to store leaf %d: %v", i, err)
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

func writeTestBatchHeader(t *testing.T, db ethdb.KeyValueWriter, batchNum *big.Int, root common.Hash, merkleSize uint64) {
	t.Helper()
	key := crypto.Keccak256(append([]byte("msgBatch"), batchNum.Bytes()...))
	header := make([]byte, 40)
	binary.BigEndian.PutUint64(header[0:8], merkleSize)
	copy(header[8:40], root.Bytes())
	if err := db.Put(key, header); err != nil {
		t.Fatalf("failed to write batch header: %v", err)
	}
}

// TestOpenClassicOutboxFromStack verifies that OpenClassicOutboxFromStack
// correctly opens a pre-existing classic-msg database with NoFreezer and
// ReadOnly options, and that data is readable through the ClassicOutboxRetriever.
// This exercises the same production code path used by CreateExecutionNode.
func TestOpenClassicOutboxFromStack(t *testing.T) {
	t.Parallel()
	dataDir := t.TempDir()
	stackConfig := testhelpers.CreateStackConfigForTest(dataDir)
	stackConfig.DBEngine = rawdb.DBPebble

	// Create and populate the classic-msg database
	stack, err := node.New(stackConfig)
	Require(t, err)

	db, err := stack.OpenDatabaseWithOptions("classic-msg", node.DatabaseOptions{
		Cache:     16,
		Handles:   16,
		NoFreezer: true,
	})
	Require(t, err)

	leaves := [][]byte{[]byte("classic-outbox-test-msg")}
	root, merkleSize := buildTestMerkleTree(t, db, leaves)
	writeTestBatchHeader(t, db, big.NewInt(0), root, merkleSize)
	db.Close()
	stack.Close()

	// Reopen through the production function
	stack2, err := node.New(stackConfig)
	Require(t, err)
	defer stack2.Close()

	retriever, err := gethexec.OpenClassicOutboxFromStack(stack2)
	Require(t, err)
	if retriever == nil {
		t.Fatal("OpenClassicOutboxFromStack returned nil for existing database")
	}

	msg, err := retriever.GetMsg(big.NewInt(0), 0)
	Require(t, err)
	if string(msg.Data) != "classic-outbox-test-msg" {
		t.Errorf("GetMsg data = %q, want %q", msg.Data, "classic-outbox-test-msg")
	}
}

// TestOpenClassicOutboxFromStackMissing verifies that OpenClassicOutboxFromStack
// returns nil (not an error) when the classic-msg database does not exist.
func TestOpenClassicOutboxFromStackMissing(t *testing.T) {
	t.Parallel()
	dataDir := t.TempDir()
	stackConfig := testhelpers.CreateStackConfigForTest(dataDir)
	stackConfig.DBEngine = rawdb.DBPebble

	stack, err := node.New(stackConfig)
	Require(t, err)
	defer stack.Close()

	retriever, err := gethexec.OpenClassicOutboxFromStack(stack)
	Require(t, err)
	if retriever != nil {
		t.Fatal("OpenClassicOutboxFromStack should return nil when database does not exist")
	}
}
