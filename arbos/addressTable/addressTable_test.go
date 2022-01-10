//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package addressTable

import (
	"bytes"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/offchainlabs/arbstate/arbos/burn"
	"github.com/offchainlabs/arbstate/arbos/storage"
	"github.com/offchainlabs/arbstate/util/testhelpers"
)

func TestAddressTableInit(t *testing.T) {
	sto := storage.NewMemoryBacked(&burn.SystemBurner{})
	Initialize(sto)
	atab := Open(sto)
	if size(t, atab) != 0 {
		t.Fatal()
	}

	_, found, err := atab.Lookup(common.Address{})
	Require(t, err)
	if found {
		t.Fatal()
	}
	_, found, err = atab.LookupIndex(0)
	Require(t, err)
	if found {
		t.Fatal()
	}
}

func TestAddressTable1(t *testing.T) {
	sto := storage.NewMemoryBacked(&burn.SystemBurner{})
	Initialize(sto)
	atab := Open(sto)
	addr := common.BytesToAddress(crypto.Keccak256([]byte{})[:20])
	_, err := atab.Register(addr)
	Require(t, err)
	if size(t, atab) != 1 {
		t.Fatal()
	}

	atab = Open(sto)
	if size(t, atab) != 1 {
		t.Fatal()
	}
	idx, found, err := atab.Lookup(addr)
	Require(t, err)
	if !found {
		t.Fatal()
	}
	if idx != 0 {
		t.Fatal()
	}

	_, found, err = atab.Lookup(common.Address{})
	Require(t, err)
	if found {
		t.Fatal()
	}

	addr2, found, err := atab.LookupIndex(0)
	Require(t, err)
	if !found {
		t.Fatal()
	}
	if addr2 != addr {
		t.Fatal()
	}

	_, found, err = atab.LookupIndex(1)
	Require(t, err)
	if found {
		t.Fatal()
	}
}

func TestAddressTableCompressNotInTable(t *testing.T) {
	sto := storage.NewMemoryBacked(&burn.SystemBurner{})
	Initialize(sto)
	atab := Open(sto)
	addr := common.BytesToAddress(crypto.Keccak256([]byte{})[:20])

	res, err := atab.Compress(addr)
	Require(t, err)
	if len(res) != 21 {
		t.Fatal()
	}
	if !bytes.Equal(addr.Bytes(), res[1:]) {
		t.Fatal()
	}

	dec, nbytes, err := atab.Decompress(res)
	if err != nil {
		t.Fatal(err)
	}
	if nbytes != 21 {
		t.Fatal(nbytes)
	}
	if dec != addr {
		t.Fatal()
	}
}

func TestAddressTableCompressInTable(t *testing.T) {
	sto := storage.NewMemoryBacked(&burn.SystemBurner{})
	Initialize(sto)
	atab := Open(sto)
	addr := common.BytesToAddress(crypto.Keccak256([]byte{})[:20])

	_, err := atab.Register(addr)
	Require(t, err)

	res, err := atab.Compress(addr)
	Require(t, err)
	if len(res) > 9 {
		t.Fatal(len(res))
	}

	dec, nbytes, err := atab.Decompress(res)
	if err != nil {
		t.Fatal(err)
	}
	if nbytes > 9 {
		t.Fatal(nbytes)
	}
	if dec != addr {
		t.Fatal()
	}
}

func size(t *testing.T, atab *AddressTable) uint64 {
	size, err := atab.Size()
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
