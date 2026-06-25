// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package addressfilter

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/johannesboyne/gofakes3"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/util/s3client"
	"github.com/offchainlabs/nitro/util/s3syncer"
	"github.com/offchainlabs/nitro/util/s3syncer/s3syncertest"
)

func TestHashStore_IsRestricted(t *testing.T) {
	store := NewHashStore(100)

	// Test empty store
	addr := common.HexToAddress("0xddfAbCdc4D8FfC6d5beaf154f18B778f892A0740")
	restricted, returnedID := store.IsRestricted(addr)
	if restricted {
		t.Error("empty store should not restrict any address")
	}
	if returnedID != uuid.Nil {
		t.Errorf("empty store: expected nil filter set ID, got %s", returnedID)
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
	filterSetID := uuid.New()
	store.Store(filterSetID, salt, HashingSchemeStringInput, hashes, "test-etag")

	// Test restricted addresses
	for _, addr := range addresses {
		restricted, returnedID := store.IsRestricted(addr)
		if !restricted {
			t.Errorf("address %s should be restricted", addr.Hex())
		}
		if returnedID != filterSetID {
			t.Errorf("address %s: expected filter set ID %s, got %s", addr.Hex(), filterSetID, returnedID)
		}
	}

	// Test non-restricted address
	nonRestrictedAddr := common.HexToAddress("0x4444444444444444444444444444444444444444")
	restricted, returnedID = store.IsRestricted(nonRestrictedAddr)
	if restricted {
		t.Errorf("address %s should not be restricted", nonRestrictedAddr.Hex())
	}
	if returnedID != filterSetID {
		t.Errorf("non-restricted address: expected filter set ID %s, got %s", filterSetID, returnedID)
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
	hash1 := HashStringInputWithPrefix(GetHashStringInputPrefix(salt1), addr1)

	// Store first set
	filterSetID1 := uuid.New()
	store.Store(filterSetID1, salt1, HashingSchemeStringInput, []common.Hash{hash1}, "etag1")
	restricted, returnedID := store.IsRestricted(addr1)
	if !restricted {
		t.Error("addr1 should be restricted after first load")
	}
	if returnedID != filterSetID1 {
		t.Errorf("expected filter set ID %s, got %s", filterSetID1, returnedID)
	}

	// Store second set with different salt (simulating hourly rotation)
	salt2, _ := uuid.Parse("2cef04bf-b23f-47ba-9c2f-4e7bd652c1c6")
	addr2 := common.HexToAddress("0x2222222222222222222222222222222222222222")
	hash2 := HashStringInputWithPrefix(GetHashStringInputPrefix(salt2), addr2)

	filterSetID2 := uuid.New()
	store.Store(filterSetID2, salt2, HashingSchemeStringInput, []common.Hash{hash2}, "etag2")

	// addr1 should no longer be restricted (different salt)
	restricted, returnedID = store.IsRestricted(addr1)
	if restricted {
		t.Error("addr1 should not be restricted after swap (salt changed)")
	}
	if returnedID != filterSetID2 {
		t.Errorf("expected filter set ID %s after swap, got %s", filterSetID2, returnedID)
	}
	// addr2 should now be restricted
	restricted, returnedID = store.IsRestricted(addr2)
	if !restricted {
		t.Error("addr2 should be restricted after swap")
	}
	if returnedID != filterSetID2 {
		t.Errorf("expected filter set ID %s, got %s", filterSetID2, returnedID)
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
		hash := HashStringInputWithPrefix(GetHashStringInputPrefix(salt1), addr)
		hashes1 = append(hashes1, hash)
	}
	store.Store(uuid.New(), salt1, HashingSchemeStringInput, hashes1, "etag")

	// prepare second set for swapping
	salt2, _ := uuid.Parse("2cef04bf-b23f-47ba-9c2f-4e7bd652c1c6")
	var addresses2 []common.Address
	var hashes2 []common.Hash
	for i := 0; i < 100; i++ {
		addr := common.BigToAddress(common.Big2)
		addr[18] = byte(i)
		addresses2 = append(addresses2, addr)
		hash := HashStringInputWithPrefix(GetHashStringInputPrefix(salt2), addr)
		hashes2 = append(hashes2, hash)
	}

	rawHashes1 := make([]common.Hash, len(addresses))
	for i, addr := range addresses {
		rawHashes1[i] = HashRawBytesInput(salt1, addr)
	}
	rawHashes2 := make([]common.Hash, len(addresses2))
	for i, addr := range addresses2 {
		rawHashes2[i] = HashRawBytesInput(salt2, addr)
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
						r1, _ := store.IsRestricted(addr1)
						r2, _ := store.IsRestricted(addr2)
						t.Log("addr1:", addr1.Hex(), "restricted:", r1)
						t.Log("addr2:", addr2.Hex(), "restricted:", r2)
						t.Error("concurrent access yielded inconsistent results")
					}
				}
			}
		}()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 20; i++ {
			switch i % 4 {
			case 0:
				store.Store(uuid.New(), salt1, HashingSchemeStringInput, hashes1, "salt1-str")
			case 1:
				store.Store(uuid.New(), salt2, HashingSchemeStringInput, hashes2, "salt2-str")
			case 2:
				store.Store(uuid.New(), salt1, HashingSchemeRawBytesInput, rawHashes1, "salt1-raw")
			case 3:
				store.Store(uuid.New(), salt2, HashingSchemeRawBytesInput, rawHashes2, "salt2-raw")
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
	// should follow format: {"id": "uuid-format", "salt": "uuid-format", "hashing_scheme": "sha256-stringinput", "hashes": ["hex1", "hex2", ...]}
	// Unknown top-level fields (e.g. extract_uuid, issued_at) must be ignored.
	id := uuid.New()
	validPayload := map[string]interface{}{
		"id":           id,
		"salt":         "2cef04bf-b23f-47ba-9c2f-4e7bd652c1c6",
		"extract_uuid": "2cef04bf-b23f-47ba-9c2f-4e7bd652c1c6",
		"issued_at":    "2026-05-13T12:05:45Z",
		"hashes": []string{
			hex.EncodeToString(hashed_addr1[:]),
			hex.EncodeToString(hashed_addr2[:]),
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
		"salt":   "not-UUID-salt",
		"id":     uuid.NewString(),
		"hashes": []string{hex.EncodeToString(hashed_addr1[:])},
	}
	invalidSaltJSON, _ := json.Marshal(invalidSaltPayload)
	_, err = parseHashListJSON(invalidSaltJSON)
	if err == nil {
		t.Error("expected error for invalid salt hex")
	}

	// Test invalid hash hex
	invalidHashPayload := map[string]interface{}{
		"salt":   "2cef04bf-b23f-47ba-9c2f-4e7bd652c1c6",
		"id":     uuid.NewString(),
		"hashes": []string{"not-hex"},
	}
	invalidHashJSON, _ := json.Marshal(invalidHashPayload)
	_, err = parseHashListJSON(invalidHashJSON)
	if err == nil {
		t.Error("expected error for invalid hash hex")
	}

	// Test wrong hash length
	wrongLenPayload := map[string]interface{}{
		"salt":   "2cef04bf-b23f-47ba-9c2f-4e7bd652c1c6",
		"id":     uuid.NewString(),
		"hashes": []string{"0123456789abcdef"},
	}
	wrongLenJSON, _ := json.Marshal(wrongLenPayload)
	_, err = parseHashListJSON(wrongLenJSON)
	if err == nil {
		t.Error("expected error for wrong hash length")
	}

	// Test with hashing_scheme: sha256-stringinput (should parse without error)
	sha256Payload := map[string]interface{}{
		"salt":           "2cef04bf-b23f-47ba-9c2f-4e7bd652c1c6",
		"id":             uuid.NewString(),
		"hashing_scheme": "sha256-stringinput",
		"hashes":         []string{hex.EncodeToString(hashed_addr1[:])},
	}
	sha256JSON, _ := json.Marshal(sha256Payload)
	parsedJson, err = parseHashListJSON(sha256JSON)
	if err != nil {
		t.Fatalf("failed to parse JSON with sha256-stringinput hashing_scheme: %v", err)
	}
	if len(parsedJson.Hashes) != 1 {
		t.Errorf("expected 1 hash, got %d", len(parsedJson.Hashes))
	}
	if parsedJson.Scheme != HashingSchemeStringInput {
		t.Errorf("expected scheme %q, got %q", HashingSchemeStringInput, parsedJson.Scheme)
	}

	// Test with hashing_scheme: sha256-rawbytesinput
	rawBytesPayload := map[string]interface{}{
		"salt":           "2cef04bf-b23f-47ba-9c2f-4e7bd652c1c6",
		"id":             uuid.NewString(),
		"hashing_scheme": "sha256-rawbytesinput",
		"hashes":         []string{hex.EncodeToString(hashed_addr1[:])},
	}
	rawBytesJSON, err := json.Marshal(rawBytesPayload)
	require.NoError(t, err)
	parsedJson, err = parseHashListJSON(rawBytesJSON)
	require.NoError(t, err)
	if parsedJson.Scheme != HashingSchemeRawBytesInput {
		t.Errorf("expected scheme %q, got %q", HashingSchemeRawBytesInput, parsedJson.Scheme)
	}

	// Test with unknown hashing_scheme — hard error
	unknownSchemePayload := map[string]interface{}{
		"salt":           "2cef04bf-b23f-47ba-9c2f-4e7bd652c1c6",
		"id":             uuid.NewString(),
		"hashing_scheme": "Unknown",
		"hashes":         []string{hex.EncodeToString(hashed_addr1[:])},
	}
	unknownSchemeJSON, _ := json.Marshal(unknownSchemePayload)
	if _, err := parseHashListJSON(unknownSchemeJSON); err == nil {
		t.Error("expected error for unknown hashing_scheme")
	}

	// Case-sensitivity: uppercase scheme must hard-error too (spec is lowercase).
	upperSchemePayload := map[string]interface{}{
		"salt":           "2cef04bf-b23f-47ba-9c2f-4e7bd652c1c6",
		"id":             uuid.NewString(),
		"hashing_scheme": "SHA256-RAWBYTESINPUT",
		"hashes":         []string{hex.EncodeToString(hashed_addr1[:])},
	}
	upperSchemeJSON, _ := json.Marshal(upperSchemePayload)
	if _, err := parseHashListJSON(upperSchemeJSON); err == nil {
		t.Error("expected error for uppercased hashing_scheme")
	}

	// Test with 0x-prefixed hashes (lowercase)
	prefixedPayload := map[string]interface{}{
		"salt": "3ccf0cbf-b23f-47ba-9c2f-4e7bd672b4c7",
		"id":   uuid.NewString(),
		"hashes": []string{
			"0x" + hex.EncodeToString(hashed_addr1[:]),
			"0X" + hex.EncodeToString(hashed_addr2[:]),
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
		"salt":   "2cef04bf-b23f-47ba-9c2f-4e7bd652c1c6",
		"id":     uuid.NewString(),
		"hashes": []string{hex.EncodeToString(hashed_addr1[:])},
	}
	noSchemeJSON, _ := json.Marshal(noSchemePayload)
	parsedJson, err = parseHashListJSON(noSchemeJSON)
	if err != nil {
		t.Fatalf("failed to parse JSON without hashing_scheme: %v", err)
	}
	if len(parsedJson.Hashes) != 1 {
		t.Errorf("expected 1 hash, got %d", len(parsedJson.Hashes))
	}
	if parsedJson.Scheme != HashingSchemeStringInput {
		t.Errorf("missing scheme should default to %q, got %q", HashingSchemeStringInput, parsedJson.Scheme)
	}
}

func TestConfig_Validate(t *testing.T) {
	// Test config with missing fields
	emptyConfig := Config{}
	if err := emptyConfig.Validate(); err == nil {
		t.Error("config with missing fields should be invalid")
	}

	// Test valid config
	validConfig := Config{
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
		hash := HashStringInputWithPrefix(GetHashStringInputPrefix(salt), addr)
		hashes = append(hashes, hash)
	}

	// Store the hashes
	store.Store(uuid.New(), salt, HashingSchemeStringInput, hashes, "test-etag")

	// Verify store works correctly with custom size
	if restricted, _ := store.IsRestricted(addresses[0]); !restricted {
		t.Error("address should be restricted")
	}
	if restricted, _ := store.IsRestricted(addresses[1]); !restricted {
		t.Error("address should be restricted")
	}

	nonRestrictedAddr := common.HexToAddress("0x3333333333333333333333333333333333333333")
	if restricted, _ := store.IsRestricted(nonRestrictedAddr); restricted {
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
	store.Store(uuid.New(), salt, HashingSchemeStringInput, nil, "etag")
	after := time.Now()

	loadedAt := store.LoadedAt()
	if loadedAt.Before(before) || loadedAt.After(after) {
		t.Errorf("LoadedAt should be between %v and %v, got %v", before, after, loadedAt)
	}
}

const filteringTestBucket = "addressfilter-test"

func newFilteringTestConfig(endpoint, key string, maxFileSizeMB int) *Config {
	cfg := DefaultConfig
	cfg.S3 = s3syncer.Config{
		Config: s3client.Config{
			Region:    "us-east-1",
			AccessKey: "dummy-access-key",
			SecretKey: "dummy-secret-key",
			Endpoint:  endpoint,
		},
		Bucket:        filteringTestBucket,
		ObjectKey:     key,
		ChunkSizeMB:   s3syncer.DefaultS3Config.ChunkSizeMB,
		MaxRetries:    s3syncer.DefaultS3Config.MaxRetries,
		Concurrency:   s3syncer.DefaultS3Config.Concurrency,
		MaxFileSizeMB: maxFileSizeMB,
	}
	return &cfg
}

func TestFilterService_Initialize_RejectsOversizedFile(t *testing.T) {
	key := "oversized.json"
	body := bytes.Repeat([]byte("X"), 2*1024*1024) // 2 MB
	endpoint, _ := s3syncertest.NewFakeS3(t, filteringTestBucket, map[string][]byte{key: body})

	tooLargeBefore := fileTooLargeCounter.Snapshot().Count()
	syncFailureBefore := syncFailureCounter.Snapshot().Count()

	service, err := NewFilterService(newFilteringTestConfig(endpoint, key, 1))
	require.NoError(t, err)

	err = service.Initialize(context.Background())
	if err == nil {
		t.Fatal("expected Initialize to return error for oversized file")
	}
	if !errors.Is(err, s3syncer.ErrObjectTooLarge) {
		t.Errorf("expected error chain to include ErrObjectTooLarge, got %v", err)
	}

	tooLargeDelta := fileTooLargeCounter.Snapshot().Count() - tooLargeBefore
	syncFailureDelta := syncFailureCounter.Snapshot().Count() - syncFailureBefore
	if tooLargeDelta != 1 {
		t.Errorf("fileTooLargeCounter delta: got %d, want 1", tooLargeDelta)
	}
	if syncFailureDelta != 1 {
		t.Errorf("syncFailureCounter delta: got %d, want 1", syncFailureDelta)
	}
}

func TestFilterService_Initialize_GenericFailure(t *testing.T) {
	endpoint, _ := s3syncertest.NewFakeS3(t, filteringTestBucket, nil) // bucket exists, key does not

	tooLargeBefore := fileTooLargeCounter.Snapshot().Count()
	syncFailureBefore := syncFailureCounter.Snapshot().Count()

	service, err := NewFilterService(newFilteringTestConfig(endpoint, "missing.json", 1))
	require.NoError(t, err)

	err = service.Initialize(context.Background())
	if err == nil {
		t.Fatal("expected Initialize to return error for missing key")
	}
	if errors.Is(err, s3syncer.ErrObjectTooLarge) {
		t.Errorf("missing-key error should not match ErrObjectTooLarge: %v", err)
	}

	tooLargeDelta := fileTooLargeCounter.Snapshot().Count() - tooLargeBefore
	syncFailureDelta := syncFailureCounter.Snapshot().Count() - syncFailureBefore
	if tooLargeDelta != 0 {
		t.Errorf("fileTooLargeCounter delta: got %d, want 0", tooLargeDelta)
	}
	if syncFailureDelta != 1 {
		t.Errorf("syncFailureCounter delta: got %d, want 1", syncFailureDelta)
	}
}

func TestFilterService_KeepsListOnOversizedSync(t *testing.T) {
	salt := uuid.New()
	restrictedAddr := common.HexToAddress("0x1111111111111111111111111111111111111111")
	hash := HashStringInputWithPrefix(GetHashStringInputPrefix(salt), restrictedAddr)
	payload := map[string]interface{}{
		"id":             uuid.NewString(),
		"salt":           salt.String(),
		"hashing_scheme": "sha256-stringinput",
		"hashes":         []string{hex.EncodeToString(hash[:])},
	}
	initialBody, err := json.Marshal(payload)
	require.NoError(t, err)

	key := "filter.json"
	endpoint, backend := s3syncertest.NewFakeS3(t, filteringTestBucket, map[string][]byte{key: initialBody})

	// 1 MB limit; initial body is well under, the swap body will be 2 MB.
	service, err := NewFilterService(newFilteringTestConfig(endpoint, key, 1))
	require.NoError(t, err)

	require.NoError(t, service.Initialize(context.Background()))

	digestBefore := service.GetHashStoreDigest()
	countBefore := service.GetHashCount()
	require.NotEmpty(t, digestBefore, "initial digest should be set")
	require.Equal(t, 1, countBefore)
	restricted, _ := service.hashStore.IsRestricted(restrictedAddr)
	require.True(t, restricted, "address should be restricted after initial load")

	// Swap the S3 object for a payload that exceeds the configured limit.
	oversized := bytes.Repeat([]byte("X"), 2*1024*1024)
	_, err = backend.PutObject(filteringTestBucket, key, map[string]string{}, bytes.NewReader(oversized), int64(len(oversized)), &gofakes3.PutConditions{})
	require.NoError(t, err)

	// Drive one sync tick directly — same call the Start() poll loop makes.
	err = service.syncMgr.Syncer.CheckAndSync(context.Background())
	if !errors.Is(err, s3syncer.ErrObjectTooLarge) {
		t.Fatalf("expected ErrObjectTooLarge from oversized swap, got %v", err)
	}

	if got := service.GetHashStoreDigest(); got != digestBefore {
		t.Errorf("digest changed after failed sync: got %q, want %q", got, digestBefore)
	}
	if got := service.GetHashCount(); got != countBefore {
		t.Errorf("hash count changed after failed sync: got %d, want %d", got, countBefore)
	}
	if restricted, _ := service.hashStore.IsRestricted(restrictedAddr); !restricted {
		t.Error("address should still be restricted after failed sync")
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

		_, restricted := data.hashes[data.hashAddress(addr)]
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

		_, restricted := data.hashes[data.hashAddress(addr)]
		data.cache.Add(addr, restricted)
		if restricted {
			return true
		}
	}
	return false
}

func TestHashRawBytesInputVendorVector(t *testing.T) {
	salt, err := uuid.Parse("ce823987-8c5b-42c8-9d44-11df313b91e9")
	require.NoError(t, err)
	addr := common.HexToAddress("0xddfabcdc4d8ffc6d5beaf154f18b778f892a0740")
	expected := common.HexToHash("0xc148590f0f751bcd1cccdc5876433aaf8acf38a31483f426da0d43043b27f193")
	got := HashRawBytesInput(salt, addr)
	if got != expected {
		t.Fatalf("HashRawBytesInput mismatch: got %s, want %s", got.Hex(), expected.Hex())
	}
}

func TestHashStore_RawBytesScheme(t *testing.T) {
	store := NewHashStore(100)

	salt, err := uuid.Parse("ce823987-8c5b-42c8-9d44-11df313b91e9")
	require.NoError(t, err)
	addrRestricted := common.HexToAddress("0xddfabcdc4d8ffc6d5beaf154f18b778f892a0740")
	addrAllowed := common.HexToAddress("0x000000000000000000000000000000000000beef")
	hashRestricted := HashRawBytesInput(salt, addrRestricted)

	store.Store(uuid.New(), salt, HashingSchemeRawBytesInput, []common.Hash{hashRestricted}, "raw")

	if restricted, _ := store.IsRestricted(addrRestricted); !restricted {
		t.Fatal("restricted address should match under raw bytes scheme")
	}
	if restricted, _ := store.IsRestricted(addrAllowed); restricted {
		t.Fatal("allowed address must not match under raw bytes scheme")
	}

	// Same hash bytes reloaded under string scheme must not match: scheme drives the lookup function.
	store.Store(uuid.New(), salt, HashingSchemeStringInput, []common.Hash{hashRestricted}, "str")
	if restricted, _ := store.IsRestricted(addrRestricted); restricted {
		t.Fatal("raw-bytes hash should not match under string input scheme")
	}
}

// End-to-end: vendor JSON with sha256-rawbytesinput parses, loads into HashStore via Store,
// and IsRestricted matches the vendor-vector address.
func TestRawBytesScheme_ParseStoreLookup(t *testing.T) {
	salt, err := uuid.Parse("ce823987-8c5b-42c8-9d44-11df313b91e9")
	require.NoError(t, err)
	addr := common.HexToAddress("0xddfabcdc4d8ffc6d5beaf154f18b778f892a0740")
	hash := HashRawBytesInput(salt, addr)

	payload := map[string]any{
		"id":             uuid.NewString(),
		"salt":           salt.String(),
		"hashing_scheme": string(HashingSchemeRawBytesInput),
		"hashes":         []string{hex.EncodeToString(hash[:])},
	}
	raw, err := json.Marshal(payload)
	require.NoError(t, err)

	parsed, err := parseHashListJSON(raw)
	require.NoError(t, err)

	store := NewHashStore(8)
	store.Store(parsed.Id, parsed.Salt, parsed.Scheme, parsed.Hashes, "etag")

	if restricted, _ := store.IsRestricted(addr); !restricted {
		t.Fatal("vendor address must be restricted after parse+Store under raw bytes scheme")
	}
	otherAddr := common.HexToAddress("0x000000000000000000000000000000000000beef")
	if restricted, _ := store.IsRestricted(otherAddr); restricted {
		t.Fatal("non-listed address must not be restricted")
	}
}
