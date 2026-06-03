// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package addressfilter

import (
	"crypto/sha256"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/lru"
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
type HashStore struct {
	data      atomic.Pointer[hashData]
	cacheSize int
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

func NewHashStore(cacheSize int) *HashStore {
	h := &HashStore{
		cacheSize: cacheSize,
	}
	h.data.Store(&hashData{
		hashes: make(map[common.Hash]struct{}),
		cache:  lru.NewCache[common.Address, bool](cacheSize),
	})
	return h
}

// Store atomically swaps in a new hash list.
// This is called after a new hash list has been downloaded and parsed.
// A new LRU cache is created for the new data, ensuring atomic consistency.
func (h *HashStore) Store(id uuid.UUID, salt uuid.UUID, scheme HashingScheme, hashes []common.Hash, digest string) {
	newData := &hashData{
		id:                    id,
		salt:                  salt,
		useRawBytesInput:      scheme == HashingSchemeRawBytesInput,
		hashStringInputPrefix: GetHashStringInputPrefix(salt),
		hashes:                make(map[common.Hash]struct{}, len(hashes)),
		digest:                digest,
		loadedAt:              time.Now(),
		cache:                 lru.NewCache[common.Address, bool](h.cacheSize),
	}
	for _, hash := range hashes {
		newData.hashes[hash] = struct{}{}
	}
	h.data.Store(newData) // Atomic pointer swap
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
