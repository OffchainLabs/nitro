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

type Programs struct {
	backingStorage *storage.Storage
	programs       *storage.Storage
	inkPrice       storage.StorageBackedUBips
	wasmMaxDepth   storage.StorageBackedUint32
	wasmHostioInk  storage.StorageBackedUint64
	freePages      storage.StorageBackedUint16
	pageGas        storage.StorageBackedUint32
	pageRamp       storage.StorageBackedUint32
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
)

var ProgramNotCompiledError func() error
var ProgramOutOfDateError func(version uint32) error
var ProgramUpToDateError func() error

const MaxWasmSize = 64 * 1024
const initialFreePages = 2
const initialPageGas = 1000
const initialPageRamp = 620674314   // targets 8MB costing 32 million gas, minus the linear term
const initialMachinePageLimit = 128 // reject wasms with memories larger than 8MB

func Initialize(sto *storage.Storage) {
	inkPrice := sto.OpenStorageBackedBips(inkPriceOffset)
	wasmMaxDepth := sto.OpenStorageBackedUint32(wasmMaxDepthOffset)
	wasmHostioInk := sto.OpenStorageBackedUint32(wasmHostioInkOffset)
	freePages := sto.OpenStorageBackedUint16(freePagesOffset)
	pageGas := sto.OpenStorageBackedUint32(pageGasOffset)
	pageRamp := sto.OpenStorageBackedUint32(pageRampOffset)
	version := sto.OpenStorageBackedUint64(versionOffset)
	_ = inkPrice.Set(1)
	_ = wasmMaxDepth.Set(math.MaxUint32)
	_ = wasmHostioInk.Set(0)
	_ = freePages.Set(initialFreePages)
	_ = pageGas.Set(initialPageGas)
	_ = pageRamp.Set(initialPageRamp)
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
		pageRamp:       sto.OpenStorageBackedUint32(pageRampOffset),
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

func (p Programs) PageRamp() (uint32, error) {
	return p.pageRamp.Get()
}

func (p Programs) SetPageRamp(ramp uint32) error {
	return p.pageRamp.Set(ramp)
}

func (p Programs) ProgramVersion(program common.Address) (uint32, error) {
	return p.programs.GetUint32(program.Hash())
}

func (p Programs) CompileProgram(statedb vm.StateDB, program common.Address, debugMode bool) (uint32, error) {
	version, err := p.StylusVersion()
	if err != nil {
		return 0, err
	}
	latest, err := p.programs.GetUint32(program.Hash())
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

	footprint, err := compileUserWasm(statedb, program, wasm, version, debugMode)
	if err != nil {
		return 0, err
	}
	programData := Program{
		footprint: footprint,
		version:   version,
		address:   program,
	}
	return version, p.programs.Set(program.Hash(), programData.serialize())
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
	params, err := p.goParams(program.version, statedb, debugMode)
	if err != nil {
		return nil, err
	}

	evm := interpreter.Evm()
	l1BlockNumber, err := evm.ProcessingHook.L1BlockNumber(evm.Context)
	if err != nil {
		return nil, err
	}

	evmData := &evmData{
		blockBasefee:    common.BigToHash(evm.Context.BaseFee),
		blockChainId:    common.BigToHash(evm.ChainConfig().ChainID),
		blockCoinbase:   evm.Context.Coinbase,
		blockDifficulty: common.BigToHash(evm.Context.Difficulty),
		blockGasLimit:   evm.Context.GasLimit,
		blockNumber:     common.BigToHash(arbmath.UintToBig(l1BlockNumber)),
		blockTimestamp:  evm.Context.Time,
		contractAddress: contract.Address(), // acting address
		msgSender:       contract.Caller(),
		msgValue:        common.BigToHash(contract.Value()),
		txGasPrice:      common.BigToHash(evm.TxContext.GasPrice),
		txOrigin:        evm.TxContext.Origin,
		footprint:       program.footprint,
	}
	return callUserWasm(program, scope, statedb, interpreter, tracingInfo, calldata, evmData, params)
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
	data, err := p.programs.Get(address.Hash())
	return Program{
		footprint: arbmath.BytesToUint16(data[26:28]),
		version:   arbmath.BytesToUint32(data[28:]),
		address:   address,
	}, err
}

type goParams struct {
	version     uint32
	maxDepth    uint32
	inkPrice    uint64
	hostioInk   uint64
	debugMode   uint32
	memoryModel goMemoryModel
}

type goMemoryModel struct {
	freePages uint16 // number of pages the tx gets for free
	pageGas   uint32 // base gas to charge per wasm page
	pageRamp  uint32 // ramps up exponential memory costs
}

func (p Programs) goParams(version uint32, statedb vm.StateDB, debug bool) (*goParams, error) {
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

	freePages, err := p.FreePages()
	if err != nil {
		return nil, err
	}
	pageGas, err := p.PageGas()
	if err != nil {
		return nil, err
	}
	pageRamp, err := p.PageRamp()
	if err != nil {
		return nil, err
	}
	memParams := goMemoryModel{
		freePages: freePages,
		pageGas:   pageGas,
		pageRamp:  pageRamp,
	}

	config := &goParams{
		version:     version,
		maxDepth:    maxDepth,
		inkPrice:    inkPrice.Uint64(),
		hostioInk:   hostioInk,
		memoryModel: memParams,
	}
	if debug {
		config.debugMode = 1
	}
	return config, nil
}

type evmData struct {
	blockBasefee    common.Hash
	blockChainId    common.Hash
	blockCoinbase   common.Address
	blockDifficulty common.Hash
	blockGasLimit   uint64
	blockNumber     common.Hash
	blockTimestamp  uint64
	contractAddress common.Address
	msgSender       common.Address
	msgValue        common.Hash
	txGasPrice      common.Hash
	txOrigin        common.Address
	footprint       uint16
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
