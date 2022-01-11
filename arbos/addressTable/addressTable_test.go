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
		Fail(t)
	}

	_, found, err := atab.Lookup(common.Address{})
	Require(t, err)
	if found {
		Fail(t)
	}
	_, found, err = atab.LookupIndex(0)
	Require(t, err)
	if found {
		Fail(t)
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
		Fail(t)
	}

	atab = Open(sto)
	if size(t, atab) != 1 {
		Fail(t)
	}
	idx, found, err := atab.Lookup(addr)
	Require(t, err)
	if !found {
		Fail(t)
	}
	if idx != 0 {
		Fail(t)
	}

	_, found, err = atab.Lookup(common.Address{})
	Require(t, err)
	if found {
		Fail(t)
	}

	addr2, found, err := atab.LookupIndex(0)
	Require(t, err)
	if !found {
		Fail(t)
	}
	if addr2 != addr {
		Fail(t)
	}

	_, found, err = atab.LookupIndex(1)
	Require(t, err)
	if found {
		Fail(t)
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
		Fail(t)
	}
	if !bytes.Equal(addr.Bytes(), res[1:]) {
		Fail(t)
	}

	dec, nbytes, err := atab.Decompress(res)
	if err != nil {
		Fail(t, err)
	}
	if nbytes != 21 {
		Fail(t, nbytes)
	}
	if dec != addr {
		Fail(t)
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
		Fail(t, len(res))
	}

	dec, nbytes, err := atab.Decompress(res)
	if err != nil {
		Fail(t, err)
	}
	if nbytes > 9 {
		Fail(t, nbytes)
	}
	if dec != addr {
		Fail(t)
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
