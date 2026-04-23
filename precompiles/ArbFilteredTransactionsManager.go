// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package precompiles

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"

	"github.com/offchainlabs/nitro/arbos/filteredTransactions"
)

// ArbFilteredTransactionsManager precompile enables ability to filter transactions by authorized callers.
// Authorized callers are added/removed through ArbOwner precompile.
type ArbFilteredTransactionsManager struct {
	Address addr // 0x74

	FilteredTransactionAdded        func(ctx, mech, common.Hash) error
	FilteredTransactionAddedGasCost func(common.Hash) (uint64, error)

	FilteredTransactionDeleted        func(ctx, mech, common.Hash) error
	FilteredTransactionDeletedGasCost func(common.Hash) (uint64, error)
}

// Adds a transaction hash to the filtered transactions list
func (con ArbFilteredTransactionsManager) AddFilteredTransaction(c *Context, evm *vm.EVM, txHash common.Hash) error {
	if !con.hasAccess(c) {
		return c.BurnOut()
	}

	filteredState := filteredTransactions.Open(evm.StateDB, c)
	if err := filteredState.Add(txHash); err != nil {
		return err
	}

	return con.FilteredTransactionAdded(c, evm, txHash)
}

// Deletes a transaction hash from the filtered transactions list
func (con ArbFilteredTransactionsManager) DeleteFilteredTransaction(c *Context, evm *vm.EVM, txHash common.Hash) error {
	if !con.hasAccess(c) {
		return c.BurnOut()
	}

	filteredState := filteredTransactions.Open(evm.StateDB, c)
	if err := filteredState.Delete(txHash); err != nil {
		return err
	}

	return con.FilteredTransactionDeleted(c, evm, txHash)
}

// Checks if a transaction hash is in the filtered transactions list
func (con ArbFilteredTransactionsManager) IsTransactionFiltered(c *Context, evm *vm.EVM, txHash common.Hash) (bool, error) {
	filteredState := filteredTransactions.Open(evm.StateDB, c)
	return filteredState.IsFiltered(txHash)
}

func (con ArbFilteredTransactionsManager) hasAccess(c *Context) bool {
	manager, err := c.State.TransactionFilterers().IsMember(c.caller)
	return manager && err == nil
}
