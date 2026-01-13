// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package restrictedaddr

import (
	"crypto/sha256"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

// hashData holds the immutable hash list data.
// Once created, this struct is never modified, making it safe for concurrent reads.
type hashData struct {
	salt     []byte
	hashes   map[[32]byte]struct{}
	digest   string
	loadedAt time.Time
}

// HashStore provides thread-safe access to restricted address hashes.
// It uses atomic.Pointer for lock-free reads during updates, implementing
// a double-buffering strategy where new data is prepared in the background
// and then atomically swapped in.
type HashStore struct {
	data atomic.Pointer[hashData]
}

func NewHashStore() *HashStore {
	h := &HashStore{}
	h.data.Store(&hashData{
		hashes: make(map[[32]byte]struct{}),
	})
	return h
}

// Load atomically swaps in a new hash list.
// This is called after a new hash list has been downloaded and parsed.
func (h *HashStore) Load(salt []byte, hashes [][32]byte, digest string) {
	newData := &hashData{
		salt:     salt,
		hashes:   make(map[[32]byte]struct{}, len(hashes)),
		digest:   digest,
		loadedAt: time.Now(),
	}
	for _, hash := range hashes {
		newData.hashes[hash] = struct{}{}
	}
	h.data.Store(newData) // Atomic pointer swap
}

// IsRestricted checks if an address is in the restricted list.
// This method is lock-free and safe to call concurrently.
func (h *HashStore) IsRestricted(addr common.Address) bool {
	data := h.data.Load() // Atomic load - no lock needed
	if len(data.salt) == 0 {
		return false // Not initialized
	}
	hash := sha256.Sum256(append(data.salt, addr.Bytes()...))
	_, exists := data.hashes[hash]
	return exists
}

// IsAllRestricted checks if all provided addresses are in the restricted list
// from same hash-store snapshot.
func (h *HashStore) IsAllRestricted(addr []common.Address) bool {
	data := h.data.Load() // Atomic load - no lock needed
	if len(data.salt) == 0 {
		return false // Not initialized
	}
	for _, a := range addr {
		hash := sha256.Sum256(append(data.salt, a.Bytes()...))
		_, exists := data.hashes[hash]
		if !exists {
			return false
		}
	}
	return true
}

// IsAnyRestricted checks if any of the provided addresses are in the restricted list
// from same hash-store snapshot.
func (h *HashStore) IsAnyRestricted(addr []common.Address) bool {
	data := h.data.Load() // Atomic load - no lock needed
	if len(data.salt) == 0 {
		return false // Not initialized
	}
	for _, a := range addr {
		hash := sha256.Sum256(append(data.salt, a.Bytes()...))
		_, exists := data.hashes[hash]
		if exists {
			return true
		}
	}
	return false
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
