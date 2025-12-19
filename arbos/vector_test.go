// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbos

import (
	"testing"

	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/arbos/storage"
	"github.com/offchainlabs/nitro/arbos/util"
)

// This file contains tests for the storage submodule, but it needs to be in the top-level module to
// avoid cross dependencies.

func TestSubStorageVector(t *testing.T) {
	state, statedb := arbosState.NewArbosMemoryBackedArbOSState()
	sto := state.BackingStorage().OpenCachedSubStorage([]byte{})
	vec := storage.OpenSubStorageVector(sto)

	stateBefore := statedb.IntermediateRoot(false)

	getLength := func() uint64 {
		length, err := vec.Length()
		Require(t, err)
		return length
	}

	// Check it starts empty
	if got, want := getLength(), uint64(0); got != want {
		t.Fatalf("wrong vector length: got %v, want %v", got, want)
	}

	// Adds n elements
	const n = uint64(100)
	for i := range n {
		subStorage, err := vec.Push()
		Require(t, err)
		err = subStorage.SetByUint64(0, util.UintToHash(i))
		Require(t, err)
	}

	// Check the length with n elements
	if got, want := getLength(), uint64(100); got != want {
		t.Fatalf("wrong vector length: got %v, want %v", got, want)
	}

	// Check each element
	for i := range n {
		subStorage := vec.At(i)
		got, err := subStorage.GetByUint64(0)
		Require(t, err)
		want := util.UintToHash(i)
		if got != want {
			t.Errorf("wrong sub-storage: got %v, want %v", got, want)
		}
	}

	// Pop each element and clear storage
	for i := range n {
		subStorage, err := vec.Pop()
		Require(t, err)
		got, err := subStorage.GetByUint64(0)
		Require(t, err)
		want := util.UintToHash(n - i - 1)
		if got != want {
			t.Errorf("wrong sub-storage: got %v, want %v", got, want)
		}
		err = subStorage.ClearByUint64(0)
		Require(t, err)
	}

	// Check it ends empty
	if got, want := getLength(), uint64(0); got != want {
		t.Fatalf("wrong vector length: got %v, want %v", got, want)
	}
	if stateBefore != statedb.IntermediateRoot(false) {
		Fail(t, "state is not clear")
	}
}
