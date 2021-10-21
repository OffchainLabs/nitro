//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbos

import (
	"bytes"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/offchainlabs/arbstate/arbos/storage"
	"github.com/offchainlabs/arbstate/arbos/util"
	"math/big"
	"testing"
)

// Create a memory-backed ArbOS state
func OpenArbosStateForTest() *ArbosState {
	raw := rawdb.NewMemoryDatabase()
	db := state.NewDatabase(raw)
	statedb, err := state.New(common.Hash{}, db, nil)
	if err != nil {
		panic("failed to init empty statedb")
	}
	return OpenArbosState(statedb)
}

func TestStorageOpenFromEmpty(t *testing.T) {
	storage := OpenArbosStateForTest()
	_ = storage
}

func TestMemoryBackingEvmStorage(t *testing.T) {
	st := storage.NewMemoryBacked()
	if st.Get(common.Hash{}) != (common.Hash{}) {
		t.Fail()
	}

	loc1 := util.IntToHash(99)
	val1 := util.IntToHash(1351908)

	st.Set(loc1, val1)
	if st.Get(common.Hash{}) != (common.Hash{}) {
		t.Fail()
	}
	if st.Get(loc1) != val1 {
		t.Fail()
	}
}

func TestStorageSegmentAllocation(t *testing.T) {
	state := OpenArbosStateForTest()
	seg := state.backingStorage.Open(state.backingStorage.UniqueKey().Bytes())
	res := seg.GetByInt64(19)
	if res != (common.Hash{}) {
		t.Fail()
	}
}

func TestStorageSegmentAllocationBytes(t *testing.T) {
	state := OpenArbosStateForTest()
	buf := []byte("This is a long string. The quick brown fox jumped over the lazy dog. Cogito ergo sum.")
	seg, _ := state.backingStorage.AllocateForBytes(buf)
	reread := seg.GetBytes()
	if bytes.Compare(buf, reread) != 0 {
		t.Fail()
	}
}

func TestStorageBackedInt64(t *testing.T) {
	state := OpenArbosStateForTest()
	storage := state.backingStorage
	offset := common.BigToHash(big.NewInt(7895463))

	valuesToTry := []int64{0, 7, -7, 56487423567, -7586427647}

	for _, val := range valuesToTry {
		storage.OpenStorageBackedInt64(offset).Set(val)
		res := storage.OpenStorageBackedInt64(offset).Get()
		if val != res {
			t.Fatal(val, res)
		}
	}
}
