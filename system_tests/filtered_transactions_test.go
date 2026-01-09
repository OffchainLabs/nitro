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

func TestManageTransactionFilterers(t *testing.T) {
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

	// Adding a filterer should be disabled by default by ArbFiltereredTransactionManagerFromTime
	_, err = arbOwner.AddTransactionFilterer(&ownerTxOpts, userTxOpts.From)
	require.Error(t, err)

	// Make sure transaction filtering can not be enabled before one week delay
	hdr, err := builder.L2.Client.HeaderByNumber(ctx, nil)
	require.NoError(t, err)
	tryEnableAt := hdr.Time + (5 * 24 * 60 * 60) // 5 days in the future
	_, err = arbOwner.SetTransactionFilteringFrom(&ownerTxOpts, tryEnableAt)
	require.Error(t, err)

	// Enable transaction filtering feature 7 days in the future and warp time forward
	enableAt := hdr.Time + precompiles.TransactionFilteringEnableDelay
	tx, err := arbOwner.SetTransactionFilteringFrom(&ownerTxOpts, enableAt)
	require.NoError(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	require.NoError(t, err)

	warpL1Time(t, builder, ctx, hdr.Time, precompiles.TransactionFilteringEnableDelay+1)

	// Owner grants user transaction filterer role
	tx, err = arbOwner.AddTransactionFilterer(&ownerTxOpts, userTxOpts.From)
	require.NoError(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	require.NoError(t, err)

	isFilterer, err := arbOwner.IsTransactionFilterer(ownerCallOpts, userTxOpts.From)
	require.NoError(t, err)
	require.True(t, isFilterer)

	// Owner is still not a filterer, so owner still cannot call the manager
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

	// Check that the FilteredTransactionAdded event was emitted
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

	// Check that the FilteredTransactionDeleted event was emitted
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
	tx, err = arbOwner.RemoveTransactionFilterer(&ownerTxOpts, userTxOpts.From)
	require.NoError(t, err)
	require.NotNil(t, tx)

	isFilterer, err = arbOwner.IsTransactionFilterer(ownerCallOpts, userTxOpts.From)
	require.NoError(t, err)
	require.False(t, isFilterer)

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

	arbOSInit := &params.ArbOSInit{
		TransactionFilteringEnabled: true,
	}

	builder := NewNodeBuilder(ctx).
		DefaultConfig(t, true).
		WithArbOSVersion(params.ArbosVersion_60).
		WithArbOSInit(arbOSInit)

	cleanup := builder.Build(t)
	defer cleanup()

	ownerTxOpts := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)

	filtererName := "Filterer"
	builder.L2Info.GenerateAccount(filtererName)

	builder.L2.TransferBalance(t, "Owner", filtererName, big.NewInt(1e16), builder.L2Info)
	filtererTxOpts := builder.L2Info.GetDefaultTransactOpts(filtererName, ctx)
	filtererTxOpts.GasLimit = 32000000

	txHash := common.BytesToHash([]byte{1, 2, 3, 4, 5})

	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, builder.L2.Client)
	require.NoError(t, err)

	arbFilteredTxs, err := precompilesgen.NewArbFilteredTransactionsManager(
		types.ArbFilteredTransactionsManagerAddress,
		builder.L2.Client,
	)
	require.NoError(t, err)

	// Owner grants filterer transaction filterer role
	tx, err := arbOwner.AddTransactionFilterer(&ownerTxOpts, filtererTxOpts.From)
	require.NoError(t, err)
	require.NotNil(t, tx)

	// Filterer filters the tx
	tx, err = arbFilteredTxs.AddFilteredTransaction(&filtererTxOpts, txHash)
	require.NoError(t, err)
	receipt, err := builder.L2.EnsureTxSucceeded(tx)
	require.NoError(t, err)

	// AddFilteredTransaction use storage set, but it should be free for filterers
	require.Equal(t, uint64(0), receipt.MultiGasUsed.Get(multigas.ResourceKindStorageAccess))

	// Filterer unfilters the tx
	tx, err = arbFilteredTxs.DeleteFilteredTransaction(&filtererTxOpts, txHash)
	require.NoError(t, err)
	receipt, err = builder.L2.EnsureTxSucceeded(tx)
	require.NoError(t, err)

	// DeleteFilteredTransaction use storage clear, but it should be free for filterers
	require.Equal(t, uint64(0), receipt.MultiGasUsed.Get(multigas.ResourceKindStorageAccess))
}
