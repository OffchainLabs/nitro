//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package precompiles

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
)

// All calls to this precompile are authorized by the OwnerPrecompile wrapper,
// which ensures only a chain owner can access these methods. For methods that
// are safe for non-owners to call, see ArbOwnerOld
type ArbOwner struct {
	Address addr
}

func (con ArbOwner) AddChainOwner(c ctx, evm mech, newOwner addr) error {
	return c.state.ChainOwners().Add(newOwner)
}

func (con ArbOwner) GetAllChainOwners(c ctx, evm mech) ([]common.Address, error) {
	return c.state.ChainOwners().AllMembers()
}

func (con ArbOwner) IsChainOwner(c ctx, evm mech, addr addr) (bool, error) {
	return c.state.ChainOwners().IsMember(addr)
}

func (con ArbOwner) RemoveChainOwner(c ctx, evm mech, addr addr) error {
	member, _ := con.IsChainOwner(c, evm, addr)
	if !member {
		return errors.New("tried to remove non-owner")
	}
	return c.state.ChainOwners().Remove(addr)
}

func (con ArbOwner) SetL1GasPriceEstimate(c ctx, evm mech, priceInWei huge) error {
	return c.state.L1PricingState().SetL1GasPriceEstimateWei(priceInWei)
}

func (con ArbOwner) SetL2GasPrice(c ctx, evm mech, priceInWei huge) error {
	return c.state.SetGasPriceWei(priceInWei)
}
