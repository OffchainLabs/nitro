//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package addressTable

import (
	"bytes"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/offchainlabs/arbstate/arbos/storage"
	"testing"
)

func TestAddressTableInit(t *testing.T) {
	sto := storage.NewMemoryBacked()
	Initialize(sto)
	atab := Open(sto)
	if atab.Size() != 0 {
		t.Fatal()
	}

	_, found := atab.Lookup(common.Address{})
	if found {
		t.Fatal()
	}
	_, found = atab.LookupIndex(0)
	if found {
		t.Fatal()
	}
}

func TestAddressTable1(t *testing.T) {
	sto := storage.NewMemoryBacked()
	Initialize(sto)
	atab := Open(sto)
	addr := common.BytesToAddress(crypto.Keccak256([]byte{})[:20])
	atab.Register(addr)
	if atab.Size() != 1 {
		t.Fatal()
	}

	atab = Open(sto)
	if atab.Size() != 1 {
		t.Fatal()
	}
	idx, found := atab.Lookup(addr)
	if !found {
		t.Fatal()
	}
	if idx != 0 {
		t.Fatal()
	}

	_, found = atab.Lookup(common.Address{})
	if found {
		t.Fatal()
	}

	addr2, found := atab.LookupIndex(0)
	if !found {
		t.Fatal()
	}
	if addr2 != addr {
		t.Fatal()
	}

	_, found = atab.LookupIndex(1)
	if found {
		t.Fatal()
	}
}

func TestAddressTableCompressNotInTable(t *testing.T) {
	sto := storage.NewMemoryBacked()
	Initialize(sto)
	atab := Open(sto)
	addr := common.BytesToAddress(crypto.Keccak256([]byte{})[:20])

	res := atab.Compress(addr)
	if len(res) != 21 {
		t.Fatal()
	}
	if res[0] != RLPPrefixFor20Bytes {
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
	sto := storage.NewMemoryBacked()
	Initialize(sto)
	atab := Open(sto)
	addr := common.BytesToAddress(crypto.Keccak256([]byte{})[:20])

	_ = atab.Register(addr)

	res := atab.Compress(addr)
	if len(res) != 9 {
		t.Fatal()
	}
	if res[0] != RLPPrefixFor8Bytes {
		t.Fatal()
	}
	if !bytes.Equal(make([]byte, 8), res[1:]) {
		t.Fatal()
	}

	dec, nbytes, err := atab.Decompress(res)
	if err != nil {
		t.Fatal(err)
	}
	if nbytes != 9 {
		t.Fatal(nbytes)
	}
	if dec != addr {
		t.Fatal()
	}
}
