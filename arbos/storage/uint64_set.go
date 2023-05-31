package storage

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/util/arbmath"
)

type Uint64Set struct {
	storage *Storage
	// TODO(magic) do we want to track number of members?
}

func InitializeUint64Set(sto *Storage) {
	// no need for initialization
}

func OpenUint64Set(sto *Storage) *Uint64Set {
	return &Uint64Set{
		sto,
	}
}

// returns true if value was added,
// false if it was already a member or when an error is returned
func (s *Uint64Set) Add(element uint64) (bool, error) {
	return s.setBit(element, 1)
}

// returns true if element was removed
func (s *Uint64Set) Remove(element uint64) (bool, error) {
	return s.setBit(element, 0)
}

func (s *Uint64Set) AddMaxGasCost() uint64 {
	return s.setBitMaxGasCost()
}

func (s *Uint64Set) RemoveMaxGasCost() uint64 {
	return s.setBitMaxGasCost()
}

func (s *Uint64Set) setBitMaxGasCost() uint64 {
	return StorageReadCost + arbmath.MaxInt(StorageWriteCost, StorageWriteZeroCost)
}

func (s *Uint64Set) setBit(element uint64, to uint) (bool, error) {
	k, v := s.splitKeyValue(element)
	bitset, err := s.storage.GetByUint64(k)
	if err != nil {
		return false, err
	}
	bits := bitset.Big()
	if bits.Bit(v) == to {
		return false, nil
	}
	bits.SetBit(bits, v, to)
	err = s.storage.SetByUint64(k, common.BigToHash(bits))
	return err == nil, err
}

func (s *Uint64Set) IsMember(element uint64) (bool, error) {
	k, v := s.splitKeyValue(element)
	bitset, err := s.storage.GetByUint64(k)
	if err != nil {
		return false, err
	}
	return bitset.Big().Bit(v) == 1, nil
}

func (s *Uint64Set) splitKeyValue(element uint64) (uint64, int) {
	return element & (^uint64(0xff)), int(element & 0xff)
}
