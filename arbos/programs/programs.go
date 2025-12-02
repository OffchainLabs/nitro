// Copyright 2022-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package programs

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/arbitrum/multigas"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	gethParams "github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbcompress"
	"github.com/offchainlabs/nitro/arbos/addressSet"
	"github.com/offchainlabs/nitro/arbos/storage"
	"github.com/offchainlabs/nitro/arbos/util"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/util/arbmath"
)

type Programs struct {
	ArbosVersion   uint64
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

type uint24 = arbmath.Uint24

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

func Initialize(arbosVersion uint64, sto *storage.Storage) {
	initStylusParams(arbosVersion, sto.OpenSubStorage(paramsKey))
	initDataPricer(sto.OpenSubStorage(dataPricerKey))
	_ = addressSet.Initialize(sto.OpenCachedSubStorage(cacheManagersKey))
}

func Open(arbosVersion uint64, sto *storage.Storage) *Programs {
	return &Programs{
		ArbosVersion:   arbosVersion,
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

func (p Programs) ActivateProgram(evm *vm.EVM, address common.Address, runCtx *core.MessageRunContext, debugMode bool) (
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
	wasm, err := getWasm(statedb, address, params.MaxWasmSize)
	if err != nil {
		return 0, codeHash, common.Hash{}, nil, false, err
	}

	// require the program's footprint not exceed the remaining memory budget
	pageLimit := arbmath.SaturatingUSub(params.PageLimit, statedb.GetStylusPagesOpen())

	info, err := activateProgram(statedb, address, codeHash, wasm, pageLimit, stylusVersion, p.ArbosVersion, debugMode, burner, runCtx)
	if err != nil {
		return 0, codeHash, common.Hash{}, nil, true, err
	}

	// remove prev asm
	if cached {
		oldModuleHash, err := p.moduleHashes.Get(codeHash)
		if err != nil {
			return 0, codeHash, common.Hash{}, nil, true, err
		}

		evictProgram(statedb, oldModuleHash, currentVersion, debugMode, runCtx, expired)
	}
	if err := p.moduleHashes.Set(codeHash, info.moduleHash); err != nil {
		return 0, codeHash, common.Hash{}, nil, true, err
	}

	estimateKb, err := arbmath.IntToUint24(arbmath.DivCeil(info.asmEstimate, 1024)) // stored in kilobytes
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
		code := statedb.GetCode(address)
		cacheProgram(statedb, info.moduleHash, programData, address, code, codeHash, params, debugMode, time, runCtx)
	}

	return stylusVersion, codeHash, info.moduleHash, dataFee, false, p.setProgram(codeHash, programData)
}

func (p Programs) CallProgram(
	scope *vm.ScopeContext,
	statedb vm.StateDB,
	evm *vm.EVM,
	tracingInfo *util.TracingInfo,
	calldata []byte,
	reentrant bool,
	runCtx *core.MessageRunContext,
) ([]byte, error) {
	contract := scope.Contract
	codeHash := contract.CodeHash
	startingGas := contract.Gas
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

	recentWasmsCacheHit := false
	if p.ArbosVersion >= gethParams.ArbosVersion_60 {
		recentWasmsCacheHit = statedb.GetRecentWasms().Insert(codeHash, params.BlockCacheSize)
	}
	cached := program.cached || recentWasmsCacheHit
	if cached || program.version > 1 { // in version 1 cached cost is part of init cost
		callCost = arbmath.SaturatingUAdd(callCost, program.cachedGas(params))
	}
	if !cached {
		callCost = arbmath.SaturatingUAdd(callCost, program.initGas(params))
	}
	if err := contract.BurnGas(callCost); err != nil {
		return nil, err
	}
	statedb.AddStylusPages(program.footprint)
	defer statedb.SetStylusPagesOpen(open)

	asmMap, err := getCompiledProgram(statedb, moduleHash, contract.Address(), contract.Code, contract.CodeHash, params.MaxWasmSize, params.PageLimit, evm.Context.Time, debugMode, program, runCtx)
	var ok bool
	var localAsm []byte
	if asmMap != nil {
		localAsm, ok = asmMap[rawdb.LocalTarget()]
	}
	if err != nil || !ok {
		panic(fmt.Sprintf("failed to get compiled program for activated program, program: %v, local target missing: %v, err: %v", contract.Address().Hex(), !ok, err))
	}

	evmData := &EvmData{
		arbosVersion:    evm.Context.ArbOSVersion,
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
		reentrant:       arbmath.BoolToUint32(reentrant),
		cached:          program.cached,
		tracing:         tracingInfo != nil,
	}

	address := contract.Address()
	metrics.GetOrRegisterCounter(fmt.Sprintf("arb/arbos/stylus/program_calls/%s", runCtx.RunModeMetricName()), nil).Inc(1)
	ret, err := callProgram(address, moduleHash, localAsm, scope, evm, tracingInfo, calldata, evmData, goParams, model, runCtx)
	if len(ret) > 0 && p.ArbosVersion >= gethParams.ArbosVersion_StylusFixes {
		// Ensure that return data costs as least as much as it would in the EVM.
		evmCost := evmMemoryCost(uint64(len(ret)))
		if startingGas < evmCost {
			// burn all remaining gas for this call
			contract.Gas = 0
			attributeWasmComputation(contract, startingGas)
			// #nosec G115
			metrics.GetOrRegisterCounter(fmt.Sprintf("arb/arbos/stylus/gas_used/%s", runCtx.RunModeMetricName()), nil).Inc(int64(startingGas))
			return nil, vm.ErrOutOfGas
		}

		maxGasToReturn := startingGas - evmCost
		contract.Gas = arbmath.MinInt(contract.Gas, maxGasToReturn)
	}
	attributeWasmComputation(contract, startingGas)

	// #nosec G115
	metrics.GetOrRegisterCounter(fmt.Sprintf("arb/arbos/stylus/gas_used/%s", runCtx.RunModeMetricName()), nil).Inc(int64(startingGas - contract.Gas))
	return ret, err
}

// attributeWasmComputation attributes the residual WASM computation gas so that
// UsedMultiGas.SingleGas() matches the gross used gas for this stylus call.
func attributeWasmComputation(contract *vm.Contract, startingGas uint64) {
	usedGas := startingGas - contract.Gas
	accountedGas := contract.UsedMultiGas.SingleGas()

	var residual uint64
	if accountedGas > usedGas {
		log.Trace("negative WASM computation residual", "usedGas", usedGas, "accountedGas", accountedGas)
		residual = 0
	} else {
		residual = usedGas - accountedGas
	}

	var overflow bool
	if contract.UsedMultiGas, overflow = contract.UsedMultiGas.SafeIncrement(multigas.ResourceKindWasmComputation, residual); overflow {
		log.Trace("WASM computation gas overflow", "residual", residual)
	}
}

func evmMemoryCost(size uint64) uint64 {
	// It would take 100GB to overflow this calculation, so no need to worry about that
	words := (size + 31) / 32
	linearCost := words * gethParams.MemoryGas
	squareCost := (words * words) / gethParams.QuadCoeffDiv
	return linearCost + squareCost
}

func getWasm(statedb vm.StateDB, program common.Address, maxWasmSize uint32) ([]byte, error) {
	prefixedWasm := statedb.GetCode(program)
	return getWasmFromContractCode(prefixedWasm, maxWasmSize)
}

func getWasmFromContractCode(prefixedWasm []byte, maxWasmSize uint32) ([]byte, error) {
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
	return arbcompress.DecompressWithDictionary(wasm, int(maxWasmSize), dict)
}

// Gets a program entry, which may be expired or not yet activated.
func (p Programs) getProgram(codeHash common.Hash, time uint64) (Program, error) {
	data, err := p.programs.Get(codeHash)
	program := Program{
		version:       arbmath.BytesToUint16(data[:2]),
		initCost:      arbmath.BytesToUint16(data[2:4]),
		cachedCost:    arbmath.BytesToUint16(data[4:6]),
		footprint:     arbmath.BytesToUint16(data[6:8]),
		activatedAt:   arbmath.BytesToUint24(data[8:11]),
		asmEstimateKb: arbmath.BytesToUint24(data[11:14]),
		cached:        arbmath.BytesToBool(data[14:15]),
	}
	program.ageSeconds = hoursToAge(time, program.activatedAt)
	return program, err
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
	if program.ageSeconds > arbmath.DaysToSeconds(params.ExpiryDays) {
		return program, ProgramExpiredError(program.ageSeconds)
	}
	return program, nil
}

func (p Programs) setProgram(codehash common.Hash, program Program) error {
	data := common.Hash{}
	copy(data[0:], arbmath.Uint16ToBytes(program.version))
	copy(data[2:], arbmath.Uint16ToBytes(program.initCost))
	copy(data[4:], arbmath.Uint16ToBytes(program.cachedCost))
	copy(data[6:], arbmath.Uint16ToBytes(program.footprint))
	copy(data[8:], arbmath.Uint24ToBytes(program.activatedAt))
	copy(data[11:], arbmath.Uint24ToBytes(program.asmEstimateKb))
	copy(data[14:], arbmath.BoolToBytes(program.cached))
	return p.programs.Set(codehash, data)
}

func (p Programs) programExists(codeHash common.Hash, time uint64, params *StylusParams) (uint16, bool, bool, error) {
	program, err := p.getProgram(codeHash, time)
	if err != nil {
		return 0, false, false, err
	}
	activatedAt := program.activatedAt
	expired := activatedAt == 0 || hoursToAge(time, activatedAt) > arbmath.DaysToSeconds(params.ExpiryDays)
	return program.version, expired, program.cached, err
}

func (p Programs) ProgramKeepalive(codeHash common.Hash, time uint64, params *StylusParams) (*big.Int, error) {
	program, err := p.getActiveProgram(codeHash, time, params)
	if err != nil {
		return nil, err
	}
	if program.ageSeconds < arbmath.DaysToSeconds(params.KeepaliveDays) {
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
	return arbmath.BytesToBool(data[14:15]), err
}

// Sets whether a program is cached. Errors if trying to cache an expired program.
// `address` must be present if setting cache to true as of ArbOS 31,
// and if `address` is present it must have the specified codeHash.
func (p Programs) SetProgramCached(
	emitEvent func() error,
	db vm.StateDB,
	codeHash common.Hash,
	address common.Address,
	cache bool,
	time uint64,
	params *StylusParams,
	runCtx *core.MessageRunContext,
	debug bool,
) error {
	program, err := p.getProgram(codeHash, time)
	if err != nil {
		return err
	}
	expired := program.ageSeconds > arbmath.DaysToSeconds(params.ExpiryDays)

	if program.version != params.Version && cache {
		return ProgramNeedsUpgradeError(program.version, params.Version)
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
	if err := p.programs.Burner().Burn(multigas.ResourceKindStorageAccess, uint64(program.initCost)); err != nil {
		return err
	}
	moduleHash, err := p.moduleHashes.Get(codeHash)
	if err != nil {
		return err
	}
	if cache {
		// Not passing in an address is supported pre-Verkle, as in Blockchain's ContractCodeWithPrefix method.
		code, err := db.Reader().Code(common.Address{}, codeHash)
		if err != nil {
			return err
		}
		if len(code) == 0 {
			return fmt.Errorf("code not found for codeHash: %x", codeHash)
		}
		cacheProgram(db, moduleHash, program, address, code, codeHash, params, debug, time, runCtx)
	} else {
		evictProgram(db, moduleHash, program.version, debug, runCtx, expired)
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
	expirySeconds := arbmath.DaysToSeconds(params.ExpiryDays)
	if age > expirySeconds {
		return 0, ProgramExpiredError(age)
	}
	return arbmath.SaturatingUSub(expirySeconds, age), nil
}

func (p Programs) ProgramInitGas(codeHash common.Hash, time uint64, params *StylusParams) (uint64, uint64, error) {
	program, err := p.getActiveProgram(codeHash, time, params)
	cachedGas := program.cachedGas(params)
	initGas := program.initGas(params)
	if params.Version > 1 {
		initGas += cachedGas
	}
	return initGas, cachedGas, err
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
	return arbmath.SaturatingUMul(p.asmEstimateKb.ToUint32(), 1024)
}

func (p Program) initGas(params *StylusParams) uint64 {
	base := uint64(params.MinInitGas) * MinInitGasUnits
	dyno := arbmath.SaturatingUMul(uint64(p.initCost), uint64(params.InitCostScalar)*CostScalarPercent)
	return arbmath.SaturatingUAdd(base, arbmath.DivCeil(dyno, 100))
}

func (p Program) cachedGas(params *StylusParams) uint64 {
	base := uint64(params.MinCachedInitGas) * MinCachedGasUnits
	dyno := arbmath.SaturatingUMul(uint64(p.cachedCost), uint64(params.CachedCostScalar)*CostScalarPercent)
	return arbmath.SaturatingUAdd(base, arbmath.DivCeil(dyno, 100))
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
	arbosVersion    uint64
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
	return arbmath.SaturatingUUCast[uint24]((arbmath.SaturatingUSub(time, ArbitrumStartTime)) / 3600)
}

// Computes program age in seconds from the hours passed since Arbitrum began.
func hoursToAge(time uint64, hours uint24) uint64 {
	seconds := arbmath.SaturatingUMul(uint64(hours), 3600)
	activatedAt := arbmath.SaturatingUAdd(ArbitrumStartTime, seconds)
	return arbmath.SaturatingUSub(time, activatedAt)
}
