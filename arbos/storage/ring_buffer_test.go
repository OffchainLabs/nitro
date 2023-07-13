package storage

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/arbos/burn"
	"github.com/offchainlabs/nitro/arbos/util"
)

func testRingHeaderFirst(t *testing.T, h ringHeader, expectedFirst uint64) {
	t.Helper()
	if have, want := h.first(), expectedFirst; have != want {
		Fatal(t, "unexpected first, have:", have, "want:", want, "header:", fmt.Sprintf("%+v", h))
	}
}

func testRingHeaderNextIndex(t *testing.T, h ringHeader, index, expectedNext uint64) {
	t.Helper()
	if have, want := h.nextIndex(index), expectedNext; have != want {
		Fatal(t, "unexpected next, have:", have, "want:", want, "index:", index, "header:", fmt.Sprintf("%+v", h))
	}
}

func testRingHeaderPrevIndex(t *testing.T, h ringHeader, index, expectedPrev uint64) {
	t.Helper()
	if have, want := h.prevIndex(index), expectedPrev; have != want {
		Fatal(t, "unexpected first, have:", have, "want:", want, "header:", fmt.Sprintf("%+v", h))
	}
}

func TestRingHeaderFirst(t *testing.T) {
	testRingHeaderFirst(t, ringHeader{last: 1, size: 1, capacity: 1, extra: 0}, 1)

	testRingHeaderFirst(t, ringHeader{last: 1, size: 1, capacity: 2, extra: 0}, 1)
	testRingHeaderFirst(t, ringHeader{last: 2, size: 1, capacity: 2, extra: 0}, 2)
	testRingHeaderFirst(t, ringHeader{last: 1, size: 2, capacity: 2, extra: 0}, 2)
	testRingHeaderFirst(t, ringHeader{last: 2, size: 2, capacity: 2, extra: 0}, 1)

	testRingHeaderFirst(t, ringHeader{last: 1, size: 1, capacity: 3, extra: 0}, 1)
	testRingHeaderFirst(t, ringHeader{last: 2, size: 1, capacity: 3, extra: 0}, 2)
	testRingHeaderFirst(t, ringHeader{last: 3, size: 1, capacity: 3, extra: 0}, 3)
	testRingHeaderFirst(t, ringHeader{last: 1, size: 2, capacity: 3, extra: 0}, 3)
	testRingHeaderFirst(t, ringHeader{last: 2, size: 2, capacity: 3, extra: 0}, 1)
	testRingHeaderFirst(t, ringHeader{last: 3, size: 2, capacity: 3, extra: 0}, 2)
	testRingHeaderFirst(t, ringHeader{last: 1, size: 3, capacity: 3, extra: 0}, 2)
	testRingHeaderFirst(t, ringHeader{last: 2, size: 3, capacity: 3, extra: 0}, 3)
	testRingHeaderFirst(t, ringHeader{last: 3, size: 3, capacity: 3, extra: 0}, 1)
}

func TestRingHeaderNextIndex(t *testing.T) {
	testRingHeaderNextIndex(t, ringHeader{last: 1, size: 0, capacity: 1, extra: 0}, 1, 1)
	testRingHeaderNextIndex(t, ringHeader{last: 1, size: 1, capacity: 1, extra: 0}, 1, 1)

	// last and size shouldn't matter
	for l := uint64(1); l <= 2; l++ {
		for s := uint64(0); s <= 2; s++ {
			testRingHeaderNextIndex(t, ringHeader{last: l, size: s, capacity: 2, extra: 0}, 1, 2)
			testRingHeaderNextIndex(t, ringHeader{last: l, size: s, capacity: 2, extra: 0}, 2, 1)
		}
	}

	// last and size shouldn't matter
	for l := uint64(1); l <= 2; l++ {
		for s := uint64(0); s <= 2; s++ {
			testRingHeaderNextIndex(t, ringHeader{last: l, size: s, capacity: 3, extra: 0}, 1, 2)
			testRingHeaderNextIndex(t, ringHeader{last: l, size: s, capacity: 3, extra: 0}, 2, 3)
			testRingHeaderNextIndex(t, ringHeader{last: l, size: s, capacity: 3, extra: 0}, 3, 1)
		}
	}
}

func TestRingHeaderPrevIndex(t *testing.T) {
	testRingHeaderPrevIndex(t, ringHeader{last: 1, size: 0, capacity: 1, extra: 0}, 1, 1)
	testRingHeaderPrevIndex(t, ringHeader{last: 1, size: 1, capacity: 1, extra: 0}, 1, 1)

	// last and size shouldn't matter
	for l := uint64(1); l <= 2; l++ {
		for s := uint64(0); s <= 2; s++ {
			testRingHeaderPrevIndex(t, ringHeader{last: l, size: s, capacity: 2, extra: 0}, 1, 2)
			testRingHeaderPrevIndex(t, ringHeader{last: l, size: s, capacity: 2, extra: 0}, 2, 1)
		}
	}

	// last and size shouldn't matter
	for l := uint64(1); l <= 2; l++ {
		for s := uint64(0); s <= 2; s++ {
			testRingHeaderPrevIndex(t, ringHeader{last: l, size: s, capacity: 3, extra: 0}, 1, 3)
			testRingHeaderPrevIndex(t, ringHeader{last: l, size: s, capacity: 3, extra: 0}, 2, 1)
			testRingHeaderPrevIndex(t, ringHeader{last: l, size: s, capacity: 3, extra: 0}, 3, 2)
		}
	}
}

