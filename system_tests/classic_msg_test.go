// Copyright 2024-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/node"

	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

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
	root, merkleSize, err := gethexec.BuildClassicMerkleTree(db, leaves)
	Require(t, err)
	Require(t, gethexec.WriteClassicBatchHeader(db, big.NewInt(0), root, merkleSize))
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
