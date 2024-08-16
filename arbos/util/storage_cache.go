// Copyright 2024, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package util

import (
	"github.com/ethereum/go-ethereum/common"
)

type storageCacheEntry struct {
	Value common.Hash
	Known *common.Hash
}

func (e storageCacheEntry) dirty() bool {
	return e.Known == nil || e.Value != *e.Known
}

type storageCacheStores struct {
	Key   common.Hash
	Value common.Hash
}

// storageCache mirrors the stylus storage cache on arbos when tracing a call.
// This is useful for correctly reporting the SLOAD and SSTORE opcodes.
type storageCache struct {
	cache map[common.Hash]storageCacheEntry
}

func newStorageCache() *storageCache {
	return &storageCache{
		cache: make(map[common.Hash]storageCacheEntry),
	}
}

// Load adds a value to the cache and returns true if the logger should emit a load opcode.
func (s *storageCache) Load(key, value common.Hash) bool {
	_, ok := s.cache[key]
	if !ok {
		// The value was not in cache, so it came from EVM
		s.cache[key] = storageCacheEntry{
			Value: value,
			Known: &value,
		}
	}
	return !ok
}

// Store updates the value on the cache.
func (s *storageCache) Store(key, value common.Hash) {
	entry := s.cache[key]
	entry.Value = value // Do not change known value
	s.cache[key] = entry
}

// Flush returns the store operations that should be logged.
func (s *storageCache) Flush() []storageCacheStores {
	stores := []storageCacheStores{}
	for key, entry := range s.cache {
		if entry.dirty() {
			v := entry.Value // Create new var to avoid alliasing
			entry.Known = &v
			s.cache[key] = entry
			stores = append(stores, storageCacheStores{
				Key:   key,
				Value: entry.Value,
			})
		}
	}
	return stores
}

// Clear clears the cache.
func (s *storageCache) Clear() {
	s.cache = make(map[common.Hash]storageCacheEntry)
}
