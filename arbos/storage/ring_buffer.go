package storage

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/util/arbmath"
)

var ErrNotEnoughElements = errors.New("not enough elements")

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
	return h.nextIndexN(index, 1)
}

func (h *ringHeader) nextIndexN(index uint64, n uint64) uint64 {
	n = n % h.capacity
	if n <= h.capacity-index {
		return index + n
	}
	return n - (h.capacity - index)
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

func (r *RingBuffer) SetExtra(extra uint64) error {
	header, err := r.header()
	if err != nil {
		return err
	}
	header.extra = extra
	return r.setHeader(header)
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
func (r *RingBuffer) Rotate(value common.Hash) error {
	return r.RotateN([]common.Hash{value})
}

func (r *RingBuffer) RotateN(values []common.Hash) error {
	header, err := r.header()
	if err != nil {
		return err
	}
	if header.capacity == 0 {
		return nil
	}
	for _, value := range values {
		place := header.nextIndex(header.last)
		err := r.storage.SetByUint64(place, value)
		if err != nil {
			return err
		}
		header.last = place
		if header.size < header.capacity {
			header.size++
		}
	}
	return r.setHeader(header)
}

// ForEach applies a closure on the enumerated elements of the queue, index relative to the first (oldest) element
// If closure returns an error, ForEach stops iteration and returns the error
// If closure returns true, iteration is stopped
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
		done, err := closure(i, value)
		if err != nil {
			return err
		}
		if done {
			return nil
		}
		place = header.nextIndex(place)
	}
	return nil
}

// ForSome applies a closure on the enumerated elements of the queue, starting from first + offeset, then skipping step - 1 elements
func (r *RingBuffer) ForSome(closure func(uint64, common.Hash) (bool, error), offset, step uint64) error {
	header, err := r.header()
	if err != nil {
		return err
	}
	place := header.nextIndexN(header.first(), offset)
	for i := uint64(0); i < header.size; i += step {
		value, err := r.storage.GetByUint64(place)
		if err != nil {
			return err
		}
		done, err := closure(i, value)
		if err != nil {
			return err
		}
		if done {
			return nil
		}
		place = header.nextIndexN(place, step)
	}
	return nil
}

func (r *RingBuffer) Peek() (common.Hash, error) {
	values, err := r.PeekN(1)
	if err != nil || len(values) == 0 {
		return common.Hash{}, err
	}
	return values[0], nil
}

func (r *RingBuffer) PeekN(n uint64) ([]common.Hash, error) {
	header, err := r.header()
	if err != nil {
		return nil, err
	}
	if header.size == 0 {
		return nil, nil
	}
	if header.size < n {
		return nil, ErrNotEnoughElements
	}
	var values []common.Hash
	place := header.last
	for i := uint64(0); i < n; i++ {
		value, err := r.storage.GetByUint64(place)
		if err != nil {
			return nil, err
		}
		values = append(values, value)
		place = header.prevIndex(place)
	}
	return values, nil
}

func (r *RingBuffer) Pop() (common.Hash, error) {
	values, err := r.PopN(1)
	if err != nil || len(values) == 0 {
		return common.Hash{}, err
	}
	return values[0], nil
}

func (r *RingBuffer) PopN(n uint64) ([]common.Hash, error) {
	header, err := r.header()
	if err != nil {
		return nil, err
	}
	if header.size == 0 {
		return nil, nil
	}
	if header.size < n {
		return nil, ErrNotEnoughElements
	}
	var values []common.Hash
	place := header.last
	for i := uint64(0); i < n; i++ {
		value, err := r.storage.GetByUint64(place)
		if err != nil {
			return nil, err
		}
		values = append(values, value)
		header.pop()
		place = header.last
	}
	if err := r.setHeader(header); err != nil {
		return nil, err
	}
	return values, nil
}
