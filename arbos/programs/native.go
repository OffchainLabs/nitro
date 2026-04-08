// Copyright 2022-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

//go:build !wasm

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
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/arbitrum/multigas"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"

	"github.com/offchainlabs/nitro/arbos/burn"
	"github.com/offchainlabs/nitro/arbos/util"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/util/containers"
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

// savedState captures the contract state before a Stylus call so it can be
// restored if the call hits a native stack overflow and needs to be retried
// with cranelift (at the original or doubled stack size, depending on whether
// this is the first overflow).
type savedState struct {
	gas          uint64
	usedMultiGas multigas.MultiGas
	openPages    uint16
	everPages    uint16
}

// allowFallback controls whether compilation failures fall back to an alternative compiler.
// Set once at startup via SetAllowFallback; defaults to true (fallback enabled).
var allowFallback atomic.Bool

func init() {
	allowFallback.Store(true)
}

// SetAllowFallback configures whether to fall back to an alternative compiler on failure.
func SetAllowFallback(enabled bool) {
	allowFallback.Store(enabled)
	log.Info("Compiler fallback for Stylus compilation configured", "enabled", enabled)
}

// GetAllowFallback returns the current compiler fallback setting.
func GetAllowFallback() bool {
	return allowFallback.Load()
}

// nativeStackBaseline stores the stack size set at startup so that
// doubleNativeStackSize can double from a known baseline. A non-zero value
// means doubling is available; doubleNativeStackSize atomically CAS's it to 0
// to claim the one-time doubling. Zero means either not yet initialized or
// already doubled.
var nativeStackBaseline atomic.Uint64

// SetInitialNativeStackSize configures the Wasmer coroutine stack size and
// records it as the baseline for overflow recovery. Call once at node startup.
// Storing a non-zero baseline re-enables doubleNativeStackSize.
func SetInitialNativeStackSize(size uint64) {
	SetNativeStackSize(size)
	// Always capture the true Wasmer stack size (whether newly set or the
	// existing default when size == 0) so doubleNativeStackSize can
	// double from a real baseline.
	nativeStackBaseline.Store(GetNativeStackSize())
}

// doubleNativeStackSize doubles the process-wide Wasmer coroutine stack size
// from the configured baseline, capped at MaxNativeStackSize. Returns true if
// the stack was successfully doubled. The baseline is atomically set to 0 on
// success, so subsequent calls return false (one-time doubling).
func doubleNativeStackSize() bool {
	baseStackSize := nativeStackBaseline.Load()
	if baseStackSize == 0 {
		// Zero means either not yet initialized (SetInitialNativeStackSize
		// was never called) or already doubled. Both cases correctly mean
		// "don't double": uninitialized has no baseline to double from,
		// and already-doubled must not double again.
		return false
	}
	newStackSize := baseStackSize * 2
	if newStackSize > MaxNativeStackSize {
		newStackSize = MaxNativeStackSize
	}
	if newStackSize <= baseStackSize {
		return false
	}
	if !nativeStackBaseline.CompareAndSwap(baseStackSize, 0) {
		return false // another goroutine doubled first
	}
	SetNativeStackSize(newStackSize)
	DrainStackPool()
	return true
}

// SetNativeStackSize configures the Wasmer coroutine stack size for Stylus execution.
// If size is 0, the existing default (1 MB) is kept.
// Does NOT update the configured baseline — use SetInitialNativeStackSize at startup.
func SetNativeStackSize(size uint64) {
	C.stylus_set_native_stack_size(u64(size))
}

// GetNativeStackSize returns the current process-wide default Wasmer coroutine stack size in bytes.
func GetNativeStackSize() uint64 {
	return uint64(C.stylus_get_native_stack_size())
}

// DrainStackPool discards all cached Wasmer coroutine stacks so that
// subsequent allocations use the current process-wide stack size.
func DrainStackPool() {
	C.stylus_drain_stack_pool()
}

// MinNativeStackSize is the floor enforced by Wasmer's set_stack_size (must match wasmer_vm clamping
// in crates/tools/wasmer/lib/vm/src/trap/traphandlers.rs, set_stack_size()).
const MinNativeStackSize = 8 * 1024 // 8 KB

