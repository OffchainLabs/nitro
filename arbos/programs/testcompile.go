// Copyright 2024-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

//go:build !wasm

package programs

// This file provides CGo test helpers for native Stylus compilation and execution.
// It lives here (not in _test.go) because cgo isn't allowed in test files.

/*
#cgo CFLAGS: -g -I../../target/include/
#include "arbitrator.h"

typedef uint16_t u16;
typedef uint32_t u32;
typedef uint64_t u64;
typedef size_t usize;

void handleReqWrap(usize api, u32 req_type, RustSlice *data, u64 *out_cost, GoSliceData *out_result, GoSliceData *out_raw_data);
*/
import "C"

import (
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/holiman/uint256"

	"github.com/ethereum/go-ethereum/arbitrum/multigas"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
)

// A program that recurses a fixed number of times (500). At 32KB native
// stack this overflows before completing. At 1MB native stack it completes
// and returns 0. The counter is stored in WASM memory at offset 0.
var recursiveStackOverflowWat = []byte(`(module
	(memory 1 1)
	(export "memory" (memory 0))
	(func $main (export "user_entrypoint") (param $args_len i32) (result i32)
		;; Load counter from memory[0]
		i32.const 0
		i32.load
		;; If counter >= 500, return 0
		i32.const 500
		i32.ge_u
		if (result i32)
			i32.const 0
		else
			;; Increment counter in memory[0]
			i32.const 0
			i32.const 0
			i32.load
			i32.const 1
			i32.add
			i32.store
			;; Recurse
			i32.const 0
			call $main
		end
	)
)`)

func Wat2Wasm(wat []byte) ([]byte, error) {
	output := &rustBytes{}

	status := C.wat_to_wasm(goSlice(wat), output)

	if status != 0 {
		return nil, fmt.Errorf("failed reading wat file: %v", string(rustBytesIntoBytes(output)))
	}

	return rustBytesIntoBytes(output), nil
}

func testCompileArch(store bool, cranelift bool) error {

	localTarget := rawdb.LocalTarget()
	nativeArm64 := localTarget == rawdb.TargetArm64
	nativeAmd64 := localTarget == rawdb.TargetAmd64
	timeout := time.Minute

	nameSuffix := ".bin"
	if cranelift {
		nameSuffix = "_cranelift.bin"
	}
	_, err := fmt.Print("starting test.. native arm? ", nativeArm64, " amd? ", nativeAmd64, " GOARCH/GOOS: ", runtime.GOARCH+"/"+runtime.GOOS, "\n")
	if err != nil {
		return err
	}

	err = SetTarget(rawdb.TargetArm64, DefaultTargetDescriptionArm, nativeArm64)
	if err != nil {
		return err
	}

	err = SetTarget(rawdb.TargetAmd64, DefaultTargetDescriptionX86, nativeAmd64)
	if err != nil {
		return err
	}

	if !(nativeArm64 || nativeAmd64) {
		err = SetTarget(localTarget, "", true)
		if err != nil {
			return err
		}
	}

	source, err := os.ReadFile("../../crates/stylus/tests/add.wat")
	if err != nil {
		return fmt.Errorf("failed reading stylus contract: %w", err)
	}

	wasm, err := Wat2Wasm(source)
	if err != nil {
		return err
	}

	if store {
		_, err := fmt.Print("storing compiled files to ../../target/testdata/\n")
		if err != nil {
			return err
		}
		err = os.MkdirAll("../../target/testdata", 0755)
		if err != nil {
			return err
		}
	}

	_, err = compileNative(wasm, 2, true, "booga", false, timeout)
	if err == nil {
		return fmt.Errorf("succeeded compiling non-existent arch: %w", err)
	}

	outBytes, err := compileNative(wasm, 1, true, localTarget, cranelift, timeout)

	if err != nil {
		return fmt.Errorf("failed compiling native: %w", err)
	}
	if store && !nativeAmd64 && !nativeArm64 {
		_, err := fmt.Printf("writing host file\n")
		if err != nil {
			return err
		}

		err = os.WriteFile("../../target/testdata/host"+nameSuffix, outBytes, 0644)
		if err != nil {
			return err
		}
	}

	outBytes, err = compileNative(wasm, 1, true, rawdb.TargetArm64, cranelift, timeout)

	if err != nil {
		return fmt.Errorf("failed compiling arm: %w", err)
	}
	if store {
		_, err := fmt.Printf("writing arm file\n")
		if err != nil {
			return err
		}

		err = os.WriteFile("../../target/testdata/arm64"+nameSuffix, outBytes, 0644)
		if err != nil {
			return err
		}
	}

	outBytes, err = compileNative(wasm, 1, true, rawdb.TargetAmd64, cranelift, timeout)

	if err != nil {
		return fmt.Errorf("failed compiling amd: %w", err)
	}
	if store {
		_, err := fmt.Printf("writing amd64 file\n")
		if err != nil {
			return err
		}

		err = os.WriteFile("../../target/testdata/amd64"+nameSuffix, outBytes, 0644)
		if err != nil {
			return err
		}
	}

	return nil
}

func resetNativeTarget() error {
	output := &rustBytes{}

	_, err := fmt.Print("resetting native target\n")
	if err != nil {
		return err
	}

	localCompileName := []byte("local")

	status := C.stylus_target_set(goSlice(localCompileName),
		goSlice([]byte{}),
		output,
		cbool(true))

	if status != 0 {
		return fmt.Errorf("failed setting compilation target arm: %v", string(rustBytesIntoBytes(output)))
	}

	return nil
}

