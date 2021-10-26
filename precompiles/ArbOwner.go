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

func (con ArbOwner) AddChainOwner(caller addr, evm mech, newOwner addr) error {
	owners := arbos.OpenArbosState(evm.StateDB).ChainOwners()
	if !owners.IsMember(caller) {
		return UnauthorizedError
	}
	owners.Add(newOwner)
	return nil
}

func (con ArbOwner) AddChainOwnerGasCost(newOwner addr) uint64 {
	return 3 * params.SstoreSetGas
}

func (con ArbOwner) GetAllChainOwners(caller addr, evm mech) ([]common.Address, error) {
	return arbos.OpenArbosState(evm.StateDB).ChainOwners().AllMembers(), nil
}

func (con ArbOwner) GetAllChainOwnersGasCost() uint64 {
	return 5 * params.SstoreSetGas
}

func (con ArbOwner) IsChainOwner(caller addr, evm mech, addr addr) (bool, error) {
	return arbos.OpenArbosState(evm.StateDB).ChainOwners().IsMember(addr), nil
}

func (con ArbOwner) IsChainOwnerGasCost(addr addr) uint64 {
	return 3 * params.SstoreSetGas
}

func (con ArbOwner) RemoveChainOwner(caller addr, evm mech, addr addr) error {
	owners := arbos.OpenArbosState(evm.StateDB).ChainOwners()
	if !owners.IsMember(caller) {
		return UnauthorizedError
	}
	owners.Remove(addr)
	return nil
}

func (con ArbOwner) RemoveChainOwnerGasCost(addr addr) uint64 {
	return 3 * params.SstoreSetGas
}
