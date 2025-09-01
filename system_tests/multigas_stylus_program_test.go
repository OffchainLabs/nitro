// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbos/multigascollector"
	"github.com/offchainlabs/nitro/arbos/multigascollector/proto"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

func TestMultigasStylus_GetBytes32(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	outDir := t.TempDir()

	// Build a node with the multigas collector enabled
	builder := NewNodeBuilder(ctx).
		DefaultConfig(t, false).
		WithMultigasCollector(multigascollector.CollectorConfig{
			OutputDir:      outDir,
			BatchSize:      5,
			ClearOutputDir: true,
		})
	cleanup := builder.Build(t) // no defer

	l2info := builder.L2Info
	l2client := builder.L2.Client

	// Deploy programs
	owner := l2info.GetDefaultTransactOpts("Owner", ctx)
	storage := deployWasm(t, ctx, owner, l2client, rustFile("storage"))

	// Send tx to call getBytes32
	key := testhelpers.RandomHash()
	readArgs := argsForStorageRead(key)

	tx := l2info.PrepareTxTo("Owner", &storage, l2info.TransferGas, nil, readArgs)
	require.NoError(t, l2client.SendTransaction(ctx, tx))
	_, err := EnsureTxSucceeded(ctx, l2client, tx)
	require.NoError(t, err)

	// Stop node to flush collector
	cleanup()

	var blocks = readCollectorBatches(t, outDir, -1)
	require.NotEmpty(t, blocks, "no multigas data found")

	var allTxs []*proto.TransactionMultiGasData
	for _, blk := range blocks {
		allTxs = append(allTxs, blk.Transactions...)
	}

	// Find transactions in the all transactions
	var found bool
	for _, ptx := range allTxs {
		if bytes.Equal(ptx.TxHash, tx.Hash().Bytes()) {
			require.Equal(t, params.ColdSloadCostEIP2929-params.WarmStorageReadCostEIP2929, ptx.MultiGas.StorageAccess)
			require.Equal(t, params.WarmStorageReadCostEIP2929, ptx.MultiGas.Computation)

			// TODO: Use hard-coded value until all the places are instrumented,
			// after expected wasm computation can be calculated from total (single gas)
			expectedWasmComputation := uint64(10423)
			require.Equal(t, expectedWasmComputation, ptx.MultiGas.WasmComputation)

			found = true
			break
		}
	}

	if !found {
		require.Fail(t, "transactions not found in multigas collector data")
	}
}
