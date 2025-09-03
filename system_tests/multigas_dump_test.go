// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"context"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/arbos/multigascollector"
)

// TestMultigasCollector_System spins up an L2 node with the multigas collector enabled,
// sends a couple of transactions, and validates the on-disk protobuf batches.
func TestMultigasCollectorFromNode(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	outDir := t.TempDir()

	// Build a node with collector enabled
	builder := NewNodeBuilder(ctx).
		DefaultConfig(t, false).
		WithMultigasCollector(multigascollector.CollectorConfig{
			OutputDir:      outDir,
			BatchSize:      5, // small to force multiple batches for 20 txs
			ClearOutputDir: true,
		})
	cleanup := builder.Build(t) // no defer

	// Generate a L2 user and send 20 transactions
	builder.L2Info.GenerateAccount("Alice")
	want := make(map[common.Hash]uint64)
	for i := 0; i < 20; i++ {
		// unique value to avoid duplicate txs
		value := big.NewInt(1e12 + int64(i))

		tx := builder.L2Info.PrepareTx(
			"Owner", "Alice",
			builder.L2Info.TransferGas,
			value,
			nil,
		)
		require.NoError(t, builder.L2.Client.SendTransaction(ctx, tx))
		rcpt, err := builder.L2.EnsureTxSucceeded(tx)
		require.NoError(t, err)

		want[tx.Hash()] = rcpt.GasUsed
	}

	// Stop the node; collector.StopAndWait() is called under cleanup()
	cleanup()

	var blocks = readCollectorBatches(t, outDir, 5)

	// Scan for our tx hashes
	found := 0
	for _, b := range blocks {
		for _, ptx := range b.Transactions {
			h := common.BytesToHash(ptx.TxHash)
			var gas uint64
			var ok bool
			if gas, ok = want[h]; !ok {
				continue
			}
			require.NotNil(t, ptx.GetMultiGas(), "missing multigas for tx %s", h)
			require.Equal(t, gas, ptx.GetMultiGas().SingleGas, "single gas mismatch for tx %s", h)
			found++
		}
	}

	require.Equal(t, len(want), found, "not all 20 sent txs were found in multigas batches")
}
