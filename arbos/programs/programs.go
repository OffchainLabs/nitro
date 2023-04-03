// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package programs

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core"
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
	wasmGasPrice    storage.StorageBackedUBips
	wasmMaxDepth    storage.StorageBackedUint32
	wasmHostioCost  storage.StorageBackedUint64
	version         storage.StorageBackedUint32
}

var machineVersionsKey = []byte{0}

const (
	versionOffset uint64 = iota
	wasmGasPriceOffset
	wasmMaxDepthOffset
	wasmHostioCostOffset
)

var ProgramNotCompiledError func() error
var ProgramOutOfDateError func(version uint32) error
var ProgramUpToDateError func() error

func Initialize(sto *storage.Storage) {
	wasmGasPrice := sto.OpenStorageBackedBips(wasmGasPriceOffset)
	wasmMaxDepth := sto.OpenStorageBackedUint32(wasmMaxDepthOffset)
	wasmHostioCost := sto.OpenStorageBackedUint32(wasmHostioCostOffset)
	version := sto.OpenStorageBackedUint64(versionOffset)
	_ = wasmGasPrice.Set(1)
	_ = wasmMaxDepth.Set(math.MaxUint32)
	_ = wasmHostioCost.Set(0)
	_ = version.Set(1)
}

func Open(sto *storage.Storage) *Programs {
	return &Programs{
		backingStorage:  sto,
		machineVersions: sto.OpenSubStorage(machineVersionsKey),
		wasmGasPrice:    sto.OpenStorageBackedUBips(wasmGasPriceOffset),
		wasmMaxDepth:    sto.OpenStorageBackedUint32(wasmMaxDepthOffset),
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
	if price == 0 {
		return errors.New("wasm gas price must be nonzero")
	}
	return p.wasmGasPrice.Set(price)
}

func (p Programs) WasmMaxDepth() (uint32, error) {
	return p.wasmMaxDepth.Get()
}

func (p Programs) SetWasmMaxDepth(depth uint32) error {
	return p.wasmMaxDepth.Set(depth)
}

func (p Programs) WasmHostioCost() (uint64, error) {
	return p.wasmHostioCost.Get()
}

func (p Programs) SetWasmHostioCost(cost uint64) error {
	return p.wasmHostioCost.Set(cost)
}

func (p Programs) WasmProgramVersion(program common.Address) (uint32, error) {
	latest, err := p.machineVersions.GetUint32(program.Hash())
	if err != nil {
		return 0, err
	}
	return latest, nil
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
	msg core.Message,
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
	return callUserWasm(scope, statedb, interpreter, tracingInfo, msg, calldata, params)
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
	version      uint32
	maxDepth     uint32
	wasmGasPrice uint64
	hostioCost   uint64
	debugMode    uint64
}

func (p Programs) goParams(version uint32, debug bool) (*goParams, error) {
	maxDepth, err := p.WasmMaxDepth()
	if err != nil {
		return nil, err
	}
	wasmGasPrice, err := p.WasmGasPrice()
	if err != nil {
		return nil, err
	}
	hostioCost, err := p.WasmHostioCost()
	if err != nil {
		return nil, err
	}
	config := &goParams{
		version:      version,
		maxDepth:     maxDepth,
		wasmGasPrice: wasmGasPrice.Uint64(),
		hostioCost:   hostioCost,
	}
	if debug {
		config.debugMode = 1
	}
	return config, nil
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
