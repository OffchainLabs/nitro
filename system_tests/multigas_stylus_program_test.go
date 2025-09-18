// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/arbitrum/multigas"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/cmd/chaininfo"
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
	require.Equal(t, params.TxGas+params.WarmStorageReadCostEIP2929, receipt.MultiGasUsed.Get(multigas.ResourceKindComputation))
	require.Equal(t, receipt.GasUsed, receipt.MultiGasUsed.SingleGas())

	// TODO(NIT-3793, NIT-3794): Once all WASM operations are instrumented, WasmComputation
	// should be derived as the residual from SingleGas instead of asserted directly.
	require.Greater(t, receipt.MultiGasUsed.Get(multigas.ResourceKindWasmComputation), uint64(0))
}

func TestMultigasStylus_AccountAccessHostIOs(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	builder.execConfig.ExposeMultiGas = true
	cleanup := builder.Build(t)
	defer cleanup()

	l2info := builder.L2Info
	l2client := builder.L2.Client
	owner := l2info.GetDefaultTransactOpts("Owner", ctx)

	hostio := deployWasm(t, ctx, owner, l2client, rustFile("hostio-test"))

	target := common.HexToAddress("0xbeefdead00000000000000000000000000000000")

	tests := []struct {
		name               string
		selectorSignature  string
		withCode           bool
		expectedAccessGas  uint64
		expectedComputeGas uint64
	}{
		{
			name:              "accountBalance",
			selectorSignature: "accountBalance(address)",
			withCode:          false,
		},
		{
			name:              "accountCode",
			selectorSignature: "accountCode(address)",
			withCode:          true,
		},
		{
			name:              "accountCodehash",
			selectorSignature: "accountCodehash(address)",
			withCode:          false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			selector := crypto.Keccak256([]byte(tc.selectorSignature))[:4]
			callData := append([]byte{}, selector...)
			callData = append(callData, common.LeftPadBytes(target.Bytes(), 32)...)

			tx := l2info.PrepareTxTo("Owner", &hostio, l2info.TransferGas, nil, callData)
			require.NoError(t, l2client.SendTransaction(ctx, tx))

			receipt, err := EnsureTxSucceeded(ctx, l2client, tx)
			require.NoError(t, err)

			expectedAccess := params.ColdAccountAccessCostEIP2929 - params.WarmStorageReadCostEIP2929
			expectedCompute := params.WarmStorageReadCostEIP2929 + params.TxGas
			if tc.withCode {
				maxCodeSize := chaininfo.ArbitrumDevTestChainConfig().MaxCodeSize()
				extCodeCost := maxCodeSize / params.DefaultMaxCodeSize * params.ExtcodeSizeGasEIP150
				expectedAccess += extCodeCost
			}

			require.Equal(t, expectedAccess,
				receipt.MultiGasUsed.Get(multigas.ResourceKindStorageAccess),
			)
			require.Equal(t, expectedCompute,
				receipt.MultiGasUsed.Get(multigas.ResourceKindComputation),
			)
		})
	}
}

func TestMultigasStylus_EmitLog(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	builder.execConfig.ExposeMultiGas = true
	cleanup := builder.Build(t)
	defer cleanup()

	l2info := builder.L2Info
	l2client := builder.L2.Client
	owner := l2info.GetDefaultTransactOpts("Owner", ctx)

	// Deploy log contract
	logAddr := deployWasm(t, ctx, owner, l2client, rustFile("log"))

	encode := func(topics []common.Hash, data []byte) []byte {
		args := []byte{byte(len(topics))}
		for _, topic := range topics {
			args = append(args, topic[:]...)
		}
		args = append(args, data...)
		return args
	}

	// Send transaction with 2 topics, 64-byte payload (valid: 64 >= 2*32)
	topics := []common.Hash{testhelpers.RandomHash(), testhelpers.RandomHash()}
	data := testhelpers.RandomSlice(64)
	args := encode(topics, data)

	tx := l2info.PrepareTxTo("Owner", &logAddr, l2info.TransferGas, nil, args)
	require.NoError(t, l2client.SendTransaction(ctx, tx))

	receipt, err := EnsureTxSucceeded(ctx, l2client, tx)
	require.NoError(t, err)

	// Expected history growth calculation
	numTopics := uint64(len(topics))
	dataBytes := uint64(len(data))
	topicBytes := uint64(32)
	topicHistPer := topicBytes * params.LogDataGas
	payloadBytes := dataBytes - topicBytes*numTopics
	expectedHistoryGrowth := topicHistPer*numTopics + payloadBytes*params.LogDataGas

	require.Equal(t,
		expectedHistoryGrowth,
		receipt.MultiGasUsed.Get(multigas.ResourceKindHistoryGrowth),
	)

	require.Equalf(t,
		receipt.GasUsed,
		receipt.MultiGasUsed.SingleGas(),
		"Used gas mismatch: GasUsed=%d, MultiGas=%d, Difference=%d",
		receipt.GasUsed, receipt.MultiGasUsed.SingleGas(), receipt.GasUsed-receipt.MultiGasUsed.SingleGas(),
	)
}
