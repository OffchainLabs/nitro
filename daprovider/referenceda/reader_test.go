// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package referenceda

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

// TestCustomDAReaderRecoverPayload tests the CustomDA reader's ability to recover payload from a batch
func TestReferenceDAReaderRecoverPayload(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create validator and writer
	validator := NewMockValidator()
	writer := NewWriter(validator)
	reader := NewReader(validator)

	// Create test batch data
	batchData := make([]byte, 2048)
	_, err := rand.Read(batchData)
	testhelpers.RequireImpl(t, err, "Failed to generate random batch data")

	// Prepend CustomDA header flag
	message := append([]byte{daprovider.CustomDAMessageHeaderFlag}, batchData...)

	// Store the batch via the writer
	certificate, err := writer.Store(ctx, message, uint64(time.Now().Add(time.Hour).Unix()), false) // #nosec G115
	testhelpers.RequireImpl(t, err, "Failed to store batch")

	// Create a preimages map
	preimagesMap := make(daprovider.PreimagesMap)

	// Recover payload from the batch using the reader
	payload, newPreimages, err := reader.RecoverPayloadFromBatch(
		ctx,
		123,           // batchNum (arbitrary for test)
		common.Hash{}, // batchBlockHash (arbitrary for test)
		certificate,
		preimagesMap,
		true, // validateSeqMsg
	)
	testhelpers.RequireImpl(t, err, "Failed to recover payload from batch")

	// Verify the recovered payload matches the original batch data
	if !bytes.Equal(payload, batchData) {
		testhelpers.FailImpl(t, "Recovered payload doesn't match original batch data")
	}

	// Verify preimages were added to the map
	if len(newPreimages) == 0 {
		testhelpers.FailImpl(t, "No preimages recovered")
	}

	// Check if custom preimage type is in the map
	if customPreimages, exists := newPreimages[arbutil.CustomDAPreimageType]; !exists {
		testhelpers.FailImpl(t, "No CustomPreimageType preimages in the map")
	} else if len(customPreimages) == 0 {
		testhelpers.FailImpl(t, "Empty CustomPreimageType preimages map")
	}
}

// TestCustomDAReaderInvalidCertificate tests how the reader handles invalid certificates
func TestReferenceDAReaderInvalidCertificate(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	validator := NewMockValidator()
	reader := NewReader(validator)

	// Create a preimages map
	preimagesMap := make(daprovider.PreimagesMap)

	// Test cases with invalid certificates
	testCases := []struct {
		name        string
		certificate []byte
	}{
		{
			"Empty certificate",
			[]byte{},
		},
		{
			"Too short certificate",
			[]byte{daprovider.CustomDAMessageHeaderFlag, 0x01, 0x02, 0x03},
		},
		{
			"Wrong header byte",
			[]byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := reader.RecoverPayloadFromBatch(
				ctx,
				123,           // batchNum
				common.Hash{}, // batchBlockHash
				tc.certificate,
				preimagesMap,
				true, // validateSeqMsg
			)

			// Should return an error for invalid certificates
			if err == nil {
				testhelpers.FailImpl(t, "Expected error for invalid certificate, but got none")
			}
		})
	}
}

// TestCustomDAReaderWithMalformedCertificate tests how the reader handles malformed but valid-length certificates
func TestReferenceDAReaderWithMalformedCertificate(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	validator := NewMockValidator()
	reader := NewReader(validator)

	// Create valid-looking certificate with wrong hash
	certificate := make([]byte, 85) // Correct length
	certificate[0] = daprovider.CustomDAMessageHeaderFlag

	// Set a hash that doesn't exist in our validator
	nonExistentHash := common.HexToHash("0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef")
	copy(certificate[1:33], nonExistentHash.Bytes())

	// Set some dummy SHA256 hash
	dummySha256 := sha256.Sum256([]byte("dummy data"))
	copy(certificate[33:65], dummySha256[:])

	// Set some dummy nonce
	for i := 65; i < 81; i++ {
		certificate[i] = byte(i)
	}

	// Set length
	binary.BigEndian.PutUint32(certificate[81:85], 100)

	// Create a preimages map
	preimagesMap := make(daprovider.PreimagesMap)

	// Recover should fail since the hash doesn't exist
	_, _, err := reader.RecoverPayloadFromBatch(
		ctx,
		123,           // batchNum
		common.Hash{}, // batchBlockHash
		certificate,
		preimagesMap,
		true, // validateSeqMsg
	)

	// Should return an error
	if err == nil {
		testhelpers.FailImpl(t, "Expected error for certificate with non-existent hash, but got none")
	}
}

// TestCustomDAReaderEndToEnd tests a complete flow from storing to recovering
func TestReferenceDAReaderEndToEnd(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create validator, writer, and reader
	validator := NewMockValidator()
	writer := NewWriter(validator)
	reader := NewReader(validator)

	// Create test batch data
	originalData := []byte("This is a test batch with some custom data " + time.Now().String())

	// Prepend CustomDA header flag
	message := append([]byte{daprovider.CustomDAMessageHeaderFlag}, originalData...)

	// Store the batch
	certificate, err := writer.Store(ctx, message, uint64(time.Now().Add(time.Hour).Unix()), false) // #nosec G115
	testhelpers.RequireImpl(t, err, "Failed to store batch")

	// Extract the keccak256 hash from the certificate
	keccak256Hash := common.BytesToHash(certificate[1:33])

	// Verify it matches the expected hash
	expectedHash := crypto.Keccak256Hash(originalData)
	if keccak256Hash != expectedHash {
		testhelpers.FailImpl(t, "Certificate contains incorrect keccak256 hash")
	}

	// Create a preimages map
	preimagesMap := make(daprovider.PreimagesMap)

	// Recover the payload
	recoveredPayload, newPreimages, err := reader.RecoverPayloadFromBatch(
		ctx,
		123,           // batchNum
		common.Hash{}, // batchBlockHash
		certificate,
		preimagesMap,
		true, // validateSeqMsg
	)
	testhelpers.RequireImpl(t, err, "Failed to recover payload")

	// Verify the recovered payload matches the original data
	if !bytes.Equal(recoveredPayload, originalData) {
		testhelpers.FailImpl(t, "Recovered payload doesn't match original data")
	}

	// Verify the preimage was added to the map
	customPreimages, exists := newPreimages[arbutil.CustomDAPreimageType]
	if !exists {
		testhelpers.FailImpl(t, "No CustomPreimageType preimages in the map")
	}

	// Verify the preimage hash matches
	preimage, exists := customPreimages[keccak256Hash]
	if !exists {
		testhelpers.FailImpl(t, "Preimage with expected hash not found in the map")
	}

	// Verify preimage content matches
	if !bytes.Equal(preimage, originalData) {
		testhelpers.FailImpl(t, "Preimage content doesn't match original data")
	}
}
