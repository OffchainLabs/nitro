//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package precompiles

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/arbstate/arbos"
)

type ArbOwner struct {
	Address addr
}

var UnauthorizedError = errors.New("unauthorized caller to access-controlled method")

func (con ArbOwner) AddChainOwner(c ctx, evm mech, newOwner addr) error {
	if err := c.burn(3 * params.SloadGas); err != nil { // charge less because only owner can call this
		return err
	}
	owners := arbos.OpenArbosState(evm.StateDB).ChainOwners()
	if !owners.IsMember(c.caller) {
		return UnauthorizedError
	}
	owners.Add(newOwner)
	return nil
}

func (con ArbOwner) GetAllChainOwners(c ctx, evm mech) ([]common.Address, error) {
	if err := c.burn(6 * params.SloadGas); err != nil {
		return []addr{}, err
	}
	return arbos.OpenArbosState(evm.StateDB).ChainOwners().AllMembers(), nil
}

func (con ArbOwner) IsChainOwner(c ctx, evm mech, addr addr) (bool, error) {
	if err := c.burn(3 * params.SloadGas); err != nil {
		return false, err
	}
	return arbos.OpenArbosState(evm.StateDB).ChainOwners().IsMember(addr), nil
}

func (con ArbOwner) RemoveChainOwner(c ctx, evm mech, addr addr) error {
	if err := c.burn(3 * params.SloadGas); err != nil { // charge less because only owner can call this
		return err
	}
	owners := arbos.OpenArbosState(evm.StateDB).ChainOwners()
	if !owners.IsMember(c.caller) {
		return UnauthorizedError
	}
	owners.Remove(addr)
	return nil
}

func (con ArbOwner) SetL2GasPrice(c ctx, evm mech, priceInWei huge) error {
	if err := c.burn(3 * params.SloadGas); err != nil { // charge less because only owner can call this
		return err
	}
	state := arbos.OpenArbosState(evm.StateDB)
	owners := state.ChainOwners()
	if !owners.IsMember(c.caller) {
		return UnauthorizedError
	}
	state.SetGasPriceWei(priceInWei)
	return nil
}
