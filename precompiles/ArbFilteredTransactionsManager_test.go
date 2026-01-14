// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package precompiles

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/arbitrum/multigas"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/arbos/filteredTransactions"
)

func setupFilteredTransactionsHandles(
	t *testing.T,
) (
	*vm.EVM,
	*arbosState.ArbosState,
	*Context,
	*ArbFilteredTransactionsManager,
) {
	t.Helper()

	evm := newMockEVMForTesting()
	caller := common.BytesToAddress(crypto.Keccak256([]byte("caller"))[:20])

	callCtx := testContext(caller, evm)
	callCtx.gasSupplied = 100000
	callCtx.gasUsed = multigas.ZeroGas()

	state, err := arbosState.OpenArbosState(evm.StateDB, callCtx)
	require.NoError(t, err)

	con := &ArbFilteredTransactionsManager{
		FilteredTransactionAdded:   func(ctx ctx, evm mech, txHash common.Hash) error { return nil },
		FilteredTransactionDeleted: func(ctx ctx, evm mech, txHash common.Hash) error { return nil },
	}

	return evm, state, callCtx, con
}

func TestFilteredTransactionsManagerBurnOutForNonFilterer(t *testing.T) {
	t.Parallel()

	evm, _, callCtx, con := setupFilteredTransactionsHandles(t)

	txHash := common.BytesToHash([]byte{1, 2, 3, 4, 5})

	err := con.AddFilteredTransaction(callCtx, evm, txHash)
	require.ErrorIs(t, err, vm.ErrOutOfGas)
}

func TestFilteredTransactionsManagerAddDeleteForFilterer(t *testing.T) {
	t.Parallel()

	evm, state, callCtx, con := setupFilteredTransactionsHandles(t)

	txHash := common.BytesToHash([]byte{5, 4, 3, 2, 1})

	err := state.TransactionFilterers().Add(callCtx.caller)
	require.NoError(t, err)

	err = con.AddFilteredTransaction(callCtx, evm, txHash)
	require.NoError(t, err)

	filteredState := filteredTransactions.Open(evm.StateDB, callCtx)
	isFiltered, err := filteredState.IsFiltered(txHash)
	require.NoError(t, err)
	require.True(t, isFiltered)

	err = con.DeleteFilteredTransaction(callCtx, evm, txHash)
	require.NoError(t, err)

	isFiltered, err = filteredState.IsFiltered(txHash)
	require.NoError(t, err)
	require.False(t, isFiltered)
}
