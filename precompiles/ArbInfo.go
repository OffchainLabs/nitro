//
// Copyright 2021-2022, Offchain Labs, Inc. All rights reserved.
//

package precompiles

import (
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/util"
)

// Provides the ability to lookup basic info about accounts and contracts.
type ArbInfo struct {
	Address addr // 0x65
}

// Retrieves an account's balance
func (con ArbInfo) GetBalance(c ctx, evm mech, account addr) (huge, error) {
	if err := c.Burn(params.BalanceGasEIP1884); err != nil {
		return nil, err
	}
	return evm.StateDB.GetBalance(account), nil
}

// Retrieves a contract's deployed code
func (con ArbInfo) GetCode(c ctx, evm mech, account addr) ([]byte, error) {
	if err := c.Burn(params.ColdSloadCostEIP2929); err != nil {
		return nil, err
	}
	code := evm.StateDB.GetCode(account)
	if err := c.Burn(params.CopyGas * util.WordsForBytes(uint64(len(code)))); err != nil {
		return nil, err
	}
	return code, nil
}
