// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"context"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
)

// TestMultigasDataFromReceipts spins up an L2 node with ancd checks if multigas data is present in receipts
func TestMultigasDataFromReceipts(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Build a node with collector enabled
	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	cleanup := builder.Build(t)
	defer cleanup()

	// Generate a L2 user and send 20 transactions
	builder.L2Info.GenerateAccount("Alice")
	txs := make(map[common.Hash]uint64)
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

		txs[tx.Hash()] = rcpt.GasUsed
	}

	// Restart the node to ensure multigas data is persisted
	builder.RestartL2Node(t)

	for tx, gas := range txs {
		rcpt, err := builder.L2.Client.TransactionReceipt(ctx, tx) // wait for inclusion
		require.NoError(t, err)

		// TODO(NIT-3552): after instrumenting intrinsic gas this difference should be zero
		creation := rcpt.ContractAddress != (common.Address{}) // or infer from tx.To()==nil
		var gasDifference uint64
		if creation {
			gasDifference = params.TxGasContractCreation
		} else {
			gasDifference = params.TxGas
		}
		require.Equal(t, gas, rcpt.MultiGasUsed.SingleGas()+gasDifference)
	}
}
