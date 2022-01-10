//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package precompiles

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/params"
)

type ArbAddressTable struct {
	Address addr
}

func (con ArbAddressTable) AddressExists(c ctx, evm mech, addr addr) (bool, error) {
	return c.state.AddressTable().AddressExists(addr)
}

func (con ArbAddressTable) Compress(c ctx, evm mech, addr addr) ([]uint8, error) {
	return c.state.AddressTable().Compress(addr)
}

func (con ArbAddressTable) Decompress(c ctx, evm mech, buf []uint8, offset huge) (addr, huge, error) {
	if !offset.IsInt64() {
		return addr{}, nil, errors.New("invalid offset in ArbAddressTable.Decompress")
	}
	ioffset := offset.Int64()
	if ioffset > int64(len(buf)) {
		return addr{}, nil, errors.New("invalid offset in ArbAddressTable.Decompress")
	}
	result, nbytes, err := c.state.AddressTable().Decompress(buf[ioffset:])
	return result, big.NewInt(int64(nbytes)), err
}

func (con ArbAddressTable) Lookup(c ctx, evm mech, addr addr) (huge, error) {
	result, exists, err := c.state.AddressTable().Lookup(addr)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.New("address does not exist in AddressTable")
	}
	return big.NewInt(int64(result)), nil
}

func (con ArbAddressTable) LookupIndex(c ctx, evm mech, index huge) (addr, error) {
	if !index.IsUint64() {
		return addr{}, errors.New("invalid index in ArbAddressTable.LookupIndex")
	}
	result, exists, err := c.state.AddressTable().LookupIndex(index.Uint64())
	if err != nil {
		return addr{}, err
	}
	if !exists {
		return addr{}, errors.New("index does not exist in AddressTable")
	}
	return result, nil
}

func (con ArbAddressTable) Register(c ctx, evm mech, addr addr) (huge, error) {
	if err := c.burn(params.SstoreSetGas); err != nil {
		return nil, err
	}
	slot, err := c.state.AddressTable().Register(addr)
	return big.NewInt(int64(slot)), err
}

func (con ArbAddressTable) Size(c ctx, evm mech) (huge, error) {
	if err := c.burn(params.SloadGas); err != nil {
		return nil, err
	}
	size, err := c.state.AddressTable().Size()
	return big.NewInt(int64(size)), err
}
