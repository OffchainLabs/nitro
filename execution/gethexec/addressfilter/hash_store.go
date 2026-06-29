// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package addressfilter

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
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

// hashData holds the immutable hash list data.
// Once created, this struct is never modified, making it safe for concurrent reads.
// The cache is included here so it gets swapped atomically with the hash data.
type hashData struct {
	id                    uuid.UUID
	salt                  uuid.UUID
	useRawBytesInput      bool
	hashStringInputPrefix string
	hashes                map[common.Hash]struct{}
	digest                string
	loadedAt              time.Time
	cache                 *lru.Cache[common.Address, bool] // LRU cache for address lookup results
}

// HashStore provides thread-safe access to restricted address hashes.
// It uses atomic.Pointer for lock-free reads during updates, implementing
// a double-buffering strategy where new data is prepared in the background
// and then atomically swapped in.
//
// When maxHashes > 0 the store preallocates two ping-pong hashData buffers and
// reuses them on every Store, so a reload performs no large allocation. Store
// recycles a buffer in place, so it must not touch a buffer a reader still
// holds. A reader holds a snapshot only for a single IsRestricted call, so Store
// recycles a buffer only after it has been free (unpublished) for reuseGrace,
// blocking until then; freeAt records when each buffer was retired. With the
// grace far larger than any read, no reader can reference the buffer being
// cleared. Store is single-writer (serialized by the syncer mutex), so active,
// freeAt, and reuseGrace need no synchronization.
type HashStore struct {
	data       atomic.Pointer[hashData]
	cacheSize  int
	maxHashes  int
	buffers    [2]*hashData
	active     int
	freeAt     [2]time.Time
	reuseGrace time.Duration
}

// defaultBufferReuseGracePeriod is how long a preallocated buffer must sit
// unpublished before Store may recycle it. It is far larger than any plausible
// IsRestricted call yet far smaller than the default poll interval, so a
// steady-state reload never waits.
const defaultBufferReuseGracePeriod = 10 * time.Second

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
		h.reuseGrace = defaultBufferReuseGracePeriod
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
// In preallocated mode it reuses a ping-pong buffer, blocking until the buffer
// has been free for reuseGrace so no reader can still hold it; otherwise it
// builds a new hashData. Either way the LRU cache is reset so it stays
// consistent with the new data. Store is single-writer (serialized by the syncer
// mutex). It returns ctx.Err() if the context is cancelled while waiting, in
// which case the currently published data is left untouched.
func (h *HashStore) Store(ctx context.Context, id uuid.UUID, salt uuid.UUID, scheme HashingScheme, hashes []common.Hash, digest string) error {
	if h.maxHashes == 0 {
		newData := &hashData{
			hashes: make(map[common.Hash]struct{}, len(hashes)),
			cache:  lru.NewCache[common.Address, bool](h.cacheSize),
		}
		fillData(newData, id, salt, scheme, hashes, digest)
		h.data.Store(newData) // Atomic pointer swap
		return nil
	}
	next := 1 - h.active

	// Reuse the non-published buffer, but only once it has been unpublished for
	// reuseGrace (see the HashStore doc), which guarantees no reader still holds
	// it. freeAt is when the buffer was retired; its zero value means it was never
	// published, so the first reuse is immediate. Go timers never fire early, so a
	// single wait is exact. Cancelling leaves the live data untouched.
	if wait := h.reuseGrace - time.Since(h.freeAt[next]); wait > 0 {
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}

	// The buffer is now quiescent: clear and refill it in place, then publish.
	d := h.buffers[next]
	clear(d.hashes) // retains bucket memory
	d.cache.Purge()
	fillData(d, id, salt, scheme, hashes, digest)
	h.data.Store(d)                 // atomic pointer swap
	h.freeAt[h.active] = time.Now() // the buffer just retired becomes free now
	h.active = next
	return nil
}

// IsRestricted returns whether the address is restricted and the filter set ID,
// both read from the same snapshot.
func (h *HashStore) IsRestricted(addr common.Address) (bool, uuid.UUID) {
	data := h.data.Load() // Atomic load - no lock needed
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
	return h.data.Load().digest
}

func (h *HashStore) Size() int {
	return len(h.data.Load().hashes)
}

func (h *HashStore) LoadedAt() time.Time {
	return h.data.Load().loadedAt
}
