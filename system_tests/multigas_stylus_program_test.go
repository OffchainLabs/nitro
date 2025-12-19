// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"context"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/arbitrum/multigas"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

func TestMultigasStylus_GetBytes32(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
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

	require.GreaterOrEqual(t, receipt.MultiGasUsed.Get(multigas.ResourceKindWasmComputation), uint64(12_000))
	require.Equal(t, receipt.MultiGasUsed.Get(multigas.ResourceKindComputation), params.TxGas+params.WarmStorageReadCostEIP2929)
}

func TestMultigasStylus_AccountAccessHostIOs(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
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

	cases := []struct {
		name       string
		numTopics  uint64
		payloadLen uint64
	}{
		{"no_topics_no_data", 0, 0},
		{"one_topic_empty_payload", 1, 0},
		{"two_topics_64_bytes", 2, 64},
		{"three_topics_96_bytes", 3, 96},
		{"four_topics_128_bytes", 4, 128},
		{"one_topic_large_payload", 1, 1024},
		{"four_topics_zero_payload", 4, 0}, // pure topic cost
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Send a transaction with specified topics and data
			topics := make([]common.Hash, tc.numTopics)
			for i := range topics {
				topics[i] = testhelpers.RandomHash()
			}
			data := make([]byte, tc.payloadLen)
			args := encode(topics, data)

			tx := l2info.PrepareTxTo("Owner", &logAddr, l2info.TransferGas, nil, args)
			require.NoError(t, l2client.SendTransaction(ctx, tx))

			receipt, err := EnsureTxSucceeded(ctx, l2client, tx)
			require.NoError(t, err)

			// Expected history growth calculation
			expectedHistoryGrowth := params.LogTopicHistoryGas*tc.numTopics + tc.payloadLen*params.LogDataGas

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
		})
	}
}

func TestMultigasStylus_Create(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	cleanup := builder.Build(t)
	defer cleanup()

	l2info := builder.L2Info
	l2client := builder.L2.Client
	owner := l2info.GetDefaultTransactOpts("Owner", ctx)

	// Deploy the factory (create) contract
	createAddr := deployWasm(t, ctx, owner, l2client, rustFile("create"))
	storageWasm, _ := readWasmFile(t, rustFile("storage"))
	deployCode := deployContractInitCode(storageWasm, false)

	// CREATE1 call
	create1Args := []byte{0x01} // selector for CREATE1
	create1Args = append(create1Args, common.BigToHash(big.NewInt(0)).Bytes()...)
	create1Args = append(create1Args, deployCode...)

	tx := l2info.PrepareTxTo("Owner", &createAddr, 1_000_000_000, nil, create1Args)
	require.NoError(t, l2client.SendTransaction(ctx, tx))
	receipt1, err := EnsureTxSucceeded(ctx, l2client, tx)
	require.NoError(t, err)

	require.Greater(t,
		receipt1.MultiGasUsed.Get(multigas.ResourceKindComputation),
		uint64(21000+32000), // intrinsic + CREATE
	)

	require.Equalf(t,
		receipt1.GasUsed,
		receipt1.MultiGasUsed.SingleGas(),
		"Used gas mismatch: GasUsed=%d, MultiGas=%d",
		receipt1.GasUsed, receipt1.MultiGasUsed.SingleGas(),
	)

	// CREATE2 call
	salt := testhelpers.RandomHash()
	create2Args := []byte{0x02} // selector for CREATE2
	create2Args = append(create2Args, common.BigToHash(big.NewInt(0)).Bytes()...)
	create2Args = append(create2Args, salt[:]...)
	create2Args = append(create2Args, deployCode...)

	tx2 := l2info.PrepareTxTo("Owner", &createAddr, 1_000_000_000, nil, create2Args)
	require.NoError(t, l2client.SendTransaction(ctx, tx2))
	receipt2, err := EnsureTxSucceeded(ctx, l2client, tx2)
	require.NoError(t, err)

	// CREATE2: expect additional keccak cost relative to CREATE1
	keccakWords := arbmath.WordsForBytes(uint64(len(deployCode)))
	expectedExtra := arbmath.SaturatingUMul(params.Keccak256WordGas, keccakWords)

	require.Equal(t,
		receipt1.MultiGasUsed.Get(multigas.ResourceKindComputation)+expectedExtra,
		receipt2.MultiGasUsed.Get(multigas.ResourceKindComputation),
	)

	require.Equalf(t,
		receipt2.GasUsed,
		receipt2.MultiGasUsed.SingleGas(),
		"Used gas mismatch: GasUsed=%d, MultiGas=%d",
		receipt2.GasUsed, receipt2.MultiGasUsed.SingleGas(),
	)
}

