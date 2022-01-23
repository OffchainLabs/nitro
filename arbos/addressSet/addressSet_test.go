//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package addressSet

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/offchainlabs/arbstate/arbos/burn"
	"github.com/offchainlabs/arbstate/arbos/storage"
	"github.com/offchainlabs/arbstate/util/testhelpers"
)

func TestEmptyAddressSet(t *testing.T) {
	sto := storage.NewMemoryBacked(burn.NewSystemBurner(true))
	Require(t, Initialize(sto))
	aset := OpenAddressSet(sto)

	if size(t, aset) != 0 {
		Fail(t)
	}
	if isMember(t, aset, common.Address{}) {
		Fail(t)
	}
	err := aset.Remove(common.Address{})
	Require(t, err)
	if size(t, aset) != 0 {
		Fail(t)
	}
	if isMember(t, aset, common.Address{}) {
		Fail(t)
	}
}

func TestAddressSet(t *testing.T) {
	sto := storage.NewMemoryBacked(burn.NewSystemBurner(true))
	Require(t, Initialize(sto))
	aset := OpenAddressSet(sto)

	addr1 := common.BytesToAddress(crypto.Keccak256([]byte{1})[:20])
	addr2 := common.BytesToAddress(crypto.Keccak256([]byte{2})[:20])
	addr3 := common.BytesToAddress(crypto.Keccak256([]byte{3})[:20])

	Require(t, aset.Add(addr1))
	if size(t, aset) != 1 {
		Fail(t)
	}
	Require(t, aset.Add(addr2))
	if size(t, aset) != 2 {
		Fail(t)
	}
	Require(t, aset.Add(addr1))
	if size(t, aset) != 2 {
		Fail(t)
	}
	if !isMember(t, aset, addr1) {
		Fail(t)
	}
	if !isMember(t, aset, addr2) {
		Fail(t)
	}
	if isMember(t, aset, addr3) {
		Fail(t)
	}

	Require(t, aset.Remove(addr1))
	if size(t, aset) != 1 {
		Fail(t)
	}
	if isMember(t, aset, addr1) {
		Fail(t)
	}
	if !isMember(t, aset, addr2) {
		Fail(t)
	}

	Require(t, aset.Add(addr3))
	if size(t, aset) != 2 {
		Fail(t)
	}
	Require(t, aset.Remove(addr3))
	if size(t, aset) != 1 {
		Fail(t)
	}

	Require(t, aset.Add(addr1))
	all, err := aset.AllMembers()
	Require(t, err)
	if len(all) != 2 {
		Fail(t)
	}
	if all[0] == addr1 {
		if all[1] != addr2 {
			Fail(t)
		}
	} else {
		if (all[0] != addr2) || (all[1] != addr1) {
			Fail(t)
		}
	}
}

func isMember(t *testing.T, aset *AddressSet, address common.Address) bool {
	t.Helper()
	present, err := aset.IsMember(address)
	Require(t, err)
	return present
}

func size(t *testing.T, aset *AddressSet) uint64 {
	t.Helper()
	size, err := aset.Size()
	Require(t, err)
	return size
}

func Require(t *testing.T, err error, text ...string) {
	t.Helper()
	testhelpers.RequireImpl(t, err, text...)
}

func Fail(t *testing.T, printables ...interface{}) {
	t.Helper()
	testhelpers.FailImpl(t, printables...)
}
