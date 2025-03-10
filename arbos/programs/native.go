// Copyright 2022-2024, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

//go:build !wasm
// +build !wasm

package programs

/*
#cgo CFLAGS: -g -I../../target/include/
#cgo LDFLAGS: ${SRCDIR}/../../target/lib/libstylus.a -ldl -lm
#include "arbitrator.h"

typedef uint8_t u8;
typedef uint16_t u16;
typedef uint32_t u32;
typedef uint64_t u64;
typedef size_t usize;
*/
import "C"

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"

	"github.com/offchainlabs/nitro/arbos/burn"
	"github.com/offchainlabs/nitro/arbos/util"
	"github.com/offchainlabs/nitro/arbutil"
)

type u8 = C.uint8_t
type u16 = C.uint16_t
type u32 = C.uint32_t
type u64 = C.uint64_t
type usize = C.size_t
type cbool = C._Bool
type bytes20 = C.Bytes20
type bytes32 = C.Bytes32
type rustBytes = C.RustBytes
type rustSlice = C.RustSlice

var (
	stylusLRUCacheSizeBytesGauge    = metrics.NewRegisteredGauge("arb/arbos/stylus/cache/lru/size_bytes", nil)
	stylusLRUCacheCountGauge        = metrics.NewRegisteredGauge("arb/arbos/stylus/cache/lru/count", nil)
	stylusLRUCacheHitsCounter       = metrics.NewRegisteredCounter("arb/arbos/stylus/cache/lru/hits", nil)
	stylusLRUCacheMissesCounter     = metrics.NewRegisteredCounter("arb/arbos/stylus/cache/lru/misses", nil)
	stylusLRUCacheDoesNotFitCounter = metrics.NewRegisteredCounter("arb/arbos/stylus/cache/lru/does_not_fit", nil)

	stylusLongTermCacheSizeBytesGauge = metrics.NewRegisteredGauge("arb/arbos/stylus/cache/long_term/size_bytes", nil)
	stylusLongTermCacheCountGauge     = metrics.NewRegisteredGauge("arb/arbos/stylus/cache/long_term/count", nil)
	stylusLongTermCacheHitsCounter    = metrics.NewRegisteredCounter("arb/arbos/stylus/cache/long_term/hits", nil)
	stylusLongTermCacheMissesCounter  = metrics.NewRegisteredCounter("arb/arbos/stylus/cache/long_term/misses", nil)
)

func activateProgram(
	db vm.StateDB,
	program common.Address,
	codehash common.Hash,
	wasm []byte,
	page_limit uint16,
	stylusVersion uint16,
	arbosVersionForGas uint64,
	debug bool,
	burner burn.Burner,
) (*activationInfo, error) {
	targets := db.Database().WasmTargets()
	moduleActivationMandatory := true
	info, asmMap, err := activateProgramInternal(program, codehash, wasm, page_limit, stylusVersion, arbosVersionForGas, debug, burner.GasLeft(), targets, moduleActivationMandatory)
	if err != nil {
		return nil, err
	}
	db.ActivateWasm(info.moduleHash, asmMap)
	return info, nil
}

func activateModule(
	addressForLogging common.Address,
	codehash common.Hash,
	wasm []byte,
	page_limit uint16,
	stylusVersion uint16,
	arbosVersionForGas uint64,
	debug bool,
	gasLeft *uint64,
) (*activationInfo, []byte, error) {
	output := &rustBytes{}
	moduleHash := &bytes32{}
	stylusData := &C.StylusData{}
	codeHash := hashToBytes32(codehash)

	status_mod := userStatus(C.stylus_activate(
		goSlice(wasm),
		u16(page_limit),
		u16(stylusVersion),
		u64(arbosVersionForGas),
		cbool(debug),
		output,
		&codeHash,
		moduleHash,
		stylusData,
		(*u64)(gasLeft),
	))

	module, msg, err := status_mod.toResult(rustBytesIntoBytes(output), debug)
	if err != nil {
		if debug {
			log.Warn("activation failed", "err", err, "msg", msg, "program", addressForLogging)
		}
		if errors.Is(err, vm.ErrExecutionReverted) {
			return nil, nil, fmt.Errorf("%w: %s", ErrProgramActivation, msg)
		}
		return nil, nil, err
	}
	info := &activationInfo{
		moduleHash:    bytes32ToHash(moduleHash),
		initGas:       uint16(stylusData.init_cost),
		cachedInitGas: uint16(stylusData.cached_init_cost),
		asmEstimate:   uint32(stylusData.asm_estimate),
		footprint:     uint16(stylusData.footprint),
	}
	return info, module, nil
}

