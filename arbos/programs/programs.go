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
	"github.com/offchainlabs/nitro/arbos/burn"
	"github.com/offchainlabs/nitro/arbos/storage"
	"github.com/offchainlabs/nitro/arbos/util"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/util/arbmath"
)

type Programs struct {
	backingStorage *storage.Storage
	programs       *storage.Storage
	inkPrice       storage.StorageBackedUint24
	maxStackDepth  storage.StorageBackedUint32
	freePages      storage.StorageBackedUint16
	pageGas        storage.StorageBackedUint16
	pageRamp       storage.StorageBackedUint64
	pageLimit      storage.StorageBackedUint16
	callScalar     storage.StorageBackedUint16
	version        storage.StorageBackedUint16
}

type Program struct {
	wasmSize  uint16 // Unit is half of a kb
	footprint uint16
	version   uint16
	address   common.Address // not saved in state
}

type uint24 = arbmath.Uint24

var programDataKey = []byte{0}

const (
	versionOffset uint64 = iota
	inkPriceOffset
	maxStackDepthOffset
	freePagesOffset
	pageGasOffset
	pageRampOffset
	pageLimitOffset
	callScalarOffset
)

var ProgramNotCompiledError func() error
var ProgramOutOfDateError func(version uint16) error
var ProgramUpToDateError func() error

const MaxWasmSize = 128 * 1024
const initialFreePages = 2
const initialPageGas = 1000
const initialPageRamp = 620674314 // targets 8MB costing 32 million gas, minus the linear term
const initialPageLimit = 128      // reject wasms with memories larger than 8MB
const initialInkPrice = 10000     // 1 evm gas buys 10k ink
const initialCallScalar = 8       // call cost per half kb.

func Initialize(sto *storage.Storage) {
	inkPrice := sto.OpenStorageBackedUint24(inkPriceOffset)
	maxStackDepth := sto.OpenStorageBackedUint32(maxStackDepthOffset)
	freePages := sto.OpenStorageBackedUint16(freePagesOffset)
	pageGas := sto.OpenStorageBackedUint16(pageGasOffset)
	pageRamp := sto.OpenStorageBackedUint64(pageRampOffset)
	pageLimit := sto.OpenStorageBackedUint16(pageLimitOffset)
	callScalar := sto.OpenStorageBackedUint16(callScalarOffset)
	version := sto.OpenStorageBackedUint16(versionOffset)
	_ = inkPrice.Set(initialInkPrice)
	_ = maxStackDepth.Set(math.MaxUint32)
	_ = freePages.Set(initialFreePages)
	_ = pageGas.Set(initialPageGas)
	_ = pageRamp.Set(initialPageRamp)
	_ = pageLimit.Set(initialPageLimit)
	_ = callScalar.Set(initialCallScalar)
	_ = version.Set(1)
}

func Open(sto *storage.Storage) *Programs {
	return &Programs{
		backingStorage: sto,
		programs:       sto.OpenSubStorage(programDataKey),
		inkPrice:       sto.OpenStorageBackedUint24(inkPriceOffset),
		maxStackDepth:  sto.OpenStorageBackedUint32(maxStackDepthOffset),
		freePages:      sto.OpenStorageBackedUint16(freePagesOffset),
		pageGas:        sto.OpenStorageBackedUint16(pageGasOffset),
		pageRamp:       sto.OpenStorageBackedUint64(pageRampOffset),
		pageLimit:      sto.OpenStorageBackedUint16(pageLimitOffset),
		callScalar:     sto.OpenStorageBackedUint16(callScalarOffset),
		version:        sto.OpenStorageBackedUint16(versionOffset),
	}
}

func (p Programs) StylusVersion() (uint16, error) {
	return p.version.Get()
}

func (p Programs) InkPrice() (uint24, error) {
	return p.inkPrice.Get()
}

func (p Programs) SetInkPrice(value uint32) error {
	ink, err := arbmath.IntToUint24(value)
	if err != nil || ink == 0 {
		return errors.New("ink price must be a positive uint24")
	}
	return p.inkPrice.Set(ink)
}

func (p Programs) MaxStackDepth() (uint32, error) {
	return p.maxStackDepth.Get()
}

func (p Programs) SetMaxStackDepth(depth uint32) error {
	return p.maxStackDepth.Set(depth)
}

func (p Programs) FreePages() (uint16, error) {
	return p.freePages.Get()
}

func (p Programs) SetFreePages(pages uint16) error {
	return p.freePages.Set(pages)
}

func (p Programs) PageGas() (uint16, error) {
	return p.pageGas.Get()
}

