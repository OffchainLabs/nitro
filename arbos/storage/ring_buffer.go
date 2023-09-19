package storage

import (
	"errors"
	"fmt"
	"math"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

var ErrBufferCorrupted = errors.New("ring buffer corrupted")
var ErrInvalidArgument = errors.New("invalid argument")

type RingBuffer struct {
	storage       *Storage
	capacitySlots uint64
	slotsPerElem  uint64
}

const ExtraLenght = common.HashLength - 16

type ringHeader struct {
	last  uint64
	size  uint64
	extra [ExtraLenght]byte
}

func ringHeaderFromHash(hash common.Hash) *ringHeader {
	last := new(big.Int).SetBytes(hash[:8]).Uint64()
	size := new(big.Int).SetBytes(hash[8:16]).Uint64()
	extra := [ExtraLenght]byte(hash[16 : 16+ExtraLenght])
	return &ringHeader{
		last:  last,
		size:  size,
		extra: extra,
	}
}

func (h *ringHeader) toHash() common.Hash {
	bytes := make([]byte, 16+ExtraLenght)
	new(big.Int).SetUint64(h.last).FillBytes(bytes[:8])
	new(big.Int).SetUint64(h.size).FillBytes(bytes[8:16])
	copy(bytes[16:], h.extra[:])
	return common.BytesToHash(bytes)
}

func (b *RingBuffer) first(h *ringHeader) uint64 {
	if h.last >= h.size {
		return h.last - h.size + 1
	}
	return b.capacitySlots - h.size + h.last + 1
}

func (b *RingBuffer) nextIndex(index uint64) uint64 {
	return b.nextIndexN(index, 1)
}

func (b *RingBuffer) nextIndexN(index uint64, n uint64) uint64 {
	n = n % b.capacitySlots
	if n <= b.capacitySlots-index {
		return index + n
	}
	return n - (b.capacitySlots - index)
}

// func (b *RingBuffer) prevIndex(index uint64) uint64 {
//	return b.prevIndexN(index, 1)
//}

func (b *RingBuffer) prevIndexN(index uint64, n uint64) uint64 {
	n = n % b.capacitySlots
	if index > n {
		return index - n
	}
	return b.capacitySlots - n + index
}

func (b *RingBuffer) nextElement(index uint64) uint64 {
	return b.nextIndexN(index, b.slotsPerElem)
}

// func (b *RingBuffer) prevElement(index uint64) uint64 {
//	return b.prevIndexN(index, b.slotsPerElem)
//}

func InitializeRingBuffer(sto *Storage, capacity, slotsPerElem uint64) error {
	// slotsPerElem has to be < math.MaxInt to fit in go slice
	if slotsPerElem == 0 || capacity == 0 || slotsPerElem > math.MaxInt {
		return ErrInvalidArgument
	}
	capacitySlots := capacity * slotsPerElem
	if capacitySlots/slotsPerElem != capacity {
		return ErrInvalidArgument
	}
	ring := OpenRingBuffer(sto, capacity, slotsPerElem)
	return ring.setHeader(&ringHeader{
		last: capacitySlots,
		size: 0,
	})
}

// slotsPerElem can't be 0, capacity * slotsPerElem must not overflow uint64
func OpenRingBuffer(sto *Storage, capacity, slotsPerElem uint64) *RingBuffer {
	capacitySlots := capacity * slotsPerElem
	if slotsPerElem == 0 || capacity == 0 || slotsPerElem > math.MaxInt {
		panic("OpenRingBuffer called with illegal arguments")
	}
	if capacitySlots/slotsPerElem != capacity {
		panic("OpenRingBuffer called with illegal arguments overflowing uint64")
	}
	return &RingBuffer{
		storage:       sto,
		capacitySlots: capacitySlots,
		slotsPerElem:  slotsPerElem,
	}
}

func (b *RingBuffer) header() (*ringHeader, error) {
	hash, err := b.storage.GetByUint64(0)
	if err != nil {
		return nil, fmt.Errorf("failed to get slot value: %w", err)
	}
	return ringHeaderFromHash(hash), nil
}

func (b *RingBuffer) setHeader(header *ringHeader) error {
	if err := b.storage.SetByUint64(0, header.toHash()); err != nil {
		return fmt.Errorf("failed to set slot value: %w", err)
	}
	return nil
}

func (b *RingBuffer) Capacity() (uint64, error) {
	if b.slotsPerElem == 0 {
		return 0, ErrBufferCorrupted
	}
	return b.capacitySlots / b.slotsPerElem, nil
}

func (b *RingBuffer) CapacitySlots() uint64 {
	return b.capacitySlots
}

func (b *RingBuffer) Size() (uint64, error) {
	header, err := b.header()
	if err != nil {
		return 0, err
	}
	if b.slotsPerElem == 0 {
		return 0, ErrBufferCorrupted
	}
	return header.size / b.slotsPerElem, nil
}

func (b *RingBuffer) SizeSlots() (uint64, error) {
	header, err := b.header()
	if err != nil {
		return 0, err
	}
	return header.size, nil
}

func (b *RingBuffer) Rotate(values []common.Hash) error {
	header, err := b.header()
	if err != nil {
		return fmt.Errorf("failed to get ring buffer header: %w", err)
	}
	if uint64(len(values)) != b.slotsPerElem {
		return ErrInvalidArgument
	}
	if b.capacitySlots == 0 {
		return nil
	}
	for _, value := range values {
		place := b.nextIndex(header.last)
		err := b.storage.SetByUint64(place, value)
		if err != nil {
			return fmt.Errorf("failed to set slot value: %w", err)
		}
		header.last = place
		if header.size < b.capacitySlots {
			header.size++
		}
	}
	if err := b.setHeader(header); err != nil {
		return fmt.Errorf("failed to set ring buffer header: %w", err)
	}
	return nil
}

// ForEach applies a closure on the enumerated elements of the queue, index relative to the first (oldest) element
// If closure returns an error, ForEach stops iteration and returns the error
// If closure returns true, iteration is stopped
func (b *RingBuffer) ForEach(closure func([]common.Hash) (bool, error)) error {
	header, err := b.header()
	if err != nil {
		return err
	}
	place := b.first(header)
	for i := uint64(0); i < header.size; i += b.slotsPerElem {
		elem, err := b.elementAt(place)
		if err != nil {
			return err
		}
		done, err := closure(elem)
		if err != nil {
			return err
		}
		if done {
			return nil
		}
		place = b.nextElement(place)
	}
	return nil
}

// ForEachWithSlotIdx applies closure on slotIdx slot of each element
func (b *RingBuffer) ForEachWithSlotIdx(closure func(common.Hash) (bool, error), slotIdx uint64) error {
	header, err := b.header()
	if err != nil {
		return err
	}
	place := b.nextIndexN(b.first(header), slotIdx)
	for i := uint64(0); i < header.size; i += b.slotsPerElem {
		value, err := b.storage.GetByUint64(place)
		if err != nil {
			return fmt.Errorf("failed to get slot value: %w", err)
		}
		done, err := closure(value)
		if err != nil {
			return err
		}
		if done {
			return nil
		}
		place = b.nextIndexN(place, b.slotsPerElem)
	}
	return nil
}

func (b *RingBuffer) elementAt(place uint64) ([]common.Hash, error) {
	elem := make([]common.Hash, b.slotsPerElem)
	for i := uint64(0); i < b.slotsPerElem; i++ {
		value, err := b.storage.GetByUint64(place)
		if err != nil {
			return nil, fmt.Errorf("failed to get slot value: %w", err)
		}
		elem[i] = value
		place = b.nextIndex(place)
	}
	return elem, nil
}

// reads slotIdx slot of the last element
func (b *RingBuffer) PeekSlot(slotIdx uint64) (common.Hash, error) {
	header, err := b.header()
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to get ring buffer header: %w", err)
	}
	if slotIdx >= b.slotsPerElem {
		return common.Hash{}, ErrInvalidArgument
	}
	place := b.prevIndexN(header.last, b.slotsPerElem-slotIdx-1)
	value, err := b.storage.GetByUint64(place)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to get slot value: %w", err)
	}
	return value, nil
}