func compileNative(
	wasm []byte,
	stylusVersion uint16,
	debug bool,
	target ethdb.WasmTarget,
) ([]byte, error) {
	output := &rustBytes{}
	status_asm := C.stylus_compile(
		goSlice(wasm),
		u16(stylusVersion),
		cbool(debug),
		goSlice([]byte(target)),
		output,
	)
	asm := rustBytesIntoBytes(output)
	if status_asm != 0 {
		return nil, fmt.Errorf("%w: %s", ErrProgramActivation, string(asm))
	}
	return asm, nil
}

func activateProgramInternal(
	addressForLogging common.Address,
	codehash common.Hash,
	wasm []byte,
	page_limit uint16,
	stylusVersion uint16,
	arbosVersionForGas uint64,
	debug bool,
	gasLeft *uint64,
	targets []ethdb.WasmTarget,
	moduleActivationMandatory bool,
) (*activationInfo, map[ethdb.WasmTarget][]byte, error) {
	var wavmFound bool
	var nativeTargets []ethdb.WasmTarget
	for _, target := range targets {
		if target == rawdb.TargetWavm {
			wavmFound = true
		} else {
			nativeTargets = append(nativeTargets, target)
		}
	}
	type result struct {
		target ethdb.WasmTarget
		asm    []byte
		err    error
	}
	results := make(chan result)
	// info will be set in separate thread, make sure to wait before reading
	var info *activationInfo
	asmMap := make(map[ethdb.WasmTarget][]byte, len(nativeTargets)+1)
	if moduleActivationMandatory || wavmFound {
		go func() {
			var err error
			var module []byte
			info, module, err = activateModule(addressForLogging, codehash, wasm, page_limit, stylusVersion, arbosVersionForGas, debug, gasLeft)
			results <- result{rawdb.TargetWavm, module, err}
		}()
	}
	if moduleActivationMandatory {
		// wait for the module activation before starting compilation for other targets
		res := <-results
		if res.err != nil {
			return nil, nil, res.err
		} else if wavmFound {
			asmMap[res.target] = res.asm
		}
	}
	for _, target := range nativeTargets {
		target := target
		go func() {
			asm, err := compileNative(wasm, stylusVersion, debug, target)
			results <- result{target, asm, err}
		}()
	}
	expectedResults := len(nativeTargets)
	if !moduleActivationMandatory && wavmFound {
		// we didn't wait for module activation result, so wait for it too
		expectedResults++
	}
	var err error
	for i := 0; i < expectedResults; i++ {
		res := <-results
		if res.err != nil {
			err = errors.Join(res.err, fmt.Errorf("%s:%w", res.target, err))
		} else {
			asmMap[res.target] = res.asm
		}
	}
	if err != nil && moduleActivationMandatory {
		log.Error(
			"Compilation failed for one or more targets despite activation succeeding",
			"address", addressForLogging,
			"codehash", codehash,
			"moduleHash", info.moduleHash,
			"targets", targets,
			"err", err,
		)
		panic(fmt.Sprintf("Compilation of %v failed for one or more targets despite activation succeeding: %v", addressForLogging, err))
	}
	return info, asmMap, err
}

