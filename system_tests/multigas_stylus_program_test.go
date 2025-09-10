// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/arbitrum/multigas"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/util/testhelpers"
)

func TestMultigasStylus_GetBytes32(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	builder.execConfig.ExposeMultiGas = true
	cleanup := builder.Build(t)
	defer cleanup()

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
	receipt, err := EnsureTxSucceeded(ctx, l2client, tx)
	require.NoError(t, err)

	require.Equal(t, params.ColdSloadCostEIP2929-params.WarmStorageReadCostEIP2929, receipt.MultiGasUsed.Get(multigas.ResourceKindStorageAccess))
	require.Equal(t, params.WarmStorageReadCostEIP2929, receipt.MultiGasUsed.Get(multigas.ResourceKindComputation))

	// TODO(NIT-3552): after instrumenting intrinsic gas and gasChargingHook this difference should be zero
	gasDifference := params.TxGas + uint64(516)
	require.Equal(t, receipt.GasUsed, receipt.MultiGasUsed.SingleGas()+gasDifference)

	// TODO(NIT-3793, NIT-3793, NIT-3795): Once all WASM operations are instrumented, WasmComputation
	// should be derived as the residual from SingleGas instead of asserted directly.
	require.Greater(t, receipt.MultiGasUsed.Get(multigas.ResourceKindWasmComputation), uint64(0))
}
