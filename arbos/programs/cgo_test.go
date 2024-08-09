// Copyright 2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

//go:build !wasm
// +build !wasm

package programs

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestConstants(t *testing.T) {
	err := testConstants()
	if err != nil {
		t.Fatal(err)
	}
}

// normal test will not write anything to disk
// to test cross-compilation:
// * run test with TEST_COMPILE=STORE on one machine
// * copy target/testdata to the other machine
// * run test with TEST_COMPILE=LOAD on the other machine
func TestCompileArch(t *testing.T) {
	compile_env := os.Getenv("TEST_COMPILE")
	if compile_env == "" {
		fmt.Print("use TEST_COMPILE=[STORE|LOAD] to allow store/load in compile test")
	}
	store := strings.Contains(compile_env, "STORE")
	err := testCompileArch(store)
	if err != nil {
		t.Fatal(err)
	}
	if store || strings.Contains(compile_env, "LOAD") {
		err = testCompileLoad()
		if err != nil {
			t.Fatal(err)
		}
	}
}
