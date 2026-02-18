// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

//go:build !wasm

package programs

import (
	"bytes"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/offchainlabs/nitro/arbcompress"
)

// testModule holds the artifacts needed to call getCompiledProgram for a
// single wasm program.
type testModule struct {
	code       []byte
	codehash   common.Hash
	moduleHash common.Hash
	asmMap     map[rawdb.WasmTarget][]byte
}

func prepareTestModule(t *testing.T, watFile string, targets []rawdb.WasmTarget) testModule {
	t.Helper()
	source, err := os.ReadFile(watFile)
	if err != nil {
		t.Fatal(err)
	}
	wasm, err := Wat2Wasm(source)
	if err != nil {
		t.Fatal(err)
	}
	compressed, err := arbcompress.Compress(wasm, 1, arbcompress.EmptyDictionary)
	if err != nil {
		t.Fatal(err)
	}
	code := append(append([]byte{}, state.StylusDiscriminant...), 0) // dict byte 0 = EmptyDictionary
	code = append(code, compressed...)
	codehash := crypto.Keccak256Hash(code)

	zeroGas := uint64(0)
	info, asmMap, err := activateProgramInternal(
		common.Address{}, codehash, wasm,
		initialPageLimit, 1, 0, true, &zeroGas, targets, true,
	)
	if err != nil {
		t.Fatal(err)
	}
	return testModule{code: code, codehash: codehash, moduleHash: info.moduleHash, asmMap: asmMap}
}

// TestGetCompiledProgram verifies that getCompiledProgram recompiles only
// missing targets and persists the results to the wasm store.
func TestGetCompiledProgram(t *testing.T) {
	localTarget := rawdb.LocalTarget()
	if err := SetTarget(localTarget, "", true); err != nil {
		t.Fatal(err)
	}
	crossTarget, crossDesc := rawdb.TargetArm64, DefaultTargetDescriptionArm
	if localTarget == rawdb.TargetArm64 {
		crossTarget, crossDesc = rawdb.TargetAmd64, DefaultTargetDescriptionX86
	}
	if err := SetTarget(crossTarget, crossDesc, false); err != nil {
		t.Fatal(err)
	}
	allTargets := []rawdb.WasmTarget{localTarget, crossTarget}

	mod := prepareTestModule(t, "../../crates/stylus/tests/add.wat", allTargets)

	// runWith creates a fresh statedb, pre-populates the wasm store with
	// prePopulated, calls getCompiledProgram requesting requestTargets, and
	// verifies every requested target is present in the result.
	runWith := func(t *testing.T, m testModule, prePopulated map[rawdb.WasmTarget][]byte, requestTargets []rawdb.WasmTarget) map[rawdb.WasmTarget][]byte {
		t.Helper()
		db := state.NewDatabaseForTesting()
		statedb, err := state.New(types.EmptyRootHash, db)
		if err != nil {
			t.Fatal(err)
		}
		if len(prePopulated) > 0 {
			batch := db.WasmStore().NewBatch()
			rawdb.WriteActivation(batch, m.moduleHash, prePopulated)
			if err := batch.Write(); err != nil {
				t.Fatal(err)
			}
		}
		result, err := getCompiledProgram(
			statedb, m.moduleHash, common.Address{}, m.code, m.codehash,
			&StylusParams{PageLimit: initialPageLimit, MaxWasmSize: 128 * 1024, Version: 1},
			uint64(ArbitrumStartTime+3600*1000), true,
			Program{version: 1, activatedAt: 1},
			core.NewMessageCommitContext(requestTargets),
		)
		if err != nil {
			t.Fatal(err)
		}
		for _, target := range requestTargets {
			if len(result[target]) == 0 {
				t.Fatalf("missing asm for target %s", target)
			}
		}
		return result
	}

	run := func(t *testing.T, prePopulated map[rawdb.WasmTarget][]byte) map[rawdb.WasmTarget][]byte {
		t.Helper()
		return runWith(t, mod, prePopulated, allTargets)
	}

	t.Run("all targets present", func(t *testing.T) {
		run(t, mod.asmMap)
	})

	t.Run("some targets missing", func(t *testing.T) {
		result := run(t, map[rawdb.WasmTarget][]byte{localTarget: mod.asmMap[localTarget]})
		if !bytes.Equal(result[localTarget], mod.asmMap[localTarget]) {
			t.Error("pre-existing target asm was recompiled instead of reused")
		}
	})

	t.Run("all targets missing", func(t *testing.T) {
		run(t, nil)
	})

	t.Run("fewer targets requested than stored", func(t *testing.T) {
		result := runWith(t, mod, mod.asmMap, []rawdb.WasmTarget{localTarget})
		if !bytes.Equal(result[localTarget], mod.asmMap[localTarget]) {
			t.Error("asm for local target changed when requesting fewer targets")
		}
		if _, has := result[crossTarget]; has {
			t.Error("unrequested cross target was returned")
		}
	})

	t.Run("multiple modules", func(t *testing.T) {
		mod2 := prepareTestModule(t, "../../crates/stylus/tests/memory.wat", allTargets)
		if mod.moduleHash == mod2.moduleHash {
			t.Fatal("test requires two distinct modules")
		}
		// Pre-populate only mod1; mod2 must be recompiled from scratch.
		db := state.NewDatabaseForTesting()
		statedb, err := state.New(types.EmptyRootHash, db)
		if err != nil {
			t.Fatal(err)
		}
		batch := db.WasmStore().NewBatch()
		rawdb.WriteActivation(batch, mod.moduleHash, mod.asmMap)
		if err := batch.Write(); err != nil {
			t.Fatal(err)
		}

		ctx := core.NewMessageCommitContext(allTargets)
		params := &StylusParams{PageLimit: initialPageLimit, MaxWasmSize: 128 * 1024, Version: 1}
		prog := Program{version: 1, activatedAt: 1}
		ts := uint64(ArbitrumStartTime + 3600*1000)

		result1, err := getCompiledProgram(statedb, mod.moduleHash, common.Address{}, mod.code, mod.codehash, params, ts, true, prog, ctx)
		if err != nil {
			t.Fatal(err)
		}
		result2, err := getCompiledProgram(statedb, mod2.moduleHash, common.Address{}, mod2.code, mod2.codehash, params, ts, true, prog, ctx)
		if err != nil {
			t.Fatal(err)
		}

		// Each module returns its own asm.
		for _, target := range allTargets {
			if len(result1[target]) == 0 {
				t.Fatalf("mod1: missing asm for target %s", target)
			}
			if len(result2[target]) == 0 {
				t.Fatalf("mod2: missing asm for target %s", target)
			}
			if bytes.Equal(result1[target], result2[target]) {
				t.Fatalf("mod1 and mod2 produced identical asm for target %s", target)
			}
		}
	})
}
