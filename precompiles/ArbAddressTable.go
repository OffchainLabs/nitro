//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package precompiles

import (
	"errors"
)

type ArbAddressTable struct{}

func (con ArbAddressTable) AddressExists(caller addr, st *stateDB, addr addr) (bool, error) {
	return false, errors.New("unimplemented")
}

func (con ArbAddressTable) AddressExistsGasCost(addr addr) uint64 {
	return 0
}

func (con ArbAddressTable) Compress(caller addr, st *stateDB, addr addr) ([]uint8, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbAddressTable) CompressGasCost(addr addr) uint64 {
	return 0
}

func (con ArbAddressTable) Decompress(caller addr, buf []uint8, offset huge) (addr, huge, error) {
	return addr{}, nil, errors.New("unimplemented")
}

func (con ArbAddressTable) DecompressGasCost(buf []uint8, offset huge) uint64 {
	return 0
}

func (con ArbAddressTable) Lookup(caller addr, st *stateDB, addr addr) (huge, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbAddressTable) LookupGasCost(addr addr) uint64 {
	return 0
}

func (con ArbAddressTable) LookupIndex(caller addr, st *stateDB, index huge) (addr, error) {
	return addr{}, errors.New("unimplemented")
}

func (con ArbAddressTable) LookupIndexGasCost(index huge) uint64 {
	return 0
}

func (con ArbAddressTable) Register(caller addr, st *stateDB, addr addr) (huge, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbAddressTable) RegisterGasCost(addr addr) uint64 {
	return 0
}

func (con ArbAddressTable) Size(caller addr, st *stateDB) (huge, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbAddressTable) SizeGasCost() uint64 {
	return 0
}
