// Copyright 2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

//go:build !wasm
// +build !wasm

package programs

// This file exists because cgo isn't allowed in tests

/*
#cgo CFLAGS: -g -Wall -I../../target/include/
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

	"github.com/wasmerio/wasmer-go/wasmer"
)

func isNativeArm() bool {
	return runtime.GOOS == "linux" && runtime.GOARCH == "arm64"
}

func isNativeX86() bool {
	return runtime.GOOS == "linux" && runtime.GOARCH == "amd64"
}

func testCompileArch(store bool) error {

	nativeArm64 := isNativeArm()
	nativeAmd64 := isNativeX86()

	arm64CompileName := []byte("arm64")
	amd64CompileName := []byte("amd64")

	arm64TargetString := []byte("arm64-linux-unknown+neon")
	amd64TargetString := []byte("x86_64-linux-unknown+sse4.2")

	output := &rustBytes{}

	_, err := fmt.Print("starting test.. native arm? ", nativeArm64, " amd? ", nativeAmd64, " GOARCH/GOOS: ", runtime.GOARCH+"/"+runtime.GOOS, "\n")
	if err != nil {
		return err
	}

	status := C.stylus_target_set(goSlice(arm64CompileName),
		goSlice(arm64TargetString),
		output,
		cbool(nativeArm64))

	if status != 0 {
		return fmt.Errorf("failed setting compilation target arm: %v", string(output.intoBytes()))
	}

	status = C.stylus_target_set(goSlice(amd64CompileName),
		goSlice(amd64TargetString),
		output,
		cbool(nativeAmd64))

	if status != 0 {
		return fmt.Errorf("failed setting compilation target amd: %v", string(output.intoBytes()))
	}

	source, err := os.ReadFile("../../arbitrator/stylus/tests/add.wat")
	if err != nil {
		return fmt.Errorf("failed reading stylus contract: %w", err)
	}

	wasm, err := wasmer.Wat2Wasm(string(source))
	if err != nil {
		return err
	}

	if store {
		_, err := fmt.Print("storing compiled files to ../../target/testdata/\n")
		if err != nil {
			return err
		}
		err = os.MkdirAll("../../target/testdata", 0644)
		if err != nil {
			return err
		}
	}

	status = C.stylus_compile(
		goSlice(wasm),
		u16(1),
		cbool(true),
		goSlice([]byte("booga")),
		output,
	)
	if status == 0 {
		return fmt.Errorf("succeeded compiling non-existent arch: %v", string(output.intoBytes()))
	}

	status = C.stylus_compile(
		goSlice(wasm),
		u16(1),
		cbool(true),
		goSlice([]byte{}),
		output,
	)
	if status != 0 {
		return fmt.Errorf("failed compiling native: %v", string(output.intoBytes()))
	}
	if store && !nativeAmd64 && !nativeArm64 {
		_, err := fmt.Printf("writing host file\n")
		if err != nil {
			return err
		}

		err = os.WriteFile("../../target/testdata/host.bin", output.intoBytes(), 0644)
		if err != nil {
			return err
		}
	}

	status = C.stylus_compile(
		goSlice(wasm),
		u16(1),
		cbool(true),
		goSlice(arm64CompileName),
		output,
	)
	if status != 0 {
		return fmt.Errorf("failed compiling arm: %v", string(output.intoBytes()))
	}
	if store {
		_, err := fmt.Printf("writing arm file\n")
		if err != nil {
			return err
		}

		err = os.WriteFile("../../target/testdata/arm64.bin", output.intoBytes(), 0644)
		if err != nil {
			return err
		}
	}

	status = C.stylus_compile(
		goSlice(wasm),
		u16(1),
		cbool(true),
		goSlice(amd64CompileName),
		output,
	)
	if status != 0 {
		return fmt.Errorf("failed compiling amd: %v", string(output.intoBytes()))
	}
	if store {
		_, err := fmt.Printf("writing amd64 file\n")
		if err != nil {
			return err
		}

		err = os.WriteFile("../../target/testdata/amd64.bin", output.intoBytes(), 0644)
		if err != nil {
			return err
		}
	}

	return nil
}

func testCompileLoad() error {
	filePath := "../../target/testdata/host.bin"
	if isNativeArm() {
		filePath = "../../target/testdata/arm64.bin"
	}
	if isNativeX86() {
		filePath = "../../target/testdata/amd64.bin"
	}

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

	_, msg, err := status.toResult(output.intoBytes(), true)
	if status == userFailure {
		err = fmt.Errorf("%w: %v", err, msg)
	}

	return err
}
