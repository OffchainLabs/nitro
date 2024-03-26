// Copyright 2022-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

package programs

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
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
	dataPricer     *DataPricer
}

type Program struct {
	version       uint16
	initGas       uint24
	asmEstimateKb uint24 // Predicted size of the asm
	footprint     uint16
	activatedAt   uint64 // Last activation timestamp
	secondsLeft   uint64 // Not stored in state
}

type uint24 = arbmath.Uint24

var paramsKey = []byte{0}
var programDataKey = []byte{1}
var moduleHashesKey = []byte{2}
var dataPricerKey = []byte{3}

var ErrProgramActivation = errors.New("program activation failed")

var ProgramNotActivatedError func() error
var ProgramNeedsUpgradeError func(version, stylusVersion uint16) error
var ProgramExpiredError func(age uint64) error
var ProgramUpToDateError func() error
var ProgramKeepaliveTooSoon func(age uint64) error

func Initialize(sto *storage.Storage) {
	initStylusParams(sto.OpenSubStorage(paramsKey))
	initDataPricer(sto.OpenSubStorage(dataPricerKey))
}

func Open(sto *storage.Storage) *Programs {
	return &Programs{
		backingStorage: sto,
		programs:       sto.OpenSubStorage(programDataKey),
		moduleHashes:   sto.OpenSubStorage(moduleHashesKey),
		dataPricer:     openDataPricer(sto.OpenSubStorage(dataPricerKey)),
	}
}

func (p Programs) DataPricer() *DataPricer {
	return p.dataPricer
}

func (p Programs) ActivateProgram(evm *vm.EVM, address common.Address, debugMode bool) (
	uint16, common.Hash, common.Hash, *big.Int, bool, error,
) {
	statedb := evm.StateDB
	codeHash := statedb.GetCodeHash(address)
	burner := p.programs.Burner()
	time := evm.Context.Time

	params, err := p.Params()
	if err != nil {
		return 0, codeHash, common.Hash{}, nil, false, err
	}

	stylusVersion := params.Version
	currentVersion, expired, err := p.programExists(codeHash, time, params)
	if err != nil {
		return 0, codeHash, common.Hash{}, nil, false, err
	}
	if currentVersion == stylusVersion && !expired {
		// already activated and up to date
		return 0, codeHash, common.Hash{}, nil, false, ProgramUpToDateError()
	}
	wasm, err := getWasm(statedb, address)
	if err != nil {
		return 0, codeHash, common.Hash{}, nil, false, err
	}

	// require the program's footprint not exceed the remaining memory budget
	pageLimit := arbmath.SaturatingUSub(params.PageLimit, statedb.GetStylusPagesOpen())

	info, err := activateProgram(statedb, address, wasm, pageLimit, stylusVersion, debugMode, burner)
	if err != nil {
		return 0, codeHash, common.Hash{}, nil, true, err
	}
	if err := p.moduleHashes.Set(codeHash, info.moduleHash); err != nil {
		return 0, codeHash, common.Hash{}, nil, true, err
	}

	estimateKb, err := arbmath.IntToUint24(arbmath.DivCeil(info.asmEstimate, 1024)) // stored in kilobytes
	if err != nil {
		return 0, codeHash, common.Hash{}, nil, true, err
	}
	initGas24, err := arbmath.IntToUint24(info.initGas)
	if err != nil {
		return 0, codeHash, common.Hash{}, nil, true, err
	}

	dataFee, err := p.dataPricer.UpdateModel(info.asmEstimate, time)
	if err != nil {
		return 0, codeHash, common.Hash{}, nil, true, err
	}

	programData := Program{
		version:       stylusVersion,
		initGas:       initGas24,
		asmEstimateKb: estimateKb,
		footprint:     info.footprint,
		activatedAt:   time,
	}
	return stylusVersion, codeHash, info.moduleHash, dataFee, false, p.setProgram(codeHash, programData)
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

	params, err := p.Params()
	if err != nil {
		return nil, err
	}

	program, err := p.getProgram(contract.CodeHash, evm.Context.Time, params)
	if err != nil {
		return nil, err
	}
	moduleHash, err := p.moduleHashes.Get(contract.CodeHash)
	if err != nil {
		return nil, err
	}
	goParams := p.goParams(program.version, debugMode, params)
	l1BlockNumber, err := evm.ProcessingHook.L1BlockNumber(evm.Context)
	if err != nil {
		return nil, err
	}

	// pay for program init
	open, ever := statedb.GetStylusPages()
	model := NewMemoryModel(params.FreePages, params.PageGas)
	memoryCost := model.GasCost(program.footprint, open, ever)
	callCost := uint64(program.initGas) + uint64(params.MinInitGas)
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
		tracingInfo, calldata, evmData, goParams, model,
	)
}

