// Copyright 2024-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

//go:build !wasm

package programs

import (
	"fmt"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/core/rawdb"

	testflag "github.com/offchainlabs/nitro/util/testhelpers/flag"
)

func TestConstants(t *testing.T) {
	err := testConstants()
	if err != nil {
		t.Fatal(err)
	}
}

// normal test will not write anything to disk
// to test cross-compilation:
// * run test with -test_compile=STORE on one machine
// * copy target/testdata to the other machine
// * run test with -test_compile=LOAD on the other machine
func TestCompileArch(t *testing.T) {
	if *testflag.CompileFlag == "" {
		fmt.Print("use -test_compile=[STORE|LOAD] to allow store/load in compile test")
	}
	store := strings.Contains(*testflag.CompileFlag, "STORE")
	err := testCompileArch(store, false)
	if err != nil {
		t.Fatal(err)
	}
	err = testCompileArch(store, true)
	if err != nil {
		t.Fatal(err)
	}
	if store || strings.Contains(*testflag.CompileFlag, "LOAD") {
		err = testCompileLoad()
		if err != nil {
			t.Fatal(err)
		}
		err = resetNativeTarget()
		if err != nil {
			t.Fatal(err)
		}
		err = testCompileLoad()
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestNativeStackSize(t *testing.T) {
	defer SetNativeStackSize(1024 * 1024) // restore default even on panic
	err := testNativeStackSize()
	if err != nil {
		t.Fatal(err)
	}
}

func TestNativeStackSizeMaxCap(t *testing.T) {
	defer SetNativeStackSize(1024 * 1024) // restore default even on panic
	err := testNativeStackSizeMaxCap()
	if err != nil {
		t.Fatal(err)
	}
}

func TestRetryOnStackOverflow(t *testing.T) {
	defer SetNativeStackSize(1024 * 1024)
	err := testRetryOnStackOverflow()
	if err != nil {
		t.Fatal(err)
	}
}

func TestCraneliftCompilationAndCache(t *testing.T) {
	defer SetNativeStackSize(1024 * 1024)
	err := testCraneliftCompilationAndCache()
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetCraneliftAsmErrors(t *testing.T) {
	err := testGetCraneliftAsmErrors()
	if err != nil {
		t.Fatal(err)
	}
}

func TestStackDoublingGivesUp(t *testing.T) {
	defer SetNativeStackSize(1024 * 1024)
	err := testStackDoublingGivesUp()
	if err != nil {
		t.Fatal(err)
	}
}

func TestCraneliftFallbackInRetry(t *testing.T) {
	defer SetNativeStackSize(1024 * 1024)
	err := testCraneliftFallbackInRetry()
	if err != nil {
		t.Fatal(err)
	}
}

func TestSelectLocalAsm(t *testing.T) {
	localTarget := rawdb.LocalTarget()
	craneliftTarget, err := rawdb.CraneliftTarget(localTarget)
	if err != nil {
		t.Fatal(err)
	}

	singlepassAsm := []byte("singlepass-asm")
	craneliftAsm := []byte("cranelift-asm")

	// Singlepass takes precedence when both exist.
	asmMap := map[rawdb.WasmTarget][]byte{
		localTarget:     singlepassAsm,
		craneliftTarget: craneliftAsm,
	}
	asm, ok := selectLocalAsm(asmMap)
	if !ok || string(asm) != "singlepass-asm" {
		t.Fatalf("expected singlepass precedence, got ok=%v asm=%q", ok, asm)
	}

	// Cranelift-only: returned when singlepass is absent.
	asmMap = map[rawdb.WasmTarget][]byte{
		craneliftTarget: craneliftAsm,
	}
	asm, ok = selectLocalAsm(asmMap)
	if !ok || string(asm) != "cranelift-asm" {
		t.Fatalf("expected cranelift fallback, got ok=%v asm=%q", ok, asm)
	}

	// Neither exists: returns false.
	asmMap = map[rawdb.WasmTarget][]byte{}
	_, ok = selectLocalAsm(asmMap)
	if ok {
		t.Fatal("expected ok=false for empty map")
	}

	// Nil map: returns false.
	_, ok = selectLocalAsm(nil)
	if ok {
		t.Fatal("expected ok=false for nil map")
	}
}
