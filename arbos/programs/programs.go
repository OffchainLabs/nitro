// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package programs

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/offchainlabs/nitro/arbcompress"
	"github.com/offchainlabs/nitro/arbos/storage"
	"github.com/offchainlabs/nitro/util/arbmath"
)

const MaxWASMSize = 64 * 1024

type Programs struct {
	backingStorage  *storage.Storage
	machineVersions *storage.Storage
	wasmGasPrice    storage.StorageBackedUBips
	wasmMaxDepth    storage.StorageBackedUint32
	wasmHeapBound   storage.StorageBackedUint32
	wasmHostioCost  storage.StorageBackedUint64
	version         storage.StorageBackedUint32
}

var machineVersionsKey = []byte{0}

const (
	versionOffset uint64 = iota
	wasmGasPriceOffset
	wasmMaxDepthOffset
	wasmHeapBoundOffset
	wasmHostioCostOffset
)

func Initialize(sto *storage.Storage) {
	wasmGasPrice := sto.OpenStorageBackedBips(wasmGasPriceOffset)
	wasmMaxDepth := sto.OpenStorageBackedUint32(wasmMaxDepthOffset)
	wasmHeapBound := sto.OpenStorageBackedUint32(wasmHeapBoundOffset)
	wasmHostioCost := sto.OpenStorageBackedUint32(wasmHostioCostOffset)
	version := sto.OpenStorageBackedUint64(versionOffset)
	_ = wasmGasPrice.Set(0)
	_ = wasmMaxDepth.Set(math.MaxUint32)
	_ = wasmHeapBound.Set(math.MaxUint32)
	_ = wasmHostioCost.Set(0)
	_ = version.Set(1)
}

func Open(sto *storage.Storage) *Programs {
	return &Programs{
		backingStorage:  sto,
		machineVersions: sto.OpenSubStorage(machineVersionsKey),
		wasmGasPrice:    sto.OpenStorageBackedUBips(wasmGasPriceOffset),
		wasmMaxDepth:    sto.OpenStorageBackedUint32(wasmMaxDepthOffset),
		wasmHeapBound:   sto.OpenStorageBackedUint32(wasmHeapBoundOffset),
		wasmHostioCost:  sto.OpenStorageBackedUint64(wasmHostioCostOffset),
		version:         sto.OpenStorageBackedUint32(versionOffset),
	}
}

func (p Programs) StylusVersion() (uint32, error) {
	return p.version.Get()
}

func (p Programs) WasmGasPrice() (arbmath.UBips, error) {
	return p.wasmGasPrice.Get()
}

func (p Programs) SetWasmGasPrice(price arbmath.UBips) error {
	return p.wasmGasPrice.Set(price)
}

func (p Programs) WasmMaxDepth() (uint32, error) {
	return p.wasmMaxDepth.Get()
}

func (p Programs) SetWasmMaxDepth(depth uint32) error {
	return p.wasmMaxDepth.Set(depth)
}

func (p Programs) WasmHeapBound() (uint32, error) {
	return p.wasmHeapBound.Get()
}

func (p Programs) SetWasmHeapBound(bound uint32) error {
	return p.wasmHeapBound.Set(bound)
}

func (p Programs) WasmHostioCost() (uint64, error) {
	return p.wasmHostioCost.Get()
}

func (p Programs) SetWasmHostioCost(cost uint64) error {
	return p.wasmHostioCost.Set(cost)
}

func (p Programs) CompileProgram(statedb vm.StateDB, program common.Address) (uint32, error) {
	version, err := p.StylusVersion()
	if err != nil {
		return 0, err
	}
	latest, err := p.machineVersions.GetUint32(program.Hash())
	if err != nil {
		return 0, err
	}
	if latest >= version {
		return 0, errors.New("program is current")
	}

	params, err := p.goParams(version)
	if err != nil {
		return 0, err
	}
	wasm, err := getWasm(statedb, program)
	if err != nil {
		return 0, err
	}
	if err := compileUserWasm(statedb, program, wasm, params); err != nil {
		return 0, err
	}
	return version, p.machineVersions.SetUint32(program.Hash(), version)
}

func (p Programs) CallProgram(
	statedb vm.StateDB,
	program common.Address,
	calldata []byte,
	gas *uint64,
) (uint32, []byte, error) {
	version, err := p.StylusVersion()
	if err != nil {
		return 0, nil, err
	}
	if version == 0 {
		return 0, nil, errors.New("wasm not compiled")
	}
	params, err := p.goParams(version)
	if err != nil {
		return 0, nil, err
	}
	return callUserWasm(statedb, program, calldata, gas, params)
}

func getWasm(statedb vm.StateDB, program common.Address) ([]byte, error) {
	wasm := statedb.GetCode(program)
	if wasm == nil {
		return nil, fmt.Errorf("missing wasm at address %v", program)
	}
	return arbcompress.Decompress(wasm, MaxWASMSize)
}

type goParams struct {
	version        uint32
	max_depth      uint32
	heap_bound     uint32
	wasm_gas_price uint64
	hostio_cost    uint64
}

func (p Programs) goParams(version uint32) (*goParams, error) {
	max_depth, err := p.WasmMaxDepth()
	if err != nil {
		return nil, err
	}
	heap_bound, err := p.WasmHeapBound()
	if err != nil {
		return nil, err
	}
	wasm_gas_price, err := p.WasmGasPrice()
	if err != nil {
		return nil, err
	}
	hostio_cost, err := p.WasmHostioCost()
	if err != nil {
		return nil, err
	}
	config := &goParams{
		version:        version,
		max_depth:      max_depth,
		heap_bound:     heap_bound,
		wasm_gas_price: wasm_gas_price.Uint64(),
		hostio_cost:    hostio_cost,
	}
	return config, nil
}
