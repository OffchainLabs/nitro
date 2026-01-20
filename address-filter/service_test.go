// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package addressfilter

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/util/s3syncer"
)

func TestHashStore_IsRestricted(t *testing.T) {
	store := NewHashStore()

	// Test empty store
	addr := common.HexToAddress("0x1234567890123456789012345678901234567890")
	if store.IsRestricted(addr) {
		t.Error("empty store should not restrict any address")
	}

	// Create test data
	salt := []byte("test-salt")
	addresses := []common.Address{
		common.HexToAddress("0x1111111111111111111111111111111111111111"),
		common.HexToAddress("0x2222222222222222222222222222222222222222"),
		common.HexToAddress("0x3333333333333333333333333333333333333333"),
	}

	// Pre-compute hashes
	var hashes []common.Hash
	for _, addr := range addresses {
		hash := sha256.Sum256(append(salt, addr.Bytes()...))
		hashes = append(hashes, hash)
	}

	// Load the hashes
	store.Load(salt, hashes, "test-etag")

	// Test restricted addresses
	for _, addr := range addresses {
		if !store.IsRestricted(addr) {
			t.Errorf("address %s should be restricted", addr.Hex())
		}
	}

	// Test non-restricted address
	nonRestrictedAddr := common.HexToAddress("0x4444444444444444444444444444444444444444")
	if store.IsRestricted(nonRestrictedAddr) {
		t.Errorf("address %s should not be restricted", nonRestrictedAddr.Hex())
	}

	// Test metadata
	if store.Digest() != "test-etag" {
		t.Errorf("expected etag 'test-etag', got '%s'", store.Digest())
	}
	if store.Size() != 3 {
		t.Errorf("expected size 3, got %d", store.Size())
	}
}

func TestHashStore_AtomicSwap(t *testing.T) {
	store := NewHashStore()

	salt1 := []byte("salt1")
	addr1 := common.HexToAddress("0x1111111111111111111111111111111111111111")
	hash1 := sha256.Sum256(append(salt1, addr1.Bytes()...))

	// Load first set
	store.Load(salt1, []common.Hash{hash1}, "etag1")
	if !store.IsRestricted(addr1) {
		t.Error("addr1 should be restricted after first load")
	}

	// Load second set with different salt (simulating hourly rotation)
	salt2 := []byte("salt2")
	addr2 := common.HexToAddress("0x2222222222222222222222222222222222222222")
	hash2 := sha256.Sum256(append(salt2, addr2.Bytes()...))

	store.Load(salt2, []common.Hash{hash2}, "etag2")

	// addr1 should no longer be restricted (different salt)
	if store.IsRestricted(addr1) {
		t.Error("addr1 should not be restricted after swap (salt changed)")
	}
	// addr2 should now be restricted
	if !store.IsRestricted(addr2) {
		t.Error("addr2 should be restricted after swap")
	}
	if store.Digest() != "etag2" {
		t.Errorf("expected etag 'etag2', got '%s'", store.Digest())
	}
}

func TestHashStore_ConcurrentAccess(t *testing.T) {
	store := NewHashStore()

	salt1 := []byte("test-salt")
	var addresses []common.Address
	var hashes1 []common.Hash
	for i := 0; i < 100; i++ {
		addr := common.BigToAddress(common.Big1)
		addr[18] = byte(i)
		addresses = append(addresses, addr)
		hash := sha256.Sum256(append(salt1, addr.Bytes()...))
		hashes1 = append(hashes1, hash)
	}
	store.Load(salt1, hashes1, "etag")

	// prepare second set for swapping
	salt2 := []byte("new-salt")
	var addresses2 []common.Address
	var hashes2 []common.Hash
	for i := 0; i < 100; i++ {
		addr := common.BigToAddress(common.Big2)
		addr[18] = byte(i)
		addresses2 = append(addresses2, addr)
		hash := sha256.Sum256(append(salt2, addr.Bytes()...))
		hashes2 = append(hashes2, hash)
	}

	// Run concurrent reads
	var wg sync.WaitGroup
	for p := 0; p < 10; p++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				for i := 0; i < 100; i++ {
					addr1 := addresses[i]
					addr2 := addresses2[i]

					if store.isAllRestricted([]common.Address{addr1, addr2}) ||
						!store.isAnyRestricted([]common.Address{addr1, addr2}) {
						// One should be restricted, the other not, atomic swap should ensure consistency
						t.Log("addr1:", addr1.Hex(), "restricted:", store.IsRestricted(addr1))
						t.Log("addr2:", addr2.Hex(), "restricted:", store.IsRestricted(addr2))
						t.Error("concurrent access yielded inconsistent results")
					}
				}
			}
		}()
	}

	// Run concurrent swap
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 10; i++ {
			if i%2 == 0 {
				store.Load(salt1, hashes1, "etag")
			} else {
				store.Load(salt2, hashes2, "new-etag")
			}
			time.Sleep(time.Millisecond)
		}
	}()

	wg.Wait()
}

