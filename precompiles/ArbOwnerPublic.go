//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package precompiles

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
)

// The calls to this precompile do not require the sender be a chain owner.
// For those that are, see ArbOwner
type ArbOwnerPublic struct {
	Address addr
}

func (con ArbOwnerPublic) GetAllChainOwners(c ctx, evm mech) ([]common.Address, error) {
	if err := c.burn(6 * params.SloadGas); err != nil {
		return []addr{}, err
	}
	return c.state.ChainOwners().AllMembers(), nil
}

func (con ArbOwnerPublic) IsChainOwner(c ctx, evm mech, addr addr) (bool, error) {
	if err := c.burn(3 * params.SloadGas); err != nil {
		return false, err
	}
	return c.state.ChainOwners().IsMember(addr), nil
}
