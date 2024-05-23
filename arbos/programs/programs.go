// Copyright 2022-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

package programs

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbcompress"
	"github.com/offchainlabs/nitro/arbos/addressSet"
	"github.com/offchainlabs/nitro/arbos/storage"
	"github.com/offchainlabs/nitro/arbos/util"
	"github.com/offchainlabs/nitro/arbutil"
	am "github.com/offchainlabs/nitro/util/arbmath"
)

type Programs struct {
	backingStorage *storage.Storage
	programs       *storage.Storage
	moduleHashes   *storage.Storage
	dataPricer     *DataPricer
	cacheManagers  *addressSet.AddressSet
}

type Program struct {
	version       uint16
	initCost      uint16
	cachedCost    uint16
	footprint     uint16
	asmEstimateKb uint24 // Predicted size of the asm
	activatedAt   uint24 // Hours since Arbitrum began
	ageSeconds    uint64 // Not stored in state
	cached        bool
}

type uint24 = am.Uint24

var paramsKey = []byte{0}
var programDataKey = []byte{1}
var moduleHashesKey = []byte{2}
var dataPricerKey = []byte{3}
var cacheManagersKey = []byte{4}

var ErrProgramActivation = errors.New("program activation failed")

var ProgramNotWasmError func() error
var ProgramNotActivatedError func() error
var ProgramNeedsUpgradeError func(version, stylusVersion uint16) error
var ProgramExpiredError func(age uint64) error
var ProgramUpToDateError func() error
var ProgramKeepaliveTooSoon func(age uint64) error

func Initialize(sto *storage.Storage) {
	initStylusParams(sto.OpenSubStorage(paramsKey))
	initDataPricer(sto.OpenSubStorage(dataPricerKey))
	_ = addressSet.Initialize(sto.OpenCachedSubStorage(cacheManagersKey))
}

func Open(sto *storage.Storage) *Programs {
	return &Programs{
		backingStorage: sto,
		programs:       sto.OpenSubStorage(programDataKey),
		moduleHashes:   sto.OpenSubStorage(moduleHashesKey),
		dataPricer:     openDataPricer(sto.OpenCachedSubStorage(dataPricerKey)),
		cacheManagers:  addressSet.OpenAddressSet(sto.OpenCachedSubStorage(cacheManagersKey)),
	}
}

func (p Programs) DataPricer() *DataPricer {
	return p.dataPricer
}

func (p Programs) CacheManagers() *addressSet.AddressSet {
	return p.cacheManagers
}

