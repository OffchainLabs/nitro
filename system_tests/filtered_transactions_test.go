// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"context"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/arbitrum/multigas"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/precompiles"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
)

func TestManageTransactionCensors(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).
		DefaultConfig(t, true).
		WithArbOSVersion(params.ArbosVersion_60)

	cleanup := builder.Build(t)
	defer cleanup()

	ownerTxOpts := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)

	builder.L2Info.GenerateAccount("User")
	builder.L2Info.GenerateAccount("User2") // For time warp
	builder.L2.TransferBalance(t, "Owner", "User", big.NewInt(1e16), builder.L2Info)
	userTxOpts := builder.L2Info.GetDefaultTransactOpts("User", ctx)

	ownerCallOpts := &bind.CallOpts{Context: ctx, From: ownerTxOpts.From}
	userCallOpts := &bind.CallOpts{Context: ctx, From: userTxOpts.From}

	txHash := common.BytesToHash([]byte{1, 2, 3, 4, 5})

	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, builder.L2.Client)
	require.NoError(t, err)

	filteredTransactionsManagerABI, err := precompilesgen.ArbFilteredTransactionsManagerMetaData.GetAbi()
	Require(t, err)
	addedTopic := filteredTransactionsManagerABI.Events["FilteredTransactionAdded"].ID
	deletedTopic := filteredTransactionsManagerABI.Events["FilteredTransactionDeleted"].ID

	arbFilteredTxs, err := precompilesgen.NewArbFilteredTransactionsManager(
		types.ArbFilteredTransactionsManagerAddress,
		builder.L2.Client,
	)
	require.NoError(t, err)

	// Initially neither owner nor user can access the filtered tx manager
	_, err = arbFilteredTxs.IsTransactionFiltered(ownerCallOpts, txHash)
	require.Error(t, err)

	_, err = arbFilteredTxs.IsTransactionFiltered(userCallOpts, txHash)
	require.Error(t, err)

	// Adding a censor should be disabled by default by ArbCensoredTransactionManagerFromTime
	_, err = arbOwner.AddTransactionCensor(&ownerTxOpts, userTxOpts.From)
	require.Error(t, err)

	// Enable transaction filtering feature 7 days in the future and warp time forward
	hdr, err := builder.L2.Client.HeaderByNumber(ctx, nil)
	require.NoError(t, err)
	enableAt := hdr.Time + precompiles.TransactionFilteringEnableDelay

	tx, err := arbOwner.SetTransactionFilteringFrom(&ownerTxOpts, enableAt)
	require.NoError(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	require.NoError(t, err)

	warpL1Time(t, builder, ctx, hdr.Time, precompiles.TransactionFilteringEnableDelay+1)

	// Owner grants user transaction censor role
	tx, err = arbOwner.AddTransactionCensor(&ownerTxOpts, userTxOpts.From)
	require.NoError(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	require.NoError(t, err)

	isCensor, err := arbOwner.IsTransactionCensor(ownerCallOpts, userTxOpts.From)
	require.NoError(t, err)
	require.True(t, isCensor)

	// Owner is still not a censor, so owner still cannot call the manager
	_, err = arbFilteredTxs.IsTransactionFiltered(ownerCallOpts, txHash)
	require.Error(t, err)

	// User can call the manager and the tx is initially not filtered
	filtered, err := arbFilteredTxs.IsTransactionFiltered(userCallOpts, txHash)
	require.NoError(t, err)
	require.False(t, filtered)

	// User filters the tx
	tx, err = arbFilteredTxs.AddFilteredTransaction(&userTxOpts, txHash)
	require.NoError(t, err)
	receipt, err := builder.L2.EnsureTxSucceeded(tx)
	require.NoError(t, err)

	foundAdded := false
	for _, lg := range receipt.Logs {
		if lg.Topics[0] != addedTopic {
			continue
		}
		ev, parseErr := arbFilteredTxs.ParseFilteredTransactionAdded(*lg)
		if parseErr != nil {
			continue
		}
		require.Equal(t, txHash, common.BytesToHash(ev.TxHash[:]))
		foundAdded = true
		break
	}
	require.True(t, foundAdded)

	filtered, err = arbFilteredTxs.IsTransactionFiltered(userCallOpts, txHash)
	require.NoError(t, err)
	require.True(t, filtered)

	// User unfilters the tx
	tx, err = arbFilteredTxs.DeleteFilteredTransaction(&userTxOpts, txHash)
	require.NoError(t, err)
	receipt, err = builder.L2.EnsureTxSucceeded(tx)
	require.NoError(t, err)

	foundDeleted := false
	for _, lg := range receipt.Logs {
		if lg.Topics[0] != deletedTopic {
			continue
		}
		ev, parseErr := arbFilteredTxs.ParseFilteredTransactionDeleted(*lg)
		if parseErr != nil {
			continue
		}
		require.Equal(t, txHash, common.BytesToHash(ev.TxHash[:]))
		foundDeleted = true
		break
	}
	require.True(t, foundDeleted)

	filtered, err = arbFilteredTxs.IsTransactionFiltered(userCallOpts, txHash)
	require.NoError(t, err)
	require.False(t, filtered)

	// Owner revokes the role
	tx, err = arbOwner.RemoveTransactionCensor(&ownerTxOpts, userTxOpts.From)
	require.NoError(t, err)
	require.NotNil(t, tx)

	isCensor, err = arbOwner.IsTransactionCensor(ownerCallOpts, userTxOpts.From)
	require.NoError(t, err)
	require.False(t, isCensor)

	// User is no longer authorised
	_, err = arbFilteredTxs.IsTransactionFiltered(userCallOpts, txHash)
	require.Error(t, err)

	// Disable transaction filtering feature again
	tx, err = arbOwner.SetTransactionFilteringFrom(&ownerTxOpts, 0)
	require.NoError(t, err)
	receipt, err = builder.L2.EnsureTxSucceeded(tx)
	require.NoError(t, err)
}

func TestFilteredTransactionsManagerFreeOps(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).
		DefaultConfig(t, true).
		WithArbOSVersion(params.ArbosVersion_60)

	cleanup := builder.Build(t)
	defer cleanup()

	ownerTxOpts := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)

	censorName := "Censor"
	builder.L2Info.GenerateAccount(censorName)
	builder.L2Info.GenerateAccount("User2") // For time warp

	builder.L2.TransferBalance(t, "Owner", censorName, big.NewInt(1e16), builder.L2Info)
	censorTxOpts := builder.L2Info.GetDefaultTransactOpts(censorName, ctx)
	censorTxOpts.GasLimit = 32000000

	txHash := common.BytesToHash([]byte{1, 2, 3, 4, 5})

	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, builder.L2.Client)
	require.NoError(t, err)

	arbFilteredTxs, err := precompilesgen.NewArbFilteredTransactionsManager(
		types.ArbFilteredTransactionsManagerAddress,
		builder.L2.Client,
	)
	require.NoError(t, err)

	// Enable transaction filtering feature 7 days in the future and warp time forward
	hdr, err := builder.L2.Client.HeaderByNumber(ctx, nil)
	require.NoError(t, err)
	enableAt := hdr.Time + precompiles.TransactionFilteringEnableDelay

	tx, err := arbOwner.SetTransactionFilteringFrom(&ownerTxOpts, enableAt)
	require.NoError(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	require.NoError(t, err)

	warpL1Time(t, builder, ctx, hdr.Time, precompiles.TransactionFilteringEnableDelay+1)

	// Owner grants censor transaction censor role
	tx, err = arbOwner.AddTransactionCensor(&ownerTxOpts, censorTxOpts.From)
	require.NoError(t, err)
	require.NotNil(t, tx)

	// Censor filters the tx
	tx, err = arbFilteredTxs.AddFilteredTransaction(&censorTxOpts, txHash)
	require.NoError(t, err)
	receipt, err := builder.L2.EnsureTxSucceeded(tx)
	require.NoError(t, err)

	// AddFilteredTransaction use storage set, but it should be free for censors
	require.Equal(t, uint64(0), receipt.MultiGasUsed.Get(multigas.ResourceKindStorageAccess))

	// Censor unfilters the tx
	tx, err = arbFilteredTxs.DeleteFilteredTransaction(&censorTxOpts, txHash)
	require.NoError(t, err)
	receipt, err = builder.L2.EnsureTxSucceeded(tx)
	require.NoError(t, err)

	// DeleteFilteredTransaction use storage clear, but it should be free for censors
	require.Equal(t, uint64(0), receipt.MultiGasUsed.Get(multigas.ResourceKindStorageAccess))
}
