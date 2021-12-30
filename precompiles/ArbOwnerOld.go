//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package precompiles

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/arbstate/arbos"
)

// The calls to this precompile are not authenticated.
// For those that are, see ArbOwner
type ArbOwnerOld struct {
	Address addr
}

func (con ArbOwnerOld) GetAllChainOwners(c ctx, evm mech) ([]common.Address, error) {
	if err := c.burn(6 * params.SloadGas); err != nil {
		return []addr{}, err
	}
	return arbos.OpenArbosState(evm.StateDB).ChainOwners().AllMembers(), nil
}

func (con ArbOwnerOld) IsChainOwner(c ctx, evm mech, addr addr) (bool, error) {
	if err := c.burn(3 * params.SloadGas); err != nil {
		return false, err
	}
	return arbos.OpenArbosState(evm.StateDB).ChainOwners().IsMember(addr), nil
}
