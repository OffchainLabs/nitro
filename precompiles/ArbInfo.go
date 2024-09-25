// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package precompiles

import (
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/util/arbmath"
)

// ArbInfo povides the ability to lookup basic info about accounts and contracts.
type ArbInfo struct {
	Address addr // 0x65
}

// GetBalance retrieves an account's balance
func (con ArbInfo) GetBalance(c ctx, evm mech, account addr) (huge, error) {
	if err := c.Burn(params.BalanceGasEIP1884); err != nil {
		return nil, err
	}
	return evm.StateDB.GetBalance(account).ToBig(), nil
}

// GetCode retrieves a contract's deployed code
func (con ArbInfo) GetCode(c ctx, evm mech, account addr) ([]byte, error) {
	if err := c.Burn(params.ColdSloadCostEIP2929); err != nil {
		return nil, err
	}
	code := evm.StateDB.GetCode(account)
	if err := c.Burn(params.CopyGas * arbmath.WordsForBytes(uint64(len(code)))); err != nil {
		return nil, err
	}
	return code, nil
}