func testCompileLoadFor(filePath string) error {
	_, err := fmt.Print("starting load test. FilePath: ", filePath, " GOARCH/GOOS: ", runtime.GOARCH+"/"+runtime.GOOS, "\n")
	if err != nil {
		return err
	}

	localAsm, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	calldata := []byte{}

	evmData := EvmData{}
	progParams := ProgParams{
		MaxDepth:  1000000,
		InkPrice:  1,
		DebugMode: true,
	}
	reqHandler := C.NativeRequestHandler{
		handle_request_fptr: (*[0]byte)(C.handleReqWrap),
		id:                  0,
	}

	inifiniteGas := u64(0xfffffffffffffff)

	output := &rustBytes{}

	_, err = fmt.Print("launching program..\n")
	if err != nil {
		return err
	}

	status := userStatus(C.stylus_call(
		goSlice(localAsm),
		goSlice(calldata),
		progParams.encode(),
		reqHandler,
		evmData.encode(),
		cbool(true),
		output,
		&inifiniteGas,
		u32(0),
	))

	_, err = fmt.Print("returned: ", status, "\n")
	if err != nil {
		return err
	}

	_, msg, err := status.toResult(rustBytesIntoBytes(output), true)
	if status == userFailure {
		err = fmt.Errorf("%w: %v", err, msg)
	}

	return err
}

// testNativeStackSize tests that:
// 1. SetNativeStackSize correctly configures the Wasmer coroutine stack size.
// 2. A program that overflows a tiny native stack returns NativeStackOverflow.
// 3. Increasing the stack size and re-running succeeds.
//
// It compiles and runs a WAT program with deep recursion (500 calls), using a very
// high MaxStackDepth and a very small native stack so it overflows.
func testNativeStackSize() error {
	localTarget := rawdb.LocalTarget()
	err := SetTarget(localTarget, "", true)
	if err != nil {
		return fmt.Errorf("failed setting target: %w", err)
	}

	wasm, err := Wat2Wasm(recursiveStackOverflowWat)
	if err != nil {
		return fmt.Errorf("failed compiling WAT: %w", err)
	}

	localAsm, err := compileNative(wasm, 1, true, localTarget, false, time.Minute)
	if err != nil {
		return fmt.Errorf("failed compiling native: %w", err)
	}

	// Set a very small native stack (32 KB) to force overflow quickly.
	SetNativeStackSize(32 * 1024)

	calldata := []byte{}
	evmData := EvmData{}
	progParams := ProgParams{
		MaxDepth:  1000000,
		InkPrice:  1,
		DebugMode: true,
	}
	reqHandler := C.NativeRequestHandler{
		handle_request_fptr: (*[0]byte)(C.handleReqWrap),
		id:                  0,
	}

	gas := u64(0xfffffffffffffff)
	output := &rustBytes{}

	// With a 32KB stack the recursive program will overflow immediately.
	// Rust now returns NativeStackOverflow directly (Go handles retries).
	status := userStatus(C.stylus_call(
		goSlice(localAsm),
		goSlice(calldata),
		progParams.encode(),
		reqHandler,
		evmData.encode(),
		cbool(true),
		output,
		&gas,
		u32(0),
	))

	rustBytesIntoBytes(output)

	if status != userNativeStackOverflow {
		return fmt.Errorf("expected userNativeStackOverflow with 32KB stack, got status %d", status)
	}

	// Now increase the stack and run again to verify the program succeeds
	// with a larger stack (simulating what future calls get after doubling).
	SetNativeStackSize(1024 * 1024)
	DrainStackPool()

	gas = u64(0xfffffffffffffff)
	output = &rustBytes{}
	reqHandler2 := C.NativeRequestHandler{
		handle_request_fptr: (*[0]byte)(C.handleReqWrap),
		id:                  0,
	}

	status = userStatus(C.stylus_call(
		goSlice(localAsm),
		goSlice(calldata),
		progParams.encode(),
		reqHandler2,
		evmData.encode(),
		cbool(true),
		output,
		&gas,
		u32(0),
	))

	rustBytesIntoBytes(output)

	// With 1MB stack the program should complete its 500 recursions successfully.
	if status == userNativeStackOverflow {
		return fmt.Errorf("still got NativeStackOverflow with 1MB stack")
	}
	if status != userSuccess {
		return fmt.Errorf("expected success after stack increase, got status %d", status)
	}

	// Verify SetNativeStackSize(0) is a no-op.
	SetNativeStackSize(0)
	if got := GetNativeStackSize(); got != 1024*1024 {
		return fmt.Errorf("SetNativeStackSize(0) should be a no-op, but stack size changed to %d", got)
	}

	_, err = fmt.Printf("testNativeStackSize: passed (status=%d, gas_left=%d)\n", status, gas)
	if err != nil {
		return err
	}

	return nil
}