func getLocalAsm(statedb vm.StateDB, moduleHash common.Hash, addressForLogging common.Address, code []byte, codehash common.Hash, pagelimit uint16, time uint64, debugMode bool, program Program) ([]byte, error) {
	localTarget := rawdb.LocalTarget()
	localAsm, err := statedb.TryGetActivatedAsm(localTarget, moduleHash)
	if err == nil && len(localAsm) > 0 {
		return localAsm, nil
	}

	// addressForLogging may be empty or may not correspond to the code, so we need to be careful to use the code passed in separately
	wasm, err := getWasmFromContractCode(code)
	if err != nil {
		log.Error("Failed to reactivate program: getWasm", "address", addressForLogging, "expected moduleHash", moduleHash, "err", err)
		return nil, fmt.Errorf("failed to reactivate program address: %v err: %w", addressForLogging, err)
	}

	// don't charge gas
	zeroArbosVersion := uint64(0)
	zeroGas := uint64(0)

	targets := statedb.Database().WasmTargets()
	// we know program is activated, so it must be in correct version and not use too much memory
	moduleActivationMandatory := false
	info, asmMap, err := activateProgramInternal(addressForLogging, codehash, wasm, pagelimit, program.version, zeroArbosVersion, debugMode, &zeroGas, targets, moduleActivationMandatory)
	if err != nil {
		log.Error("failed to reactivate program", "address", addressForLogging, "expected moduleHash", moduleHash, "err", err)
		return nil, fmt.Errorf("failed to reactivate program address: %v err: %w", addressForLogging, err)
	}

	if info != nil && info.moduleHash != moduleHash {
		log.Error("failed to reactivate program", "address", addressForLogging, "expected moduleHash", moduleHash, "got", info.moduleHash)
		return nil, fmt.Errorf("failed to reactivate program. address: %v, expected ModuleHash: %v", addressForLogging, moduleHash)
	}

	currentHoursSince := hoursSinceArbitrum(time)
	if currentHoursSince > program.activatedAt {
		// stylus program is active on-chain, and was activated in the past
		// so we store it directly to database
		batch := statedb.Database().WasmStore().NewBatch()
		rawdb.WriteActivation(batch, moduleHash, asmMap)
		if err := batch.Write(); err != nil {
			log.Error("failed writing re-activation to state", "address", addressForLogging, "err", err)
		}
	} else {
		// program activated recently, possibly in this eth_call
		// store it to statedb. It will be stored to database if statedb is commited
		statedb.ActivateWasm(moduleHash, asmMap)
	}
	asm, exists := asmMap[localTarget]
	if !exists {
		var availableTargets []ethdb.WasmTarget
		for target := range asmMap {
			availableTargets = append(availableTargets, target)
		}
		log.Error("failed to reactivate program - missing asm for local target", "address", addressForLogging, "local target", localTarget, "available targets", availableTargets)
		return nil, fmt.Errorf("failed to reactivate program - missing asm for local target, address: %v, local target: %v, available targets: %v", addressForLogging, localTarget, availableTargets)
	}
	return asm, nil
}

func callProgram(
	address common.Address,
	moduleHash common.Hash,
	localAsm []byte,
	scope *vm.ScopeContext,
	interpreter *vm.EVMInterpreter,
	tracingInfo *util.TracingInfo,
	calldata []byte,
	evmData *EvmData,
	stylusParams *ProgParams,
	memoryModel *MemoryModel,
	arbos_tag uint32,
) ([]byte, error) {
	db := interpreter.Evm().StateDB
	debug := stylusParams.DebugMode

	if len(localAsm) == 0 {
		log.Error("missing asm", "program", address, "module", moduleHash)
		panic("missing asm")
	}

	if stateDb, ok := db.(*state.StateDB); ok {
		stateDb.RecordProgram(db.Database().WasmTargets(), moduleHash)
	}

	evmApi := newApi(interpreter, tracingInfo, scope, memoryModel)
	defer evmApi.drop()

	output := &rustBytes{}
	status := userStatus(C.stylus_call(
		goSlice(localAsm),
		goSlice(calldata),
		stylusParams.encode(),
		evmApi.cNative,
		evmData.encode(),
		cbool(debug),
		output,
		(*u64)(&scope.Contract.Gas),
		u32(arbos_tag),
	))

	depth := interpreter.Depth()
	data, msg, err := status.toResult(rustBytesIntoBytes(output), debug)
	if status == userFailure && debug {
		log.Warn("program failure", "err", err, "msg", msg, "program", address, "depth", depth)
	}
	if tracingInfo != nil {
		tracingInfo.CaptureStylusExit(uint8(status), data, err, scope.Contract.Gas)
	}
	return data, err
}

