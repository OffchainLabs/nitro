// Copyright 2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

//go:build !wasm

package programs

// This file exists because cgo isn't allowed in tests

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

	"github.com/ethereum/go-ethereum/core/rawdb"
)

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

	source, err := os.ReadFile("../../arbitrator/stylus/tests/add.wat")
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
		MaxDepth:  10000,
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