// testNativeStackSizeMaxCap tests that a program which overflows at smaller
// stack sizes runs successfully when the stack is set to MaxNativeStackSize
// (100 MB), verifying the maximum cap is correctly applied.
func testNativeStackSizeMaxCap() error {
	localTarget := rawdb.LocalTarget()
	err := SetTarget(localTarget, "", true)
	if err != nil {
		return fmt.Errorf("failed setting target: %w", err)
	}

	wasm, err := Wat2Wasm(recursiveStackOverflowWat)
	if err != nil {
		return fmt.Errorf("failed compiling WAT: %w", err)
	}

	localAsm, err := compileNative(wasm, 1, true, localTarget, false, time.Minute)
	if err != nil {
		return fmt.Errorf("failed compiling native: %w", err)
	}

	// Set the stack to MAX_STACK_SIZE (100 MB).
	SetNativeStackSize(100 * 1024 * 1024)

	calldata := []byte{}
	evmData := EvmData{}
	progParams := ProgParams{
		MaxDepth:  1000000,
		InkPrice:  1,
		DebugMode: true,
	}
	reqHandler := C.NativeRequestHandler{
		handle_request_fptr: (*[0]byte)(C.handleReqWrap),
		id:                  0,
	}

	gas := u64(0xfffffffffffffff)
	output := &rustBytes{}

	status := userStatus(C.stylus_call(
		goSlice(localAsm),
		goSlice(calldata),
		progParams.encode(),
		reqHandler,
		evmData.encode(),
		cbool(true),
		output,
		&gas,
		u32(0),
	))

	rustBytesIntoBytes(output)

	// At 100MB stack, the program should complete its 500 recursions successfully.
	if status == userNativeStackOverflow {
		return fmt.Errorf("got NativeStackOverflow even at 100MB stack")
	}
	if status != userSuccess {
		return fmt.Errorf("expected success at max cap, got status %d", status)
	}

	// Verify the stack size is unchanged.
	currentSize := GetNativeStackSize()
	if currentSize != 100*1024*1024 {
		return fmt.Errorf("expected stack size to remain at 100MB, got %d bytes", currentSize)
	}

	_, err = fmt.Printf("testNativeStackSizeMaxCap: passed (status=%d) at max cap\n", status)
	if err != nil {
		return err
	}

	return nil
}

// testDoubleNativeStackSize verifies doubleNativeStackSize:
// 1. Returns true and doubles the stack on first call.
// 2. Returns false on second call (baseline already consumed via CAS to 0).
// 3. Returns false when baseline was never initialized (zero).
// 4. Returns false when baseline is already at MaxNativeStackSize.
// 5. SetInitialNativeStackSize re-enables doubling by restoring a non-zero baseline.
func testDoubleNativeStackSize() error {
	// 1. First call doubles the stack.
	SetInitialNativeStackSize(32 * 1024)
	if !doubleNativeStackSize() {
		return fmt.Errorf("first call: expected true (should double from 32KB to 64KB)")
	}
	if got := GetNativeStackSize(); got != 64*1024 {
		return fmt.Errorf("first call: expected 64KB, got %d", got)
	}

	// 2. Second call returns false (baseline consumed to 0).
	if doubleNativeStackSize() {
		return fmt.Errorf("second call: expected false (already doubled)")
	}
	// Stack size should remain at 64KB.
	if got := GetNativeStackSize(); got != 64*1024 {
		return fmt.Errorf("second call: expected stack to remain at 64KB, got %d", got)
	}

	// 3. Zero baseline (not initialized) returns false.
	nativeStackBaseline.Store(0)
	if doubleNativeStackSize() {
		return fmt.Errorf("zero baseline: expected false")
	}

	// 4. Baseline at MaxNativeStackSize returns false (doubled would exceed cap).
	SetInitialNativeStackSize(MaxNativeStackSize)
	if doubleNativeStackSize() {
		return fmt.Errorf("max baseline: expected false (cannot double beyond max)")
	}
	if got := GetNativeStackSize(); got != MaxNativeStackSize {
		return fmt.Errorf("max baseline: expected stack to remain at %d, got %d", MaxNativeStackSize, got)
	}

	// 5. SetInitialNativeStackSize re-enables doubling.
	SetInitialNativeStackSize(32 * 1024)
	if !doubleNativeStackSize() {
		return fmt.Errorf("re-enabled: expected true after SetInitialNativeStackSize reset")
	}
	if got := GetNativeStackSize(); got != 64*1024 {
		return fmt.Errorf("re-enabled: expected 64KB, got %d", got)
	}

	fmt.Printf("testDoubleNativeStackSize: passed\n")
	return nil
}

func testCompileLoad() error {
	filePathStart := "../../target/testdata/host"
	localTarget := rawdb.LocalTarget()
	if localTarget == rawdb.TargetArm64 {
		filePathStart = "../../target/testdata/arm64"
	}
	if localTarget == rawdb.TargetAmd64 {
		filePathStart = "../../target/testdata/amd64"
	}
	err := testCompileLoadFor(filePathStart + ".bin")
	if err != nil {
		return err
	}
	return testCompileLoadFor(filePathStart + "_cranelift.bin")
}

// makeTestEVMScope creates minimal EVM and ScopeContext objects suitable for
// running test Stylus programs that don't invoke EVM host calls.
func makeTestEVMScope(gas uint64) (*vm.EVM, *vm.ScopeContext, vm.StateDB) {
	statedb, _ := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
	evm := vm.NewEVM(vm.BlockContext{}, statedb, params.TestChainConfig, vm.Config{})
	contract := vm.NewContract(common.Address{}, common.Address{1}, new(uint256.Int), gas, nil)
	scope := &vm.ScopeContext{Contract: contract}
	return evm, scope, statedb
}

