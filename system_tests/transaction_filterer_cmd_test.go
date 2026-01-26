// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/cmd/transaction-filterer/api"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/util/rpcclient"
)

func TestTransactionFiltererCmd(t *testing.T) {
	ctx := t.Context()

	arbOSInit := &params.ArbOSInit{
		TransactionFilteringEnabled: true,
	}
	builder := NewNodeBuilder(ctx).
		DefaultConfig(t, false).
		WithArbOSVersion(params.ArbosVersion_TransactionFiltering).
		WithArbOSInit(arbOSInit)
	cleanup := builder.Build(t)
	defer cleanup()

	transactionFiltererStackConf := api.DefaultStackConfig
	// use arbitrary available ports
	transactionFiltererStackConf.HTTPPort = 0
	transactionFiltererStackConf.WSPort = 0
	transactionFiltererStackConf.AuthPort = 0

	filtererName := "Filterer"
	builder.L2Info.GenerateAccount(filtererName)
	builder.L2.TransferBalance(t, "Owner", filtererName, big.NewInt(1e16), builder.L2Info)
	filtererTxOpts := builder.L2Info.GetDefaultTransactOpts("Filterer", ctx)

	transactionFiltererStack, err := api.NewStack(&transactionFiltererStackConf, &filtererTxOpts, builder.L2.Client)
	require.NoError(t, err)
	err = transactionFiltererStack.Start()
	require.NoError(t, err)
	defer transactionFiltererStack.Close()

	transactionFiltererRPCClientConfigFetcher := func() *rpcclient.ClientConfig {
		config := rpcclient.DefaultClientConfig
		config.URL = transactionFiltererStack.HTTPEndpoint()
		return &config
	}
	transactionFiltererRPCClient := rpcclient.NewRpcClient(transactionFiltererRPCClientConfigFetcher, nil)
	err = transactionFiltererRPCClient.Start(ctx)
	require.NoError(t, err)
	defer transactionFiltererRPCClient.Close()

	arbFilteredTransactionsManager, err := precompilesgen.NewArbFilteredTransactionsManager(
		types.ArbFilteredTransactionsManagerAddress,
		builder.L2.Client,
	)
	require.NoError(t, err)

	txHash := common.BytesToHash([]byte{1, 2, 3, 4, 5})

	// Ensure transaction is not filtered
	callOpts := &bind.CallOpts{Context: ctx}
	filtered, err := arbFilteredTransactionsManager.IsTransactionFiltered(callOpts, txHash)
	require.NoError(t, err)
	require.False(t, filtered)

	// Filterer not added to the filterers set yet, should fail
	err = transactionFiltererRPCClient.CallContext(ctx, nil, "transactionfilterer_filter", txHash)
	require.Error(t, err)

	// Add filterer
	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, builder.L2.Client)
	require.NoError(t, err)
	ownerTxOpts := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	tx, err := arbOwner.AddTransactionFilterer(&ownerTxOpts, filtererTxOpts.From)
	require.NoError(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	require.NoError(t, err)

	// Now filtering should work
	err = transactionFiltererRPCClient.CallContext(ctx, nil, "transactionfilterer_filter", txHash)
	require.NoError(t, err)

	// Ensure transaction is now filtered
	filtered, err = arbFilteredTransactionsManager.IsTransactionFiltered(callOpts, txHash)
	require.NoError(t, err)
	require.True(t, filtered)
}
