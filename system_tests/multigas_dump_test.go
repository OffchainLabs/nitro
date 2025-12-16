// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"context"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/core/rawdb"
)

// TestMultigasDataFromReceipts spins up an L2 node with ancd checks if multigas data is present in receipts
func TestMultigasDataFromReceipts(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	cleanup := builder.Build(t)
	defer cleanup()

	// Generate a L2 user and send 20 transactions
	builder.L2Info.GenerateAccount("Alice")
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

		require.Equal(t, rcpt.GasUsed, rcpt.MultiGasUsed.SingleGas())
	}
}

func TestMultigasDataCanBeDisabled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false).WithDatabase(rawdb.DBPebble)
	builder.execConfig.ExposeMultiGas = false
	cleanup := builder.Build(t)
	defer cleanup()

	tx := builder.L2Info.PrepareTx(
		"Owner", "Owner",
		builder.L2Info.TransferGas,
		big.NewInt(1),
		nil,
	)
	require.NoError(t, builder.L2.Client.SendTransaction(ctx, tx))
	receipt, err := builder.L2.EnsureTxSucceeded(tx)
	require.NoError(t, err)

	require.True(t, receipt.MultiGasUsed.IsZero())
}
