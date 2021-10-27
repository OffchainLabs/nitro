//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package precompiles

import (
	"errors"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/arbstate/arbos"
	"math/big"
)

type ArbAddressTable struct {
	Address addr
}

func (con ArbAddressTable) AddressExists(c ctx, evm mech, addr addr) (bool, error) {
	if err := c.burn(params.SloadGas); err != nil {
		return false, err
	}
	return arbos.OpenArbosState(evm.StateDB).AddressTable().AddressExists(addr), nil
}

func (con ArbAddressTable) Compress(c ctx, evm mech, addr addr) ([]uint8, error) {
	if err := c.burn(params.SloadGas); err != nil {
		return nil, err
	}
	return arbos.OpenArbosState(evm.StateDB).AddressTable().Compress(addr), nil
}

func (con ArbAddressTable) Decompress(c ctx, evm mech, buf []uint8, offset huge) (addr, huge, error) {
	if err := c.burn(params.SloadGas); err != nil {
		return addr{}, nil, err
	}
	if !offset.IsInt64() {
		return addr{}, nil, errors.New("invalid offset in ArbAddressTable.Decompress")
	}
	ioffset := offset.Int64()
	if ioffset > int64(len(buf)) {
		return addr{}, nil, errors.New("invalid offset in ArbAddressTable.Decompress")
	}
	result, nbytes, err := arbos.OpenArbosState(evm.StateDB).AddressTable().Decompress(buf[ioffset:])
	return result, big.NewInt(int64(nbytes)), err
}

func (con ArbAddressTable) Lookup(c ctx, evm mech, addr addr) (huge, error) {
	if err := c.burn(params.SloadGas); err != nil {
		return nil, err
	}
	result, exists := arbos.OpenArbosState(evm.StateDB).AddressTable().Lookup(addr)
	if !exists {
		return nil, errors.New("address does not exist in AddressTable")
	}
	return big.NewInt(int64(result)), nil
}

func (con ArbAddressTable) LookupIndex(c ctx, evm mech, index huge) (addr, error) {
	if err := c.burn(params.SloadGas); err != nil {
		return addr{}, err
	}
	if !index.IsUint64() {
		return addr{}, errors.New("invalid index in ArbAddressTable.LookupIndex")
	}
	result, exists := arbos.OpenArbosState(evm.StateDB).AddressTable().LookupIndex(index.Uint64())
	if !exists {
		return addr{}, errors.New("index does not exist in AddressTable")
	}
	return result, nil
}

func (con ArbAddressTable) Register(c ctx, evm mech, addr addr) (huge, error) {
	if err := c.burn(params.SstoreSetGas); err != nil {
		return nil, err
	}
	return big.NewInt(int64(arbos.OpenArbosState(evm.StateDB).AddressTable().Register(addr))), nil
}

func (con ArbAddressTable) Size(c ctx, evm mech) (huge, error) {
	if err := c.burn(params.SloadGas); err != nil {
		return nil, err
	}
	return big.NewInt(int64(arbos.OpenArbosState(evm.StateDB).AddressTable().Size())), nil
}
