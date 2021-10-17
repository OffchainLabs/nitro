//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package precompiles

import (
	"errors"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"math/big"
)

type ArbAddressTable struct{}

func (con ArbAddressTable) AddressExists(st *state.StateDB, addr common.Address) (bool, error) {
	return false, errors.New("unimplemented")
}

func (con ArbAddressTable) Compress(st *state.StateDB, addr common.Address) ([]uint8, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbAddressTable) Decompress(buf []uint8, offset *big.Int) (common.Address, *big.Int, error) {
	return common.Address{}, nil, errors.New("unimplemented")
}

func (con ArbAddressTable) Lookup(st *state.StateDB, addr common.Address) (*big.Int, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbAddressTable) LookupIndex(st *state.StateDB, index *big.Int) (common.Address, error) {
	return common.Address{}, errors.New("unimplemented")
}

func (con ArbAddressTable) Register(st *state.StateDB, addr common.Address) (*big.Int, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbAddressTable) Size(st *state.StateDB) (*big.Int, error) {
	return nil, errors.New("unimplemented")
}
