// Copyright 2022-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

package programs

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbcompress"
	"github.com/offchainlabs/nitro/arbos/storage"
	"github.com/offchainlabs/nitro/arbos/util"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/util/arbmath"
)

type Programs struct {
	backingStorage *storage.Storage
	programs       *storage.Storage
	moduleHashes   *storage.Storage
	dataPricer     *dataPricer
	inkPrice       storage.StorageBackedUint24
	maxStackDepth  storage.StorageBackedUint32
	freePages      storage.StorageBackedUint16
	pageGas        storage.StorageBackedUint16
	pageRamp       storage.StorageBackedUint64
	pageLimit      storage.StorageBackedUint16
	callScalar     storage.StorageBackedUint16
	expiryDays     storage.StorageBackedUint16
	keepaliveDays  storage.StorageBackedUint16
	version        storage.StorageBackedUint16 // Must only be changed during ArbOS upgrades
}

type Program struct {
	version     uint16
	wasmSize    uint16 // Unit is half of a kb
	footprint   uint16
	activatedAt uint64 // Last activation timestamp
	secondsLeft uint64 // Not stored in state
}

type uint24 = arbmath.Uint24

var programDataKey = []byte{0}
var moduleHashesKey = []byte{1}
var dataPricerKey = []byte{2}

const (
	versionOffset uint64 = iota
	inkPriceOffset
	maxStackDepthOffset
	freePagesOffset
	pageGasOffset
	pageRampOffset
	pageLimitOffset
	callScalarOffset
	expiryDaysOffset
	keepaliveDaysOffset
)

var ErrProgramActivation = errors.New("program activation failed")

var ProgramNotActivatedError func() error
var ProgramNeedsUpgradeError func(version, stylusVersion uint16) error
var ProgramExpiredError func(age uint64) error
var ProgramUpToDateError func() error
var ProgramKeepaliveTooSoon func(age uint64) error

const MaxWasmSize = 128 * 1024
const initialFreePages = 2
const initialPageGas = 1000
const initialPageRamp = 620674314 // targets 8MB costing 32 million gas, minus the linear term.
const initialPageLimit = 128      // reject wasms with memories larger than 8MB.
const initialInkPrice = 10000     // 1 evm gas buys 10k ink.
const initialCallScalar = 8       // call cost per half kb.
const initialExpiryDays = 365     // deactivate after 1 year.
const initialKeepaliveDays = 31   // wait a month before allowing reactivation

func Initialize(sto *storage.Storage) {
	inkPrice := sto.OpenStorageBackedUint24(inkPriceOffset)
	maxStackDepth := sto.OpenStorageBackedUint32(maxStackDepthOffset)
	freePages := sto.OpenStorageBackedUint16(freePagesOffset)
	pageGas := sto.OpenStorageBackedUint16(pageGasOffset)
	pageRamp := sto.OpenStorageBackedUint64(pageRampOffset)
	pageLimit := sto.OpenStorageBackedUint16(pageLimitOffset)
	callScalar := sto.OpenStorageBackedUint16(callScalarOffset)
	expiryDays := sto.OpenStorageBackedUint16(expiryDaysOffset)
	keepaliveDays := sto.OpenStorageBackedUint16(keepaliveDaysOffset)
	version := sto.OpenStorageBackedUint16(versionOffset)
	_ = inkPrice.Set(initialInkPrice)
	_ = maxStackDepth.Set(math.MaxUint32)
	_ = freePages.Set(initialFreePages)
	_ = pageGas.Set(initialPageGas)
	_ = pageRamp.Set(initialPageRamp)
	_ = pageLimit.Set(initialPageLimit)
	_ = callScalar.Set(initialCallScalar)
	_ = expiryDays.Set(initialExpiryDays)
	_ = keepaliveDays.Set(initialKeepaliveDays)
	_ = version.Set(1)
}

