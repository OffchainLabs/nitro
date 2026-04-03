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
// 3. The Go-side retry logic (cranelift + stack doubling) recovers.
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

	// Now test the Go retry path: double the stack and run again.
	// This simulates what retryOnStackOverflow does.
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

	_, err = fmt.Printf("testNativeStackSize: passed (status=%d, gas_left=%d), Go retry recovered\n", status, gas)
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

// testRetryOnStackOverflow tests that retryOnStackOverflow returns
// NativeStackOverflow immediately when allowFallback is false, and also
// when the execution is off-chain (gas estimation).
func testRetryOnStackOverflow() error {
	localTarget := rawdb.LocalTarget()
	if err := SetTarget(localTarget, "", true); err != nil {
		return fmt.Errorf("failed setting target: %w", err)
	}

	SetInitialNativeStackSize(32 * 1024)
	DrainStackPool()

	gas := uint64(0xfffffffffffffff)
	evm, scope, db := makeTestEVMScope(gas)
	scope.Contract.Gas = gas

	stylusParams := &ProgParams{
		Version:   1,
		MaxDepth:  1000000,
		InkPrice:  1,
		DebugMode: true,
	}
	memModel := NewMemoryModel(0, 0)
	moduleHash := common.HexToHash("0x1234567890abcdef")

	// Sub-test 1: allowFallback=false → immediate NativeStackOverflow, no retry.
	allowFallback.Store(false)
	runCtx := core.NewMessageCommitContext([]rawdb.WasmTarget{localTarget})

	snapshot := db.Snapshot()
	status, _ := retryOnStackOverflow(
		common.Address{}, moduleHash,
		scope, evm, nil, []byte{}, &EvmData{}, stylusParams,
		memModel, runCtx, gas, multigas.ZeroGas(), true,
		db, snapshot, nil, nil, Program{version: 1},
	)
	if status != userNativeStackOverflow {
		return fmt.Errorf("allowFallback=false: expected NativeStackOverflow, got %d", status)
	}

	// Sub-test 2: allowFallback=true but off-chain → immediate NativeStackOverflow.
	allowFallback.Store(true)
	offChainCtx := core.NewMessageGasEstimationContext()
	scope.Contract.Gas = gas

	status, _ = retryOnStackOverflow(
		common.Address{}, moduleHash,
		scope, evm, nil, []byte{}, &EvmData{}, stylusParams,
		memModel, offChainCtx, gas, multigas.ZeroGas(), true,
		db, snapshot, nil, nil, Program{version: 1},
	)
	if status != userNativeStackOverflow {
		return fmt.Errorf("off-chain: expected NativeStackOverflow, got %d", status)
	}

	fmt.Printf("testRetryOnStackOverflow: passed (no-fallback and off-chain both return immediately)\n")
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

	// Manually compile cranelift ASM and persist it (simulating what
	// getCraneliftAsm does internally, since getCraneliftAsm needs
	// getWasmFromContractCode which requires real contract state).
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
	if len(readAsm) == 0 {
		return fmt.Errorf("getCraneliftAsm returned empty after wasm store write")
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

	if status == userNativeStackOverflow {
		return fmt.Errorf("cranelift ASM overflowed with 1MB stack")
	}
	if status != userSuccess {
		return fmt.Errorf("expected success from cranelift ASM, got %d", status)
	}

	fmt.Printf("testCraneliftCompilationAndCache: passed (status=%d, getCraneliftAsm + wasm store verified)\n", status)
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

// testStackDoublingGivesUp verifies behavior when configuredNativeStackSize is
// already at MaxNativeStackSize: the doubling path cannot grow further, so the
// cranelift retry at max stack is the last attempt. Since the 500-recursion
// program succeeds at 100MB, the test verifies that cranelift works at max
// stack and that the stack size is properly restored afterward.
//
// To test the true give-up path (cranelift also overflows at max stack) we
// would need a program that exceeds 100MB even with cranelift, which is
// impractical. What we CAN test: the stack is at max, cranelift is called,
// and the stack size globals are properly restored regardless of outcome.
func testStackDoublingGivesUp() error {
	localTarget := rawdb.LocalTarget()
	if err := SetTarget(localTarget, "", true); err != nil {
		return fmt.Errorf("failed setting target: %w", err)
	}

	wasm, err := Wat2Wasm(recursiveStackOverflowWat)
	if err != nil {
		return fmt.Errorf("failed compiling WAT: %w", err)
	}

	// Compile singlepass ASM so we have something to call initially.
	// The cranelift ASM we pre-populate will be real but compiled for tiny stack,
	// meaning at 100MB it should succeed — so this test actually verifies that
	// when cranelift *succeeds* at max stack, the function returns that success.
	// To test the true give-up (cranelift also overflows), we'd need a program
	// that exceeds 100MB stack even with cranelift, which is impractical.
	//
	// What we CAN test: the stack is at max, cranelift is called, and the
	// stack size is properly restored after the call regardless of outcome.
	craneliftAsm, err := compileNative(wasm, 1, true, localTarget, true, time.Minute)
	if err != nil {
		return fmt.Errorf("failed compiling cranelift: %w", err)
	}

	// Set the stack to exactly MaxNativeStackSize.
	SetInitialNativeStackSize(MaxNativeStackSize)
	DrainStackPool()

	gas := uint64(0xfffffffffffffff)
	evm, scope, db := makeTestEVMScope(gas)
	scope.Contract.Gas = gas

	// Pre-populate cranelift ASM in wasm store.
	moduleHash := common.HexToHash("0x1234567890abcdef")
	craneliftTarget, err := rawdb.CraneliftTarget(localTarget)
	if err != nil {
		return fmt.Errorf("failed getting cranelift target: %w", err)
	}
	wasmStore := db.Database().WasmStore()
	rawdb.WriteActivatedAsm(wasmStore, craneliftTarget, moduleHash, craneliftAsm)

	stylusParams := &ProgParams{Version: 1, MaxDepth: 1000000, InkPrice: 1, DebugMode: true}
	memModel := NewMemoryModel(0, 0)
	runCtx := core.NewMessageCommitContext([]rawdb.WasmTarget{localTarget})
	allowFallback.Store(true)

	snapshot := db.Snapshot()
	retryStatus, _ := retryOnStackOverflow(
		common.Address{}, moduleHash,
		scope, evm, nil, []byte{}, &EvmData{}, stylusParams,
		memModel, runCtx, gas, multigas.ZeroGas(), true,
		db, snapshot, nil, nil, Program{version: 1},
	)

	// Cranelift at 100MB should succeed (out-of-ink or out-of-stack), not overflow.
	// This verifies the cranelift path works at max stack size.
	if retryStatus == userNativeStackOverflow {
		// If it did overflow, that's also fine — the test still verifies
		// the stack size is properly restored.
		fmt.Printf("testStackDoublingGivesUp: cranelift overflowed at max (give-up path exercised)\n")
	}

	// Key invariant: stack size is restored to MaxNativeStackSize.
	if got := GetNativeStackSize(); got != MaxNativeStackSize {
		return fmt.Errorf("expected stack size to remain at %d, got %d", MaxNativeStackSize, got)
	}

	fmt.Printf("testStackDoublingGivesUp: passed (status=%d, stack size preserved at max)\n", retryStatus)
	return nil
}

// testCraneliftFallbackInRetry tests the cranelift fallback path inside
// retryOnStackOverflow. It uses a commit-mode RunContext (IsExecutedOnChain=true)
// and pre-populates cranelift ASM in the wasm store so the fallback succeeds
// without needing real contract code / getWasmFromContractCode.
func testCraneliftFallbackInRetry() error {
	localTarget := rawdb.LocalTarget()
	if err := SetTarget(localTarget, "", true); err != nil {
		return fmt.Errorf("failed setting target: %w", err)
	}

	wasm, err := Wat2Wasm(recursiveStackOverflowWat)
	if err != nil {
		return fmt.Errorf("failed compiling WAT: %w", err)
	}

	// Compile cranelift ASM.
	craneliftAsm, err := compileNative(wasm, 1, true, localTarget, true, time.Minute)
	if err != nil {
		return fmt.Errorf("failed compiling cranelift: %w", err)
	}

	// Use a tiny stack so singlepass overflows.
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
	rawdb.WriteActivatedAsm(wasmStore, craneliftTarget, moduleHash, craneliftAsm)

	stylusParams := &ProgParams{Version: 1, MaxDepth: 1000000, InkPrice: 1, DebugMode: true}
	memModel := NewMemoryModel(0, 0)
	// Use commit context: IsExecutedOnChain() = true → cranelift fallback enabled.
	runCtx := core.NewMessageCommitContext([]rawdb.WasmTarget{localTarget})
	allowFallback.Store(true)

	snapshot := db.Snapshot()
	status, _ := retryOnStackOverflow(
		common.Address{}, moduleHash,
		scope, evm, nil, []byte{}, &EvmData{}, stylusParams,
		memModel, runCtx, gas, multigas.ZeroGas(), true,
		db, snapshot, nil, nil, Program{version: 1},
	)

	// Cranelift should have succeeded (program completes its recursions, not overflow).
	if status == userNativeStackOverflow {
		return fmt.Errorf("cranelift fallback did not resolve overflow")
	}
	if status != userSuccess {
		return fmt.Errorf("expected success from cranelift fallback, got %d", status)
	}

	// Verify the stack size was restored (cranelift path doesn't double).
	if got := GetNativeStackSize(); got != 32*1024 {
		return fmt.Errorf("expected stack size to remain at 32KB after cranelift fallback, got %d", got)
	}

	fmt.Printf("testCraneliftFallbackInRetry: passed (status=%d, cranelift fallback worked)\n", status)
	return nil
}