func (p Programs) SetPageGas(gas uint16) error {
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

func (p Programs) CallScalar() (uint16, error) {
	return p.callScalar.Get()
}

func (p Programs) SetCallScalar(scalar uint16) error {
	return p.callScalar.Set(scalar)
}

func (p Programs) ActivateProgram(evm *vm.EVM, program common.Address, debugMode bool) (uint16, bool, error) {
	statedb := evm.StateDB

	version, err := p.StylusVersion()
	if err != nil {
		return 0, false, err
	}
	latest, err := p.ProgramVersion(program)
	if err != nil {
		return 0, false, err
	}
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

	// charge 3 million up front to begin compilation
	burner := p.programs.Burner()
	if err := burner.Burn(3000000); err != nil {
		return 0, false, err
	}
	info, err := compileUserWasm(statedb, program, wasm, pageLimit, version, debugMode, burner)
	if err != nil {
		return 0, true, err
	}

	// wasmSize is stored as half kb units, rounding up
	wasmSize := arbmath.SaturatingUCast[uint16]((len(wasm) + 511) / 512)

	programData := Program{
		wasmSize:  wasmSize,
		footprint: info.footprint,
		version:   version,
		address:   program,
	}
	return version, false, p.programs.Set(program.Hash(), programData.serialize())
}

func (p Programs) CallProgram(
	scope *vm.ScopeContext,
	statedb vm.StateDB,
	interpreter *vm.EVMInterpreter,
	tracingInfo *util.TracingInfo,
	calldata []byte,
	reentrant bool,
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
	memoryCost := model.GasCost(program.footprint, open, ever)
	callScalar, err := p.CallScalar()
	if err != nil {
		return nil, err
	}
	callCost := uint64(program.wasmSize) * uint64(callScalar)
	cost := common.SaturatingUAdd(memoryCost, callCost)
	if err := contract.BurnGas(cost); err != nil {
		return nil, err
	}
	statedb.AddStylusPages(program.footprint)
	defer statedb.SetStylusPagesOpen(open)

	evmData := &evmData{
		blockBasefee:    common.BigToHash(evm.Context.BaseFee),
		chainId:         evm.ChainConfig().ChainID.Uint64(),
		blockCoinbase:   evm.Context.Coinbase,
		blockGasLimit:   evm.Context.GasLimit,
		blockNumber:     l1BlockNumber,
		blockTimestamp:  evm.Context.Time,
		contractAddress: contract.Address(), // acting address
		msgSender:       contract.Caller(),
		msgValue:        common.BigToHash(contract.Value()),
		txGasPrice:      common.BigToHash(evm.TxContext.GasPrice),
		txOrigin:        evm.TxContext.Origin,
		reentrant:       arbmath.BoolToUint32(reentrant),
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

func (p Programs) getProgram(contract *vm.Contract) (Program, error) {
	address := contract.Address()
	if contract.CodeAddr != nil {
		address = *contract.CodeAddr
	}
	return p.deserializeProgram(address)
}

func (p Programs) deserializeProgram(address common.Address) (Program, error) {
	data, err := p.programs.Get(address.Hash())
	return Program{
		wasmSize:  arbmath.BytesToUint16(data[26:28]),
		footprint: arbmath.BytesToUint16(data[28:30]),
		version:   arbmath.BytesToUint16(data[30:]),
		address:   address,
	}, err
}

func (p Program) serialize() common.Hash {
	data := common.Hash{}
	copy(data[26:], arbmath.Uint16ToBytes(p.wasmSize))
	copy(data[28:], arbmath.Uint16ToBytes(p.footprint))
	copy(data[30:], arbmath.Uint16ToBytes(p.version))
	return data
}

func (p Programs) ProgramVersion(address common.Address) (uint16, error) {
	program, err := p.deserializeProgram(address)
	return program.version, err
}

func (p Programs) ProgramSize(address common.Address) (uint32, error) {
	program, err := p.deserializeProgram(address)
	// wasmSize represents the number of half kb units, return as bytes
	return uint32(program.wasmSize) * 512, err
}

func (p Programs) ProgramMemoryFootprint(address common.Address) (uint16, error) {
	program, err := p.deserializeProgram(address)
	return program.footprint, err
}

type goParams struct {
	version   uint16
	maxDepth  uint32
	inkPrice  uint24
	debugMode uint32
}

func (p Programs) goParams(version uint16, debug bool) (*goParams, error) {
	maxDepth, err := p.MaxStackDepth()
	if err != nil {
		return nil, err
	}
	inkPrice, err := p.InkPrice()
	if err != nil {
		return nil, err
	}

	config := &goParams{
		version:  version,
		maxDepth: maxDepth,
		inkPrice: inkPrice,
	}
	if debug {
		config.debugMode = 1
	}
	return config, nil
}

type evmData struct {
	blockBasefee    common.Hash
	chainId         uint64
	blockCoinbase   common.Address
	blockGasLimit   uint64
	blockNumber     uint64
	blockTimestamp  uint64
	contractAddress common.Address
	msgSender       common.Address
	msgValue        common.Hash
	txGasPrice      common.Hash
	txOrigin        common.Address
	reentrant       uint32
}

type userStatus uint8

const (
	userSuccess userStatus = iota
	userRevert
	userFailure
	userOutOfInk
	userOutOfStack
)

func (status userStatus) toResult(data []byte, debug bool) ([]byte, string, error) {
	details := func() string {
		if debug {
			return arbutil.ToStringOrHex(data)
		}
		return ""
	}
	switch status {
	case userSuccess:
		return data, "", nil
	case userRevert:
		return data, details(), vm.ErrExecutionReverted
	case userFailure:
		return nil, details(), vm.ErrExecutionReverted
	case userOutOfInk:
		return nil, "", vm.ErrOutOfGas
	case userOutOfStack:
		return nil, "", vm.ErrDepth
	default:
		log.Error("program errored with unknown status", "status", status, "data", common.Bytes2Hex(data))
		return nil, details(), vm.ErrExecutionReverted
	}
}

type wasmPricingInfo struct {
	footprint uint16
	size      uint32
}

// Pay for compilation. Right now this is a fixed amount of gas.
// In the future, costs will be variable and based on the wasm.
// Note: memory expansion costs are baked into compilation charging.
func payForCompilation(burner burn.Burner, _info *wasmPricingInfo) error {
	return burner.Burn(11000000)
}
