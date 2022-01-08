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
	state := arbosState.OpenArbosStateForTesting(t)
	sto := state.BackingStorage().OpenSubStorage([]byte{})
	storage.InitializeQueue(sto)
	q := storage.OpenQueue(sto)

	if !q.IsEmpty() {
		t.Fail()
	}

	val0 := uint64(853139508)
	for i := uint64(0); i < 150; i++ {
		val := util.UintToHash(val0 + i)
		q.Put(val)
		if q.IsEmpty() {
			t.Fail()
		}
	}

	for i := uint64(0); i < 150; i++ {
		val := util.UintToHash(val0 + i)
		res := q.Get()
		if res.Big().Cmp(val.Big()) != 0 {
			t.Fail()
		}
	}

	if !q.IsEmpty() {
		t.Fail()
	}
}
