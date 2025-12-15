package arbtest

import (
	"context"
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/core/rawdb"
)

func TestAccessingPathSchemeState(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	builder := NewNodeBuilder(ctx).DefaultConfig(t, false).WithDatabase(rawdb.DBPebble)

	// This test is PathScheme specific, it shouldn't be run with HashScheme
	builder.RequireScheme(t, rawdb.PathScheme)

	// Build a node with history past the 128 block diff threshold
	cancelNode := buildWithHistory(t, ctx, builder, 150)
	execNode, l2client := builder.L2.ExecNode, builder.L2.Client
	defer cancelNode()
	bc := execNode.Backend.ArbInterface().BlockChain()

	header := bc.CurrentBlock()
	if header == nil {
		Fatal(t, "failed to get current block header")
	}

	head := header.Number.Uint64()
	if head < 129 {
		t.Fatalf("chain height (%d) too low — need at least 129 blocks to check last 128", head)
	}
	start := head - 128

	for height := head; height > start; height-- {
		_, err := l2client.BalanceAt(ctx, GetTestAddressForAccountName(t, "User2"), new(big.Int).SetUint64(height))
		Require(t, err, "Failed to get balance at height", height)
	}

	// Now try to access state older than 128 blocks ago, which should be missing
	//
	// We don't want to see the error
	// `missing trie node X (path ) state Y is not available, not found`
	// because that indicates a failure to find data that should exist. Implying our state backend has a bug.
	heightWhereStateShouldBeMissing := head - 129
	_, err := l2client.BalanceAt(ctx, GetTestAddressForAccountName(t, "User2"), new(big.Int).SetUint64(heightWhereStateShouldBeMissing))
	require.Error(t, err, "expected BalanceAt to fail for missing historical state")
	require.Contains(t, err.Error(), "historical state", "unexpected error message: %v", err)
	require.Contains(t, err.Error(), "is not available", "unexpected error message: %v", err)
}

func TestAccessingPathSchemeArchivalState(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	builder := NewNodeBuilder(ctx).DefaultConfig(t, false).WithDatabase(rawdb.DBPebble)
	builder.execConfig.Caching.Archive = true
	builder.execConfig.Caching.StateHistory = 2
	// There's a race condition for when persisted state ID is updated and checked against
	// the first history block, meaning sometimes state pruning is skipped to make sure
	// the persisted state ID is ahead. NoAsyncFlush config makes the flush synchronous
	// when set to true, that way disklayer.writeStateHistory won't skip calls to
	// trancateFromTail.
	builder.TrieNoAsyncFlush = true

	// This test is PathScheme specific, it shouldn't be run with HashScheme
	builder.RequireScheme(t, rawdb.PathScheme)

	// Build a node with history past the 128 block diff threshold
	cancelNode := buildWithHistory(t, ctx, builder, 150)
	fmt.Println("bluebird 5-3", builder.execConfig.Caching.StateScheme)
	execNode, l2client := builder.L2.ExecNode, builder.L2.Client
	defer cancelNode()
	bc := execNode.Backend.ArbInterface().BlockChain()

	header := bc.CurrentBlock()
	if header == nil {
		Fatal(t, "failed to get current block header")
	}

	head := header.Number.Uint64()
	if head < 132 {
		t.Fatalf("chain height (%d) too low — need at least 129 blocks to check last 128", head)
	}
	start := head - 131

	for height := head; height > start; height-- {
		_, err := l2client.BalanceAt(ctx, GetTestAddressForAccountName(t, "User2"), new(big.Int).SetUint64(height))
		Require(t, err, "Failed to get balance at height", height)
	}

	// Now try to access state older than 131 blocks ago, which should be missing
	heightWhereStateShouldBeMissing := head - 132
	_, err := l2client.BalanceAt(ctx, GetTestAddressForAccountName(t, "User2"), new(big.Int).SetUint64(heightWhereStateShouldBeMissing))
	require.Error(t, err, "expected BalanceAt to fail for missing historical state")
	// `metadata is not found` is the error returned when archival data is pruned for some reason
	require.Contains(t, err.Error(), "metadata is not found", "unexpected error message: %v", err)
}
