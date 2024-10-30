// Copyright 2021-2024, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package server_arb

import (
	"path"
	"reflect"
	"runtime"
	"sort"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

func TestEntriesAreDeletedFromPreimageResolversGlobalMap(t *testing.T) {
	resolver := func(arbutil.PreimageType, common.Hash) ([]byte, error) {
		return nil, nil
	}

	sortedKeys := func() []int64 {
		keys := preimageResolvers.Keys()
		sort.Slice(keys, func(i, j int) bool {
			return keys[i] < keys[j]
		})
		return keys
	}

	// clear global map before running test
	preimageKeys := sortedKeys()
	for _, key := range preimageKeys {
		preimageResolvers.Delete(key)
	}

	_, filename, _, _ := runtime.Caller(0)
	wasmDir := path.Join(path.Dir(filename), "../../arbitrator/prover/test-cases/")
	wasmPath := path.Join(wasmDir, "global-state.wasm")
	modulePaths := []string{path.Join(wasmDir, "global-state-wrapper.wasm")}

	machine1, err := LoadSimpleMachine(wasmPath, modulePaths, true)
	testhelpers.RequireImpl(t, err)
	err = machine1.SetPreimageResolver(resolver)
	testhelpers.RequireImpl(t, err)

	machine2, err := LoadSimpleMachine(wasmPath, modulePaths, true)
	testhelpers.RequireImpl(t, err)
	err = machine2.SetPreimageResolver(resolver)
	testhelpers.RequireImpl(t, err)

	machine1_clone1 := machine1.Clone()
	machine1_clone2 := machine1.Clone()

	checkKeys := func(expectedKeys []int64, scenario string) {
		keys := sortedKeys()
		if !reflect.DeepEqual(keys, expectedKeys) {
			t.Fatal("Unexpected preimageResolversKeys got", keys, "expected", expectedKeys, "scenario", scenario)
		}
	}

	checkKeys([]int64{machine1.contextId, machine2.contextId}, "initial")

	machine1_clone1.Destroy()
	checkKeys([]int64{machine1.contextId, machine2.contextId}, "after machine1_clone1 destroy")

	machine1_clone2.Destroy()
	checkKeys([]int64{machine1.contextId, machine2.contextId}, "after machine1_clone2 destroy")

	machine2.Destroy()
	checkKeys([]int64{machine1.contextId}, "after machine2 destroy")

	machine1.Destroy()
	checkKeys([]int64{}, "after machine1 destroy")
}
