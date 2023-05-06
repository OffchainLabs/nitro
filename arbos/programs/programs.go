// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package programs

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbcompress"
	"github.com/offchainlabs/nitro/arbos/storage"
	"github.com/offchainlabs/nitro/arbos/util"
	"github.com/offchainlabs/nitro/util/arbmath"
)

const MaxWasmSize = 64 * 1024

type Programs struct {
	backingStorage  *storage.Storage
	machineVersions *storage.Storage
	inkPrice        storage.StorageBackedUBips
	wasmMaxDepth    storage.StorageBackedUint32
	wasmHostioInk   storage.StorageBackedUint64
	version         storage.StorageBackedUint32
}

var machineVersionsKey = []byte{0}

const (
	versionOffset uint64 = iota
	inkPriceOffset
	wasmMaxDepthOffset
	wasmHostioInkOffset
)

var ProgramNotCompiledError func() error
var ProgramOutOfDateError func(version uint32) error
var ProgramUpToDateError func() error

func Initialize(sto *storage.Storage) {
	inkPrice := sto.OpenStorageBackedBips(inkPriceOffset)
	wasmMaxDepth := sto.OpenStorageBackedUint32(wasmMaxDepthOffset)
	wasmHostioInk := sto.OpenStorageBackedUint32(wasmHostioInkOffset)
	version := sto.OpenStorageBackedUint64(versionOffset)
	_ = inkPrice.Set(1)
	_ = wasmMaxDepth.Set(math.MaxUint32)
	_ = wasmHostioInk.Set(0)
	_ = version.Set(1)
}

func Open(sto *storage.Storage) *Programs {
	return &Programs{
		backingStorage:  sto,
		machineVersions: sto.OpenSubStorage(machineVersionsKey),
		inkPrice:        sto.OpenStorageBackedUBips(inkPriceOffset),
		wasmMaxDepth:    sto.OpenStorageBackedUint32(wasmMaxDepthOffset),
		wasmHostioInk:   sto.OpenStorageBackedUint64(wasmHostioInkOffset),
		version:         sto.OpenStorageBackedUint32(versionOffset),
	}
}

func (p Programs) StylusVersion() (uint32, error) {
	return p.version.Get()
}

func (p Programs) InkPrice() (arbmath.UBips, error) {
	return p.inkPrice.Get()
}

func (p Programs) SetInkPrice(price arbmath.UBips) error {
	if price == 0 {
		return errors.New("ink price must be nonzero")
	}
	return p.inkPrice.Set(price)
}

func (p Programs) WasmMaxDepth() (uint32, error) {
	return p.wasmMaxDepth.Get()
}

func (p Programs) SetWasmMaxDepth(depth uint32) error {
	return p.wasmMaxDepth.Set(depth)
}

func (p Programs) WasmHostioInk() (uint64, error) {
	return p.wasmHostioInk.Get()
}

func (p Programs) SetWasmHostioInk(cost uint64) error {
	return p.wasmHostioInk.Set(cost)
}

func (p Programs) ProgramVersion(program common.Address) (uint32, error) {
	return p.machineVersions.GetUint32(program.Hash())
}

func (p Programs) CompileProgram(statedb vm.StateDB, program common.Address, debugMode bool) (uint32, error) {
	version, err := p.StylusVersion()
	if err != nil {
		return 0, err
	}
	latest, err := p.machineVersions.GetUint32(program.Hash())
	if err != nil {
		return 0, err
	}
	if latest >= version {
		return 0, ProgramUpToDateError()
	}

	wasm, err := getWasm(statedb, program)
	if err != nil {
		return 0, err
	}
	if err := compileUserWasm(statedb, program, wasm, version, debugMode); err != nil {
		return 0, err
	}
	return version, p.machineVersions.SetUint32(program.Hash(), version)
}

func (p Programs) CallProgram(
	scope *vm.ScopeContext,
	statedb vm.StateDB,
	interpreter *vm.EVMInterpreter,
	tracingInfo *util.TracingInfo,
	calldata []byte,
) ([]byte, error) {
	stylusVersion, err := p.StylusVersion()
	if err != nil {
		return nil, err
	}
	programVersion, err := p.machineVersions.GetUint32(scope.Contract.Address().Hash())
	if err != nil {
		return nil, err
	}
	if programVersion == 0 {
		return nil, ProgramNotCompiledError()
	}
	if programVersion != stylusVersion {
		return nil, ProgramOutOfDateError(programVersion)
	}
	params, err := p.goParams(programVersion, interpreter.Evm().ChainConfig().DebugMode())
	if err != nil {
		return nil, err
	}
	evm := interpreter.Evm()
	evmData := &evmData{
		origin: evm.TxContext.Origin,
	}
	return callUserWasm(scope, statedb, interpreter, tracingInfo, calldata, evmData, params)
}

func getWasm(statedb vm.StateDB, program common.Address) ([]byte, error) {
	prefixedWasm := statedb.GetCode(program)
	if prefixedWasm == nil {
		return nil, fmt.Errorf("missing wasm at address %v", program)
	}
	wasm, err := state.StripStylusPrefix(prefixedWasm)
	if err != nil {
		return nil, err
	}
	return arbcompress.Decompress(wasm, MaxWasmSize)
}

type goParams struct {
	version   uint32
	maxDepth  uint32
	inkPrice  uint64
	hostioInk uint64
	debugMode uint32
}

func (p Programs) goParams(version uint32, debug bool) (*goParams, error) {
	maxDepth, err := p.WasmMaxDepth()
	if err != nil {
		return nil, err
	}
	inkPrice, err := p.InkPrice()
	if err != nil {
		return nil, err
	}
	hostioInk, err := p.WasmHostioInk()
	if err != nil {
		return nil, err
	}
	config := &goParams{
		version:   version,
		maxDepth:  maxDepth,
		inkPrice:  inkPrice.Uint64(),
		hostioInk: hostioInk,
	}
	if debug {
		config.debugMode = 1
	}
	return config, nil
}

type evmData struct {
	origin common.Address
}

type userStatus uint8

const (
	userSuccess userStatus = iota
	userRevert
	userFailure
	userOutOfGas
	userOutOfStack
)

func (status userStatus) output(data []byte) ([]byte, error) {
	switch status {
	case userSuccess:
		return data, nil
	case userRevert:
		return data, vm.ErrExecutionReverted
	case userFailure:
		return nil, vm.ErrExecutionReverted
	case userOutOfGas:
		return nil, vm.ErrOutOfGas
	case userOutOfStack:
		return nil, vm.ErrDepth
	default:
		log.Error("program errored with unknown status", "status", status, "data", common.Bytes2Hex(data))
		return nil, vm.ErrExecutionReverted
	}
}