func TestParseHashListJSON(t *testing.T) {
	hashed_addr1 := sha256.Sum256(common.BigToAddress(common.Big1).Bytes())
	hashed_addr2 := sha256.Sum256(common.BigToAddress(common.Big2).Bytes())
	// Test valid JSON
	// should follow format: {"salt": "hex...", "address_hashes": [{"hash": "hex1", "max_risk_score_level":1}, {"hash": "hex2", "max_risk_score_level":3}, ...]}
	validPayload := map[string]interface{}{
		"salt": hex.EncodeToString([]byte("test-salt")),
		"address_hashes": []map[string]interface{}{
			{
				"hash":                 hex.EncodeToString(hashed_addr1[:]),
				"max_risk_score_level": 2,
			},
			{
				"hash":                 hex.EncodeToString(hashed_addr2[:]),
				"max_risk_score_level": 3,
			},
		},
	}
	validJSON, _ := json.Marshal(validPayload)

	salt, hashes, err := parseHashListJSON(validJSON)
	if err != nil {
		t.Fatalf("failed to parse valid JSON: %v", err)
	}
	if string(salt) != "test-salt" {
		t.Errorf("expected salt 'test-salt', got '%s'", string(salt))
	}
	if len(hashes) != 2 {
		t.Errorf("expected 2 hashes, got %d", len(hashes))
	}

	// Test invalid JSON
	_, _, err = parseHashListJSON([]byte("not json"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}

	// Test invalid salt hex
	invalidSaltPayload := map[string]interface{}{
		"salt":           "not-hex",
		"address_hashes": []map[string]interface{}{{"hash": hex.EncodeToString(hashed_addr1[:])}},
	}
	invalidSaltJSON, _ := json.Marshal(invalidSaltPayload)
	_, _, err = parseHashListJSON(invalidSaltJSON)
	if err == nil {
		t.Error("expected error for invalid salt hex")
	}

	// Test invalid hash hex
	invalidHashPayload := map[string]interface{}{
		"salt":           hex.EncodeToString([]byte("test-salt")),
		"address_hashes": []map[string]interface{}{{"hash": "not-hex"}},
	}
	invalidHashJSON, _ := json.Marshal(invalidHashPayload)
	_, _, err = parseHashListJSON(invalidHashJSON)
	if err == nil {
		t.Error("expected error for invalid hash hex")
	}

	// Test wrong hash length
	wrongLenPayload := map[string]interface{}{
		"salt":           hex.EncodeToString([]byte("test-salt")),
		"address_hashes": []map[string]interface{}{{"hash": "0123456789abcdef"}},
	}
	wrongLenJSON, _ := json.Marshal(wrongLenPayload)
	_, _, err = parseHashListJSON(wrongLenJSON)
	if err == nil {
		t.Error("expected error for wrong hash length")
	}
}

func TestConfig_Validate(t *testing.T) {
	// Test disabled config (should always be valid)
	disabledConfig := Config{Enable: false}
	if err := disabledConfig.Validate(); err != nil {
		t.Errorf("disabled config should be valid: %v", err)
	}

	// Test enabled config with missing fields
	enabledConfig := Config{Enable: true}
	if err := enabledConfig.Validate(); err == nil {
		t.Error("enabled config with missing fields should be invalid")
	}

	// Test valid enabled config
	validConfig := Config{
		Enable: true,
		S3: s3syncer.Config{
			Bucket:    "test-bucket",
			Region:    "us-east-1",
			ObjectKey: "hashlists/current.json",
		},
		PollInterval: 5 * time.Minute,
	}
	if err := validConfig.Validate(); err != nil {
		t.Errorf("valid config should pass validation: %v", err)
	}

	// Test invalid poll interval
	invalidPollConfig := validConfig
	invalidPollConfig.PollInterval = 0
	if err := invalidPollConfig.Validate(); err == nil {
		t.Error("config with zero poll interval should be invalid")
	}
}

func TestHashStore_LoadedAt(t *testing.T) {
	store := NewHashStore()

	// Empty store should have zero time
	if !store.LoadedAt().IsZero() {
		t.Error("empty store should have zero LoadedAt")
	}

	// After load, should have current time
	before := time.Now()
	store.Load([]byte("salt"), nil, "etag")
	after := time.Now()

	loadedAt := store.LoadedAt()
	if loadedAt.Before(before) || loadedAt.After(after) {
		t.Errorf("LoadedAt should be between %v and %v, got %v", before, after, loadedAt)
	}
}

// IsAllRestricted checks if all provided addresses are in the restricted list
// from same hash-store snapshot. Results are cached in the LRU cache.
func (h *HashStore) isAllRestricted(addrs []common.Address) bool {
	data := h.data.Load() // Atomic load - no lock needed
	if len(data.salt) == 0 {
		return false // Not initialized
	}
	for _, addr := range addrs {
		// Check cache first (cache is per-data snapshot)
		if restricted, ok := data.cache.Get(addr); ok {
			if !restricted {
				return false
			}
			continue
		}

		hash := sha256.Sum256(append(data.salt, addr.Bytes()...))
		_, restricted := data.hashes[hash]
		data.cache.Add(addr, restricted)
		if !restricted {
			return false
		}
	}
	return true
}

// IsAnyRestricted checks if any of the provided addresses are in the restricted list
// from same hash-store snapshot. Results are cached in the LRU cache.
func (h *HashStore) isAnyRestricted(addrs []common.Address) bool {
	data := h.data.Load() // Atomic load - no lock needed
	if len(data.salt) == 0 {
		return false // Not initialized
	}
	for _, addr := range addrs {
		// Check cache first (cache is per-data snapshot)
		if restricted, ok := data.cache.Get(addr); ok {
			if restricted {
				return true
			}
			continue
		}

		hash := sha256.Sum256(append(data.salt, addr.Bytes()...))
		_, restricted := data.hashes[hash]
		data.cache.Add(addr, restricted)
		if restricted {
			return true
		}
	}
	return false
}