// testHandleNativeStackOverflow tests that handleNativeStackOverflow:
// 1. Does not retry when allowFallback is false.
// 2. Does not retry for off-chain execution.
// 3. Doubles stack and retries with cranelift on first on-chain overflow.
// 4. Returns overflow immediately when cranelift is unavailable.
func testHandleNativeStackOverflow() error {
	savedFallback := GetAllowFallback()
	defer SetAllowFallback(savedFallback)

	localTarget := rawdb.LocalTarget()
	if err := SetTarget(localTarget, "", true); err != nil {
		return fmt.Errorf("failed setting target: %w", err)
	}

	wasm, err := Wat2Wasm(recursiveStackOverflowWat)
	if err != nil {
		return fmt.Errorf("failed compiling WAT: %w", err)
	}

	// Compile cranelift ASM and pre-populate the wasm store for sub-test 3.
	craneliftAsm, err := compileNative(wasm, 1, true, localTarget, true, time.Minute)
	if err != nil {
		return fmt.Errorf("failed compiling cranelift: %w", err)
	}

	SetInitialNativeStackSize(32 * 1024)
	DrainStackPool()

	moduleHash := common.HexToHash("0x1234567890abcdef")
	stylusParams := &ProgParams{Version: 1, MaxDepth: 1000000, InkPrice: 1, DebugMode: true}
	memModel := NewMemoryModel(0, 0)

	gas := uint64(0xfffffffffffffff)
	evm, scope, db := makeTestEVMScope(gas)
	scope.Contract.Gas = gas

	// Pre-populate cranelift ASM in the wasm store so getCraneliftAsm finds it.
	craneliftTarget, err := rawdb.CraneliftTarget(localTarget)
	if err != nil {
		return fmt.Errorf("failed getting cranelift target: %w", err)
	}
	wasmStore := db.Database().WasmStore()
	batch := wasmStore.NewBatch()
	rawdb.WriteActivatedAsm(batch, craneliftTarget, moduleHash, craneliftAsm)
	if err := batch.Write(); err != nil {
		return fmt.Errorf("failed to persist cranelift ASM to wasm store: %w", err)
	}
	// Sub-test 1: allowFallback=false → no retry, returns overflow.
	allowFallback.Store(false)
	runCtx := core.NewMessageCommitContext([]rawdb.WasmTarget{localTarget})

	snapshot := db.Snapshot()
	status, _ := handleNativeStackOverflow(
		common.Address{}, moduleHash,
		scope, evm, nil, []byte{}, &EvmData{}, stylusParams,
		memModel, runCtx, savedState{gas: gas}, true,
		db, snapshot, nil, nil, Program{version: 1},
	)
	if status != userNativeStackOverflow {
		return fmt.Errorf("allowFallback=false: expected NativeStackOverflow, got %d", status)
	}
	if nativeStackBaseline.Load() == 0 {
		return fmt.Errorf("allowFallback=false: stack should not have been doubled")
	}

	// Sub-test 2: allowFallback=true but off-chain → no retry.
	allowFallback.Store(true)
	offChainCtx := core.NewMessageGasEstimationContext()
	scope.Contract.Gas = gas

	status, _ = handleNativeStackOverflow(
		common.Address{}, moduleHash,
		scope, evm, nil, []byte{}, &EvmData{}, stylusParams,
		memModel, offChainCtx, savedState{gas: gas}, true,
		db, snapshot, nil, nil, Program{version: 1},
	)
	if status != userNativeStackOverflow {
		return fmt.Errorf("off-chain: expected NativeStackOverflow, got %d", status)
	}
	if nativeStackBaseline.Load() == 0 {
		return fmt.Errorf("off-chain: stack should not have been doubled")
	}

	// Sub-test 3: on-chain with allowFallback=true → doubles stack and retries
	// with cranelift. The stack should go from 32KB to 64KB, and the cranelift
	// retry at 64KB should succeed for the 500-recursion program.
	scope.Contract.Gas = gas
	snapshot = db.Snapshot()
	status, _ = handleNativeStackOverflow(
		common.Address{}, moduleHash,
		scope, evm, nil, []byte{}, &EvmData{}, stylusParams,
		memModel, runCtx, savedState{gas: gas}, true,
		db, snapshot, nil, nil, Program{version: 1},
	)
	if status != userSuccess {
		return fmt.Errorf("on-chain: expected success after stack doubling + cranelift retry, got %d", status)
	}
	if got := GetNativeStackSize(); got != 64*1024 {
		return fmt.Errorf("on-chain: expected stack to be doubled to 64KB, got %d", got)
	}

	// Sub-test 4: no cranelift available → returns overflow immediately.
	// Use a fresh DB with no cranelift ASM in the wasm store so
	// getCraneliftAsm fails and handleNativeStackOverflow returns without
	// any retry or doubling attempt.
	// Note: nativeStackBaseline is 0 here (consumed by sub-test 3's doubling),
	// but that doesn't affect this test since it returns before reaching doubleNativeStackSize.
	SetNativeStackSize(32 * 1024)
	DrainStackPool()
	evm4, scope4, db4 := makeTestEVMScope(gas)
	scope4.Contract.Gas = gas
	snapshot = db4.Snapshot()
	status, _ = handleNativeStackOverflow(
		common.Address{}, moduleHash,
		scope4, evm4, nil, []byte{}, &EvmData{}, stylusParams,
		memModel, runCtx, savedState{gas: gas}, true,
		db4, snapshot, nil, nil, Program{version: 1},
	)
	if status != userNativeStackOverflow {
		return fmt.Errorf("no cranelift available: expected NativeStackOverflow, got %d", status)
	}

	fmt.Printf("testHandleNativeStackOverflow: passed\n")
	return nil
}