//export handleReqImpl
func handleReqImpl(apiId usize, req_type u32, data *rustSlice, costPtr *u64, out_response *C.GoSliceData, out_raw_data *C.GoSliceData) {
	api := getApi(apiId)
	reqData := readRustSlice(data)
	reqType := RequestType(req_type - EvmApiMethodReqOffset)
	response, raw_data, cost := api.handler(reqType, reqData)
	*costPtr = u64(cost)
	api.pinAndRef(response, out_response)
	api.pinAndRef(raw_data, out_raw_data)
}

// Caches a program in Rust. We write a record so that we can undo on revert.
// For gas estimation and eth_call, we ignore permanent updates and rely on Rust's LRU.
func cacheProgram(db vm.StateDB, module common.Hash, program Program, addressForLogging common.Address, code []byte, codehash common.Hash, params *StylusParams, debug bool, time uint64, runMode core.MessageRunMode) {
	if runMode == core.MessageCommitMode {
		// address is only used for logging
		asm, err := getLocalAsm(db, module, addressForLogging, code, codehash, params.PageLimit, time, debug, program)
		if err != nil {
			panic("unable to recreate wasm")
		}
		tag := db.Database().WasmCacheTag()
		state.CacheWasmRust(asm, module, program.version, tag, debug)
		db.RecordCacheWasm(state.CacheWasm{ModuleHash: module, Version: program.version, Tag: tag, Debug: debug})
	}
}

// Evicts a program in Rust. We write a record so that we can undo on revert, unless we don't need to (e.g. expired)
// For gas estimation and eth_call, we ignore permanent updates and rely on Rust's LRU.
func evictProgram(db vm.StateDB, module common.Hash, version uint16, debug bool, runMode core.MessageRunMode, forever bool) {
	if runMode == core.MessageCommitMode {
		tag := db.Database().WasmCacheTag()
		state.EvictWasmRust(module, version, tag, debug)
		if !forever {
			db.RecordEvictWasm(state.EvictWasm{ModuleHash: module, Version: version, Tag: tag, Debug: debug})
		}
	}
}

func init() {
	state.CacheWasmRust = func(asm []byte, moduleHash common.Hash, version uint16, tag uint32, debug bool) {
		C.stylus_cache_module(goSlice(asm), hashToBytes32(moduleHash), u16(version), u32(tag), cbool(debug))
	}
	state.EvictWasmRust = func(moduleHash common.Hash, version uint16, tag uint32, debug bool) {
		C.stylus_evict_module(hashToBytes32(moduleHash), u16(version), u32(tag), cbool(debug))
	}
}

func SetWasmLruCacheCapacity(capacityBytes uint64) {
	C.stylus_set_cache_lru_capacity(u64(capacityBytes))
}

func UpdateWasmCacheMetrics() {
	metrics := &C.CacheMetrics{}
	C.stylus_get_cache_metrics(metrics)

	stylusLRUCacheSizeBytesGauge.Update(int64(metrics.lru.size_bytes))
	stylusLRUCacheCountGauge.Update(int64(metrics.lru.count))
	stylusLRUCacheHitsCounter.Inc(int64(metrics.lru.hits))
	stylusLRUCacheMissesCounter.Inc(int64(metrics.lru.misses))
	stylusLRUCacheDoesNotFitCounter.Inc(int64(metrics.lru.does_not_fit))

	stylusLongTermCacheSizeBytesGauge.Update(int64(metrics.long_term.size_bytes))
	stylusLongTermCacheCountGauge.Update(int64(metrics.long_term.count))
	stylusLongTermCacheHitsCounter.Inc(int64(metrics.long_term.hits))
	stylusLongTermCacheMissesCounter.Inc(int64(metrics.long_term.misses))
}

// Used for testing
type WasmLruCacheMetrics struct {
	SizeBytes uint64
	Count     uint32
}

// Used for testing
type WasmLongTermCacheMetrics struct {
	SizeBytes uint64
	Count     uint32
}

// Used for testing
type WasmCacheMetrics struct {
	Lru      WasmLruCacheMetrics
	LongTerm WasmLongTermCacheMetrics
}

// Used for testing
func GetWasmCacheMetrics() *WasmCacheMetrics {
	metrics := &C.CacheMetrics{}
	C.stylus_get_cache_metrics(metrics)

	return &WasmCacheMetrics{
		Lru: WasmLruCacheMetrics{
			SizeBytes: uint64(metrics.lru.size_bytes),
			Count:     uint32(metrics.lru.count),
		},
		LongTerm: WasmLongTermCacheMetrics{
			SizeBytes: uint64(metrics.long_term.size_bytes),
			Count:     uint32(metrics.long_term.count),
		},
	}
}

