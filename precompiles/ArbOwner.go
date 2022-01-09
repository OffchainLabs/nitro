//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package precompiles

import (
	"github.com/ethereum/go-ethereum/common"
)

// All calls to this precompile are authorized by the OwnerPrecompile wrapper,
// which ensures only a chain owner can access these methods. For methods that
// are safe for non-owners to call, see ArbOwnerOld
type ArbOwner struct {
	Address addr
}

func (con ArbOwner) AddChainOwner(c ctx, evm mech, newOwner addr) error {
	owners := c.state.ChainOwners()
	owners.Add(newOwner)
	return nil
}

func (con ArbOwner) GetAllChainOwners(c ctx, evm mech) ([]common.Address, error) {
	return c.state.ChainOwners().AllMembers(), nil
}

func (con ArbOwner) IsChainOwner(c ctx, evm mech, addr addr) (bool, error) {
	return c.state.ChainOwners().IsMember(addr), nil
}

func (con ArbOwner) RemoveChainOwner(c ctx, evm mech, addr addr) error {
	owners := c.state.ChainOwners()
	owners.Remove(addr)
	return nil
}

func (con ArbOwner) SetL2GasPrice(c ctx, evm mech, priceInWei huge) error {
	c.state.SetGasPriceWei(priceInWei)
	return nil
}
