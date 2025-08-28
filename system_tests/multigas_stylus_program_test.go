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

	// Find tx in last block and assert StorageAccess was charged
	txHash := tx.Hash().Bytes()
	var found bool
	for _, ptx := range blocks[len(blocks)-1].Transactions {
		if !bytes.Equal(ptx.TxHash, txHash) {
			continue
		}
		require.NotNil(t, ptx.MultiGas, "missing multigas for tx %s", tx.Hash())

		require.Equal(t, uint64(params.ColdSloadCostEIP2929), ptx.MultiGas.StorageAccess)
		require.Greater(t, ptx.MultiGas.WasmComputation, uint64(0))

		// TODO(NIT-3767): check total gas value
		// require.Equal(t, mg.StorageAccess+mg.WasmComputation, mg.Total)

		found = true
		break
	}

	if !found {
		require.Fail(t, "tx %s not found in multigas data", tx.Hash().Hex())
	}
}
