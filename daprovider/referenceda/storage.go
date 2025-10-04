// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package referenceda

import (
	"crypto/sha256"
	"sync"

	"github.com/ethereum/go-ethereum/common"
)

// InMemoryStorage implements PreimageStorage interface for in-memory storage
type InMemoryStorage struct {
	mu        sync.RWMutex
	preimages map[common.Hash][]byte
}

var (
	// singleton instance of InMemoryStorage
	storageInstance *InMemoryStorage
	storageOnce     sync.Once
)

// GetInMemoryStorage returns the singleton instance of InMemoryStorage
func GetInMemoryStorage() *InMemoryStorage {
	storageOnce.Do(func() {
		storageInstance = &InMemoryStorage{
			preimages: make(map[common.Hash][]byte),
		}
	})
	return storageInstance
}

func (s *InMemoryStorage) Store(data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	hash := sha256.Sum256(data)
	s.preimages[common.BytesToHash(hash[:])] = data
	return nil
}

func (s *InMemoryStorage) GetByHash(hash common.Hash) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, exists := s.preimages[hash]
	if !exists {
		return nil, nil
	}
	return data, nil
}