func TestMultigasStylus_Calls(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	cleanup := builder.Build(t)
	defer cleanup()

	l2info := builder.L2Info
	l2client := builder.L2.Client
	owner := l2info.GetDefaultTransactOpts("Owner", ctx)

	// deploy multicall + storage targets
	callsAddr := deployWasm(t, ctx, owner, l2client, rustFile("multicall"))
	storeAddr := deployWasm(t, ctx, owner, l2client, rustFile("storage"))

	cases := []struct {
		name   string
		opcode vm.OpCode
	}{
		{"call", vm.CALL},
		{"delegatecall", vm.DELEGATECALL},
		{"staticcall", vm.STATICCALL},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var calldata []byte
			var expectedStorageAccess uint64
			switch tc.opcode {
			case vm.CALL:
				key := testhelpers.RandomHash()
				storageVal := testhelpers.RandomHash()
				calldata = argsForMulticall(vm.CALL, storeAddr, nil, argsForStorageWrite(key, storageVal))

				expectedStorageAccess = params.ColdAccountAccessCostEIP2929 - params.WarmStorageReadCostEIP2929 + params.ColdSloadCostEIP2929

			case vm.DELEGATECALL:
				calldata = argsForMulticall(vm.DELEGATECALL, callsAddr, nil, []byte{0})

				expectedStorageAccess = 0

			case vm.STATICCALL:
				key := testhelpers.RandomHash()

				// now read it with STATICCALL
				calldata = argsForMulticall(vm.STATICCALL, storeAddr, nil, argsForStorageRead(key))

				// One cold account access + one cold storage read
				expectedStorageAccess = params.ColdAccountAccessCostEIP2929 + params.ColdSloadCostEIP2929 - params.WarmStorageReadCostEIP2929*2
			}

			tx := l2info.PrepareTxTo("Owner", &callsAddr, 1e9, nil, calldata)
			require.NoError(t, l2client.SendTransaction(ctx, tx))

			receipt, err := EnsureTxSucceeded(ctx, l2client, tx)
			require.NoError(t, err)

			require.Equalf(t,
				receipt.GasUsed,
				receipt.MultiGasUsed.SingleGas(),
				"Used gas mismatch: GasUsed=%d, MultiGas=%d, Difference=%d",
				receipt.GasUsed, receipt.MultiGasUsed.SingleGas(), receipt.GasUsed-receipt.MultiGasUsed.SingleGas(),
			)

			require.Equal(t,
				expectedStorageAccess,
				receipt.MultiGasUsed.Get(multigas.ResourceKindStorageAccess),
			)
		})
	}
}

func TestMultigasStylus_StorageWrite(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	cleanup := builder.Build(t)
	defer cleanup()

	l2info := builder.L2Info
	l2client := builder.L2.Client
	owner := l2info.GetDefaultTransactOpts("Owner", ctx)

	storage := deployWasm(t, ctx, owner, l2client, rustFile("storage"))

	key := testhelpers.RandomHash()
	val := testhelpers.RandomHash()
	writeArgs := argsForStorageWrite(key, val)

	cases := []struct {
		name     string
		gasLimit uint64
		expectOK bool
	}{
		{"success", 1_000_000_000, true},
		{"out_of_gas", 1_500_000, false}, // above intrinsic cost, below storage create slot cost
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tx := l2info.PrepareTxTo("Owner", &storage, tc.gasLimit, nil, writeArgs)
			require.NoError(t, l2client.SendTransaction(ctx, tx))

			receipt, err := EnsureTxSucceeded(ctx, l2client, tx)
			if tc.expectOK {
				require.NoError(t, err)

				// Expected multigas for create slot operation
				require.Equal(t, receipt.GasUsed, receipt.MultiGasUsed.SingleGas())
				require.Equal(t, params.ColdSloadCostEIP2929, receipt.MultiGasUsed.Get(multigas.ResourceKindStorageAccess))
				require.Equal(t, params.SstoreSetGasEIP2200, receipt.MultiGasUsed.Get(multigas.ResourceKindStorageGrowth))
			} else {
				require.Error(t, err)
				receipt, err := l2client.TransactionReceipt(ctx, tx.Hash())
				require.NoError(t, err)
				require.Equal(t, receipt.GasUsed, receipt.MultiGasUsed.SingleGas())
			}
		})
	}
}