// testHandleNativeStackOverflowAtMax verifies that handleNativeStackOverflow
// works correctly at MaxNativeStackSize with cranelift available. The program
// (500 recursions) should succeed with cranelift at 100MB, and the stack size
// must remain at MaxNativeStackSize (doubling is a no-op since the stack is
// already at max).
func testHandleNativeStackOverflowAtMax() error {
	savedFallback := GetAllowFallback()
	defer SetAllowFallback(savedFallback)

	localTarget := rawdb.LocalTarget()
	if err := SetTarget(localTarget, "", true); err != nil {
		return fmt.Errorf("failed setting target: %w", err)
	}

	wasm, err := Wat2Wasm(recursiveStackOverflowWat)
	if err != nil {
		return fmt.Errorf("failed compiling WAT: %w", err)
	}

	craneliftAsm, err := compileNative(wasm, 1, true, localTarget, true, time.Minute)
	if err != nil {
		return fmt.Errorf("failed compiling cranelift: %w", err)
	}

	SetInitialNativeStackSize(MaxNativeStackSize)
	DrainStackPool()
	allowFallback.Store(true)

	moduleHash := common.HexToHash("0x1234567890abcdef")
	stylusParams := &ProgParams{Version: 1, MaxDepth: 1000000, InkPrice: 1, DebugMode: true}
	memModel := NewMemoryModel(0, 0)
	gas := uint64(0xfffffffffffffff)
	evm, scope, db := makeTestEVMScope(gas)
	scope.Contract.Gas = gas
	runCtx := core.NewMessageCommitContext([]rawdb.WasmTarget{localTarget})

	// Pre-populate cranelift ASM in the wasm store.
	craneliftTarget, err := rawdb.CraneliftTarget(localTarget)
	if err != nil {
		return fmt.Errorf("failed getting cranelift target: %w", err)
	}
	wasmStore := db.Database().WasmStore()
	batch := wasmStore.NewBatch()
	rawdb.WriteActivatedAsm(batch, craneliftTarget, moduleHash, craneliftAsm)
	if err := batch.Write(); err != nil {
		return fmt.Errorf("failed to persist cranelift ASM: %w", err)
	}

	snapshot := db.Snapshot()
	status, _ := handleNativeStackOverflow(
		common.Address{}, moduleHash,
		scope, evm, nil, []byte{}, &EvmData{}, stylusParams,
		memModel, runCtx, savedState{gas: gas}, true,
		db, snapshot, nil, nil, Program{version: 1},
	)
	// Cranelift at 100MB should succeed (500 recursions is trivial at this stack size).
	if status != userSuccess {
		return fmt.Errorf("expected success from cranelift at max stack, got %d", status)
	}
	// Stack size must remain at MaxNativeStackSize (no doubling attempted).
	if got := GetNativeStackSize(); got != MaxNativeStackSize {
		return fmt.Errorf("expected stack to remain at %d, got %d", MaxNativeStackSize, got)
	}
	if nativeStackBaseline.Load() == 0 {
		return fmt.Errorf("baseline should not have been consumed (no doubling at max)")
	}

	fmt.Printf("testHandleNativeStackOverflowAtMax: passed\n")
	return nil
}

