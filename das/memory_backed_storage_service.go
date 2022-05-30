// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"context"
	"errors"
	"github.com/ethereum/go-ethereum/crypto"
	"sync"
)

type MemoryBackedStorageService struct { // intended for testing and debugging
	contents map[[32]byte][]byte
	rwmutex  sync.RWMutex
	closed   bool
}

var ErrClosed = errors.New("cannot access a StorageService that has been Closed")

func NewMemoryBackedStorageService(ctx context.Context) StorageService {
	return &MemoryBackedStorageService{
		contents: make(map[[32]byte][]byte),
	}
}

func (m *MemoryBackedStorageService) GetByHash(ctx context.Context, key []byte) ([]byte, error) {
	m.rwmutex.RLock()
	defer m.rwmutex.RUnlock()
	if m.closed {
		return nil, ErrClosed
	}
	var h32 [32]byte
	copy(h32[:], key)
	res, found := m.contents[h32]
	if !found {
		return nil, ErrNotFound
	}
	return res, nil
}

func (m *MemoryBackedStorageService) Put(ctx context.Context, data []byte, expirationTime uint64) error {
	m.rwmutex.Lock()
	defer m.rwmutex.Unlock()
	if m.closed {
		return ErrClosed
	}
	var h32 [32]byte
	h := crypto.Keccak256(data)
	copy(h32[:], h)
	m.contents[h32] = append([]byte{}, data...)
	return nil
}

func (m *MemoryBackedStorageService) Sync(ctx context.Context) error {
	m.rwmutex.RLock()
	defer m.rwmutex.RUnlock()
	if m.closed {
		return ErrClosed
	}
	return nil
}

func (m *MemoryBackedStorageService) Close(ctx context.Context) error {
	m.rwmutex.Lock()
	defer m.rwmutex.Unlock()
	m.closed = true
	return nil
}

func (m *MemoryBackedStorageService) ExpirationPolicy(ctx context.Context) ExpirationPolicy {
	return KeepForever
}

func (m *MemoryBackedStorageService) String() string {
	return "MemoryBackedStorageService"
}

func (m *MemoryBackedStorageService) HealthCheck(ctx context.Context) error {
	return nil
}
