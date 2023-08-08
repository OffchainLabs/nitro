// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

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

type Programs struct {
	backingStorage *storage.Storage
	programs       *storage.Storage
	inkPrice       storage.StorageBackedUBips
	wasmMaxDepth   storage.StorageBackedUint32
	wasmHostioInk  storage.StorageBackedUint64
	freePages      storage.StorageBackedUint16
	pageGas        storage.StorageBackedUint32
	pageRamp       storage.StorageBackedUint64
	pageLimit      storage.StorageBackedUint16
	version        storage.StorageBackedUint32
}

type Program struct {
	footprint uint16
	version   uint32
	address   common.Address // not saved in state
}

var machineVersionsKey = []byte{0}

const (
	versionOffset uint64 = iota
	inkPriceOffset
	wasmMaxDepthOffset
	wasmHostioInkOffset
	freePagesOffset
	pageGasOffset
	pageRampOffset
	pageLimitOffset
)

var ProgramNotCompiledError func() error
var ProgramUpToDateError func() error
var ProgramOutOfDateError func(version uint32) error

const MaxWasmSize = 64 * 1024
const initialFreePages = 2
const initialPageGas = 1000
const initialPageRamp = 620674314 // targets 8MB costing 32 million gas, minus the linear term
const initialPageLimit = 128      // reject wasms with memories larger than 8MB

func Initialize(sto *storage.Storage) {
	inkPrice := sto.OpenStorageBackedBips(inkPriceOffset)
	wasmMaxDepth := sto.OpenStorageBackedUint32(wasmMaxDepthOffset)
	wasmHostioInk := sto.OpenStorageBackedUint32(wasmHostioInkOffset)
	freePages := sto.OpenStorageBackedUint16(freePagesOffset)
	pageGas := sto.OpenStorageBackedUint32(pageGasOffset)
	pageRamp := sto.OpenStorageBackedUint64(pageRampOffset)
	pageLimit := sto.OpenStorageBackedUint16(pageLimitOffset)
	version := sto.OpenStorageBackedUint64(versionOffset)
	_ = inkPrice.Set(1)
	_ = wasmMaxDepth.Set(math.MaxUint32)
	_ = wasmHostioInk.Set(0)
	_ = freePages.Set(initialFreePages)
	_ = pageGas.Set(initialPageGas)
	_ = pageRamp.Set(initialPageRamp)
	_ = pageLimit.Set(initialPageLimit)
	_ = version.Set(1)
}