// testRetryRestoresStylusPages verifies that handleNativeStackOverflow correctly
// restores openWasmPages and everWasmPages before retrying. These fields are not
// journaled, so RevertToSnapshot alone does not restore them — the retry path
// must do it explicitly via savedState.
func testRetryRestoresStylusPages() error {
	savedFallback := GetAllowFallback()
	defer SetAllowFallback(savedFallback)

	localTarget := rawdb.LocalTarget()
	if err := SetTarget(localTarget, "", true); err != nil {
		return fmt.Errorf("failed setting target: %w", err)
	}

	wasm, err := Wat2Wasm(recursiveStackOverflowWat)
	if err != nil {
		return fmt.Errorf("failed compiling WAT: %w", err)
	}

	// Compile cranelift ASM for the retry.
	craneliftAsm, err := compileNative(wasm, 1, true, localTarget, true, time.Minute)
	if err != nil {
		return fmt.Errorf("failed compiling cranelift: %w", err)
	}

	// Use a tiny stack so the first call overflows and the retry is triggered.
	SetInitialNativeStackSize(32 * 1024)
	DrainStackPool()

	gas := uint64(0xfffffffffffffff)
	evm, scope, db := makeTestEVMScope(gas)
	scope.Contract.Gas = gas

	// Pre-populate cranelift ASM in the wasm store.
	moduleHash := common.HexToHash("0x1234567890abcdef")
	craneliftTarget, err := rawdb.CraneliftTarget(localTarget)
	if err != nil {
		return fmt.Errorf("failed getting cranelift target: %w", err)
	}
	wasmStore := db.Database().WasmStore()
	batch := wasmStore.NewBatch()
	rawdb.WriteActivatedAsm(batch, craneliftTarget, moduleHash, craneliftAsm)
	if err := batch.Write(); err != nil {
		return fmt.Errorf("failed to persist cranelift ASM to wasm store: %w", err)
	}

	// Set initial page counts to simulate a prior Stylus call in the same tx.
	initialOpen := uint16(10)
	initialEver := uint16(15)
	db.SetStylusPages(initialOpen, initialEver)

	// Inflate pages to simulate what memory.grow during the first (failed)
	// attempt would have done. The retry must undo this.
	db.AddStylusPages(20)
	inflatedOpen, inflatedEver := db.GetStylusPages()
	if inflatedOpen <= initialOpen || inflatedEver <= initialEver {
		return fmt.Errorf("inflation failed: open=%d ever=%d", inflatedOpen, inflatedEver)
	}

	// Set a non-zero UsedMultiGas to verify it is restored on retry.
	// UsedMultiGas is modified by host I/O callbacks (api.go) and lives on
	// the contract scope, not the StateDB, so RevertToSnapshot does not
	// restore it — restoreState must do it explicitly.
	initialMultiGas := multigas.NewMultiGas(multigas.ResourceKindComputation, 42)
	scope.Contract.UsedMultiGas = initialMultiGas

	stylusParams := &ProgParams{Version: 1, MaxDepth: 1000000, InkPrice: 1, DebugMode: true}
	memModel := NewMemoryModel(0, 0)
	runCtx := core.NewMessageCommitContext([]rawdb.WasmTarget{localTarget})
	allowFallback.Store(true)

	saved := savedState{
		gas:          gas,
		usedMultiGas: initialMultiGas,
		openPages:    initialOpen,
		everPages:    initialEver,
	}

	snapshot := db.Snapshot()
	status, _ := handleNativeStackOverflow(
		common.Address{}, moduleHash,
		scope, evm, nil, []byte{}, &EvmData{}, stylusParams,
		memModel, runCtx, saved, true,
		db, snapshot, nil, nil, Program{version: 1},
	)

	if status == userNativeStackOverflow {
		return fmt.Errorf("retry did not resolve overflow")
	}

	// The recursive_stack_overflow program doesn't call memory.grow, so
	// openWasmPages should equal initialOpen after the retry's revert +
	// re-execution.
	gotOpen, gotEver := db.GetStylusPages()
	if gotOpen != initialOpen {
		return fmt.Errorf("openWasmPages not restored: got %d, want %d", gotOpen, initialOpen)
	}
	if gotEver != initialEver {
		return fmt.Errorf("everWasmPages not restored: got %d, want %d", gotEver, initialEver)
	}

	// The test WAT program does no host I/O, so UsedMultiGas is only modified
	// by restoreState. Verify it was restored to the saved value.
	gotMultiGas := scope.Contract.UsedMultiGas.SingleGas()
	wantMultiGas := initialMultiGas.SingleGas()
	if gotMultiGas != wantMultiGas {
		return fmt.Errorf("usedMultiGas not restored: got %d, want %d", gotMultiGas, wantMultiGas)
	}

	fmt.Printf("testRetryRestoresStylusPages: passed (open=%d, ever=%d, multiGas=%d after retry)\n", gotOpen, gotEver, gotMultiGas)
	return nil
}

// testGetCraneliftAsmErrors verifies getCraneliftAsm returns specific errors
// when the wasm store has no cached ASM and compilation is not possible.
func testGetCraneliftAsmErrors() error {
	localTarget := rawdb.LocalTarget()
	if err := SetTarget(localTarget, "", true); err != nil {
		return fmt.Errorf("failed setting target: %w", err)
	}

	moduleHash := common.HexToHash("0xdeadbeef")
	gas := uint64(0xfffffffffffffff)
	_, _, db := makeTestEVMScope(gas)

	// With no cached ASM and invalid code, getWasmFromContractCode should fail,
	// and getCraneliftAsm should return that error instead of a bare nil.
	// Use non-empty invalid bytes so we don't hit the ProgramNotWasmError()
	// function pointer (which is only initialized by the precompiles package).
	asm, err := getCraneliftAsm(moduleHash, common.Address{}, db, []byte{0xff}, &StylusParams{}, 1, true)
	if err == nil {
		return fmt.Errorf("expected error from getCraneliftAsm with no cached ASM and invalid code, got nil (asm len=%d)", len(asm))
	}
	if len(asm) != 0 {
		return fmt.Errorf("expected nil ASM on error, got %d bytes", len(asm))
	}

	fmt.Printf("testGetCraneliftAsmErrors: passed (err=%v)\n", err)
	return nil
}

