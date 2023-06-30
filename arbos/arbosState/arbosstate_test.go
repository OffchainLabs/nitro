// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/nitro/blob/master/LICENSE

package arbosState

import (
	"bytes"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/arbos/burn"
	"github.com/offchainlabs/nitro/arbos/storage"
	"github.com/offchainlabs/nitro/arbos/util"
	"github.com/offchainlabs/nitro/util/colors"
)

func TestStorageOpenFromEmpty(t *testing.T) {
	NewArbosMemoryBackedArbOSState()
}

func TestMemoryBackingEvmStorage(t *testing.T) {
	sto := storage.NewMemoryBacked(burn.NewSystemBurner(nil, false))
	value, err := sto.Get(common.Hash{})
	Require(t, err)
	if value != (common.Hash{}) {
		Fail(t)
	}

	loc1 := util.UintToHash(99)
	val1 := util.UintToHash(1351908)

	Require(t, sto.Set(loc1, val1))
	value, err = sto.Get(common.Hash{})
	Require(t, err)
	if value != (common.Hash{}) {
		Fail(t)
	}

	value, err = sto.Get(loc1)
	Require(t, err)
	if value != val1 {
		Fail(t)
	}
}

func TestStorageBackedInt64(t *testing.T) {
	state, _ := NewArbosMemoryBackedArbOSState()
	storage := state.backingStorage
	offset := uint64(7895463)

	valuesToTry := []int64{0, 7, -7, 56487423567, -7586427647}

	for _, val := range valuesToTry {
		sbi := storage.OpenStorageBackedInt64(offset)
		Require(t, sbi.Set(val))
		sbi = storage.OpenStorageBackedInt64(offset)
		res, err := sbi.Get()
		Require(t, err)
		if val != res {
			Fail(t, val, res)
		}
	}
}

func TestStorageSlots(t *testing.T) {
	state, _ := NewArbosMemoryBackedArbOSState()
	sto := state.BackingStorage().OpenSubStorage([]byte{})

	println("nil address", colors.Blue, storage.NilAddressRepresentation.String(), colors.Clear)

	a := sto.GetStorageSlot(util.IntToHash(0))
	b := sto.GetStorageSlot(util.IntToHash(1))
	c := sto.GetStorageSlot(util.IntToHash(255))
	d := sto.GetStorageSlot(util.IntToHash(256)) // should be in its own page

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
