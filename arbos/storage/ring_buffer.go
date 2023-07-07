package storage

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/util/arbmath"
)

type RingBuffer struct {
	storage *Storage
}

type ringHeader struct {
	last     uint64
	size     uint64
	capacity uint64
	extra    uint64
}

func ringHeaderFromHash(hash common.Hash) *ringHeader {
	last, size, capacity, extra := arbmath.QuadUint64FromHash(hash)
	return &ringHeader{
		last:     last,
		size:     size,
		capacity: capacity,
		extra:    extra,
	}
}

func (h *ringHeader) toHash() common.Hash {
	return arbmath.QuadUint64ToHash(h.last, h.size, h.capacity, h.extra)
}

func (h *ringHeader) first() uint64 {
	if h.last >= h.size {
		return h.last - h.size + 1
	}
	return h.capacity - h.size + h.last + 1
}

func (h *ringHeader) nextIndex(index uint64) uint64 {
	if index == h.capacity {
		return 1
	}
	return index + 1
}

func (h *ringHeader) prevIndex(index uint64) uint64 {
	if index == 1 {
		return h.capacity
	}
	return index - 1
}

func (h *ringHeader) pop() {
	if h.size > 0 {
		h.last = h.prevIndex(h.last)
		h.size--
	}
}

func InitializeRingBuffer(sto *Storage, capacity uint64) error {
	ring := OpenRingBuffer(sto)
	return ring.setHeader(&ringHeader{
		last:     capacity,
		size:     0,
		capacity: capacity,
		extra:    0,
	})
}

func OpenRingBuffer(sto *Storage) *RingBuffer {
	return &RingBuffer{
		sto,
	}
}

func (r *RingBuffer) header() (*ringHeader, error) {
	hash, err := r.storage.GetByUint64(0)
	if err != nil {
		return nil, err
	}
	return ringHeaderFromHash(hash), nil
}
func (r *RingBuffer) setHeader(header *ringHeader) error {
	return r.storage.SetByUint64(0, header.toHash())
}

func (r *RingBuffer) Extra() (uint64, error) {
	header, err := r.header()
	if err != nil {
		return 0, err
	}
	return header.extra, nil
}

func (r *RingBuffer) Capacity() (uint64, error) {
	header, err := r.header()
	if err != nil {
		return 0, err
	}
	return header.capacity, nil
}

func (r *RingBuffer) Size() (uint64, error) {
	header, err := r.header()
	if err != nil {
		return 0, err
	}
	return header.size, nil
}

func (r *RingBuffer) rotateInternal(header *ringHeader, value common.Hash) (*ringHeader, error) {
	if header.capacity == 0 {
		return header, nil
	}
	place := header.nextIndex(header.last)
	err := r.storage.SetByUint64(place, value)
	if err != nil {
		return nil, err
	}
	header.last = place
	if header.size < header.capacity {
		header.size++
	}
	return header, nil
}

func (r *RingBuffer) Rotate(value common.Hash) error {
	header, err := r.header()
	if err != nil {
		return err
	}
	header, err = r.rotateInternal(header, value)
	if err != nil {
		return err
	}
	return r.setHeader(header)
}

// closure gets extra as argument, should return:
// 1. bool - if true, ring should be rotated
// 2. common.Hash - value, which should be added to the ring on rotation
// 3. uint64 - new extra value, ignored if first returned value is false
// 4. error
func (r *RingBuffer) RotateAndSetExtraConditionaly(closure func(uint64) (bool, common.Hash, uint64, error)) error {
	header, err := r.header()
	if err != nil {
		return err
	}
	rotate, value, extra, err := closure(header.extra)
	if err != nil {
		return err
	}
	if rotate {
		header, err = r.rotateInternal(header, value)
		if err != nil {
			return err
		}
		header.extra = extra
		return r.setHeader(header)
	}
	return nil
}

// ForEach apply a closure on the enumerated elements of the queue, index relative to the first (oldest) element
// If closure returns an error, ForEach stops iteration and returns the error
// If closure returns false, iteration is stopped
func (r *RingBuffer) ForEach(closure func(uint64, common.Hash) (bool, error)) error {
	header, err := r.header()
	if err != nil {
		return err
	}
	place := header.first()
	for i := uint64(0); i < header.size; i++ {
		value, err := r.storage.GetByUint64(place)
		if err != nil {
			return err
		}
		proceed, err := closure(i, value)
		if err != nil {
			return err
		}
		if !proceed {
			return nil
		}
		place = header.nextIndex(place)
	}
	return nil
}

func (r *RingBuffer) Peak() (common.Hash, error) {
	header, err := r.header()
	if err != nil {
		return common.Hash{}, err
	}
	if header.size == 0 {
		return common.Hash{}, nil
	}
	value, err := r.storage.GetByUint64(header.last)
	if err != nil {
		return common.Hash{}, err
	}
	return value, nil
}

func (r *RingBuffer) Pop() (common.Hash, error) {
	header, err := r.header()
	if err != nil {
		return common.Hash{}, err
	}
	if header.size == 0 {
		return common.Hash{}, nil
	}
	value, err := r.storage.GetByUint64(header.last)
	if err != nil {
		return common.Hash{}, err
	}
	header.pop()
	if err := r.setHeader(header); err != nil {
		return common.Hash{}, err
	}
	return value, nil
}