func (p Programs) ActivateProgram(evm *vm.EVM, address common.Address, runMode core.MessageRunMode, debugMode bool) (
	uint16, common.Hash, common.Hash, *big.Int, bool, error,
) {
	statedb := evm.StateDB
	codeHash := statedb.GetCodeHash(address)
	burner := p.programs.Burner()
	time := evm.Context.Time

	if statedb.HasSelfDestructed(address) {
		return 0, codeHash, common.Hash{}, nil, false, errors.New("self destructed")
	}

	params, err := p.Params()
	if err != nil {
		return 0, codeHash, common.Hash{}, nil, false, err
	}

	stylusVersion := params.Version
	currentVersion, expired, cached, err := p.programExists(codeHash, time, params)
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
	pageLimit := am.SaturatingUSub(params.PageLimit, statedb.GetStylusPagesOpen())

	info, err := activateProgram(statedb, address, codeHash, wasm, pageLimit, stylusVersion, debugMode, burner)
	if err != nil {
		return 0, codeHash, common.Hash{}, nil, true, err
	}

	// remove prev asm
	if cached {
		oldModuleHash, err := p.moduleHashes.Get(codeHash)
		if err != nil {
			return 0, codeHash, common.Hash{}, nil, true, err
		}
		evictProgram(statedb, oldModuleHash, currentVersion, debugMode, runMode, expired)
	}
	if err := p.moduleHashes.Set(codeHash, info.moduleHash); err != nil {
		return 0, codeHash, common.Hash{}, nil, true, err
	}

	estimateKb, err := am.IntToUint24(am.DivCeil(info.asmEstimate, 1024)) // stored in kilobytes
	if err != nil {
		return 0, codeHash, common.Hash{}, nil, true, err
	}

	dataFee, err := p.dataPricer.UpdateModel(info.asmEstimate, time)
	if err != nil {
		return 0, codeHash, common.Hash{}, nil, true, err
	}

	programData := Program{
		version:       stylusVersion,
		initCost:      info.initGas,
		cachedCost:    info.cachedInitGas,
		footprint:     info.footprint,
		asmEstimateKb: estimateKb,
		activatedAt:   hoursSinceArbitrum(time),
		cached:        cached,
	}
	// replace the cached asm
	if cached {
		cacheProgram(statedb, info.moduleHash, programData, params, debugMode, time, runMode)
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
	runmode core.MessageRunMode,
) ([]byte, error) {
	evm := interpreter.Evm()
	contract := scope.Contract
	codeHash := contract.CodeHash
	debugMode := evm.ChainConfig().DebugMode()

	params, err := p.Params()
	if err != nil {
		return nil, err
	}

	program, err := p.getActiveProgram(codeHash, evm.Context.Time, params)
	if err != nil {
		return nil, err
	}
	moduleHash, err := p.moduleHashes.Get(codeHash)
	if err != nil {
		return nil, err
	}
	goParams := p.progParams(program.version, debugMode, params)
	l1BlockNumber, err := evm.ProcessingHook.L1BlockNumber(evm.Context)
	if err != nil {
		return nil, err
	}

	// pay for memory init
	open, ever := statedb.GetStylusPages()
	model := NewMemoryModel(params.FreePages, params.PageGas)
	callCost := model.GasCost(program.footprint, open, ever)

	// pay for program init
	cached := program.cached || statedb.GetRecentWasms().Insert(codeHash, params.BlockCacheSize)
	if cached {
		callCost = am.SaturatingUAdd(callCost, program.cachedGas(params))
	} else {
		callCost = am.SaturatingUAdd(callCost, program.initGas(params))
	}
	if err := contract.BurnGas(callCost); err != nil {
		return nil, err
	}
	statedb.AddStylusPages(program.footprint)
	defer statedb.SetStylusPagesOpen(open)

	localAsm, err := getLocalAsm(statedb, moduleHash, contract.Address(), params.PageLimit, evm.Context.Time, debugMode, program)
	if err != nil {
		log.Crit("failed to get local wasm for activated program", "program", contract.Address())
		return nil, err
	}

	evmData := &EvmData{
		blockBasefee:    common.BigToHash(evm.Context.BaseFee),
		chainId:         evm.ChainConfig().ChainID.Uint64(),
		blockCoinbase:   evm.Context.Coinbase,
		blockGasLimit:   evm.Context.GasLimit,
		blockNumber:     l1BlockNumber,
		blockTimestamp:  evm.Context.Time,
		contractAddress: scope.Contract.Address(),
		moduleHash:      moduleHash,
		msgSender:       scope.Contract.Caller(),
		msgValue:        scope.Contract.Value().Bytes32(),
		txGasPrice:      common.BigToHash(evm.TxContext.GasPrice),
		txOrigin:        evm.TxContext.Origin,
		reentrant:       am.BoolToUint32(reentrant),
		cached:          program.cached,
		tracing:         tracingInfo != nil,
	}

	address := contract.Address()
	if contract.CodeAddr != nil {
		address = *contract.CodeAddr
	}
	var arbos_tag uint32
	if runmode == core.MessageCommitMode {
		arbos_tag = statedb.Database().WasmCacheTag()
	}
	return callProgram(address, moduleHash, localAsm, scope, interpreter, tracingInfo, calldata, evmData, goParams, model, arbos_tag)
}

func getWasm(statedb vm.StateDB, program common.Address) ([]byte, error) {
	prefixedWasm := statedb.GetCode(program)
	return getWasmFromContractCode(prefixedWasm)
}

func getWasmFromContractCode(prefixedWasm []byte) ([]byte, error) {
	if prefixedWasm == nil {
		return nil, ProgramNotWasmError()
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

// Gets a program entry, which may be expired or not yet activated.
func (p Programs) getProgram(codeHash common.Hash, time uint64) (Program, error) {
	data, err := p.programs.Get(codeHash)
	program := Program{
		version:       am.BytesToUint16(data[:2]),
		initCost:      am.BytesToUint16(data[2:4]),
		cachedCost:    am.BytesToUint16(data[4:6]),
		footprint:     am.BytesToUint16(data[6:8]),
		activatedAt:   am.BytesToUint24(data[8:11]),
		asmEstimateKb: am.BytesToUint24(data[11:14]),
		cached:        am.BytesToBool(data[14:15]),
	}
	program.ageSeconds = hoursToAge(time, program.activatedAt)
	return program, err
}

// SaveActiveProgramToWasmStore is used to save active stylus programs to wasm store during rebuilding
func (p Programs) SaveActiveProgramToWasmStore(statedb *state.StateDB, codeHash common.Hash, code []byte, time uint64, debugMode bool, rebuildingStartBlockTime uint64) error {
	params, err := p.Params()
	if err != nil {
		return err
	}

	program, err := p.getActiveProgram(codeHash, time, params)
	if err != nil {
		// The program is not active so return early
		log.Info("program is not active, getActiveProgram returned error, hence do not include in rebuilding", "err", err)
		return nil
	}

	// It might happen that node crashed some time after rebuilding commenced and before it completed, hence when rebuilding
	// resumes after node is restarted the latest diskdb derived from statedb might now have codehashes that were activated
	// during the last rebuilding session. In such cases we don't need to fetch moduleshashes but instead return early
	// since they would already be added to the wasm store
	currentHoursSince := hoursSinceArbitrum(rebuildingStartBlockTime)
	if currentHoursSince < program.activatedAt {
		return nil
	}

	moduleHash, err := p.moduleHashes.Get(codeHash)
	if err != nil {
		return err
	}

	// If already in wasm store then return early
	localAsm, err := statedb.TryGetActivatedAsm(moduleHash)
	if err == nil && len(localAsm) > 0 {
		return nil
	}

	wasm, err := getWasmFromContractCode(code)
	if err != nil {
		log.Error("Failed to reactivate program while rebuilding wasm store: getWasmFromContractCode", "expected moduleHash", moduleHash, "err", err)
		return fmt.Errorf("failed to reactivate program while rebuilding wasm store: %w", err)
	}

	unlimitedGas := uint64(0xffffffffffff)
	// We know program is activated, so it must be in correct version and not use too much memory
	// Empty program address is supplied because we dont have access to this during rebuilding of wasm store
	info, asm, module, err := activateProgramInternal(statedb, common.Address{}, codeHash, wasm, params.PageLimit, program.version, debugMode, &unlimitedGas)
	if err != nil {
		log.Error("failed to reactivate program while rebuilding wasm store", "expected moduleHash", moduleHash, "err", err)
		return fmt.Errorf("failed to reactivate program while rebuilding wasm store: %w", err)
	}

	if info.moduleHash != moduleHash {
		log.Error("failed to reactivate program while rebuilding wasm store", "expected moduleHash", moduleHash, "got", info.moduleHash)
		return fmt.Errorf("failed to reactivate program while rebuilding wasm store, expected ModuleHash: %v", moduleHash)
	}

	batch := statedb.Database().WasmStore().NewBatch()
	rawdb.WriteActivation(batch, moduleHash, asm, module)
	if err := batch.Write(); err != nil {
		log.Error("failed writing re-activation to state while rebuilding wasm store", "err", err)
		return err
	}

	return nil
}

// Gets a program entry. Errors if not active.
func (p Programs) getActiveProgram(codeHash common.Hash, time uint64, params *StylusParams) (Program, error) {
	program, err := p.getProgram(codeHash, time)
	if err != nil {
		return program, err
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
	if program.ageSeconds > am.DaysToSeconds(params.ExpiryDays) {
		return program, ProgramExpiredError(program.ageSeconds)
	}
	return program, nil
}

func (p Programs) setProgram(codehash common.Hash, program Program) error {
	data := common.Hash{}
	copy(data[0:], am.Uint16ToBytes(program.version))
	copy(data[2:], am.Uint16ToBytes(program.initCost))
	copy(data[4:], am.Uint16ToBytes(program.cachedCost))
	copy(data[6:], am.Uint16ToBytes(program.footprint))
	copy(data[8:], am.Uint24ToBytes(program.activatedAt))
	copy(data[11:], am.Uint24ToBytes(program.asmEstimateKb))
	copy(data[14:], am.BoolToBytes(program.cached))
	return p.programs.Set(codehash, data)
}

func (p Programs) programExists(codeHash common.Hash, time uint64, params *StylusParams) (uint16, bool, bool, error) {
	program, err := p.getProgram(codeHash, time)
	if err != nil {
		return 0, false, false, err
	}
	activatedAt := program.activatedAt
	expired := activatedAt == 0 || hoursToAge(time, activatedAt) > am.DaysToSeconds(params.ExpiryDays)
	return program.version, expired, program.cached, err
}

func (p Programs) ProgramKeepalive(codeHash common.Hash, time uint64, params *StylusParams) (*big.Int, error) {
	program, err := p.getActiveProgram(codeHash, time, params)
	if err != nil {
		return nil, err
	}
	if program.ageSeconds < am.DaysToSeconds(params.KeepaliveDays) {
		return nil, ProgramKeepaliveTooSoon(program.ageSeconds)
	}

	stylusVersion := params.Version
	if program.version != stylusVersion {
		return nil, ProgramNeedsUpgradeError(program.version, stylusVersion)
	}

	dataFee, err := p.dataPricer.UpdateModel(program.asmSize(), time)
	if err != nil {
		return nil, err
	}
	program.activatedAt = hoursSinceArbitrum(time)
	return dataFee, p.setProgram(codeHash, program)
}

// Gets whether a program is cached. Note that the program may be expired.
func (p Programs) ProgramCached(codeHash common.Hash) (bool, error) {
	data, err := p.programs.Get(codeHash)
	return am.BytesToBool(data[14:15]), err
}

// Sets whether a program is cached. Errors if trying to cache an expired program.
func (p Programs) SetProgramCached(
	emitEvent func() error,
	db vm.StateDB,
	codeHash common.Hash,
	cache bool,
	time uint64,
	params *StylusParams,
	runMode core.MessageRunMode,
	debug bool,
) error {
	program, err := p.getProgram(codeHash, time)
	if err != nil {
		return err
	}
	expired := program.ageSeconds > am.DaysToSeconds(params.ExpiryDays)

	if program.version == 0 && cache {
		return ProgramNeedsUpgradeError(0, params.Version)
	}
	if expired && cache {
		return ProgramExpiredError(program.ageSeconds)
	}
	if program.cached == cache {
		return nil
	}
	if err := emitEvent(); err != nil {
		return err
	}

	// pay to cache the program, or to re-cache in case of upcoming revert
	if err := p.programs.Burner().Burn(uint64(program.initCost)); err != nil {
		return err
	}
	moduleHash, err := p.moduleHashes.Get(codeHash)
	if err != nil {
		return err
	}
	if cache {
		cacheProgram(db, moduleHash, program, params, debug, time, runMode)
	} else {
		evictProgram(db, moduleHash, program.version, debug, runMode, expired)
	}
	program.cached = cache
	return p.setProgram(codeHash, program)
}

func (p Programs) CodehashVersion(codeHash common.Hash, time uint64, params *StylusParams) (uint16, error) {
	program, err := p.getActiveProgram(codeHash, time, params)
	if err != nil {
		return 0, err
	}
	return program.version, nil
}

// Gets the number of seconds left until expiration. Errors if it's already happened.
func (p Programs) ProgramTimeLeft(codeHash common.Hash, time uint64, params *StylusParams) (uint64, error) {
	program, err := p.getActiveProgram(codeHash, time, params)
	if err != nil {
		return 0, err
	}
	age := hoursToAge(time, program.activatedAt)
	expirySeconds := am.DaysToSeconds(params.ExpiryDays)
	if age > expirySeconds {
		return 0, ProgramExpiredError(age)
	}
	return am.SaturatingUSub(expirySeconds, age), nil
}

func (p Programs) ProgramInitGas(codeHash common.Hash, time uint64, params *StylusParams) (uint64, uint64, error) {
	program, err := p.getActiveProgram(codeHash, time, params)
	return program.initGas(params), program.cachedGas(params), err
}

func (p Programs) ProgramMemoryFootprint(codeHash common.Hash, time uint64, params *StylusParams) (uint16, error) {
	program, err := p.getActiveProgram(codeHash, time, params)
	return program.footprint, err
}

func (p Programs) ProgramAsmSize(codeHash common.Hash, time uint64, params *StylusParams) (uint32, error) {
	program, err := p.getActiveProgram(codeHash, time, params)
	if err != nil {
		return 0, err
	}
	return program.asmSize(), nil
}

func (p Program) asmSize() uint32 {
	return am.SaturatingUMul(p.asmEstimateKb.ToUint32(), 1024)
}

func (p Program) initGas(params *StylusParams) uint64 {
	base := uint64(params.MinInitGas) * MinInitGasUnits
	dyno := am.SaturatingUMul(uint64(p.initCost), uint64(params.InitCostScalar)*CostScalarPercent)
	return am.SaturatingUAdd(base, am.DivCeil(dyno, 100))
}

func (p Program) cachedGas(params *StylusParams) uint64 {
	base := uint64(params.MinCachedInitGas) * MinCachedGasUnits
	dyno := am.SaturatingUMul(uint64(p.cachedCost), uint64(params.CachedCostScalar)*CostScalarPercent)
	return am.SaturatingUAdd(base, am.DivCeil(dyno, 100))
}

type ProgParams struct {
	Version   uint16
	MaxDepth  uint32
	InkPrice  uint24
	DebugMode bool
}

func (p Programs) progParams(version uint16, debug bool, params *StylusParams) *ProgParams {
	return &ProgParams{
		Version:   version,
		MaxDepth:  params.MaxStackDepth,
		InkPrice:  params.InkPrice,
		DebugMode: debug,
	}
}

type EvmData struct {
	blockBasefee    common.Hash
	chainId         uint64
	blockCoinbase   common.Address
	blockGasLimit   uint64
	blockNumber     uint64
	blockTimestamp  uint64
	contractAddress common.Address
	moduleHash      common.Hash
	msgSender       common.Address
	msgValue        common.Hash
	txGasPrice      common.Hash
	txOrigin        common.Address
	reentrant       uint32
	cached          bool
	tracing         bool
}

type activationInfo struct {
	moduleHash    common.Hash
	initGas       uint16
	cachedInitGas uint16
	asmEstimate   uint32
	footprint     uint16
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

// Hours since Arbitrum began, rounded down.
func hoursSinceArbitrum(time uint64) uint24 {
	return am.SaturatingUUCast[uint24]((am.SaturatingUSub(time, ArbitrumStartTime)) / 3600)
}

// Computes program age in seconds from the hours passed since Arbitrum began.
func hoursToAge(time uint64, hours uint24) uint64 {
	seconds := am.SaturatingUMul(uint64(hours), 3600)
	activatedAt := am.SaturatingUAdd(ArbitrumStartTime, seconds)
	return am.SaturatingUSub(time, activatedAt)
}
