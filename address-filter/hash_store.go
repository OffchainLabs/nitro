// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package addressfilter

import (
	"crypto/sha256"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/lru"
)

// hashData holds the immutable hash list data.
// Once created, this struct is never modified, making it safe for concurrent reads.
// The cache is included here so it gets swapped atomically with the hash data.
type hashData struct {
	salt     []byte
	hashes   map[common.Hash]struct{}
	digest   string
	loadedAt time.Time
	cache    *lru.Cache[common.Address, bool] // LRU cache for address lookup results
}

// HashStore provides thread-safe access to restricted address hashes.
// It uses atomic.Pointer for lock-free reads during updates, implementing
// a double-buffering strategy where new data is prepared in the background
// and then atomically swapped in.
type HashStore struct {
	data      atomic.Pointer[hashData]
	cacheSize int
}

const defaultCacheSize = 10000

func NewHashStore() *HashStore {
	return NewHashStoreWithCacheSize(defaultCacheSize)
}

func NewHashStoreWithCacheSize(cacheSize int) *HashStore {
	h := &HashStore{
		cacheSize: cacheSize,
	}
	h.data.Store(&hashData{
		hashes: make(map[common.Hash]struct{}),
		cache:  lru.NewCache[common.Address, bool](cacheSize),
	})
	return h
}

// Load atomically swaps in a new hash list.
// This is called after a new hash list has been downloaded and parsed.
// A new LRU cache is created for the new data, ensuring atomic consistency.
func (h *HashStore) Load(salt []byte, hashes []common.Hash, digest string) {
	newData := &hashData{
		salt:     salt,
		hashes:   make(map[common.Hash]struct{}, len(hashes)),
		digest:   digest,
		loadedAt: time.Now(),
		cache:    lru.NewCache[common.Address, bool](h.cacheSize),
	}
	for _, hash := range hashes {
		newData.hashes[hash] = struct{}{}
	}
	h.data.Store(newData) // Atomic pointer swap
}

// IsRestricted checks if an address is in the restricted list.
// Results are cached in the LRU cache for faster subsequent lookups.
// This method is safe to call concurrently.
func (h *HashStore) IsRestricted(addr common.Address) bool {
	data := h.data.Load() // Atomic load - no lock needed
	if len(data.salt) == 0 {
		return false // Not initialized
	}

	// Check cache first (cache is per-data snapshot)
	if restricted, ok := data.cache.Get(addr); ok {
		return restricted
	}

	hash := sha256.Sum256(append(data.salt, addr.Bytes()...))
	_, restricted := data.hashes[hash]

	// Cache the result
	data.cache.Add(addr, restricted)
	return restricted
}

// Digest Return the digest of the current loaded hashstore.
func (h *HashStore) Digest() string {
	return h.data.Load().digest
}

func (h *HashStore) Size() int {
	return len(h.data.Load().hashes)
}

func (h *HashStore) LoadedAt() time.Time {
	return h.data.Load().loadedAt
}

// Salt returns a copy of the current salt.
func (h *HashStore) Salt() []byte {
	data := h.data.Load()
	if len(data.salt) == 0 {
		return nil
	}
	salt := make([]byte, len(data.salt))
	copy(salt, data.salt)
	return salt
}

// CacheLen returns the current number of entries in the LRU cache.
func (h *HashStore) CacheLen() int {
	return h.data.Load().cache.Len()
}