func (b *RingBuffer) Peek() ([]common.Hash, error) {
	header, err := b.header()
	if err != nil {
		return nil, fmt.Errorf("failed to get ring buffer header: %w", err)
	}
	if header.size == 0 {
		return nil, nil
	}
	if header.size < b.slotsPerElem {
		return nil, ErrBufferCorrupted
	}
	place := b.prevIndexN(header.last, b.slotsPerElem-1)
	return b.elementAt(place)
}

func (b *RingBuffer) SetExtra(extra [ExtraLenght]byte) error {
	header, err := b.header()
	if err != nil {
		return fmt.Errorf("failed to get ring buffer header: %w", err)
	}
	header.extra = extra
	if err := b.setHeader(header); err != nil {
		return fmt.Errorf("failed to set ring buffer header: %w", err)
	}
	return nil
}

func (b *RingBuffer) Extra() ([ExtraLenght]byte, error) {
	header, err := b.header()
	if err != nil {
		return [ExtraLenght]byte{}, fmt.Errorf("failed to get ring buffer header: %w", err)
	}
	return header.extra, nil
}

func (b *RingBuffer) SetExtraUint64(extra uint64) error {
	header, err := b.header()
	if err != nil {
		return fmt.Errorf("failed to get ring buffer header: %w", err)
	}
	new(big.Int).SetUint64(extra).FillBytes(header.extra[:8])
	if err := b.setHeader(header); err != nil {
		return fmt.Errorf("failed to set ring buffer header: %w", err)
	}
	return nil
}

func (b *RingBuffer) ExtraUint64() (uint64, error) {
	header, err := b.header()
	if err != nil {
		return 0, fmt.Errorf("failed to get ring buffer header: %w", err)
	}
	return new(big.Int).SetBytes(header.extra[:8]).Uint64(), nil
}

//
// func (b *RingBuffer) Pop() (common.Hash, error) {
//	values, err := b.PopN(1)
//	if err != nil || len(values) == 0 {
//		return common.Hash{}, err
//	}
//	return values[0], nil
//}
//
// func (b *RingBuffer) PopN(n uint64) ([]common.Hash, error) {
//	header, err := b.header()
//	if err != nil {
//		return nil, err
//	}
//	if header.size == 0 {
//		return nil, nil
//	}
//	if header.size < n {
//		return nil, ErrBufferCorrupted
//	}
//	var values []common.Hash
//	place := header.last
//	for i := uint64(0); i < n; i++ {
//		value, err := b.storage.GetByUint64(place)
//		if err != nil {
//			return nil, err
//		}
//		values = append(values, value)
//		header.pop()
//		place = header.last
//	}
//	if err := b.setHeader(header); err != nil {
//		return nil, err
//	}
//	return values, nil
//}