func getWasm(statedb vm.StateDB, program common.Address) ([]byte, error) {
	prefixedWasm := statedb.GetCode(program)
	if prefixedWasm == nil {
		return nil, fmt.Errorf("missing wasm at address %v", program)
	}
	wasm, dictByte, err := state.StripStylusPrefix(prefixedWasm)
	if err != nil {
		return nil, err
	}

	var dict arbcompress.Dictionary
	switch dictByte {
	case 0:
		dict = arbcompress.EmptyDictionary
	case 1:
		dict = arbcompress.StylusProgramDictionary
	default:
		return nil, fmt.Errorf("unsupported dictionary %v", dictByte)
	}
	return arbcompress.DecompressWithDictionary(wasm, MaxWasmSize, dict)
}

func (p Programs) getProgram(codeHash common.Hash, time uint64, params *StylusParams) (Program, error) {
	data, err := p.programs.Get(codeHash)
	if err != nil {
		return Program{}, err
	}
	program := Program{
		version:       arbmath.BytesToUint16(data[:2]),
		initGas:       arbmath.BytesToUint24(data[2:5]),
		asmEstimateKb: arbmath.BytesToUint24(data[5:8]),
		footprint:     arbmath.BytesToUint16(data[8:10]),
		activatedAt:   arbmath.BytesToUint(data[10:18]),
	}
	if program.version == 0 {
		return program, ProgramNotActivatedError()
	}

	// check that the program is up to date
	stylusVersion := params.Version
	if program.version != stylusVersion {
		return program, ProgramNeedsUpgradeError(program.version, stylusVersion)
	}

	// ensure the program hasn't expired
	expiryDays := params.ExpiryDays
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
	copy(data[2:], arbmath.Uint24ToBytes(program.initGas))
	copy(data[5:], arbmath.Uint24ToBytes(program.asmEstimateKb))
	copy(data[8:], arbmath.Uint16ToBytes(program.footprint))
	copy(data[10:], arbmath.UintToBytes(program.activatedAt))
	return p.programs.Set(codehash, data)
}

func (p Programs) programExists(codeHash common.Hash, time uint64, params *StylusParams) (uint16, bool, error) {
	data, err := p.programs.Get(codeHash)
	if err != nil {
		return 0, false, err
	}

	version := arbmath.BytesToUint16(data[:2])
	activatedAt := arbmath.BytesToUint(data[10:18])
	expired := time-activatedAt > arbmath.DaysToSeconds(params.ExpiryDays)
	return version, expired, err
}

func (p Programs) ProgramKeepalive(codeHash common.Hash, time uint64, params *StylusParams) (*big.Int, error) {
	program, err := p.getProgram(codeHash, time, params)
	if err != nil {
		return nil, err
	}
	keepaliveDays := params.KeepaliveDays
	if program.secondsLeft < arbmath.DaysToSeconds(keepaliveDays) {
		return nil, ProgramKeepaliveTooSoon(time - program.activatedAt)
	}

	stylusVersion := params.Version
	if program.version != stylusVersion {
		return nil, ProgramNeedsUpgradeError(program.version, stylusVersion)
	}

	bytes := arbmath.SaturatingUMul(program.asmEstimateKb.ToUint32(), 1024)
	dataFee, err := p.dataPricer.UpdateModel(bytes, time)
	if err != nil {
		return nil, err
	}
	program.activatedAt = time
	return dataFee, p.setProgram(codeHash, program)

}

func (p Programs) CodehashVersion(codeHash common.Hash, time uint64, params *StylusParams) (uint16, error) {
	program, err := p.getProgram(codeHash, time, params)
	if err != nil {
		return 0, err
	}
	return program.version, nil
}

func (p Programs) ProgramTimeLeft(codeHash common.Hash, time uint64, params *StylusParams) (uint64, error) {
	program, err := p.getProgram(codeHash, time, params)
	if err != nil {
		return 0, err
	}
	return program.secondsLeft, nil
}

func (p Programs) ProgramInitGas(codeHash common.Hash, time uint64, params *StylusParams) (uint32, error) {
	program, err := p.getProgram(codeHash, time, params)
	return uint32(program.initGas), err
}

func (p Programs) ProgramMemoryFootprint(codeHash common.Hash, time uint64, params *StylusParams) (uint16, error) {
	program, err := p.getProgram(codeHash, time, params)
	return program.footprint, err
}

type goParams struct {
	version   uint16
	maxDepth  uint32
	inkPrice  uint24
	debugMode uint32
}

func (p Programs) goParams(version uint16, debug bool, params *StylusParams) *goParams {
	config := &goParams{
		version:  version,
		maxDepth: params.MaxStackDepth,
		inkPrice: params.InkPrice,
	}
	if debug {
		config.debugMode = 1
	}
	return config
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

type activationInfo struct {
	moduleHash  common.Hash
	initGas     uint32
	asmEstimate uint32
	footprint   uint16
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
