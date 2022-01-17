//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package precompiles

import (
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/arbstate/util"
)

type ArbInfo struct {
	Address addr
}

func (con ArbInfo) GetBalance(c ctx, evm mech, account addr) (huge, error) {
	if err := c.Burn(params.BalanceGasEIP1884); err != nil {
		return nil, err
	}
	return evm.StateDB.GetBalance(account), nil
}

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
