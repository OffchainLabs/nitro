package storage

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/arbos/burn"
	"github.com/offchainlabs/nitro/arbos/util"
)

func testRingBufferFirst(t *testing.T, b RingBuffer, h ringHeader, expectedFirst uint64) {
	t.Helper()
	if have, want := b.first(&h), expectedFirst; have != want {
		Fatal(t, "unexpected first, have:", have, "want:", want, "header:", fmt.Sprintf("%+v", h))
	}
}

// func testBufferHeaderNextIndex(t *testing.T, h ringHeader, index, expectedNext uint64) {
//	t.Helper()
//	if have, want := h.nextIndex(index), expectedNext; have != want {
//		Fatal(t, "unexpected next, have:", have, "want:", want, "index:", index, "header:", fmt.Sprintf("%+v", h))
//	}
//}
//
// func testRingHeaderPrevIndex(t *testing.T, h ringHeader, index, expectedPrev uint64) {
//	t.Helper()
//	if have, want := h.prevIndex(index), expectedPrev; have != want {
//		Fatal(t, "unexpected first, have:", have, "want:", want, "header:", fmt.Sprintf("%+v", h))
//	}
//}

func TestRingBufferFirst(t *testing.T) {
	testRingBufferFirst(t, RingBuffer{storage: nil, capacitySlots: 1, slotsPerElem: 1}, ringHeader{last: 1, size: 1}, 1)

	testRingBufferFirst(t, RingBuffer{storage: nil, capacitySlots: 2, slotsPerElem: 1}, ringHeader{last: 1, size: 1}, 1)
	testRingBufferFirst(t, RingBuffer{storage: nil, capacitySlots: 2, slotsPerElem: 1}, ringHeader{last: 2, size: 1}, 2)
	testRingBufferFirst(t, RingBuffer{storage: nil, capacitySlots: 2, slotsPerElem: 1}, ringHeader{last: 1, size: 2}, 2)
	testRingBufferFirst(t, RingBuffer{storage: nil, capacitySlots: 2, slotsPerElem: 1}, ringHeader{last: 2, size: 2}, 1)

	testRingBufferFirst(t, RingBuffer{storage: nil, capacitySlots: 2, slotsPerElem: 2}, ringHeader{last: 1, size: 1}, 1)
	testRingBufferFirst(t, RingBuffer{storage: nil, capacitySlots: 2, slotsPerElem: 2}, ringHeader{last: 2, size: 1}, 2)
	testRingBufferFirst(t, RingBuffer{storage: nil, capacitySlots: 2, slotsPerElem: 2}, ringHeader{last: 1, size: 2}, 2)
	testRingBufferFirst(t, RingBuffer{storage: nil, capacitySlots: 2, slotsPerElem: 2}, ringHeader{last: 2, size: 2}, 1)

	testRingBufferFirst(t, RingBuffer{storage: nil, capacitySlots: 3, slotsPerElem: 1}, ringHeader{last: 1, size: 1}, 1)
	testRingBufferFirst(t, RingBuffer{storage: nil, capacitySlots: 3, slotsPerElem: 1}, ringHeader{last: 2, size: 1}, 2)
	testRingBufferFirst(t, RingBuffer{storage: nil, capacitySlots: 3, slotsPerElem: 1}, ringHeader{last: 3, size: 1}, 3)
	testRingBufferFirst(t, RingBuffer{storage: nil, capacitySlots: 3, slotsPerElem: 1}, ringHeader{last: 1, size: 2}, 3)
	testRingBufferFirst(t, RingBuffer{storage: nil, capacitySlots: 3, slotsPerElem: 1}, ringHeader{last: 2, size: 2}, 1)
	testRingBufferFirst(t, RingBuffer{storage: nil, capacitySlots: 3, slotsPerElem: 1}, ringHeader{last: 3, size: 2}, 2)
	testRingBufferFirst(t, RingBuffer{storage: nil, capacitySlots: 3, slotsPerElem: 1}, ringHeader{last: 1, size: 3}, 2)
	testRingBufferFirst(t, RingBuffer{storage: nil, capacitySlots: 3, slotsPerElem: 1}, ringHeader{last: 2, size: 3}, 3)
	testRingBufferFirst(t, RingBuffer{storage: nil, capacitySlots: 3, slotsPerElem: 1}, ringHeader{last: 3, size: 3}, 1)

	testRingBufferFirst(t, RingBuffer{storage: nil, capacitySlots: 3, slotsPerElem: 3}, ringHeader{last: 1, size: 1}, 1)
	testRingBufferFirst(t, RingBuffer{storage: nil, capacitySlots: 3, slotsPerElem: 3}, ringHeader{last: 2, size: 1}, 2)
	testRingBufferFirst(t, RingBuffer{storage: nil, capacitySlots: 3, slotsPerElem: 3}, ringHeader{last: 3, size: 1}, 3)
	testRingBufferFirst(t, RingBuffer{storage: nil, capacitySlots: 3, slotsPerElem: 3}, ringHeader{last: 1, size: 2}, 3)
	testRingBufferFirst(t, RingBuffer{storage: nil, capacitySlots: 3, slotsPerElem: 3}, ringHeader{last: 2, size: 2}, 1)
	testRingBufferFirst(t, RingBuffer{storage: nil, capacitySlots: 3, slotsPerElem: 3}, ringHeader{last: 3, size: 2}, 2)
	testRingBufferFirst(t, RingBuffer{storage: nil, capacitySlots: 3, slotsPerElem: 3}, ringHeader{last: 1, size: 3}, 2)
	testRingBufferFirst(t, RingBuffer{storage: nil, capacitySlots: 3, slotsPerElem: 3}, ringHeader{last: 2, size: 3}, 3)
	testRingBufferFirst(t, RingBuffer{storage: nil, capacitySlots: 3, slotsPerElem: 3}, ringHeader{last: 3, size: 3}, 1)
}

