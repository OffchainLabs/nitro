// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package referenceda

import (
	"context"
	"crypto/sha256"
	"sync"

	"github.com/ethereum/go-ethereum/common"
)

// InMemoryStorage implements PreimageStorage interface for in-memory storage
type InMemoryStorage struct {
	mu        sync.RWMutex
	preimages map[common.Hash][]byte
}

// NewInMemoryStorage creates a new in-memory storage implementation
func NewInMemoryStorage() *InMemoryStorage {
	return &InMemoryStorage{
		preimages: make(map[common.Hash][]byte),
	}
}

func (s *InMemoryStorage) Store(ctx context.Context, data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	hash := sha256.Sum256(data)
	s.preimages[common.BytesToHash(hash[:])] = data
	return nil
}

func (s *InMemoryStorage) GetByHash(ctx context.Context, hash common.Hash) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, exists := s.preimages[hash]
	if !exists {
		return nil, nil
	}
	return data, nil
}
