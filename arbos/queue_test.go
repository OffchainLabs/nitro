// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbos

import (
	"testing"

	"github.com/offchainlabs/nitro/arbos/arbosState"

	"github.com/offchainlabs/nitro/arbos/storage"
	"github.com/offchainlabs/nitro/arbos/util"
)

func TestQueue(t *testing.T) {
	state, statedb := arbosState.NewArbosMemoryBackedArbOSState()
	sto := state.BackingStorage().OpenCachedSubStorage([]byte{})
	Require(t, storage.InitializeQueue(sto))
	q := storage.OpenQueue(sto)

	stateBefore := statedb.IntermediateRoot(false)

	empty := func() bool {
		empty, err := q.IsEmpty()
		Require(t, err)
		return empty
	}

	if !empty() {
		Fail(t)
	}

	val0 := uint64(853139508)
	for i := uint64(0); i < 150; i++ {
		val := util.UintToHash(val0 + i)
		Require(t, q.Put(val))
		if empty() {
			Fail(t)
		}
	}

	for i := uint64(0); i < 150; i++ {
		val := util.UintToHash(val0 + i)
		res, err := q.Get()
		Require(t, err)
		if res.Big().Cmp(val.Big()) != 0 {
			Fail(t)
		}
	}

	if !empty() {
		Fail(t)
	}
	cleared, err := q.Shift()
	Require(t, err)
	if !cleared || stateBefore != statedb.IntermediateRoot(false) {
		Fail(t, "Emptying & shifting didn't clear the state")
	}
}
