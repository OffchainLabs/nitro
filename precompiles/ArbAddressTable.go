//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package precompiles

import (
	"errors"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/arbstate/arbos"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
)

type ArbAddressTable struct{}

func (con ArbAddressTable) AddressExists(caller common.Address, st *state.StateDB, addr common.Address) (bool, error) {
	return arbos.OpenArbosState(st).AddressTable().AddressExists(addr), nil
}

func (con ArbAddressTable) AddressExistsGasCost(addr common.Address) uint64 {
	return params.SloadGas
}

func (con ArbAddressTable) Compress(caller common.Address, st *state.StateDB, addr common.Address) ([]uint8, error) {
	return arbos.OpenArbosState(st).AddressTable().Compress(addr), nil
}

func (con ArbAddressTable) CompressGasCost(addr common.Address) uint64 {
	return params.SloadGas
}

func (con ArbAddressTable) Decompress(
	caller common.Address,
	st *state.StateDB,
	buf []uint8,
	offset *big.Int,
) (common.Address, *big.Int, error) {
	if !offset.IsInt64() {
		return common.Address{}, nil, errors.New("invalid offset in ArbAddressTable.Decompress")
	}
	ioffset := offset.Int64()
	if ioffset > int64(len(buf)) {
		return common.Address{}, nil, errors.New("invalid offset in ArbAddressTable.Decompress")
	}
	result, nbytes, err := arbos.OpenArbosState(st).AddressTable().Decompress(buf[ioffset:])
	return result, big.NewInt(int64(nbytes)), err
}

func (con ArbAddressTable) DecompressGasCost(buf []uint8, offset *big.Int) uint64 {
	return params.SloadGas
}

func (con ArbAddressTable) Lookup(caller common.Address, st *state.StateDB, addr common.Address) (*big.Int, error) {
	result, exists := arbos.OpenArbosState(st).AddressTable().Lookup(addr)
	if !exists {
		return nil, errors.New("address does not exist in AddressTable")
	}
	return big.NewInt(int64(result)), nil
}

func (con ArbAddressTable) LookupGasCost(addr common.Address) uint64 {
	return params.SloadGas
}

func (con ArbAddressTable) LookupIndex(
	caller common.Address,
	st *state.StateDB,
	index *big.Int,
) (common.Address, error) {
	if !index.IsUint64() {
		return common.Address{}, errors.New("invalid index in ArbAddressTable.LookupIndex")
	}
	result, exists := arbos.OpenArbosState(st).AddressTable().LookupIndex(index.Uint64())
	if !exists {
		return common.Address{}, errors.New("index does not exist in AddressTable")
	}
	return result, nil
}

func (con ArbAddressTable) LookupIndexGasCost(index *big.Int) uint64 {
	return params.SloadGas
}

func (con ArbAddressTable) Register(caller common.Address, st *state.StateDB, addr common.Address) (*big.Int, error) {
	return big.NewInt(int64(arbos.OpenArbosState(st).AddressTable().Register(addr))), nil
}

func (con ArbAddressTable) RegisterGasCost(addr common.Address) uint64 {
	return params.SstoreSetGas
}

func (con ArbAddressTable) Size(caller common.Address, st *state.StateDB) (*big.Int, error) {
	return big.NewInt(int64(arbos.OpenArbosState(st).AddressTable().Size())), nil
}

func (con ArbAddressTable) SizeGasCost() uint64 {
	return params.SloadGas
}
