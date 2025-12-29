// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"context"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
)

func TestManageTransactionCensors(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).
		DefaultConfig(t, false).
		WithArbOSVersion(params.ArbosVersion_60)

	cleanup := builder.Build(t)
	defer cleanup()

	ownerTxOpts := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)

	builder.L2Info.GenerateAccount("User")
	builder.L2.TransferBalance(t, "Owner", "User", big.NewInt(1e16), builder.L2Info)
	userTxOpts := builder.L2Info.GetDefaultTransactOpts("User", ctx)

	ownerCallOpts := &bind.CallOpts{Context: ctx, From: ownerTxOpts.From}
	userCallOpts := &bind.CallOpts{Context: ctx, From: userTxOpts.From}

	txHash := common.BytesToHash([]byte{1, 2, 3, 4, 5})

	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, builder.L2.Client)
	require.NoError(t, err)

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

	// Owner grants user transaction censor role
	tx, err := arbOwner.AddTransactionCensor(&ownerTxOpts, userTxOpts.From)
	require.NoError(t, err)
	require.NotNil(t, tx)

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
	require.NotNil(t, tx)

	filtered, err = arbFilteredTxs.IsTransactionFiltered(userCallOpts, txHash)
	require.NoError(t, err)
	require.True(t, filtered)

	// User unfilters the tx
	tx, err = arbFilteredTxs.DeleteFilteredTransaction(&userTxOpts, txHash)
	require.NoError(t, err)
	require.NotNil(t, tx)

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
}
