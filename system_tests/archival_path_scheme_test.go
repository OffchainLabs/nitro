package arbtest

import (
	"context"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/core/rawdb"
)

func TestAccessingPathSchemeArchivalState(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	scheme := rawdb.PathScheme
	builder.defaultDbScheme = scheme
	builder.execConfig.Caching.StateScheme = scheme
	builder.execConfig.RPC.StateScheme = scheme

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
	start := head - 128
	if start < 1 {
		t.Fatalf("chain height (%d) too low — need at least 129 blocks to check last 128", head)
	}

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
