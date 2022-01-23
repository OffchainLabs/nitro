//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbos

import (
	"testing"

	"github.com/offchainlabs/arbstate/arbos/arbosState"

	"github.com/offchainlabs/arbstate/arbos/storage"
	"github.com/offchainlabs/arbstate/arbos/util"
)

func TestQueue(t *testing.T) {
	state := arbosState.OpenArbosMemoryBackedArbOSState()
	sto := state.BackingStorage().OpenSubStorage([]byte{})
	Require(t, storage.InitializeQueue(sto))
	q := storage.OpenQueue(sto)

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
}
