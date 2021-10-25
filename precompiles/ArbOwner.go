//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package precompiles

import (
	"errors"
	"github.com/ethereum/go-ethereum/common"
)

type ArbOwner struct {
	Address addr
}

func (con ArbOwner) AddChainOwner(caller addr, evm mech, newOwner addr) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) AddChainOwnerGasCost(newOwner addr) uint64 {
	return 0
}

func (con ArbOwner) GetAllChainOwners(caller addr, evm mech) ([]common.Address, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbOwner) GetAllChainOwnersGasCost() uint64 {
	return 0
}

func (con ArbOwner) IsChainOwner(caller addr, evm mech, addr addr) (bool, error) {
	return false, errors.New("unimplemented")
}

func (con ArbOwner) IsChainOwnerGasCost(addr addr) uint64 {
	return 0
}

func (con ArbOwner) RemoveChainOwner(caller addr, evm mech, addr addr) error {
	return errors.New("unimplemented")
}

func (con ArbOwner) RemoveChainOwnerGasCost(addr addr) uint64 {
	return 0
}
