//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package storage

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"testing"
)

func TestEmptyHashSet(t *testing.T) {
	sto := NewMemoryBacked()
	InitializeHashSet(sto)
	aset := OpenHashSet(sto)

	if aset.Size() != 0 {
		t.Fatal()
	}
	if aset.IsMember(common.Hash{}) {
		t.Fatal()
	}
	aset.Remove(common.Hash{})
	if aset.Size() != 0 {
		t.Fatal()
	}
	if aset.IsMember(common.Hash{}) {
		t.Fatal()
	}
}

func TestHashSet(t *testing.T) {
	sto := NewMemoryBacked()
	InitializeHashSet(sto)
	aset := OpenHashSet(sto)

	hash1 := crypto.Keccak256Hash([]byte{1})
	hash2 := crypto.Keccak256Hash([]byte{2})
	hash3 := crypto.Keccak256Hash([]byte{3})

	aset.Add(hash1)
	if aset.Size() != 1 {
		t.Fatal()
	}
	aset.Add(hash2)
	if aset.Size() != 2 {
		t.Fatal()
	}
	aset.Add(hash1)
	if aset.Size() != 2 {
		t.Fatal()
	}
	if !aset.IsMember(hash1) {
		t.Fatal()
	}
	if !aset.IsMember(hash2) {
		t.Fatal()
	}
	if aset.IsMember(hash3) {
		t.Fatal()
	}

	aset.Remove(hash1)
	if aset.Size() != 1 {
		t.Fatal()
	}
	if aset.IsMember(hash1) {
		t.Fatal()
	}
	if !aset.IsMember(hash2) {
		t.Fatal()
	}

	aset.Add(hash3)
	if aset.Size() != 2 {
		t.Fatal()
	}
	aset.Remove(hash3)
	if aset.Size() != 1 {
		t.Fatal()
	}

	aset.Add(hash1)
	all := aset.AllMembers()
	if len(all) != 2 {
		t.Fatal()
	}
	if all[0] == hash1 {
		if all[1] != hash2 {
			t.Fatal()
		}
	} else {
		if (all[0] != hash2) || (all[1] != hash1) {
			t.Fatal()
		}
	}
}
