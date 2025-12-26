// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package precompiles

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"

	"github.com/offchainlabs/nitro/arbos/filteredTransactions"
)

// ArbFilteredTransactionsManager precompile enables ability to censor transactions by authorized callers.
// Authorized callers are added/removed through ArbOwner precompile.
type ArbFilteredTransactionsManager struct {
	Address addr // 0x74
}

// Adds a transaction hash to the filtered transactions list
func (con ArbFilteredTransactionsManager) AddFilteredTransaction(c *Context, evm *vm.EVM, txHash common.Hash) error {
	hasAccess, err := con.hasAccess(c)
	if err != nil {
		return err
	}
	if !hasAccess {
		return c.BurnOut()
	}

	filteredState := filteredTransactions.Open(evm.StateDB, c)
	return filteredState.Add(txHash)
}

// Deletes a transaction hash from the filtered transactions list
func (con ArbFilteredTransactionsManager) DeleteFilteredTransaction(c *Context, evm *vm.EVM, txHash common.Hash) error {
	hasAccess, err := con.hasAccess(c)
	if err != nil {
		return err
	}
	if !hasAccess {
		return c.BurnOut()
	}

	filteredState := filteredTransactions.Open(evm.StateDB, c)
	return filteredState.Delete(txHash)
}

// Checks if a transaction hash is in the filtered transactions list
func (con ArbFilteredTransactionsManager) IsTransactionFiltered(c *Context, evm *vm.EVM, txHash common.Hash) (bool, error) {
	hasAccess, err := con.hasAccess(c)
	if err != nil {
		return false, err
	}
	if !hasAccess {
		return false, c.BurnOut()
	}
	filteredState := filteredTransactions.Open(evm.StateDB, c)
	return filteredState.IsFiltered(txHash)
}

func (con ArbFilteredTransactionsManager) hasAccess(c *Context) (bool, error) {
	return c.State.TransactionCensors().IsMember(c.caller)
}
