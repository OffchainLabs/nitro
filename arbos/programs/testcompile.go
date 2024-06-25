// Copyright 2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

//go:build !wasm
// +build !wasm

package programs

// This file exists because cgo isn't allowed in tests

/*
#cgo CFLAGS: -g -Wall -I../../target/include/
#include "arbitrator.h"
*/
import "C"
import (
	"fmt"
	"os"
	"runtime"

	"github.com/wasmerio/wasmer-go/wasmer"
)

func testCompileArch() error {

	nativeArm64 := false
	nativeAmd64 := false

	arm64CompileName := []byte("arm64")
	amd64CompileName := []byte("amd64")

	arm64TargetString := []byte("arm64-linux-unknown+neon")
	amd64TargetString := []byte("x86_64-linux-unknown+sse4.2")

	if runtime.GOOS == "linux" {
		if runtime.GOARCH == "amd64" {
			nativeAmd64 = true
		}
		if runtime.GOARCH == "arm64" {
			nativeArm64 = true
		}
	}

	output := &rustBytes{}

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

	source, err := os.ReadFile("../../arbitrator/stylus/tests/memory.wat")
	if err != nil {
		return fmt.Errorf("failed reading stylus contract: %w", err)
	}

	wasm, err := wasmer.Wat2Wasm(string(source))
	if err != nil {
		return err
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

	return nil
}
