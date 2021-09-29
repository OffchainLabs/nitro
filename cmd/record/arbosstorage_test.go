package main

import (
	"bytes"
	"github.com/ethereum/go-ethereum/common"
	"testing"
)

func TestStorageOpenFromEmpty(t *testing.T) {
	storage := OpenArbosStorage(NewMemoryBackingEvmStorage())
	_ = storage
}

func TestMemoryBackingEvmStorage(t *testing.T) {
	st := NewMemoryBackingEvmStorage()
	if st.Get(common.Hash{}) != (common.Hash{}) {
		t.Fail()
	}

	loc1 := intToHash(99)
	val1 := intToHash(1351908)

	st.Set(loc1, val1)
	if st.Get(common.Hash{}) != (common.Hash{}) {
		t.Fail()
	}
	if st.Get(loc1) != val1 {
		t.Fail()
	}
}

func TestStorageSegmentAllocation(t *testing.T) {
	storage := OpenArbosStorage(NewMemoryBackingEvmStorage())
	size := 37
	seg, err := storage.Allocate(uint64(size))
	if err != nil {
		t.Error(err)
	}
	if seg.size != 37 {
		t.Fail()
	}
	res, err := seg.Get(19)
	if err != nil {
		t.Error(err)
	}
	if res != (common.Hash{}) {
		t.Fail()
	}
	if _, err := seg.Get(uint64(size + 3)); err == nil {
		t.Fail()
	}
	if _, err := seg.Get(uint64(size)); err == nil {
		t.Fail()
	}

	val := intToHash(51985380)
	if err := seg.Set(uint64(size-2), val); err != nil {
		t.Error(err)
	}
	res, err = seg.Get(uint64(size - 2))
	if err != nil {
		t.Error(err)
	}
	if res != val {
		t.Fail()
	}
}

func TestStorageSegmentAllocationBytes(t *testing.T) {
	storage := OpenArbosStorage(NewMemoryBackingEvmStorage())
	buf := []byte("This is a long string. The quick brown fox jumped over the lazy dog. Cogito ergo sum.")
	seg, err := storage.AllocateForBytes(buf)
	if err != nil {
		t.Error(err)
	}
	if int(seg.size) != 1 + (len(buf)+31) / 32 {
		t.Fail()
	}

	reread, err := seg.GetBytes()
	if err != nil {
		t.Error(err)
	}
	if bytes.Compare(buf, reread) != 0 {
		t.Fail()
	}
}