// testCraneliftCompilationAndCache tests getCraneliftAsm:
// 1. getCraneliftAsm reads from the wasm store when cranelift ASM is cached.
// 2. The returned cranelift ASM produces valid execution results.
func testCraneliftCompilationAndCache() error {
	localTarget := rawdb.LocalTarget()
	if err := SetTarget(localTarget, "", true); err != nil {
		return fmt.Errorf("failed setting target: %w", err)
	}

	wasm, err := Wat2Wasm(recursiveStackOverflowWat)
	if err != nil {
		return fmt.Errorf("failed compiling WAT: %w", err)
	}

	moduleHash := common.HexToHash("0xdeadbeef")
	gas := uint64(0xfffffffffffffff)
	_, _, db := makeTestEVMScope(gas)

	craneliftTarget, err := rawdb.CraneliftTarget(localTarget)
	if err != nil {
		return fmt.Errorf("failed getting cranelift target: %w", err)
	}

	// Verify wasm store is initially empty for this module.
	wasmStore := db.Database().WasmStore()
	existing := rawdb.ReadActivatedAsm(wasmStore, craneliftTarget, moduleHash)
	if len(existing) > 0 {
		return fmt.Errorf("expected empty wasm store, but found %d bytes", len(existing))
	}

	// Manually compile cranelift ASM and persist it.
	craneliftAsm, err := compileNative(wasm, 1, true, localTarget, true, time.Minute)
	if err != nil {
		return fmt.Errorf("cranelift compilation failed: %w", err)
	}
	if len(craneliftAsm) == 0 {
		return fmt.Errorf("cranelift produced empty ASM")
	}

	// Persist to wasm store.
	batch := wasmStore.NewBatch()
	rawdb.WriteActivatedAsm(batch, craneliftTarget, moduleHash, craneliftAsm)
	if err := batch.Write(); err != nil {
		return fmt.Errorf("failed to persist cranelift ASM: %w", err)
	}

	// Verify getCraneliftAsm reads from the wasm store.
	readAsm, err := getCraneliftAsm(moduleHash, common.Address{}, db, nil, nil, 1, true)
	if err != nil {
		return fmt.Errorf("getCraneliftAsm failed: %w", err)
	}
	if len(readAsm) != len(craneliftAsm) {
		return fmt.Errorf("getCraneliftAsm length mismatch: expected %d, got %d", len(craneliftAsm), len(readAsm))
	}

	// Verify the cranelift ASM actually executes correctly.
	SetNativeStackSize(1024 * 1024)
	DrainStackPool()

	reqHandler := C.NativeRequestHandler{
		handle_request_fptr: (*[0]byte)(C.handleReqWrap),
		id:                  0,
	}
	cgas := u64(0xfffffffffffffff)
	output := &rustBytes{}
	progParams := ProgParams{
		MaxDepth:  1000000,
		InkPrice:  1,
		DebugMode: true,
	}

	status := userStatus(C.stylus_call(
		goSlice(readAsm),
		goSlice([]byte{}),
		progParams.encode(),
		reqHandler,
		(&EvmData{}).encode(),
		cbool(true),
		output,
		(*u64)(&cgas),
		u32(0),
	))
	rustBytesIntoBytes(output)

	if status != userSuccess {
		return fmt.Errorf("expected success from cranelift ASM, got %d", status)
	}

	fmt.Printf("testCraneliftCompilationAndCache: passed (getCraneliftAsm + wasm store verified)\n")
	return nil
}

// testActivateWithCraneliftTarget verifies activateProgramInternal handles
// cranelift targets correctly:
//  1. Cranelift target names are not in the Rust target cache — compileNative
//     must resolve them to base target names before calling the FFI.
//  2. A cranelift target compiles with cranelift=true and the result is stored
//     under the cranelift target key.
//  3. A base target still compiles with singlepass and is stored under the base key.
//  4. Both targets can be activated together, producing distinct ASM.
func testActivateWithCraneliftTarget() error {
	localTarget := rawdb.LocalTarget()
	if err := SetTarget(localTarget, "", true); err != nil {
		return fmt.Errorf("failed setting target: %w", err)
	}

	wasm, err := Wat2Wasm(recursiveStackOverflowWat)
	if err != nil {
		return fmt.Errorf("failed compiling WAT: %w", err)
	}

	craneliftTarget, err := rawdb.CraneliftTarget(localTarget)
	if err != nil {
		return fmt.Errorf("CraneliftTarget: %w", err)
	}

	// Verify invariant: cranelift target names are NOT in the Rust target cache.
	// compileNative with a cranelift target name must fail. This proves that
	// activateProgramInternal must resolve to the base target name before calling FFI.
	_, err = compileNative(wasm, 1, true, craneliftTarget, true, time.Minute)
	if err == nil {
		return fmt.Errorf("compileNative should fail with cranelift target name %v (not in Rust cache), but succeeded", craneliftTarget)
	}

	gas := uint64(0xfffffffffffffff)

	// Test 1: activate with only a cranelift target.
	// moduleActivationMandatory=false so we skip WAVM activation entirely.
	// This implicitly tests that activateProgramInternal resolves the cranelift
	// target name to the base target name before calling compileNative.
	_, asmMap, err := activateProgramInternal(
		common.Address{}, common.Hash{}, wasm, 128, 1, 0, true, &gas,
		[]rawdb.WasmTarget{craneliftTarget},
		false, false,
	)
	if err != nil {
		return fmt.Errorf("activation with cranelift target failed: %w", err)
	}
	if _, ok := asmMap[craneliftTarget]; !ok {
		return fmt.Errorf("expected ASM under cranelift target %v, got keys: %v", craneliftTarget, asmMapKeys(asmMap))
	}

	// Test 2: activate with both base and cranelift targets.
	gas = uint64(0xfffffffffffffff)
	_, asmMap, err = activateProgramInternal(
		common.Address{}, common.Hash{}, wasm, 128, 1, 0, true, &gas,
		[]rawdb.WasmTarget{localTarget, craneliftTarget},
		false, false,
	)
	if err != nil {
		return fmt.Errorf("activation with both targets failed: %w", err)
	}
	if _, ok := asmMap[localTarget]; !ok {
		return fmt.Errorf("expected ASM under base target %v, got keys: %v", localTarget, asmMapKeys(asmMap))
	}
	if _, ok := asmMap[craneliftTarget]; !ok {
		return fmt.Errorf("expected ASM under cranelift target %v, got keys: %v", craneliftTarget, asmMapKeys(asmMap))
	}

	// Verify singlepass and cranelift produce different ASM.
	baseAsm := asmMap[localTarget]
	clAsm := asmMap[craneliftTarget]
	if len(baseAsm) == 0 || len(clAsm) == 0 {
		return fmt.Errorf("ASM should not be empty: base=%d bytes, cranelift=%d bytes", len(baseAsm), len(clAsm))
	}
	sameContent := len(baseAsm) == len(clAsm)
	if sameContent {
		for i := range baseAsm {
			if baseAsm[i] != clAsm[i] {
				sameContent = false
				break
			}
		}
	}
	if sameContent {
		return fmt.Errorf("singlepass and cranelift ASM should differ, but both are %d bytes with identical content", len(baseAsm))
	}

	fmt.Printf("testActivateWithCraneliftTarget: passed (base=%d bytes, cranelift=%d bytes)\n", len(baseAsm), len(clAsm))
	return nil
}