func Open(sto *storage.Storage) *Programs {
	return &Programs{
		backingStorage: sto,
		programs:       sto.OpenSubStorage(programDataKey),
		moduleHashes:   sto.OpenSubStorage(moduleHashesKey),
		dataPricer:     openDataPricer(sto.OpenSubStorage(dataPricerKey)),
		inkPrice:       sto.OpenStorageBackedUint24(inkPriceOffset),
		maxStackDepth:  sto.OpenStorageBackedUint32(maxStackDepthOffset),
		freePages:      sto.OpenStorageBackedUint16(freePagesOffset),
		pageGas:        sto.OpenStorageBackedUint16(pageGasOffset),
		pageRamp:       sto.OpenStorageBackedUint64(pageRampOffset),
		pageLimit:      sto.OpenStorageBackedUint16(pageLimitOffset),
		callScalar:     sto.OpenStorageBackedUint16(callScalarOffset),
		expiryDays:     sto.OpenStorageBackedUint16(expiryDaysOffset),
		keepaliveDays:  sto.OpenStorageBackedUint16(keepaliveDaysOffset),
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

func (p Programs) ExpiryDays() (uint16, error) {
	return p.expiryDays.Get()
}

func (p Programs) SetExpiryDays(days uint16) error {
	return p.expiryDays.Set(days)
}

func (p Programs) KeepaliveDays() (uint16, error) {
	return p.keepaliveDays.Get()
}

func (p Programs) SetKeepaliveDays(days uint16) error {
	return p.keepaliveDays.Set(days)
}

func (p Programs) ActivateProgram(evm *vm.EVM, address common.Address, debugMode bool) (
	uint16, common.Hash, common.Hash, bool, error,
) {
	statedb := evm.StateDB
	codeHash := statedb.GetCodeHash(address)
	burner := p.programs.Burner()

	stylusVersion, err := p.StylusVersion()
	if err != nil {
		return 0, codeHash, common.Hash{}, false, err
	}
	currentVersion, err := p.programExists(codeHash)
	if err != nil {
		return 0, codeHash, common.Hash{}, false, err
	}
	if currentVersion == stylusVersion {
		// already activated and up to date
		return 0, codeHash, common.Hash{}, false, ProgramUpToDateError()
	}
	wasm, err := getWasm(statedb, address)
	if err != nil {
		return 0, codeHash, common.Hash{}, false, err
	}

	// require the program's footprint not exceed the remaining memory budget
	pageLimit, err := p.PageLimit()
	if err != nil {
		return 0, codeHash, common.Hash{}, false, err
	}
	pageLimit = arbmath.SaturatingUSub(pageLimit, statedb.GetStylusPagesOpen())

	moduleHash, footprint, err := activateProgram(statedb, address, wasm, pageLimit, stylusVersion, debugMode, burner)
	if err != nil {
		return 0, codeHash, common.Hash{}, true, err
	}
	if err := p.moduleHashes.Set(codeHash, moduleHash); err != nil {
		return 0, codeHash, common.Hash{}, true, err
	}

	// wasmSize is stored as half kb units, rounding up
	wasmSize := arbmath.SaturatingUCast[uint16]((len(wasm) + 511) / 512)

	programData := Program{
		version:     stylusVersion,
		wasmSize:    wasmSize,
		footprint:   footprint,
		activatedAt: evm.Context.Time,
	}
	return stylusVersion, codeHash, moduleHash, false, p.setProgram(codeHash, programData)
}

func (p Programs) CallProgram(
	scope *vm.ScopeContext,
	statedb vm.StateDB,
	interpreter *vm.EVMInterpreter,
	tracingInfo *util.TracingInfo,
	calldata []byte,
	reentrant bool,
) ([]byte, error) {
	evm := interpreter.Evm()
	contract := scope.Contract
	debugMode := evm.ChainConfig().DebugMode()

	program, err := p.getProgram(contract.CodeHash, evm.Context.Time)
	if err != nil {
		return nil, err
	}
	moduleHash, err := p.moduleHashes.Get(contract.CodeHash)
	if err != nil {
		return nil, err
	}
	params, err := p.goParams(program.version, debugMode)
	if err != nil {
		return nil, err
	}
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
		contractAddress: scope.Contract.Address(),
		msgSender:       scope.Contract.Caller(),
		msgValue:        common.BigToHash(scope.Contract.Value()),
		txGasPrice:      common.BigToHash(evm.TxContext.GasPrice),
		txOrigin:        evm.TxContext.Origin,
		reentrant:       arbmath.BoolToUint32(reentrant),
		tracing:         tracingInfo != nil,
	}

	address := contract.Address()
	if contract.CodeAddr != nil {
		address = *contract.CodeAddr
	}
	return callProgram(
		address, moduleHash, scope, statedb, interpreter,
		tracingInfo, calldata, evmData, params, model,
	)
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

func (p Programs) getProgram(codeHash common.Hash, time uint64) (Program, error) {
	data, err := p.programs.Get(codeHash)
	if err != nil {
		return Program{}, err
	}
	program := Program{
		version:     arbmath.BytesToUint16(data[:2]),
		wasmSize:    arbmath.BytesToUint16(data[2:4]),
		footprint:   arbmath.BytesToUint16(data[4:6]),
		activatedAt: arbmath.BytesToUint(data[6:14]),
	}
	if program.version == 0 {
		return program, ProgramNotActivatedError()
	}

	// check that the program is up to date
	stylusVersion, err := p.StylusVersion()
	if err != nil {
		return program, err
	}
	if program.version != stylusVersion {
		return program, ProgramNeedsUpgradeError(program.version, stylusVersion)
	}

	// ensure the program hasn't expired
	expiryDays, err := p.ExpiryDays()
	if err != nil {
		return program, err
	}
	age := time - program.activatedAt
	expirySeconds := arbmath.DaysToSeconds(expiryDays)
	if age > expirySeconds {
		return program, ProgramExpiredError(age)
	}
	program.secondsLeft = arbmath.SaturatingUSub(expirySeconds, age)
	return program, nil
}

func (p Programs) setProgram(codehash common.Hash, program Program) error {
	data := common.Hash{}
	copy(data[0:], arbmath.Uint16ToBytes(program.version))
	copy(data[2:], arbmath.Uint16ToBytes(program.wasmSize))
	copy(data[4:], arbmath.Uint16ToBytes(program.footprint))
	copy(data[6:], arbmath.UintToBytes(program.activatedAt))
	return p.programs.Set(codehash, data)
}

func (p Programs) programExists(codeHash common.Hash) (uint16, error) {
	data, err := p.programs.Get(codeHash)
	return arbmath.BytesToUint16(data[:2]), err
}

func (p Programs) ProgramKeepalive(codeHash common.Hash, time uint64) (*big.Int, error) {
	program, err := p.getProgram(codeHash, time)
	if err != nil {
		return nil, err
	}
	keepaliveDays, err := p.KeepaliveDays()
	if err != nil {
		return nil, err
	}
	if program.secondsLeft < arbmath.DaysToSeconds(keepaliveDays) {
		return nil, ProgramKeepaliveTooSoon(time - program.activatedAt)
	}

	stylusVersion, err := p.StylusVersion()
	if err != nil {
		return nil, err
	}
	if program.version != stylusVersion {
		return nil, ProgramNeedsUpgradeError(program.version, stylusVersion)
	}

	naive := uint32(5 * 1024 * 1024)
	cost, err := p.dataPricer.updateModel(naive, time)
	if err != nil {
		return nil, err
	}

	program.activatedAt = time
	return cost, p.setProgram(codeHash, program)

}

func (p Programs) CodehashVersion(codeHash common.Hash, time uint64) (uint16, error) {
	program, err := p.getProgram(codeHash, time)
	if err != nil {
		return 0, err
	}
	return program.version, nil
}

func (p Programs) ProgramTimeLeft(codeHash common.Hash, time uint64) (uint64, error) {
	program, err := p.getProgram(codeHash, time)
	if err != nil {
		return 0, err
	}
	return program.secondsLeft, nil
}

func (p Programs) ProgramSize(codeHash common.Hash, time uint64) (uint32, error) {
	program, err := p.getProgram(codeHash, time)
	// wasmSize represents the number of half kb units, return as bytes
	return uint32(program.wasmSize) * 512, err
}

func (p Programs) ProgramMemoryFootprint(codeHash common.Hash, time uint64) (uint16, error) {
	program, err := p.getProgram(codeHash, time)
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
	tracing         bool
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
	msg := arbutil.ToStringOrHex(data)
	switch status {
	case userSuccess:
		return data, "", nil
	case userRevert:
		return data, msg, vm.ErrExecutionReverted
	case userFailure:
		return nil, msg, vm.ErrExecutionReverted
	case userOutOfInk:
		return nil, "", vm.ErrOutOfGas
	case userOutOfStack:
		return nil, "", vm.ErrDepth
	default:
		log.Error("program errored with unknown status", "status", status, "data", msg)
		return nil, msg, vm.ErrExecutionReverted
	}
}