// MaxNativeStackSize is the hard cap on Wasmer coroutine stack size (must match wasmer_vm::MAX_STACK_SIZE).
const MaxNativeStackSize = 100 * 1024 * 1024 // 100 MB

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
	runCtx *core.MessageRunContext,
) (*activationInfo, error) {
	moduleActivationMandatory := true
	suppliedGas := burner.GasLeft()
	gasLeft := suppliedGas
	shouldAllowFallback := GetAllowFallback() && runCtx.IsExecutedOnChain()
	info, asmMap, err := activateProgramInternal(program, codehash, wasm, page_limit, stylusVersion, arbosVersionForGas, debug, &gasLeft, runCtx.WasmTargets(), moduleActivationMandatory, shouldAllowFallback)
	if gasLeft < suppliedGas {
		// Ignore the out-of-gas error because we want to return the error above
		burner.Burn(multigas.ResourceKindComputation, suppliedGas-gasLeft) //nolint:errcheck
	}
	if err != nil {
		return nil, err
	}
	if err := db.ActivateWasm(info.moduleHash, asmMap); err != nil {
		return nil, err
	}
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
	target rawdb.WasmTarget,
	cranelift bool,
	timeout time.Duration,
) ([]byte, error) {
	result := containers.NewPromise[[]byte](func() {})
	go func() {
		output := &rustBytes{}
		status_asm := C.stylus_compile(
			goSlice(wasm),
			u16(stylusVersion),
			cbool(debug),
			goSlice([]byte(target)),
			cbool(cranelift),
			output,
		)
		asm := rustBytesIntoBytes(output)
		if status_asm != 0 {
			result.ProduceError(fmt.Errorf("%w: %s", ErrProgramActivation, string(asm)))
		} else {
			result.Produce(asm)
		}
	}()
	select {
	case <-result.ReadyChan():
	case <-time.After(timeout):
	}
	return result.Current()
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
	targets []rawdb.WasmTarget,
	moduleActivationMandatory bool,
	useFallback bool,
) (*activationInfo, map[rawdb.WasmTarget][]byte, error) {
	var wavmFound bool
	var nativeTargets []rawdb.WasmTarget
	for _, target := range targets {
		if target == rawdb.TargetWavm {
			wavmFound = true
		} else {
			nativeTargets = append(nativeTargets, target)
		}
	}
	type result struct {
		target rawdb.WasmTarget
		asm    []byte
		err    error
	}
	results := make(chan result)
	// info will be set in separate thread, make sure to wait before reading
	var info *activationInfo
	asmMap := make(map[rawdb.WasmTarget][]byte, len(nativeTargets)+1)
	if moduleActivationMandatory || wavmFound {
		go func() {
			var err error
			var module []byte
			info, module, err = activateModule(addressForLogging, codehash, wasm, page_limit, stylusVersion, arbosVersionForGas, debug, gasLeft)
			results <- result{target: rawdb.TargetWavm, asm: module, err: err}
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
		go func() {
			if target == rawdb.TargetWasm {
				results <- result{target: target, asm: wasm}
			} else {
				cranelift := rawdb.IsCraneliftTarget(target)
				// compileNative needs the base target name for the Rust target
				// cache lookup (cranelift variant names are not registered there).
				// The cranelift bool selects the compiler backend independently.
				compileTarget := target
				if cranelift {
					if base, err := rawdb.BaseTarget(target); err == nil {
						compileTarget = base
					}
				}
				timeout := time.Second * 15
				asm, err := compileNative(wasm, stylusVersion, debug, compileTarget, cranelift, timeout)
				if err != nil {
					var fallbackTarget rawdb.WasmTarget
					var fallbackErr error
					if cranelift {
						fallbackTarget, fallbackErr = rawdb.BaseTarget(target)
					} else {
						fallbackTarget, fallbackErr = rawdb.CraneliftTarget(target)
					}
					if useFallback && fallbackErr == nil {
						log.Warn("stylus compilation failed, falling back to alternative compiler", "address", addressForLogging, "target", target, "fallbackTarget", fallbackTarget, "timeout", timeout, "err", err)
						asm, err = compileNative(wasm, stylusVersion, debug, compileTarget, !cranelift, timeout)
						results <- result{target: fallbackTarget, asm: asm, err: err}
						return
					} else if !useFallback {
						log.Warn("stylus compilation failed and fallback is disabled", "address", addressForLogging, "target", target, "timeout", timeout, "err", err)
					} else {
						log.Error("stylus compilation failed and fallback target lookup also failed",
							"address", addressForLogging, "target", target, "compileErr", err, "fallbackTargetErr", fallbackErr)
					}
				}
				results <- result{target: target, asm: asm, err: err}
			}
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
			err = errors.Join(err, fmt.Errorf("%s: %w", res.target, res.err))
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
			"useFallback", useFallback,
			"err", err,
		)
		panic(fmt.Sprintf("Compilation of %v failed for one or more targets despite activation succeeding (useFallback=%v): %v", addressForLogging, useFallback, err))
	}
	return info, asmMap, err
}

// getCompiledProgram gets compiled wasm for all targets and recompiles missing ones.
func getCompiledProgram(statedb vm.StateDB, moduleHash common.Hash, addressForLogging common.Address, code []byte, codehash common.Hash, params *StylusParams, time uint64, debugMode bool, program Program, runCtx *core.MessageRunContext) (map[rawdb.WasmTarget][]byte, error) {
	targets := runCtx.WasmTargets()
	// even though we need only asm for local target, make sure that all configured targets are available as they are needed during multi-target recording of a program call
	asmMap, missingTargets, err := statedb.ActivatedAsmMap(targets, moduleHash)
	if err != nil {
		// exit early in case of an unexpected error
		// e.g. caused by inconsistent recent activations stored in statedb (not yet committed activation that doesn't contain asms for all targets)
		log.Error("unexpected error reading asm map", "err", err)
		return nil, err
	}
	if len(missingTargets) == 0 {
		return asmMap, nil
	}

	// addressForLogging may be empty or may not correspond to the code, so we need to be careful to use the code passed in separately
	wasm, err := getWasmFromContractCode(statedb, code, params, nil)
	if err != nil {
		log.Error("Failed to reactivate program: getWasm", "address", addressForLogging, "expected moduleHash", moduleHash, "err", err)
		return nil, fmt.Errorf("failed to reactivate program address: %v err: %w", addressForLogging, err)
	}

	// don't charge gas
	zeroArbosVersion := uint64(0)
	zeroGas := uint64(0)

	// we know program is activated, so it must be in correct version and not use too much memory
	moduleActivationMandatory := false
	// compile only missing targets
	shouldAllowFallback := GetAllowFallback() && runCtx.IsExecutedOnChain()
	info, newlyBuilt, err := activateProgramInternal(addressForLogging, codehash, wasm, params.PageLimit, program.version, zeroArbosVersion, debugMode, &zeroGas, missingTargets, moduleActivationMandatory, shouldAllowFallback)
	if err != nil {
		log.Error("failed to reactivate program", "address", addressForLogging, "expected moduleHash", moduleHash, "err", err)
		return nil, fmt.Errorf("failed to reactivate program address: %v err: %w", addressForLogging, err)
	}

	if info != nil && info.moduleHash != moduleHash {
		log.Error("failed to reactivate program", "address", addressForLogging, "expected moduleHash", moduleHash, "got", info.moduleHash)
		return nil, fmt.Errorf("failed to reactivate program. address: %v, expected ModuleHash: %v", addressForLogging, moduleHash)
	}

	// merge in the newly built asms (includes both regular and cranelift targets)
	for target, asm := range newlyBuilt {
		asmMap[target] = asm
	}
	currentHoursSince := hoursSinceArbitrum(time)
	if currentHoursSince > program.activatedAt {
		// stylus program is active on-chain, and was activated in the past
		// so we store it directly to database (including cranelift entries)
		batch := statedb.Database().WasmStore().NewBatch()
		// rawdb.WriteActivation iterates over the asms map and writes each entry separately to wasmdb, so the writes for the same module hash can be incremental
		// we know that all targets for which asms were found initially, were read from disk as oppose to from newly activated asms from memory, as otherwise statedb.ActivatedAsmMap would have failed with an error because of missing targets within newly activated asms
		rawdb.WriteActivation(batch, moduleHash, newlyBuilt)
		if err := batch.Write(); err != nil {
			log.Error("failed writing re-activation to state", "address", addressForLogging, "err", err)
		}
	} else {
		// we need to add asms for all targets to the newly activated targets (not only the newly built) to maintain consistency the newly activated targets map
		if err := statedb.ActivateWasm(moduleHash, asmMap); err != nil {
			log.Error("Failed to store recent activation of wasm in statedb", "err", err)
			return nil, err
		}
	}
	return asmMap, nil
}

// selectLocalAsm returns the best available ASM for the local target from an asmMap.
// When allowFallback is enabled and cranelift ASM exists, cranelift is preferred.
// Cranelift ASM may be present from compile-time fallback (singlepass compilation
// failed, so cranelift was used instead during activation).
func selectLocalAsm(asmMap map[rawdb.WasmTarget][]byte) ([]byte, bool) {
	if asmMap == nil {
		return nil, false
	}
	localTarget := rawdb.LocalTarget()
	craneliftTarget, ctErr := rawdb.CraneliftTarget(localTarget)

	// When fallback is enabled and cranelift ASM exists, prefer it.
	if GetAllowFallback() && ctErr == nil {
		if asm, ok := asmMap[craneliftTarget]; ok && len(asm) > 0 {
			return asm, true
		}
	}

	// Default: use singlepass ASM.
	if asm, ok := asmMap[localTarget]; ok && len(asm) > 0 {
		return asm, true
	}

	return nil, false
}

func handleProgramPrepare(statedb vm.StateDB, moduleHash common.Hash, addressForLogging common.Address, code []byte, codehash common.Hash, params *StylusParams, time uint64, debugMode bool, program Program, runCtx *core.MessageRunContext) []byte {
	asmMap, err := getCompiledProgram(statedb, moduleHash, addressForLogging, code, codehash, params, time, debugMode, program, runCtx)
	localAsm, ok := selectLocalAsm(asmMap)
	if err != nil || !ok {
		panic(fmt.Sprintf("failed to get compiled program for activated program, program: %v, local target missing: %v, err: %v", addressForLogging.Hex(), !ok, err))
	}

	return localAsm
}

// doStylusCall performs a single CGo call to stylus_call and returns the status and output.
// Each invocation creates a fresh NativeApi because the Go-side API object is
// deregistered on drop, so the handler ID cannot be reused across calls.
func doStylusCall(
	asm []byte,
	calldata []byte,
	stylusParams *ProgParams,
	evm *vm.EVM,
	tracingInfo *util.TracingInfo,
	scope *vm.ScopeContext,
	memoryModel *MemoryModel,
	evmData *EvmData,
	debug bool,
	runCtx *core.MessageRunContext,
) (userStatus, []byte) {
	evmApi := newApi(evm, tracingInfo, scope, memoryModel)
	defer evmApi.drop()

	output := &rustBytes{}
	status := userStatus(C.stylus_call(
		goSlice(asm),
		goSlice(calldata),
		stylusParams.encode(),
		evmApi.cNative,
		evmData.encode(),
		cbool(debug),
		output,
		(*u64)(&scope.Contract.Gas),
		u32(runCtx.WasmCacheTag()),
	))
	return status, rustBytesIntoBytes(output)
}

func callProgram(
	address common.Address,
	moduleHash common.Hash,
	localAsm []byte,
	scope *vm.ScopeContext,
	evm *vm.EVM,
	tracingInfo *util.TracingInfo,
	calldata []byte,
	evmData *EvmData,
	stylusParams *ProgParams,
	memoryModel *MemoryModel,
	runCtx *core.MessageRunContext,
	code []byte,
	params *StylusParams,
	program Program,
) ([]byte, error) {
	db := evm.StateDB
	debug := stylusParams.DebugMode

	if len(localAsm) == 0 {
		log.Error("missing asm", "program", address, "module", moduleHash)
		panic("missing asm")
	}

	if runCtx.IsRecording() {
		if stateDB, ok := db.(*state.StateDB); ok {
			if err := stateDB.RecordProgram(runCtx.WasmTargets(), moduleHash); err != nil {
				log.Error("failed to record program", "program", address, "module", moduleHash, "err", err)
				panic(fmt.Sprintf("failed to record program: %v", err))
			}
		}
	}

	openPages, everPages := db.GetStylusPages()
	saved := savedState{
		gas:          scope.Contract.Gas,
		usedMultiGas: scope.Contract.UsedMultiGas,
		openPages:    openPages,
		everPages:    everPages,
	}

	// Snapshot the StateDB before the first attempt so we can revert any partial
	// state changes (storage writes, subcalls) if the call hits a native stack
	// overflow and we need to retry. Snapshot is cheap — it just records a journal ID.
	snapshot := db.Snapshot()

	// First attempt with the locally-compiled ASM (singlepass or cranelift, depending on activation).
	status, output := doStylusCall(localAsm, calldata, stylusParams, evm, tracingInfo, scope, memoryModel, evmData, debug, runCtx)

	if status == userNativeStackOverflow {
		status, output = handleNativeStackOverflow(
			address, moduleHash,
			scope, evm, tracingInfo, calldata, evmData, stylusParams,
			memoryModel, runCtx, saved, debug, db, snapshot,
			code, params, program,
		)
	}

	depth := evm.Depth()
	// Panic if native stack overflow was not resolved after all recovery
	// attempts (cranelift retry + stack doubling). An activated program that
	// cannot execute would cause this node to produce wrong state — diverging
	// from nodes that CAN run it — which is a consensus failure. This matches
	// the panics in handleProgramPrepare and cacheProgram for the same reason.
	if status == userNativeStackOverflow {
		panic(fmt.Sprintf("native stack overflow not resolved (program=%v, module=%v, depth=%d, allowFallback=%v, onChain=%v, stackSize=%d)",
			address, moduleHash, depth, GetAllowFallback(), runCtx.IsExecutedOnChain(), GetNativeStackSize()))
	}
	data, msg, err := status.toResult(output, debug)
	if status == userFailure && debug {
		log.Warn("program failure", "err", err, "msg", msg, "program", address, "depth", depth)
	}
	if tracingInfo != nil {
		tracingInfo.CaptureStylusExit(uint8(status), data, err, scope.Contract.Gas)
	}
	return data, err
}

// restoreState reverts the StateDB to the given snapshot and restores the
// contract gas, multi-gas, and Stylus page counters from the saved checkpoint.
// openWasmPages/everWasmPages are not journaled, so RevertToSnapshot alone
// does not restore them — we must do it explicitly.
func restoreState(scope *vm.ScopeContext, saved savedState, db vm.StateDB, snapshot int) {
	db.RevertToSnapshot(snapshot)
	scope.Contract.Gas = saved.gas
	scope.Contract.UsedMultiGas = saved.usedMultiGas
	db.SetStylusPages(saved.openPages, saved.everPages)
}

// handleNativeStackOverflow handles a native stack overflow by compiling
// cranelift ASM and retrying. On the first overflow it also doubles the
// process-wide Wasmer coroutine stack size (kept permanently); subsequent
// overflows retry with cranelift at the already-doubled size.
//
// Returns userNativeStackOverflow without retrying when fallback is disabled,
// off-chain, or cranelift ASM is unavailable — the caller panics in that case.
func handleNativeStackOverflow(
	address common.Address,
	moduleHash common.Hash,
	scope *vm.ScopeContext,
	evm *vm.EVM,
	tracingInfo *util.TracingInfo,
	calldata []byte,
	evmData *EvmData,
	stylusParams *ProgParams,
	memoryModel *MemoryModel,
	runCtx *core.MessageRunContext,
	saved savedState,
	debug bool,
	db vm.StateDB,
	snapshot int,
	code []byte,
	params *StylusParams,
	program Program,
) (userStatus, []byte) {
	if !GetAllowFallback() {
		log.Warn("native stack overflow, fallback disabled",
			"program", address, "module", moduleHash)
		return userNativeStackOverflow, nil
	}
	if !runCtx.IsExecutedOnChain() {
		log.Info("native stack overflow, no stack doubling for off-chain execution",
			"program", address, "module", moduleHash)
		return userNativeStackOverflow, nil
	}

	craneliftAsm, err := getCraneliftAsm(moduleHash, address, db, code, params, program.version, debug)
	if err != nil || len(craneliftAsm) == 0 {
		log.Error("native stack overflow, cranelift ASM unavailable",
			"program", address, "module", moduleHash, "err", err)
		return userNativeStackOverflow, nil
	}

	// On the first overflow, double the process-wide Wasmer coroutine stack size.
	// On subsequent overflows doubleNativeStackSize is a no-op (baseline already
	// consumed via CAS) — we still retry with cranelift at the already-doubled size.
	if doubleNativeStackSize() {
		log.Warn("native stack overflow, doubled stack size",
			"program", address, "module", moduleHash, "newSize", GetNativeStackSize())
	}

	// Revert any partial state changes the previous attempt may have made via
	// host I/O before the overflow, then retry with cranelift.
	restoreState(scope, saved, db, snapshot)

	return doStylusCall(craneliftAsm, calldata, stylusParams, evm, tracingInfo, scope, memoryModel, evmData, debug, runCtx)
}

// getCraneliftAsm returns cranelift-compiled ASM for the given module.
// It first checks the persistent wasm store. If not found, it compiles
// with cranelift and persists the result to the wasm store.
func getCraneliftAsm(
	moduleHash common.Hash,
	address common.Address,
	db vm.StateDB,
	code []byte,
	params *StylusParams,
	version uint16,
	debug bool,
) ([]byte, error) {
	craneliftTarget, err := rawdb.CraneliftTarget(rawdb.LocalTarget())
	if err != nil {
		return nil, fmt.Errorf("failed to determine cranelift target: %w", err)
	}

	// Check in-memory activatedWasms (same-block activation) then persistent wasm store.
	if asm := db.ActivatedAsm(craneliftTarget, moduleHash); len(asm) > 0 {
		return asm, nil
	}

	// Compile with cranelift from the contract code.
	if len(code) == 0 || params == nil {
		return nil, fmt.Errorf("cranelift ASM not cached and contract code/params unavailable for compilation")
	}

	log.Warn("native stack overflow, compiling with cranelift",
		"program", address, "module", moduleHash)

	wasm, err := getWasmFromContractCode(db, code, params, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get wasm for cranelift compilation: %w", err)
	}

	asm, err := compileNative(wasm, version, debug, rawdb.LocalTarget(), true, 15*time.Second)
	if err != nil {
		return nil, fmt.Errorf("cranelift compilation failed: %w", err)
	}

	// Persist to wasm store.
	wasmStore := db.Database().WasmStore()
	if wasmStore != nil {
		batch := wasmStore.NewBatch()
		rawdb.WriteActivatedAsm(batch, craneliftTarget, moduleHash, asm)
		if err := batch.Write(); err != nil {
			log.Warn("failed to persist cranelift ASM to wasm store, will recompile on next overflow",
				"program", address, "err", err)
		}
	}

	return asm, nil
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
func cacheProgram(db vm.StateDB, module common.Hash, program Program, addressForLogging common.Address, code []byte, codehash common.Hash, params *StylusParams, debug bool, time uint64, runCtx *core.MessageRunContext) {
	if runCtx.IsCommitMode() {
		// address is only used for logging
		asmMap, err := getCompiledProgram(db, module, addressForLogging, code, codehash, params, time, debug, program, runCtx)
		localAsm, ok := selectLocalAsm(asmMap)
		if err != nil || !ok {
			panic(fmt.Sprintf("failed to get compiled program for caching, program: %v, local target missing: %v, err: %v", addressForLogging.Hex(), !ok, err))
		}
		tag := runCtx.WasmCacheTag()
		state.CacheWasmRust(localAsm, module, program.version, tag, debug)
		db.RecordCacheWasm(state.CacheWasm{ModuleHash: module, Version: program.version, Tag: tag, Debug: debug})
	}
}

// Evicts a program in Rust. We write a record so that we can undo on revert, unless we don't need to (e.g. expired)
// For gas estimation and eth_call, we ignore permanent updates and rely on Rust's LRU.
func evictProgram(db vm.StateDB, module common.Hash, version uint16, debug bool, runCtx *core.MessageRunContext, forever bool) {
	if runCtx.IsCommitMode() {
		tag := runCtx.WasmCacheTag()
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

func SetTarget(name rawdb.WasmTarget, description string, native bool) error {
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
