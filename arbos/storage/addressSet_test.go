//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package storage

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"testing"
)

func TestEmptyAddressSet(t *testing.T) {
	sto := NewMemoryBacked()
	InitializeAddressSet(sto)
	aset := OpenAddressSet(sto)

	if aset.Size() != 0 {
		t.Fatal()
	}
	if aset.IsMember(common.Address{}) {
		t.Fatal()
	}
	aset.Remove(common.Address{})
	if aset.Size() != 0 {
		t.Fatal()
	}
	if aset.IsMember(common.Address{}) {
		t.Fatal()
	}
}

func TestAddressSet(t *testing.T) {
	sto := NewMemoryBacked()
	InitializeAddressSet(sto)
	aset := OpenAddressSet(sto)

	addr1 := common.BytesToAddress(crypto.Keccak256([]byte{1})[:20])
	addr2 := common.BytesToAddress(crypto.Keccak256([]byte{2})[:20])
	addr3 := common.BytesToAddress(crypto.Keccak256([]byte{3})[:20])

	aset.Add(addr1)
	if aset.Size() != 1 {
		t.Fatal()
	}
	aset.Add(addr2)
	if aset.Size() != 2 {
		t.Fatal()
	}
	aset.Add(addr1)
	if aset.Size() != 2 {
		t.Fatal()
	}
	if !aset.IsMember(addr1) {
		t.Fatal()
	}
	if !aset.IsMember(addr2) {
		t.Fatal()
	}
	if aset.IsMember(addr3) {
		t.Fatal()
	}

	aset.Remove(addr1)
	if aset.Size() != 1 {
		t.Fatal()
	}
	if aset.IsMember(addr1) {
		t.Fatal()
	}
	if !aset.IsMember(addr2) {
		t.Fatal()
	}

	aset.Add(addr3)
	if aset.Size() != 2 {
		t.Fatal()
	}
	aset.Remove(addr3)
	if aset.Size() != 1 {
		t.Fatal()
	}

	aset.Add(addr1)
	all := aset.AllMembers()
	if len(all) != 2 {
		t.Fatal()
	}
	if all[0] == addr1 {
		if all[1] != addr2 {
			t.Fatal()
		}
	} else {
		if (all[0] != addr2) || (all[1] != addr1) {
			t.Fatal()
		}
	}
}