func Open(sto *storage.Storage) *Programs {
	return &Programs{
		backingStorage: sto,
		programs:       sto.OpenSubStorage(machineVersionsKey),
		inkPrice:       sto.OpenStorageBackedUBips(inkPriceOffset),
		wasmMaxDepth:   sto.OpenStorageBackedUint32(wasmMaxDepthOffset),
		wasmHostioInk:  sto.OpenStorageBackedUint64(wasmHostioInkOffset),
		freePages:      sto.OpenStorageBackedUint16(freePagesOffset),
		pageGas:        sto.OpenStorageBackedUint32(pageGasOffset),
		pageRamp:       sto.OpenStorageBackedUint64(pageRampOffset),
		pageLimit:      sto.OpenStorageBackedUint16(pageLimitOffset),
		version:        sto.OpenStorageBackedUint32(versionOffset),
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

func (p Programs) SetWasmHostioInk(ink uint64) error {
	return p.wasmHostioInk.Set(ink)
}

func (p Programs) FreePages() (uint16, error) {
	return p.freePages.Get()
}

func (p Programs) SetFreePages(pages uint16) error {
	return p.freePages.Set(pages)
}

func (p Programs) PageGas() (uint32, error) {
	return p.pageGas.Get()
}

func (p Programs) SetPageGas(gas uint32) error {
	return p.pageGas.Set(gas)
}

func (p Programs) PageRamp() (uint64, error) {
	return p.pageRamp.Get()
}

func (p Programs) SetPageRamp(ramp uint64) error {
	return p.pageRamp.Set(ramp)
}

func (p Programs) PageLimit() (uint16, error) {
	return p.pageLimit.Get()
}

func (p Programs) SetPageLimit(limit uint16) error {
	return p.pageLimit.Set(limit)
}

func (p Programs) ProgramVersion(codeHash common.Hash) (uint32, error) {
	return p.programs.GetUint32(codeHash)
}

func (p Programs) CompileProgram(evm *vm.EVM, program common.Address, debugMode bool) (uint32, bool, error) {
	statedb := evm.StateDB
	codeHash := statedb.GetCodeHash(program)

	version, err := p.StylusVersion()
	if err != nil {
		return 0, false, err
	}
	latest, err := p.ProgramVersion(codeHash)
	if err != nil {
		return 0, false, err
	}
	// Already compiled and found in the machine versions mapping.
	if latest >= version {
		return 0, false, ProgramUpToDateError()
	}
	wasm, err := getWasm(statedb, program)
	if err != nil {
		return 0, false, err
	}

	// require the program's footprint not exceed the remaining memory budget
	pageLimit, err := p.PageLimit()
	if err != nil {
		return 0, false, err
	}
	pageLimit = arbmath.SaturatingUSub(pageLimit, statedb.GetStylusPagesOpen())

	footprint, err := compileUserWasm(statedb, program, wasm, pageLimit, version, debugMode)
	if err != nil {
		return 0, true, err
	}

	// reflect the fact that, briefly, the footprint was allocated
	// note: the actual payment for the expansion happens in Rust
	statedb.AddStylusPagesEver(footprint)

	programData := Program{
		footprint: footprint,
		version:   version,
		address:   program,
	}
	return version, false, p.programs.Set(codeHash, programData.serialize())
}

func (p Programs) CallProgram(
	scope *vm.ScopeContext,
	statedb vm.StateDB,
	interpreter *vm.EVMInterpreter,
	tracingInfo *util.TracingInfo,
	calldata []byte,
) ([]byte, error) {

	// ensure the program is runnable
	stylusVersion, err := p.StylusVersion()
	if err != nil {
		return nil, err
	}
	contract := scope.Contract
	program, err := p.getProgram(contract)
	if err != nil {
		return nil, err
	}
	if program.version == 0 {
		return nil, ProgramNotCompiledError()
	}
	if program.version != stylusVersion {
		return nil, ProgramOutOfDateError(program.version)
	}

	debugMode := interpreter.Evm().ChainConfig().DebugMode()
	params, err := p.goParams(program.version, debugMode)
	if err != nil {
		return nil, err
	}

	evm := interpreter.Evm()
	l1BlockNumber, err := evm.ProcessingHook.L1BlockNumber(evm.Context)
	if err != nil {
		return nil, err
	}

	// pay for program init
	open, ever := statedb.GetStylusPages()
	model, err := p.memoryModel()
	if err != nil {
		return nil, err
	}
	cost := model.GasCost(program.footprint, open, ever)
	if err := contract.BurnGas(cost); err != nil {
		return nil, err
	}
	statedb.AddStylusPages(program.footprint)
	defer statedb.SetStylusPagesOpen(open)

	evmData := &evmData{
		blockBasefee:    common.BigToHash(evm.Context.BaseFee),
		chainId:         common.BigToHash(evm.ChainConfig().ChainID),
		blockCoinbase:   evm.Context.Coinbase,
		blockGasLimit:   evm.Context.GasLimit,
		blockNumber:     common.BigToHash(arbmath.UintToBig(l1BlockNumber)),
		blockTimestamp:  evm.Context.Time,
		contractAddress: scope.Contract.Address(),
		msgSender:       scope.Contract.Caller(),
		msgValue:        common.BigToHash(scope.Contract.Value()),
		txGasPrice:      common.BigToHash(evm.TxContext.GasPrice),
		txOrigin:        evm.TxContext.Origin,
	}

	return callUserWasm(program, scope, statedb, interpreter, tracingInfo, calldata, evmData, params, model)
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

func (p Program) serialize() common.Hash {
	data := common.Hash{}
	copy(data[26:], arbmath.Uint16ToBytes(p.footprint))
	copy(data[28:], arbmath.Uint32ToBytes(p.version))
	return data
}

func (p Programs) getProgram(contract *vm.Contract) (Program, error) {
	address := contract.Address()
	if contract.CodeAddr != nil {
		address = *contract.CodeAddr
	}
	data, err := p.programs.Get(contract.CodeHash)
	return Program{
		footprint: arbmath.BytesToUint16(data[26:28]),
		version:   arbmath.BytesToUint32(data[28:]),
		address:   address,
	}, err
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
	blockBasefee    common.Hash
	chainId         common.Hash
	blockCoinbase   common.Address
	blockGasLimit   uint64
	blockNumber     common.Hash
	blockTimestamp  uint64
	contractAddress common.Address
	msgSender       common.Address
	msgValue        common.Hash
	txGasPrice      common.Hash
	txOrigin        common.Address
}

type userStatus uint8

const (
	userSuccess userStatus = iota
	userRevert
	userFailure
	userOutOfInk
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
	case userOutOfInk:
		return nil, vm.ErrOutOfGas
	case userOutOfStack:
		return nil, vm.ErrDepth
	default:
		log.Error("program errored with unknown status", "status", status, "data", common.Bytes2Hex(data))
		return nil, vm.ErrExecutionReverted
	}
}
