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

func (con ArbAddressTable) AddressExists(caller common.Address, st *state.StateDB, addr common.Address) (bool, error) {
	return false, errors.New("unimplemented")
}

func (con ArbAddressTable) Compress(caller common.Address, st *state.StateDB, addr common.Address) ([]uint8, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbAddressTable) Decompress(
	caller common.Address,
	buf []uint8,
	offset *big.Int,
) (common.Address, *big.Int, error) {
	return common.Address{}, nil, errors.New("unimplemented")
}

func (con ArbAddressTable) Lookup(caller common.Address, st *state.StateDB, addr common.Address) (*big.Int, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbAddressTable) LookupIndex(
	caller common.Address,
	st *state.StateDB,
	index *big.Int,
) (common.Address, error) {
	return common.Address{}, errors.New("unimplemented")
}

func (con ArbAddressTable) Register(caller common.Address, st *state.StateDB, addr common.Address) (*big.Int, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbAddressTable) Size(caller common.Address, st *state.StateDB) (*big.Int, error) {
	return nil, errors.New("unimplemented")
}
