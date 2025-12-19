// Copyright 2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package headerreader

import (
	"encoding/json"
	"math/rand"
	"os"
	"path"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"

	"github.com/offchainlabs/nitro/util/blobs"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

// createTestBlobs creates test blobs and their versioned hashes
func createTestBlobs(count int) ([]kzg4844.Blob, []common.Hash, error) {
	testData := make([]byte, count*blobs.BlobEncodableData)
	r := rand.New(rand.NewSource(1))
	_, err := r.Read(testData)
	if err != nil {
		return nil, nil, err
	}

	testBlobs, err := blobs.EncodeBlobs(testData)
	if err != nil {
		return nil, nil, err
	}

	_, versionedHashes, err := blobs.ComputeCommitmentsAndHashes(testBlobs)
	if err != nil {
		return nil, nil, err
	}

	return testBlobs, versionedHashes, nil
}

func TestSaveBlobsV1ToDisk(t *testing.T) {
	testBlobs, versionedHashes, err := createTestBlobs(3)
	Require(t, err)

	testDir := t.TempDir()
	slot := uint64(12345)

	err = saveBlobsV1ToDisk(testBlobs, versionedHashes, slot, testDir)
	Require(t, err)

	filePath := path.Join(testDir, "12345")
	data, err := os.ReadFile(filePath)
	Require(t, err)

	var storage blobStorageV1
	err = json.Unmarshal(data, &storage)
	Require(t, err)

	if storage.Version != 1 {
		t.Errorf("expected version 1, got %d", storage.Version)
	}

	if len(storage.Data) != len(testBlobs) {
		t.Errorf("expected %d blobs, got %d", len(testBlobs), len(storage.Data))
	}

	for i, hash := range versionedHashes {
		blobData, found := storage.Data[hash.Hex()]
		if !found {
			t.Errorf("blob not found for hash %s", hash.Hex())
			continue
		}
		if !reflect.DeepEqual([]byte(blobData), testBlobs[i][:]) {
			t.Errorf("blob data mismatch for index %d", i)
		}
	}
}

func TestReadBlobsV1FromDisk(t *testing.T) {
	testBlobs, versionedHashes, err := createTestBlobs(3)
	Require(t, err)

	testDir := t.TempDir()
	slot := uint64(12345)

	err = saveBlobsV1ToDisk(testBlobs, versionedHashes, slot, testDir)
	Require(t, err)

	readBlobs, err := ReadBlobsFromDisk(testDir, slot, versionedHashes)
	Require(t, err)

	if len(readBlobs) != len(testBlobs) {
		t.Fatalf("expected %d blobs, got %d", len(testBlobs), len(readBlobs))
	}

	for i := range testBlobs {
		if readBlobs[i] != testBlobs[i] {
			t.Errorf("blob mismatch at index %d", i)
		}
	}
}

func TestReadBlobsV1OrderPreservation(t *testing.T) {
	testBlobs, versionedHashes, err := createTestBlobs(3)
	Require(t, err)

	testDir := t.TempDir()
	slot := uint64(12345)

	err = saveBlobsV1ToDisk(testBlobs, versionedHashes, slot, testDir)
	Require(t, err)

	// Read in reverse order
	reversedHashes := slices.Clone(versionedHashes)
	slices.Reverse(reversedHashes)

	readBlobs, err := ReadBlobsFromDisk(testDir, slot, reversedHashes)
	Require(t, err)

	// Verify blobs are in reversed order
	for i := range testBlobs {
		expectedIdx := len(testBlobs) - 1 - i
		if readBlobs[i] != testBlobs[expectedIdx] {
			t.Errorf("blob at index %d does not match expected blob at index %d", i, expectedIdx)
		}
	}
}

func TestReadBlobsV0FromDisk(t *testing.T) {
	testDir := t.TempDir()
	slot := uint64(5)

	testBlobs, _, err := createTestBlobs(2)
	Require(t, err)

	commitment1, err := kzg4844.BlobToCommitment(&testBlobs[0])
	Require(t, err)
	commitment2, err := kzg4844.BlobToCommitment(&testBlobs[1])
	Require(t, err)

	hash1 := blobs.CommitmentToVersionedHash(commitment1)
	hash2 := blobs.CommitmentToVersionedHash(commitment2)
	versionedHashes := []common.Hash{hash1, hash2}

	// Create V0 format response with real blobs and commitments
	response := []blobResponseItem{{
		BlockRoot:       "a",
		Index:           0,
		Slot:            5,
		BlockParentRoot: "a0",
		ProposerIndex:   9,
		Blob:            testBlobs[0][:],
		KzgCommitment:   commitment1[:],
		KzgProof:        []byte{1},
	}, {
		BlockRoot:       "a",
		Index:           1,
		Slot:            5,
		BlockParentRoot: "a0",
		ProposerIndex:   10,
		Blob:            testBlobs[1][:],
		KzgCommitment:   commitment2[:],
		KzgProof:        []byte{2},
	}}

	rawData, err := json.Marshal(response)
	Require(t, err)
	err = saveBlobsV0ToDisk(rawData, slot, testDir)
	Require(t, err)

	// Read old format blobs using new reader
	readBlobs, err := ReadBlobsFromDisk(testDir, slot, versionedHashes)
	Require(t, err)

	if len(readBlobs) != 2 {
		t.Fatalf("expected 2 blobs, got %d", len(readBlobs))
	}

	if readBlobs[0] != testBlobs[0] {
		t.Errorf("blob 0 mismatch")
	}
	if readBlobs[1] != testBlobs[1] {
		t.Errorf("blob 1 mismatch")
	}
}

func TestDetectBlobFileFormat(t *testing.T) {
	// Test version 1 format
	v1Data := blobStorageV1{
		Version: 1,
		Data:    make(map[string]hexutil.Bytes),
	}
	v1JSON, err := json.Marshal(v1Data)
	Require(t, err)

	version, err := detectBlobFileFormat(v1JSON)
	Require(t, err)
	if version != 1 {
		t.Errorf("expected version 1, got %d", version)
	}

	// Test version 0 format (no version field)
	v0Data := fullResult[[]blobResponseItem]{
		Data: []blobResponseItem{},
	}
	v0JSON, err := json.Marshal(v0Data)
	Require(t, err)

	version, err = detectBlobFileFormat(v0JSON)
	Require(t, err)
	if version != 0 {
		t.Errorf("expected version 0, got %d", version)
	}
}

func TestReadBlobsFromDiskErrors(t *testing.T) {
	testDir := t.TempDir()

	// Test missing file
	_, err := ReadBlobsFromDisk(testDir, 99999, []common.Hash{common.HexToHash("0x123")})
	if err == nil {
		t.Error("expected error for missing file")
	}

	// Test empty versioned hashes
	_, err = ReadBlobsFromDisk(testDir, 12345, []common.Hash{})
	if err == nil {
		t.Error("expected error for empty versioned hashes")
	}

	// Test missing blob in V1 format
	testBlobs, versionedHashes, err := createTestBlobs(2)
	Require(t, err)

	slot := uint64(12345)
	err = saveBlobsV1ToDisk(testBlobs, versionedHashes, slot, testDir)
	Require(t, err)

	// Try to read with a hash that doesn't exist
	wrongHash := common.HexToHash("0xdeadbeef")
	_, err = ReadBlobsFromDisk(testDir, slot, []common.Hash{wrongHash})
	if err == nil {
		t.Error("expected error for missing blob")
	}
}

func TestReadBlobsV1ValidationFailure(t *testing.T) {
	testDir := t.TempDir()
	slot := uint64(12345)

	// Create two valid blobs
	testBlobs, versionedHashes, err := createTestBlobs(2)
	Require(t, err)

	// Save them normally
	err = saveBlobsV1ToDisk(testBlobs, versionedHashes, slot, testDir)
	Require(t, err)

	// Manually corrupt the file by swapping blob data between two hashes
	filePath := path.Join(testDir, "12345")
	data, err := os.ReadFile(filePath)
	Require(t, err)

	var storage blobStorageV1
	err = json.Unmarshal(data, &storage)
	Require(t, err)

	// Swap the blob data for the two hashes (so hash doesn't match blob)
	hash0Str := versionedHashes[0].Hex()
	hash1Str := versionedHashes[1].Hex()
	storage.Data[hash0Str], storage.Data[hash1Str] = storage.Data[hash1Str], storage.Data[hash0Str]

	// Write corrupted data back
	corruptedData, err := json.Marshal(storage)
	Require(t, err)
	err = os.WriteFile(filePath, corruptedData, 0600)
	Require(t, err)

	// Try to read - should fail validation
	_, err = ReadBlobsFromDisk(testDir, slot, versionedHashes)
	if err == nil {
		t.Error("expected validation error for corrupted blob")
	}
	if err != nil && !strings.Contains(err.Error(), "blob validation failed") {
		t.Errorf("expected validation error, got: %v", err)
	}
}

func TestReadBlobsV0ValidationFailure(t *testing.T) {
	testDir := t.TempDir()
	slot := uint64(5)

	// Create two valid blobs
	testBlobs, _, err := createTestBlobs(2)
	Require(t, err)

	// Compute commitments
	commitment1, err := kzg4844.BlobToCommitment(&testBlobs[0])
	Require(t, err)
	commitment2, err := kzg4844.BlobToCommitment(&testBlobs[1])
	Require(t, err)

	// Create V0 format but with swapped blobs (blob1 with commitment2 and vice versa)
	response := []blobResponseItem{{
		BlockRoot:       "a",
		Index:           0,
		Slot:            5,
		BlockParentRoot: "a0",
		ProposerIndex:   9,
		Blob:            testBlobs[1][:], // Wrong blob for this commitment
		KzgCommitment:   commitment1[:],
		KzgProof:        []byte{1},
	}, {
		BlockRoot:       "a",
		Index:           1,
		Slot:            5,
		BlockParentRoot: "a0",
		ProposerIndex:   10,
		Blob:            testBlobs[0][:], // Wrong blob for this commitment
		KzgCommitment:   commitment2[:],
		KzgProof:        []byte{2},
	}}

	// Save corrupted data
	rawData, err := json.Marshal(response)
	Require(t, err)
	err = saveBlobsV0ToDisk(rawData, slot, testDir)
	Require(t, err)

	// Compute expected versioned hashes
	hash1 := blobs.CommitmentToVersionedHash(commitment1)
	hash2 := blobs.CommitmentToVersionedHash(commitment2)
	versionedHashes := []common.Hash{hash1, hash2}

	// Try to read - should fail validation
	_, err = ReadBlobsFromDisk(testDir, slot, versionedHashes)
	if err == nil {
		t.Error("expected validation error for corrupted blob")
	}
	if err != nil && !strings.Contains(err.Error(), "blob validation failed") {
		t.Errorf("expected validation error, got: %v", err)
	}
}

func Require(t *testing.T, err error, printables ...interface{}) {
	t.Helper()
	testhelpers.RequireImpl(t, err, printables...)
}

func Fail(t *testing.T, printables ...interface{}) {
	t.Helper()
	testhelpers.FailImpl(t, printables...)
}