// func TestRingHeaderNextIndex(t *testing.T) {
//	testBufferHeaderNextIndex(t, ringHeader{last: 1, size: 0, capacity: 1, slotsPerElem: 1}, 1, 1)
//	testBufferHeaderNextIndex(t, ringHeader{last: 1, size: 1, capacity: 1, slotsPerElem: 1}, 1, 1)
//
//	// last, size, slotsPerElem shouldn't matter
//	for l := uint64(1); l <= 2; l++ {
//		for s := uint64(0); s <= 2; s++ {
//			for _, e := range []uint64{1, 2} {
//				testBufferHeaderNextIndex(t, ringHeader{last: l, size: s, capacity: 2, slotsPerElem: e}, 1, 2)
//				testBufferHeaderNextIndex(t, ringHeader{last: l, size: s, capacity: 2, slotsPerElem: e}, 2, 1)
//			}
//		}
//	}
//
//	// last, size, slotsPerElem shouldn't matter
//	for l := uint64(1); l <= 2; l++ {
//		for s := uint64(0); s <= 2; s++ {
//			for _, e := range []uint64{1, 3} {
//				testBufferHeaderNextIndex(t, ringHeader{last: l, size: s, capacity: 3, slotsPerElem: e}, 1, 2)
//				testBufferHeaderNextIndex(t, ringHeader{last: l, size: s, capacity: 3, slotsPerElem: e}, 2, 3)
//				testBufferHeaderNextIndex(t, ringHeader{last: l, size: s, capacity: 3, slotsPerElem: e}, 3, 1)
//			}
//		}
//	}
//}
//
// func TestRingHeaderPrevIndex(t *testing.T) {
//	testRingHeaderPrevIndex(t, ringHeader{last: 1, size: 0, capacity: 1, slotsPerElem: 1}, 1, 1)
//	testRingHeaderPrevIndex(t, ringHeader{last: 1, size: 1, capacity: 1, slotsPerElem: 1}, 1, 1)
//
//	// last and size shouldn't matter
//	for l := uint64(1); l <= 2; l++ {
//		for s := uint64(0); s <= 2; s++ {
//			testRingHeaderPrevIndex(t, ringHeader{last: l, size: s, capacity: 2, slotsPerElem: 1}, 1, 2)
//			testRingHeaderPrevIndex(t, ringHeader{last: l, size: s, capacity: 2, slotsPerElem: 1}, 2, 1)
//		}
//	}
//
//	// last and size shouldn't matter
//	for l := uint64(1); l <= 2; l++ {
//		for s := uint64(0); s <= 2; s++ {
//			testRingHeaderPrevIndex(t, ringHeader{last: l, size: s, capacity: 3, slotsPerElem: 1}, 1, 3)
//			testRingHeaderPrevIndex(t, ringHeader{last: l, size: s, capacity: 3, slotsPerElem: 1}, 2, 1)
//			testRingHeaderPrevIndex(t, ringHeader{last: l, size: s, capacity: 3, slotsPerElem: 1}, 3, 2)
//		}
//	}
//}

func testRingBufferRotate(t *testing.T, capacity uint64) {
	t.Helper()
	sto := NewMemoryBacked(burn.NewSystemBurner(nil, false))
	rawSlot := sto.NewSlot(0)
	err := InitializeRingBuffer(sto, capacity, 1)
	Require(t, err, "InitializeRingBuffer failed")
	ring := OpenRingBuffer(sto, capacity, 1)
	rawHeader, err := rawSlot.Get()
	Require(t, err, "rawSlot.Get() failed")
	expectedHeaderHash := (&ringHeader{
		last: capacity,
		size: 0,
	}).toHash()
	if !bytes.Equal(rawHeader.Bytes(), expectedHeaderHash.Bytes()) {
		Fatal(t, "unexpected raw header, want:", expectedHeaderHash, "have:", rawHeader)
	}

	for i := uint64(0); i < capacity; i++ {
		err = ring.Rotate([]common.Hash{util.UintToHash(i)})
		Require(t, err, "Rotate failed, i: ", i)
		expectedHeaderHash := (&ringHeader{
			last: i + 1,
			size: i + 1,
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
		err = ring.ForEach(func(value []common.Hash) (bool, error) {
			t.Helper()
			expectedValue := util.UintToHash(j)
			if len(value) != 1 || !bytes.Equal(value[0].Bytes(), expectedValue.Bytes()) {
				Fatal(t, "Unexpected value in ForEach closure, have:", value, "want:", expectedValue, "i:", i)
			}
			j++
			return false, nil
		})
		Require(t, err, "ForEach failed, i: ", i, "capacity:", capacity)
	}

	for i := capacity; i <= capacity*3; i++ {
		err = ring.Rotate([]common.Hash{util.UintToHash(i)})
		Require(t, err, "Rotate failed, i: ", i)

		k := uint64(0)
		err = ring.ForEach(func(value []common.Hash) (bool, error) {
			t.Helper()
			expectedValue := util.UintToHash(1 + i + k - capacity)
			if len(value) != 1 || !bytes.Equal(value[0].Bytes(), expectedValue.Bytes()) {
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

func TestRingBufferRotate(t *testing.T) {
	testRingBufferRotate(t, 1)
	testRingBufferRotate(t, 2)
	testRingBufferRotate(t, 3)
	testRingBufferRotate(t, 4)
	testRingBufferRotate(t, 5)
}
