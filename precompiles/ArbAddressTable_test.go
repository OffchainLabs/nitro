// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package precompiles

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

func TestArbAddressTableInit(t *testing.T) {
	evm := newMockEVMForTesting()
	atab := ArbAddressTable{}
	context := testContext(common.Address{}, evm)

	size, err := atab.Size(context, evm)
	Require(t, err)
	if (!size.IsInt64()) || (size.Int64() != 0) {
		t.Fatal()
	}

	_, shouldErr := atab.Lookup(context, evm, common.Address{})
	if shouldErr == nil {
		t.Fatal()
	}

	_, shouldErr = atab.LookupIndex(context, evm, big.NewInt(0))
	if shouldErr == nil {
		t.Fatal()
	}
}

func TestAddressTable1(t *testing.T) {
	evm := newMockEVMForTesting()
	atab := ArbAddressTable{}
	context := testContext(common.Address{}, evm)

	addr := common.BytesToAddress(crypto.Keccak256([]byte{})[:20])

	// register addr
	slot, err := atab.Register(context, evm, addr)
	Require(t, err)
	if (!slot.IsInt64()) || (slot.Int64() != 0) {
		t.Fatal()
	}

	// verify Size() is 1
	size, err := atab.Size(context, evm)
	Require(t, err)
	if (!size.IsInt64()) || (size.Int64() != 1) {
		t.Fatal()
	}

	// verify Lookup of addr returns 0
	index, err := atab.Lookup(context, evm, addr)
	Require(t, err)
	if (!index.IsInt64()) || (index.Int64() != 0) {
		t.Fatal()
	}

	// verify Lookup of nonexistent address returns error
	_, shouldErr := atab.Lookup(context, evm, common.Address{})
	if shouldErr == nil {
		t.Fatal()
	}

	// verify LookupIndex of 0 returns addr
	addr2, err := atab.LookupIndex(context, evm, big.NewInt(0))
	Require(t, err)
	if addr2 != addr {
		t.Fatal()
	}

	// verify LookupIndex of 1 returns error
	_, shouldErr = atab.LookupIndex(context, evm, big.NewInt(1))
	if shouldErr == nil {
		t.Fatal()
	}
}

func TestAddressTableCompressNotInTable(t *testing.T) {
	evm := newMockEVMForTesting()
	atab := ArbAddressTable{}
	context := testContext(common.Address{}, evm)

	addr := common.BytesToAddress(crypto.Keccak256([]byte{})[:20])

	// verify that compressing addr produces the 21-byte format
	res, err := atab.Compress(context, evm, addr)
	Require(t, err)
	if len(res) != 21 {
		t.Fatal()
	}
	if !bytes.Equal(addr.Bytes(), res[1:]) {
		t.Fatal()
	}

	// verify that decompressing res consumes 21 bytes and returns the original addr
	dec, nbytes, err := atab.Decompress(context, evm, res, big.NewInt(0))
	Require(t, err)
	if (!nbytes.IsInt64()) || (nbytes.Int64() != 21) {
		t.Fatal()
	}
	if dec != addr {
		t.Fatal()
	}
}

func TestAddressTableCompressInTable(t *testing.T) {
	evm := newMockEVMForTesting()
	atab := ArbAddressTable{}
	context := testContext(common.Address{}, evm)

	addr := common.BytesToAddress(crypto.Keccak256([]byte{})[:20])

	// Register addr
	if _, err := atab.Register(context, evm, addr); err != nil {
		t.Fatal(err)
	}

	// verify that compressing addr yields the <= 9 byte format
	res, err := atab.Compress(context, evm, addr)
	Require(t, err)
	if len(res) > 9 {
		Fail(t, len(res))
	}

	// add a byte of padding at the beginning and end of res
	res = append([]byte{99}, res...)
	res = append(res, 33)

	// verify that decompressing res consumes all but two bytes of res and produces addr
	dec, nbytes, err := atab.Decompress(context, evm, res, big.NewInt(1))
	Require(t, err)
	if (!nbytes.IsInt64()) || (nbytes.Int64()+2 != int64(len(res))) {
		Fail(t)
	}
	if dec != addr {
		Fail(t)
	}
}

func newMockEVMForTesting() *vm.EVM {
	return newMockEVMForTestingWithVersion(nil)
}

func newMockEVMForTestingWithVersionAndRunMode(version *uint64, runMode types.MessageRunMode) *vm.EVM {
	evm := newMockEVMForTestingWithVersion(version)
	evm.ProcessingHook = arbos.NewTxProcessor(evm, types.Message{TxRunMode: runMode})
	return evm
}

func newMockEVMForTestingWithVersion(version *uint64) *vm.EVM {
	chainConfig := params.ArbitrumDevTestChainConfig()
	if version != nil {
		chainConfig.ArbitrumChainParams.InitialArbOSVersion = *version
	}
	_, statedb := arbosState.NewArbosMemoryBackedArbOSState()
	context := vm.BlockContext{
		BlockNumber: big.NewInt(0),
		GasLimit:    ^uint64(0),
		Time:        0,
	}
	evm := vm.NewEVM(context, vm.TxContext{}, statedb, chainConfig, vm.Config{})
	evm.ProcessingHook = &arbos.TxProcessor{}
	return evm
}

func Require(t *testing.T, err error, printables ...interface{}) {
	t.Helper()
	testhelpers.RequireImpl(t, err, printables...)
}

func Fail(t *testing.T, printables ...interface{}) {
	t.Helper()
	testhelpers.FailImpl(t, printables...)
}
