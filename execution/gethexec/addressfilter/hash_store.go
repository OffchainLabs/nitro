// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package addressfilter

import (
	"crypto/sha256"
	"encoding/binary"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/lru"

	"github.com/offchainlabs/nitro/util/warmbuffer"
)

type HashingScheme string

const (
	HashingSchemeStringInput   HashingScheme = "sha256-stringinput"
	HashingSchemeRawBytesInput HashingScheme = "sha256-rawbytesinput"
)

// hashData holds a hash list snapshot. In preallocated mode Store recycles a
// snapshot in place, so mu guards its mutable fields: readers take the read
// lock, Store the write lock. The cache swaps atomically with the data and is
// itself safe for concurrent use.
type hashData struct {
	mu                    sync.RWMutex
	id                    uuid.UUID
	salt                  uuid.UUID
	useRawBytesInput      bool
	hashStringInputPrefix string
	hashes                map[common.Hash]struct{}
	digest                string
	loadedAt              time.Time
	cache                 *lru.Cache[common.Address, bool] // LRU cache for address lookup results
}

// HashStore provides thread-safe access to restricted address hashes using a
// double-buffering strategy: new data is prepared and then atomically swapped
// in via an atomic.Pointer, so a reader loads the current snapshot without
// blocking the swap.
//
// When maxHashes > 0 the store preallocates two ping-pong hashData buffers and
// reuses them on every Store, so a reload performs no large allocation. Store
// recycles a buffer in place, so it must not clear one a reader still holds.
// Each hashData carries an RWMutex: a reader holds the read lock for the whole
// call, and Store write-locks the buffer it is about to recycle, blocking until
// every in-flight reader of that buffer has released it. Store is single-writer
// (serialized by the syncer mutex), so active needs no synchronization.
type HashStore struct {
	data      atomic.Pointer[hashData]
	cacheSize int
	maxHashes int
	buffers   [2]*hashData
	active    int
}

func HashStringInputWithPrefix(prefix string, address common.Address) common.Hash {
	hashInput := prefix + common.Bytes2Hex(address.Bytes())
	return sha256.Sum256([]byte(hashInput))
}

func HashRawBytesInput(salt uuid.UUID, address common.Address) common.Hash {
	var buf [len(salt) + common.AddressLength]byte
	copy(buf[:len(salt)], salt[:])
	copy(buf[len(salt):], address[:])
	return sha256.Sum256(buf[:])
}

func GetHashStringInputPrefix(salt uuid.UUID) string {
	return salt.String() + "::0x"
}

func (d *hashData) hashAddress(addr common.Address) common.Hash {
	if d.useRawBytesInput {
		return HashRawBytesInput(d.salt, addr)
	}
	return HashStringInputWithPrefix(d.hashStringInputPrefix, addr)
}

// NewHashStore creates a hash store without preallocation.
func NewHashStore(cacheSize int) *HashStore {
	return newHashStore(cacheSize, 0)
}

// distinctHashGen returns a generator that yields a distinct hash on each call,
// used to fault the bucket memory of a warmed map.
func distinctHashGen() func() common.Hash {
	var counter uint64
	return func() common.Hash {
		var h common.Hash
		binary.LittleEndian.PutUint64(h[:8], counter)
		counter++
		return h
	}
}

// newHashStore creates a hash store. When maxHashes > 0 it preallocates and
// commits two ping-pong buffers sized for maxHashes and reuses them on Store.
func newHashStore(cacheSize int, maxHashes int) *HashStore {
	h := &HashStore{
		cacheSize: cacheSize,
		maxHashes: maxHashes,
	}
	if maxHashes > 0 {
		for i := range h.buffers {
			d := &hashData{
				hashes: warmbuffer.MakeWarmMap[common.Hash, struct{}](maxHashes, distinctHashGen()),
				cache:  lru.NewCache[common.Address, bool](cacheSize),
			}
			h.buffers[i] = d
		}
		h.data.Store(h.buffers[0]) // empty, salt Nil: reports uninitialized
		return h
	}
	h.data.Store(&hashData{
		hashes: make(map[common.Hash]struct{}),
		cache:  lru.NewCache[common.Address, bool](cacheSize),
	})
	return h
}

// fillData populates the scalar fields and hash map of d from a parsed list.
func fillData(d *hashData, id uuid.UUID, salt uuid.UUID, scheme HashingScheme, hashes []common.Hash, digest string) {
	d.id = id
	d.salt = salt
	d.useRawBytesInput = scheme == HashingSchemeRawBytesInput
	d.hashStringInputPrefix = GetHashStringInputPrefix(salt)
	for _, hash := range hashes {
		d.hashes[hash] = struct{}{}
	}
	d.digest = digest
	d.loadedAt = time.Now()
}

// Store atomically swaps in a new hash list.
// This is called after a new hash list has been downloaded and parsed.
// In preallocated mode it recycles a ping-pong buffer in place under the
// buffer's write lock, which blocks until every in-flight reader of that buffer
// has released it; otherwise it builds a new hashData. Either way the LRU cache
// is reset so it stays consistent with the new data. Store is single-writer
// (serialized by the syncer mutex).
func (h *HashStore) Store(id uuid.UUID, salt uuid.UUID, scheme HashingScheme, hashes []common.Hash, digest string) {
	if h.maxHashes == 0 {
		newData := &hashData{
			hashes: make(map[common.Hash]struct{}, len(hashes)),
			cache:  lru.NewCache[common.Address, bool](h.cacheSize),
		}
		fillData(newData, id, salt, scheme, hashes, digest)
		h.data.Store(newData) // Atomic pointer swap
		return
	}

	// Recycle the non-published buffer in place. Its write lock blocks until every
	// reader still holding it from a previous publish has released its read lock,
	// so the clear and refill never race a reader.
	next := 1 - h.active
	d := h.buffers[next]
	d.mu.Lock()
	defer d.mu.Unlock()
	clear(d.hashes) // retains bucket memory
	d.cache.Purge()
	fillData(d, id, salt, scheme, hashes, digest)
	h.data.Store(d) // publish under the write lock; the deferred Unlock releases readers
	h.active = next
}

// IsRestricted returns whether the address is restricted and the filter set ID,
// both read from the same snapshot.
func (h *HashStore) IsRestricted(addr common.Address) (bool, uuid.UUID) {
	data := h.data.Load() // lock-free snapshot load
	data.mu.RLock()
	defer data.mu.RUnlock()
	if data.salt == uuid.Nil {
		return false, uuid.Nil // Not initialized
	}

	// Check cache first (cache is per-data snapshot)
	if restricted, ok := data.cache.Get(addr); ok {
		return restricted, data.id
	}
	_, restricted := data.hashes[data.hashAddress(addr)]
	// Cache the result
	data.cache.Add(addr, restricted)
	return restricted, data.id
}

// Digest Return the digest of the current loaded hashstore.
func (h *HashStore) Digest() string {
	data := h.data.Load()
	data.mu.RLock()
	defer data.mu.RUnlock()
	return data.digest
}

func (h *HashStore) Size() int {
	data := h.data.Load()
	data.mu.RLock()
	defer data.mu.RUnlock()
	return len(data.hashes)
}

func (h *HashStore) LoadedAt() time.Time {
	data := h.data.Load()
	data.mu.RLock()
	defer data.mu.RUnlock()
	return data.loadedAt
}
