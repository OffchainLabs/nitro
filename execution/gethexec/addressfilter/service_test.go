// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package addressfilter

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/util/s3client"
	"github.com/offchainlabs/nitro/util/s3syncer"
)

func TestHashStore_IsRestricted(t *testing.T) {
	store := NewHashStore(100)

	// Test empty store
	addr := common.HexToAddress("0xddfAbCdc4D8FfC6d5beaf154f18B778f892A0740")
	if store.IsRestricted(addr) {
		t.Error("empty store should not restrict any address")
	}

	// Create test data
	salt, err := uuid.Parse("3ccf0cbf-b23f-47ba-9c2f-4e7bd672b4c7")
	require.NoError(t, err, "failed to parse salt UUID")

	addresses := []common.Address{
		addr,
		common.HexToAddress("0x2222222222222222222222222222222222222222"),
		common.HexToAddress("0x3333333333333333333333333333333333333333"),
	}

	// Pre-compute hashes
	hashes := []common.Hash{
		common.HexToHash("0x8fb74f22f0aed996e7548101ae1cea812ccdf86e7ad8a781eebea00f797ce4a6"),
		common.HexToHash("0xc9dd008409dbc74d6420ed5ca87c0e833ea10e85562b5d07403195271142f9bb"),
		common.HexToHash("0x615d83d8357c337c142c8d795f1a9334163de4170a870af0ce21e43b67fd5be3"),
	}

	// Store the hashes
	store.Store(uuid.New(), salt, hashes, "test-etag")

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
	store := NewHashStore(100)

	salt1, _ := uuid.Parse("3ccf0cbf-b23f-47ba-9c2f-4e7bd672b4c7")
	addr1 := common.HexToAddress("0x1111111111111111111111111111111111111111")
	hash1 := HashWithSalt(salt1, addr1)

	// Store first set
	store.Store(uuid.New(), salt1, []common.Hash{hash1}, "etag1")
	if !store.IsRestricted(addr1) {
		t.Error("addr1 should be restricted after first load")
	}

	// Store second set with different salt (simulating hourly rotation)
	salt2, _ := uuid.Parse("2cef04bf-b23f-47ba-9c2f-4e7bd652c1c6")
	addr2 := common.HexToAddress("0x2222222222222222222222222222222222222222")
	hash2 := HashWithSalt(salt2, addr2)

	store.Store(uuid.New(), salt2, []common.Hash{hash2}, "etag2")

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
	store := NewHashStore(100)

	salt1, _ := uuid.Parse("3ccf0cbf-b23f-47ba-9c2f-4e7bd672b4c7")

	var addresses []common.Address
	var hashes1 []common.Hash
	for i := 0; i < 100; i++ {
		addr := common.BigToAddress(common.Big1)
		addr[18] = byte(i)
		addresses = append(addresses, addr)
		hash := HashWithSalt(salt1, addr)
		hashes1 = append(hashes1, hash)
	}
	store.Store(uuid.New(), salt1, hashes1, "etag")

	// prepare second set for swapping
	salt2, _ := uuid.Parse("2cef04bf-b23f-47ba-9c2f-4e7bd652c1c6")
	var addresses2 []common.Address
	var hashes2 []common.Hash
	for i := 0; i < 100; i++ {
		addr := common.BigToAddress(common.Big2)
		addr[18] = byte(i)
		addresses2 = append(addresses2, addr)
		hash := HashWithSalt(salt2, addr)
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
				store.Store(uuid.New(), salt1, hashes1, "etag")
			} else {
				store.Store(uuid.New(), salt2, hashes2, "new-etag")
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
	// should follow format: {"id": "uuid-format", "salt": "uuid-format", "address_hashes": [{"hash": "hex1", "max_risk_score_level":1}, {"hash": "hex2", "max_risk_score_level":3}, ...]}
	id := uuid.New()
	validPayload := map[string]interface{}{
		"id":   id,
		"salt": "2cef04bf-b23f-47ba-9c2f-4e7bd652c1c6",
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

	parsedJson, err := parseHashListJSON(validJSON)
	if err != nil {
		t.Fatalf("failed to parse valid JSON: %v", err)
	}
	expectedSalt, _ := uuid.Parse("2cef04bf-b23f-47ba-9c2f-4e7bd652c1c6")
	if parsedJson.Salt != expectedSalt {
		t.Errorf("expected salt '%s', got '%s'", expectedSalt.String(), parsedJson.Salt.String())
	}

	if parsedJson.Id != id {
		t.Errorf("expected id '%s', got '%s'", id.String(), parsedJson.Id.String())
	}

	if len(parsedJson.Hashes) != 2 {
		t.Errorf("expected 2 hashes, got %d", len(parsedJson.Hashes))
	}

	// Test invalid JSON
	_, err = parseHashListJSON([]byte("not json"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}

	// Test invalid salt hex
	invalidSaltPayload := map[string]interface{}{
		"salt":           "not-UUID-salt",
		"id":             uuid.NewString(),
		"address_hashes": []map[string]interface{}{{"hash": hex.EncodeToString(hashed_addr1[:])}},
	}
	invalidSaltJSON, _ := json.Marshal(invalidSaltPayload)
	_, err = parseHashListJSON(invalidSaltJSON)
	if err == nil {
		t.Error("expected error for invalid salt hex")
	}

	// Test invalid hash hex
	invalidHashPayload := map[string]interface{}{
		"salt":           "2cef04bf-b23f-47ba-9c2f-4e7bd652c1c6",
		"address_hashes": []map[string]interface{}{{"hash": "not-hex"}},
	}
	invalidHashJSON, _ := json.Marshal(invalidHashPayload)
	_, err = parseHashListJSON(invalidHashJSON)
	if err == nil {
		t.Error("expected error for invalid hash hex")
	}

	// Test wrong hash length
	wrongLenPayload := map[string]interface{}{
		"salt":           "2cef04bf-b23f-47ba-9c2f-4e7bd652c1c6",
		"address_hashes": []map[string]interface{}{{"hash": "0123456789abcdef"}},
	}
	wrongLenJSON, _ := json.Marshal(wrongLenPayload)
	_, err = parseHashListJSON(wrongLenJSON)
	if err == nil {
		t.Error("expected error for wrong hash length")
	}

	// Test with hashing_scheme: Sha256 (should parse without error)
	sha256Payload := map[string]interface{}{
		"salt":           "2cef04bf-b23f-47ba-9c2f-4e7bd652c1c6",
		"id":             uuid.NewString(),
		"hashing_scheme": "Sha256",
		"address_hashes": []map[string]interface{}{
			{"hash": hex.EncodeToString(hashed_addr1[:])},
		},
	}
	sha256JSON, _ := json.Marshal(sha256Payload)
	parsedJson, err = parseHashListJSON(sha256JSON)
	if err != nil {
		t.Fatalf("failed to parse JSON with Sha256 hashing_scheme: %v", err)
	}
	if len(parsedJson.Hashes) != 1 {
		t.Errorf("expected 1 hash, got %d", len(parsedJson.Hashes))
	}

	// Test with unknown hashing_scheme (should parse but log warning - we can't easily verify log in test)
	unknownSchemePayload := map[string]interface{}{
		"salt":           "2cef04bf-b23f-47ba-9c2f-4e7bd652c1c6",
		"id":             uuid.NewString(),
		"hashing_scheme": "Unknown",
		"address_hashes": []map[string]interface{}{
			{"hash": hex.EncodeToString(hashed_addr1[:])},
		},
	}
	unknownSchemeJSON, _ := json.Marshal(unknownSchemePayload)
	parsedJson, err = parseHashListJSON(unknownSchemeJSON)
	if err != nil {
		t.Fatalf("failed to parse JSON with unknown hashing_scheme: %v", err)
	}
	if len(parsedJson.Hashes) != 1 {
		t.Errorf("expected 1 hash, got %d", len(parsedJson.Hashes))
	}

	// Test with 0x-prefixed hashes (lowercase)
	prefixedPayload := map[string]interface{}{
		"salt": "3ccf0cbf-b23f-47ba-9c2f-4e7bd672b4c7",
		"id":   uuid.NewString(),
		"address_hashes": []map[string]interface{}{
			{"hash": "0x" + hex.EncodeToString(hashed_addr1[:])},
			{"hash": "0X" + hex.EncodeToString(hashed_addr2[:])},
		},
	}
	prefixedJSON, _ := json.Marshal(prefixedPayload)
	parsedJson, err = parseHashListJSON(prefixedJSON)
	if err != nil {
		t.Fatalf("failed to parse 0x-prefixed JSON: %v", err)
	}
	if len(parsedJson.Hashes) != 2 {
		t.Errorf("expected 2 hashes, got %d", len(parsedJson.Hashes))
	}
	if parsedJson.Hashes[0] != hashed_addr1 {
		t.Errorf("hash[0] mismatch: got %x, want %x", parsedJson.Hashes[0], hashed_addr1)
	}
	// Test without hashing_scheme field (backward compatible)
	noSchemePayload := map[string]interface{}{
		"salt": "2cef04bf-b23f-47ba-9c2f-4e7bd652c1c6",
		"id":   uuid.NewString(),
		"address_hashes": []map[string]interface{}{
			{"hash": hex.EncodeToString(hashed_addr1[:])},
		},
	}
	noSchemeJSON, _ := json.Marshal(noSchemePayload)
	parsedJson, err = parseHashListJSON(noSchemeJSON)
	if err != nil {
		t.Fatalf("failed to parse JSON without hashing_scheme: %v", err)
	}
	if len(parsedJson.Hashes) != 1 {
		t.Errorf("expected 1 hash, got %d", len(parsedJson.Hashes))
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
			Config:    s3client.Config{Region: "us-east-1"},
			Bucket:    "test-bucket",
			ObjectKey: "hashlists/current.json",
		},
		PollInterval:              5 * time.Minute,
		CacheSize:                 10000,
		AddressCheckerWorkerCount: 4,
		AddressCheckerQueueSize:   8192,
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

	// Test invalid cache size (zero)
	invalidCacheConfig := validConfig
	invalidCacheConfig.PollInterval = 5 * time.Minute
	invalidCacheConfig.CacheSize = 0
	if err := invalidCacheConfig.Validate(); err == nil {
		t.Error("config with zero cache size should be invalid")
	}

	// Test invalid cache size (negative)
	invalidCacheConfig.CacheSize = -1
	if err := invalidCacheConfig.Validate(); err == nil {
		t.Error("config with negative cache size should be invalid")
	}
}

func TestHashStore_CustomCacheSize(t *testing.T) {
	// Test creating store with custom cache size
	store := NewHashStore(500)

	// Create test data
	salt, _ := uuid.Parse("2cef04bf-b23f-47ba-9c2f-4e7bd652c1c6")
	addresses := []common.Address{
		common.HexToAddress("0x1111111111111111111111111111111111111111"),
		common.HexToAddress("0x2222222222222222222222222222222222222222"),
	}

	// Pre-compute hashes
	hashes := make([]common.Hash, 0, len(addresses))
	for _, addr := range addresses {
		hash := HashWithSalt(salt, addr)
		hashes = append(hashes, hash)
	}

	// Store the hashes
	store.Store(uuid.New(), salt, hashes, "test-etag")

	// Verify store works correctly with custom size
	if !store.IsRestricted(addresses[0]) {
		t.Error("address should be restricted")
	}
	if !store.IsRestricted(addresses[1]) {
		t.Error("address should be restricted")
	}

	nonRestrictedAddr := common.HexToAddress("0x3333333333333333333333333333333333333333")
	if store.IsRestricted(nonRestrictedAddr) {
		t.Error("address should not be restricted")
	}
}

func TestHashStore_LoadedAt(t *testing.T) {
	store := NewHashStore(100)

	// Empty store should have zero time
	if !store.LoadedAt().IsZero() {
		t.Error("empty store should have zero LoadedAt")
	}

	// After load, should have current time
	before := time.Now()
	salt, _ := uuid.Parse("2cef04bf-b23f-47ba-9c2f-4e7bd652c1c6")
	store.Store(uuid.New(), salt, nil, "etag")
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
	if data.salt == uuid.Nil {
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

		hash := HashWithSalt(data.salt, addr)
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
	if data.salt == uuid.Nil {
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

		hash := HashWithSalt(data.salt, addr)
		_, restricted := data.hashes[hash]
		data.cache.Add(addr, restricted)
		if restricted {
			return true
		}
	}
	return false
}
