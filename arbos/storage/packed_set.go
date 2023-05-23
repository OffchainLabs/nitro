package storage

import "github.com/ethereum/go-ethereum/common"

type PackedSet struct {
	storage *Storage
}

func InitializePackedSet(sto *Storage) {
	// TODO(magic)
}

func OpenPackedSet(sto *Storage) *PackedSet {
	return &PackedSet{
		sto,
		// TODO(magic)
	}
}

func (s *PackedSet) Add(key common.Hash) error {
	// TODO(magic)
	return nil
}

func (s *PackedSet) Remove(key common.Hash) error {
	// TODO(magic)
	return nil
}

func (s *PackedSet) IsMember(key common.Hash) bool {
	// TODO(magic)
	return false
}