func asmMapKeys(m map[rawdb.WasmTarget][]byte) []rawdb.WasmTarget {
	keys := make([]rawdb.WasmTarget, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// testCraneliftFallbackTargetKeyMismatch reproduces a bug where
// activateProgramInternal stores cranelift-fallback ASM under the cranelift
// target key (e.g., TargetArm64Cranelift) instead of the base target key
// (TargetArm64). This breaks ActivatedAsmMap's consistency check when the
// program is called in the same block it was activated.
func testCraneliftFallbackTargetKeyMismatch() error {
	_, _, db := makeTestEVMScope(1_000_000)

	localTarget := rawdb.LocalTarget()
	craneliftTarget, err := rawdb.CraneliftTarget(localTarget)
	if err != nil {
		return fmt.Errorf("CraneliftTarget: %w", err)
	}

	module := common.HexToHash("0x01")

	// Simulate what activateProgramInternal produces when singlepass fails
	// and cranelift fallback fires: ASM stored under cranelift key only.
	fallbackAsmMap := map[rawdb.WasmTarget][]byte{
		craneliftTarget:  []byte("cranelift-asm"),
		rawdb.TargetWavm: []byte("wavm-asm"),
	}
	if err := db.ActivateWasm(module, fallbackAsmMap); err != nil {
		return fmt.Errorf("ActivateWasm: %w", err)
	}

	// getCompiledProgram requests base targets (from node config WasmTargets()).
	targets := []rawdb.WasmTarget{localTarget, rawdb.TargetWavm}
	_, _, err = db.ActivatedAsmMap(targets, module)
	if err != nil {
		return fmt.Errorf(
			"ActivatedAsmMap should accept cranelift key as substitute for base target %v, but got: %w",
			localTarget, err)
	}

	fmt.Printf("testCraneliftFallbackTargetKeyMismatch: passed\n")
	return nil
}

// testCraneliftFallbackActivateWasmConsistency verifies that ActivateWasm
// accepts a mix of normal and cranelift-fallback activations in the same
// StateDB cycle. When singlepass fails for one program but succeeds for
// another in the same block, the target key sets differ (e.g.,
// {TargetArm64, TargetWavm} vs {TargetArm64Cranelift, TargetWavm}).
// ActivateWasm's consistency check must treat cranelift keys as equivalent
// to their base counterparts.
func testCraneliftFallbackActivateWasmConsistency() error {
	_, _, db := makeTestEVMScope(1_000_000)

	localTarget := rawdb.LocalTarget()
	craneliftTarget, err := rawdb.CraneliftTarget(localTarget)
	if err != nil {
		return fmt.Errorf("CraneliftTarget: %w", err)
	}

	module1 := common.HexToHash("0x01")
	module2 := common.HexToHash("0x02")

	// Normal activation (singlepass succeeded).
	normal := map[rawdb.WasmTarget][]byte{
		localTarget:      []byte("sp-asm"),
		rawdb.TargetWavm: []byte("wavm"),
	}
	if err := db.ActivateWasm(module1, normal); err != nil {
		return fmt.Errorf("ActivateWasm(module1): %w", err)
	}

	// Fallback activation in same block (singlepass failed, cranelift succeeded).
	fallback := map[rawdb.WasmTarget][]byte{
		craneliftTarget:  []byte("cl-asm"),
		rawdb.TargetWavm: []byte("wavm-2"),
	}
	if err := db.ActivateWasm(module2, fallback); err != nil {
		return fmt.Errorf(
			"ActivateWasm should accept cranelift fallback alongside normal activation, but got: %w", err)
	}

	fmt.Printf("testCraneliftFallbackActivateWasmConsistency: passed\n")

	// Reverse direction: fallback first, then normal.
	// When previouslyActivated has cranelift keys and new asmMap has base keys,
	// the consistency check must also pass.
	_, _, db2 := makeTestEVMScope(1_000_000)

	module3 := common.HexToHash("0x03")
	module4 := common.HexToHash("0x04")

	fallback2 := map[rawdb.WasmTarget][]byte{
		craneliftTarget:  []byte("cl-asm-3"),
		rawdb.TargetWavm: []byte("wavm-3"),
	}
	if err := db2.ActivateWasm(module3, fallback2); err != nil {
		return fmt.Errorf("ActivateWasm(module3, fallback first): %w", err)
	}

	normal2 := map[rawdb.WasmTarget][]byte{
		localTarget:      []byte("sp-asm-4"),
		rawdb.TargetWavm: []byte("wavm-4"),
	}
	if err := db2.ActivateWasm(module4, normal2); err != nil {
		return fmt.Errorf(
			"ActivateWasm should accept normal activation after cranelift fallback, but got: %w", err)
	}

	fmt.Printf("testCraneliftFallbackActivateWasmConsistency (reverse): passed\n")
	return nil
}
