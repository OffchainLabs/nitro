// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package precompiles

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
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
	return errors.New("ArbCensoredTransactionsManager precompile is not yet implemented")
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
	return errors.New("ArbCensoredTransactionsManager precompile is not yet implemented")
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
	return false, errors.New("ArbCensoredTransactionsManager precompile is not yet implemented")
}

func (con ArbCensoredTransactionsManager) hasAccess(c *Context) (bool, error) {
	return c.State.TransactionCensors().IsMember(c.caller)
}