func testRingBufferRotate(t *testing.T, capacity uint64) {
	t.Helper()
	sto := NewMemoryBacked(burn.NewSystemBurner(nil, false))
	rawSlot := sto.NewSlot(0)
	err := InitializeRingBuffer(sto, capacity)
	Require(t, err, "InitializeRingBuffer failed")
	ring := OpenRingBuffer(sto)
	rawHeader, err := rawSlot.Get()
	Require(t, err, "rawSlot.Get() failed")
	expectedHeaderHash := (&ringHeader{
		last:     capacity,
		size:     0,
		capacity: capacity,
		extra:    0,
	}).toHash()
	if !bytes.Equal(rawHeader.Bytes(), expectedHeaderHash.Bytes()) {
		Fatal(t, "unexpected raw header, want:", expectedHeaderHash, "have:", rawHeader)
	}

	for i := uint64(0); i < capacity; i++ {
		err = ring.Rotate(util.UintToHash(i))
		Require(t, err, "Rotate failed, i: ", i)
		expectedHeaderHash := (&ringHeader{
			last:     i + 1,
			size:     i + 1,
			capacity: capacity,
			extra:    0,
		}).toHash()
		rawHeader, err := rawSlot.Get()
		Require(t, err, "rawSlot.Get() failed")
		if !bytes.Equal(rawHeader.Bytes(), expectedHeaderHash.Bytes()) {
			Fatal(t, "unexpected raw header, want:", expectedHeaderHash, "have:", rawHeader, "i:", i)
		}

		size, err := ring.Size()
		Require(t, err, "Size failed after Rotate, i:", i)
		if size != i+1 {
			Fatal(t, "Unexpected ring size after Rotate, have:", size, "want:", i)
		}

		j := uint64(0)
		err = ring.ForEach(func(index uint64, value common.Hash) (bool, error) {
			t.Helper()
			if index != j {
				Fatal(t, "Unexpected index in ForEach closure, have:", index, "want:", j)
			}
			expectedValue := util.UintToHash(j)
			if !bytes.Equal(value.Bytes(), expectedValue.Bytes()) {
				Fatal(t, "Unexpected value in ForEach closure, have:", value, "want:", expectedValue, "i:", i)
			}
			j++
			return false, nil
		})
		Require(t, err, "ForEach failed, i: ", i, "capacity:", capacity)
	}

	for i := capacity; i <= capacity*3; i++ {
		err = ring.Rotate(util.UintToHash(i))
		Require(t, err, "Rotate failed, i: ", i)

		k := uint64(0)
		err = ring.ForEach(func(index uint64, value common.Hash) (bool, error) {
			t.Helper()
			if index != k {
				Fatal(t, "Unexpected index in ForEach closure, have:", index, "want:", k, "i:", i)
			}
			expectedValue := util.UintToHash(1 + i + k - capacity)
			if !bytes.Equal(value.Bytes(), expectedValue.Bytes()) {
				Fatal(t, "Unexpected value in ForEach closure, have:", value, "want:", expectedValue, "i:", i, "k:", k)
			}
			k++
			return false, nil
		})
		Require(t, err, "ForEach failed, i: ", i, "capacity:", capacity)
		if k != capacity {
			Fatal(t, "ForEach closure wasn't called for all elements, expected:", capacity, "called:", k)
		}
	}
}

func testRingBufferPop(t *testing.T, capacity uint64) {
	t.Helper()
	sto := NewMemoryBacked(burn.NewSystemBurner(nil, false))
	err := InitializeRingBuffer(sto, capacity)
	Require(t, err, "InitializeRingBuffer failed")
	ring := OpenRingBuffer(sto)

	for i := uint64(0); i < capacity; i++ {
		err = ring.Rotate(util.UintToHash(i))
		Require(t, err, "Rotate failed, i: ", i)
	}
	for i := capacity; i > 0; i-- {
		j := i - 1
		value, err := ring.Pop()
		Require(t, err, "Rotate failed, i: ", j)
		expectedValue := util.UintToHash(uint64(j))
		if !bytes.Equal(value.Bytes(), expectedValue.Bytes()) {
			Fatal(t, "Pop returned unexpected Value, have:", value, "want:", expectedValue, "j:", j)
		}
	}
}

func TestRingBuffer(t *testing.T) {
	testRingBufferRotate(t, 0)
	testRingBufferRotate(t, 1)
	testRingBufferRotate(t, 2)
	testRingBufferRotate(t, 3)
	testRingBufferRotate(t, 4)
	testRingBufferRotate(t, 5)
}

func TestRingBufferPop(t *testing.T) {
	testRingBufferPop(t, 1)
	testRingBufferPop(t, 2)
	testRingBufferPop(t, 3)
	testRingBufferPop(t, 4)
	testRingBufferPop(t, 5)
}
