//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbos

import (
	"github.com/offchainlabs/arbstate/arbos/storage"
	"github.com/offchainlabs/arbstate/arbos/util"
	"testing"
)

func TestQueue(t *testing.T) {
	state := OpenArbosStateForTest()
	sto := state.backingStorage.OpenSubStorage([]byte{})
	storage.InitializeQueue(sto)
	q := storage.OpenQueue(sto)

	if !q.IsEmpty() {
		t.Fail()
	}

	val0 := int64(853139508)
	for i := 0; i < 150; i++ {
		val := util.IntToHash(val0 + int64(i))
		q.Put(val)
		if q.IsEmpty() {
			t.Fail()
		}
	}

	for i := 0; i < 150; i++ {
		val := util.IntToHash(val0 + int64(i))
		res := q.Get()
		if res.Big().Cmp(val.Big()) != 0 {
			t.Fail()
		}
	}

	if !q.IsEmpty() {
		t.Fail()
	}
}
