// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/nitro/blob/master/LICENSE.md

package programs

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/arbitrum/multigas"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	gethParams "github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbcompress"
	"github.com/offchainlabs/nitro/arbos/util"
)

type testBurner struct {
	gasSupplied uint64
	gasUsed     multigas.MultiGas
}

func newTestBurner(gasSupplied uint64) *testBurner {
	return &testBurner{gasSupplied: gasSupplied}
}

func (b *testBurner) Burn(kind multigas.ResourceKind, amount uint64) error {
	if b.GasLeft() < amount {
		return b.BurnOut()
	}
	b.gasUsed.SaturatingIncrementInto(kind, amount)
	return nil
}

func (b *testBurner) BurnMultiGas(amount multigas.MultiGas) error {
	if b.GasLeft() < amount.SingleGas() {
		return b.BurnOut()
	}
	b.gasUsed.SaturatingAddInto(amount)
	return nil
}

func (b *testBurner) Burned() uint64 {
	return b.gasUsed.SingleGas()
}

func (b *testBurner) GasLeft() uint64 {
	return b.gasSupplied - b.gasUsed.SingleGas()
}

func (b *testBurner) BurnOut() error {
	b.gasUsed.SaturatingIncrementInto(multigas.ResourceKindComputation, b.GasLeft())
	return vm.ErrOutOfGas
}

func (*testBurner) Restrict(error) {}

func (*testBurner) HandleError(err error) error {
	return err
}

func (*testBurner) ReadOnly() bool {
	return false
}

func (*testBurner) TracingInfo() *util.TracingInfo {
	return nil
}

func makeFragmentedRootForTest(t *testing.T) (*state.StateDB, []byte, []byte, common.Address, []byte) {
	t.Helper()

	statedb, err := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
	require.NoError(t, err)

	wasm := []byte("fragment gas reserve regression test")
	compressedWasm, err := arbcompress.Compress(wasm, arbcompress.LEVEL_WELL, arbcompress.EmptyDictionary)
	require.NoError(t, err)

	fragmentCode := append(state.NewStylusFragmentPrefix(), compressedWasm...)
	fragmentAddr := common.Address{1}
	statedb.SetCode(fragmentAddr, fragmentCode, tracing.CodeChangeUnspecified)

	rootCode := make([]byte, 0, 8+common.AddressLength)
	rootCode = append(rootCode, state.NewStylusRootPrefix(byte(arbcompress.EmptyDictionary))...)
	var decompressedLen [4]byte
	// #nosec G115
	binary.BigEndian.PutUint32(decompressedLen[:], uint32(len(wasm)))
	rootCode = append(rootCode, decompressedLen[:]...)
	rootCode = append(rootCode, fragmentAddr.Bytes()...)

	return statedb, rootCode, wasm, fragmentAddr, fragmentCode
}

func TestGetWasmFromRootStylusRequiresMaxFragmentReadReserve(t *testing.T) {
	statedb, rootCode, _, fragmentAddr, fragmentCode := makeFragmentedRootForTest(t)

	actualCost, err := fragmentReadGasCost(false, uint64(len(fragmentCode)))
	require.NoError(t, err)
	maxCost, err := fragmentReadGasCost(false, gethParams.DefaultMaxCodeSize)
	require.NoError(t, err)
	require.Greater(t, maxCost.SingleGas(), actualCost.SingleGas())

	burner := newTestBurner(actualCost.SingleGas())
	charger, err := newFragmentReadCharger(burner, gethParams.DefaultMaxCodeSize)
	require.NoError(t, err)

	wasm, err := getWasmFromRootStylus(statedb, rootCode, 1<<20, 1, charger)
	require.Nil(t, wasm)
	require.ErrorIs(t, err, vm.ErrOutOfGas)
	require.Equal(t, actualCost.SingleGas(), burner.Burned())
	require.False(t, statedb.AddressInAccessList(fragmentAddr))
}

func TestGetWasmFromRootStylusBurnsActualFragmentReadCostAfterPreflight(t *testing.T) {
	statedb, rootCode, expectedWasm, fragmentAddr, fragmentCode := makeFragmentedRootForTest(t)

	actualCost, err := fragmentReadGasCost(false, uint64(len(fragmentCode)))
	require.NoError(t, err)
	maxCost, err := fragmentReadGasCost(false, gethParams.DefaultMaxCodeSize)
	require.NoError(t, err)
	require.Greater(t, maxCost.SingleGas(), actualCost.SingleGas())

	burner := newTestBurner(maxCost.SingleGas())
	charger, err := newFragmentReadCharger(burner, gethParams.DefaultMaxCodeSize)
	require.NoError(t, err)

	// #nosec G115
	wasm, err := getWasmFromRootStylus(statedb, rootCode, uint32(len(expectedWasm)), 1, charger)
	require.NoError(t, err)
	require.Equal(t, expectedWasm, wasm)
	require.Equal(t, actualCost.SingleGas(), burner.Burned())
	require.Equal(t, maxCost.SingleGas()-actualCost.SingleGas(), burner.GasLeft())
	require.True(t, statedb.AddressInAccessList(fragmentAddr))
}

func TestGetWasmFromRootStylusUsesWarmFragmentReadCostWhenAddressIsWarm(t *testing.T) {
	statedb, rootCode, expectedWasm, fragmentAddr, fragmentCode := makeFragmentedRootForTest(t)
	statedb.AddAddressToAccessList(fragmentAddr)

	actualCost, err := fragmentReadGasCost(true, uint64(len(fragmentCode)))
	require.NoError(t, err)
	maxCost, err := fragmentReadGasCost(true, gethParams.DefaultMaxCodeSize)
	require.NoError(t, err)
	require.Greater(t, maxCost.SingleGas(), actualCost.SingleGas())

	burner := newTestBurner(maxCost.SingleGas())
	charger, err := newFragmentReadCharger(burner, gethParams.DefaultMaxCodeSize)
	require.NoError(t, err)

	// #nosec G115
	wasm, err := getWasmFromRootStylus(statedb, rootCode, uint32(len(expectedWasm)), 1, charger)
	require.NoError(t, err)
	require.Equal(t, expectedWasm, wasm)
	require.Equal(t, actualCost.SingleGas(), burner.Burned())
	require.Equal(t, maxCost.SingleGas()-actualCost.SingleGas(), burner.GasLeft())
	require.True(t, statedb.AddressInAccessList(fragmentAddr))
}
