package arbos

import (
	"bytes"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
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
	st := NewMemoryBackingEvmStorage()
	if st.Get(common.Hash{}) != (common.Hash{}) {
		t.Fail()
	}

	loc1 := IntToHash(99)
	val1 := IntToHash(1351908)

	st.Set(loc1, val1)
	if st.Get(common.Hash{}) != (common.Hash{}) {
		t.Fail()
	}
	if st.Get(loc1) != val1 {
		t.Fail()
	}
}

func TestStorageSegmentAllocation(t *testing.T) {
	storage := OpenArbosStateForTest()
	size := 37
	seg, err := storage.AllocateSizedSegment(uint64(size))
	if err != nil {
		t.Error(err)
	}
	if seg.size != 37 {
		t.Fail()
	}
	res := seg.Get(19)
	if res != (common.Hash{}) {
		t.Fail()
	}

	val := IntToHash(51985380)
	seg.Set(uint64(size-2), val)
	res = seg.Get(uint64(size - 2))
	if res != val {
		t.Fail()
	}
}

func TestStorageSegmentAllocationBytes(t *testing.T) {
	storage := OpenArbosStateForTest()
	buf := []byte("This is a long string. The quick brown fox jumped over the lazy dog. Cogito ergo sum.")
	seg := storage.AllocateSizedSegmentForBytes(buf)
	if int(seg.size) != 1 + (len(buf)+31) / 32 {
		t.Fail()
	}

	reread := seg.GetBytes()
	if bytes.Compare(buf, reread) != 0 {
		t.Fail()
	}
}