//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbos

import (
	"bytes"
	"testing"

	"github.com/offchainlabs/arbstate/arbos/arbosState"
	"github.com/offchainlabs/arbstate/util/colors"

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

func TestStorageSpots(t *testing.T) {
	state := arbosState.OpenArbosStateForTesting(t)
	sto := state.BackingStorage().OpenSubStorage([]byte{})

	a := sto.GetStorageSpot(util.IntToHash(0))
	b := sto.GetStorageSpot(util.IntToHash(1))
	c := sto.GetStorageSpot(util.IntToHash(255))
	d := sto.GetStorageSpot(util.IntToHash(256)) // should be in its own page

	if !bytes.Equal(a[:31], b[:31]) {
		Fail(t, "upper bytes are unequal", a.String(), b.String())
	}
	if !bytes.Equal(a[:31], c[:31]) {
		Fail(t, "upper bytes are unequal", a.String(), c.String())
	}
	if bytes.Equal(a[:31], d[:31]) {
		Fail(t, "upper bytes should be different", a.String(), d.String())
	}

	if a[31] != 0 || b[31] != 1 || c[31] != 255 || d[31] != 0 {
		println("offset 0\t", colors.Red, a.String(), colors.Clear)
		println("offset 1\t", colors.Red, b.String(), colors.Clear)
		println("offset 255\t", colors.Red, c.String(), colors.Clear)
		println("offset 256\t", colors.Red, d.String(), colors.Clear)
		Fail(t, "page offset mismatch")
	}
}