// Used for testing
func ClearWasmLruCache() {
	C.stylus_clear_lru_cache()
}

// Used for testing
func ClearWasmLongTermCache() {
	C.stylus_clear_long_term_cache()
}

// Used for testing
func GetEntrySizeEstimateBytes(module []byte, version uint16, debug bool) uint64 {
	return uint64(C.stylus_get_entry_size_estimate_bytes(goSlice(module), u16(version), cbool(debug)))
}

const DefaultTargetDescriptionArm = "arm64-linux-unknown+neon"
const DefaultTargetDescriptionX86 = "x86_64-linux-unknown+sse4.2+lzcnt+bmi"

func SetTarget(name ethdb.WasmTarget, description string, native bool) error {
	output := &rustBytes{}
	status := userStatus(C.stylus_target_set(
		goSlice([]byte(name)),
		goSlice([]byte(description)),
		output,
		cbool(native),
	))
	if status != userSuccess {
		msg := arbutil.ToStringOrHex(rustBytesIntoBytes(output))
		log.Error("failed to set stylus compilation target", "status", status, "msg", msg)
		return fmt.Errorf("failed to set stylus compilation target, status %v: %v", status, msg)
	}
	return nil
}

func bytes32ToHash(value *bytes32) common.Hash {
	hash := common.Hash{}
	for index, b := range value.bytes {
		hash[index] = byte(b)
	}
	return hash
}

func hashToBytes32(hash common.Hash) bytes32 {
	value := bytes32{}
	for index, b := range hash.Bytes() {
		value.bytes[index] = u8(b)
	}
	return value
}

func addressToBytes20(addr common.Address) bytes20 {
	value := bytes20{}
	for index, b := range addr.Bytes() {
		value.bytes[index] = u8(b)
	}
	return value
}

func readRustSlice(slice *rustSlice) []byte {
	if slice.len == 0 {
		return nil
	}
	return arbutil.PointerToSlice((*byte)(slice.ptr), int(slice.len))
}

func readRustBytes(vec *rustBytes) []byte {
	if vec.len == 0 {
		return nil
	}
	return arbutil.PointerToSlice((*byte)(vec.ptr), int(vec.len))
}

func rustBytesIntoBytes(vec *rustBytes) []byte {
	slice := readRustBytes(vec)
	dropRustBytes(vec)
	return slice
}

func dropRustBytes(vec *rustBytes) {
	C.free_rust_bytes(*vec)
}

func goSlice(slice []byte) C.GoSliceData {
	return C.GoSliceData{
		ptr: (*u8)(arbutil.SliceToPointer(slice)),
		len: usize(len(slice)),
	}
}

func (params *ProgParams) encode() C.StylusConfig {
	pricing := C.PricingParams{
		ink_price: u32(params.InkPrice.ToUint32()),
	}
	return C.StylusConfig{
		version:   u16(params.Version),
		max_depth: u32(params.MaxDepth),
		pricing:   pricing,
	}
}

func (data *EvmData) encode() C.EvmData {
	return C.EvmData{
		arbos_version:    u64(data.arbosVersion),
		block_basefee:    hashToBytes32(data.blockBasefee),
		chainid:          u64(data.chainId),
		block_coinbase:   addressToBytes20(data.blockCoinbase),
		block_gas_limit:  u64(data.blockGasLimit),
		block_number:     u64(data.blockNumber),
		block_timestamp:  u64(data.blockTimestamp),
		contract_address: addressToBytes20(data.contractAddress),
		module_hash:      hashToBytes32(data.moduleHash),
		msg_sender:       addressToBytes20(data.msgSender),
		msg_value:        hashToBytes32(data.msgValue),
		tx_gas_price:     hashToBytes32(data.txGasPrice),
		tx_origin:        addressToBytes20(data.txOrigin),
		reentrant:        u32(data.reentrant),
		return_data_len:  0,
		cached:           cbool(data.cached),
		tracing:          cbool(data.tracing),
	}
}
