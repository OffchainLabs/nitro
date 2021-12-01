//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package precompiles

import (
	"bytes"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"math/big"
	"testing"
)

func TestArbAddressTableInit(t *testing.T) {
	st := newMockEVMForTesting(t)
	atab := ArbAddressTable{}
	context := testContext(common.Address{})

	sz, err := atab.Size(context, st)
	if err != nil {
		t.Fatal(err)
	}
	if (!sz.IsInt64()) || (sz.Int64() != 0) {
		t.Fatal()
	}

	_, err = atab.Lookup(context, st, common.Address{})
	if err == nil {
		t.Fatal()
	}

	_, err = atab.LookupIndex(context, st, big.NewInt(0))
	if err == nil {
		t.Fatal()
	}
}

func TestAddressTable1(t *testing.T) {
	st := newMockEVMForTesting(t)
	atab := ArbAddressTable{}
	context := testContext(common.Address{})

	addr := common.BytesToAddress(crypto.Keccak256([]byte{})[:20])

	// register addr
	slot, err := atab.Register(context, st, addr)
	if err != nil {
		t.Fatal(err)
	}
	if (!slot.IsInt64()) || (slot.Int64() != 0) {
		t.Fatal()
	}

	// verify Size() is 1
	sz, err := atab.Size(context, st)
	if err != nil {
		t.Fatal(err)
	}
	if (!sz.IsInt64()) || (sz.Int64() != 1) {
		t.Fatal()
	}

	// verify Lookup of addr returns 0
	index, err := atab.Lookup(context, st, addr)
	if err != nil {
		t.Fatal(err)
	}
	if (!index.IsInt64()) || (index.Int64() != 0) {
		t.Fatal()
	}

	// verify Lookup of nonexistent address returns error
	_, err = atab.Lookup(context, st, common.Address{})
	if err == nil {
		t.Fatal()
	}

	// verify LookupIndex of 0 returns addr
	addr2, err := atab.LookupIndex(context, st, big.NewInt(0))
	if err != nil {
		t.Fatal(err)
	}
	if addr2 != addr {
		t.Fatal()
	}

	// verify LookupIndex of 1 returns error
	_, err = atab.LookupIndex(context, st, big.NewInt(1))
	if err == nil {
		t.Fatal()
	}
}

func TestAddressTableCompressNotInTable(t *testing.T) {
	st := newMockEVMForTesting(t)
	atab := ArbAddressTable{}
	context := testContext(common.Address{})

	addr := common.BytesToAddress(crypto.Keccak256([]byte{})[:20])

	// verify that compressing addr produces the 21-byte format
	res, err := atab.Compress(context, st, addr)
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 21 {
		t.Fatal()
	}
	if !bytes.Equal(addr.Bytes(), res[1:]) {
		t.Fatal()
	}

	// verify that decompressing res consumes 21 bytes and returns the original addr
	dec, nbytes, err := atab.Decompress(context, st, res, big.NewInt(0))
	if err != nil {
		t.Fatal(err)
	}
	if (!nbytes.IsInt64()) || (nbytes.Int64() != 21) {
		t.Fatal()
	}
	if dec != addr {
		t.Fatal()
	}
}

func TestAddressTableCompressInTable(t *testing.T) {
	st := newMockEVMForTesting(t)
	atab := ArbAddressTable{}
	context := testContext(common.Address{})

	addr := common.BytesToAddress(crypto.Keccak256([]byte{})[:20])

	// Register addr
	if _, err := atab.Register(context, st, addr); err != nil {
		t.Fatal(err)
	}

	// verify that compressing addr yields the <= 9 byte format
	res, err := atab.Compress(context, st, addr)
	if err != nil {
		t.Fatal(err)
	}
	if len(res) > 9 {
		t.Fatal(len(res))
	}

	// add a byte of padding at the beginning and end of res
	res = append([]byte{99}, res...)
	res = append(res, 33)

	// verify that decompressing res consumes all but two bytes of res and produces addr
	dec, nbytes, err := atab.Decompress(context, st, res, big.NewInt(1))
	if err != nil {
		t.Fatal(err)
	}
	if (!nbytes.IsInt64()) || (nbytes.Int64()+2 != int64(len(res))) {
		t.Fatal()
	}
	if dec != addr {
		t.Fatal()
	}
}

func newMockEVMForTesting(t *testing.T) *vm.EVM {
	raw := rawdb.NewMemoryDatabase()
	db := state.NewDatabase(raw)
	statedb, err := state.New(common.Hash{}, db, nil)
	if err != nil {
		t.Fatal(err)
	}
	return &vm.EVM{
		StateDB: statedb,
	}
}
