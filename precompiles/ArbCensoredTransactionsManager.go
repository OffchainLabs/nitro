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

func (con ArbCensoredTransactionsManager) AddCensoredTransaction(c *Context, evm *vm.EVM, txHash common.Hash) error {
	return errors.New("ArbCensoredTransactionsManager precompile is not yet implemented")
}

func (con ArbCensoredTransactionsManager) DeleteCensoredTransaction(c *Context, evm *vm.EVM, txHash common.Hash) error {
	return errors.New("ArbCensoredTransactionsManager precompile is not yet implemented")
}

func (con ArbCensoredTransactionsManager) IsTransactionCensored(c *Context, evm *vm.EVM, txHash common.Hash) (bool, error) {
	return false, errors.New("ArbCensoredTransactionsManager precompile is not yet implemented")
}
