// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package precompiles

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"

	"github.com/offchainlabs/nitro/arbos/censored_transactions"
)

// ArbCensoredTransactionsManager precompile enables ability to censor transactions by authorized callers.
// Authorized callers are added/removed through ArbOwner precompile.
type ArbCensoredTransactionsManager struct {
	Address addr // 0x74
}

// Adds a transaction hash to the censored transactions list
func (con ArbCensoredTransactionsManager) AddCensoredTransaction(c *Context, evm *vm.EVM, txHash common.Hash) error {
	hasAccess, err := con.hasAccess(c)
	if err != nil {
		return err
	}
	if !hasAccess {
		return c.BurnOut()
	}

	censoredState := censored_transactions.Open(evm.StateDB, c)
	return censoredState.Add(txHash)
}

// Deletes a transaction hash from the censored transactions list
func (con ArbCensoredTransactionsManager) DeleteCensoredTransaction(c *Context, evm *vm.EVM, txHash common.Hash) error {
	hasAccess, err := con.hasAccess(c)
	if err != nil {
		return err
	}
	if !hasAccess {
		return c.BurnOut()
	}

	censoredState := censored_transactions.Open(evm.StateDB, c)
	return censoredState.Delete(txHash)
}

// Checks if a transaction hash is in the censored transactions list
func (con ArbCensoredTransactionsManager) IsTransactionCensored(c *Context, evm *vm.EVM, txHash common.Hash) (bool, error) {
	hasAccess, err := con.hasAccess(c)
	if err != nil {
		return false, err
	}
	if !hasAccess {
		return false, c.BurnOut()
	}
	censoredState := censored_transactions.Open(evm.StateDB, c)
	return censoredState.IsCensored(txHash)
}

func (con ArbCensoredTransactionsManager) hasAccess(c *Context) (bool, error) {
	return c.State.TransactionCensors().IsMember(c.caller)
}
