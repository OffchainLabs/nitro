// Copyright 2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

//go:build !wasm

package programs

import (
	"fmt"
	"strings"
	"testing"

	"github.com/offchainlabs/nitro/util/testhelpers/flag"
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
