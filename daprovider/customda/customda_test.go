// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package customda

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

// Helper functions for test assertions
func Require(t *testing.T, err error, printables ...interface{}) {
	t.Helper()
	testhelpers.RequireImpl(t, err, printables...)
}

func Fail(t *testing.T, printables ...interface{}) {
	t.Helper()
	testhelpers.FailImpl(t, printables...)
}

// MockValidator implements daprovider.Validator interface for testing
type MockValidator struct {
	preimages map[common.Hash][]byte
}

func NewMockValidator() *MockValidator {
	return &MockValidator{
		preimages: make(map[common.Hash][]byte),
	}
}

func (v *MockValidator) RecordPreimages(
	ctx context.Context,
	batch []byte,
) ([]daprovider.PreimageWithType, error) {
	// Create a simple preimage from the batch
	hash := crypto.Keccak256Hash(batch)
	v.preimages[hash] = batch

	// Return the preimage info
	return []daprovider.PreimageWithType{
		{
			PreimageType: arbutil.CustomDAPreimageType,
			Hash:         hash,
			Data:         batch,
		},
	}, nil
}

func (v *MockValidator) GenerateProof(
	ctx context.Context,
	preimageType arbutil.PreimageType,
	hash common.Hash,
	offset uint64,
) ([]byte, error) {
	// Simple proof is just returning the entire preimage
	if data, exists := v.preimages[hash]; exists {
		return data, nil
	}
	return nil, daprovider.ErrNoSuchPreimage
}

func TestCustomDAWriterStoreAndGetBatch(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	validator := NewMockValidator()
	writer := NewWriter(validator)

	// Create test batch data
	batchData := []byte("test batch data " + time.Now().String())

	// Prepend CustomDA header flag
	message := append([]byte{daprovider.CustomDAMessageHeaderFlag}, batchData...)

	// Store the batch
	certificate, err := writer.Store(ctx, message, uint64(time.Now().Add(time.Hour).Unix()), false) // #nosec G115
	Require(t, err, "Error storing batch")

	// Verify the certificate format
	if len(certificate) < 85 || certificate[0] != daprovider.CustomDAMessageHeaderFlag {
		Fail(t, "Invalid certificate format")
	}

	// Extract hash from certificate
	keccak256Hash := common.BytesToHash(certificate[1:33])

	// Retrieve the batch using GetBatch
	retrievedBatch, exists := writer.GetBatch(keccak256Hash)
	if !exists {
		Fail(t, "Batch not found")
	}

	// Verify the retrieved batch matches the original
	if !bytes.Equal(batchData, retrievedBatch) {
		Fail(t, "Retrieved batch doesn't match original")
	}
}

func TestCustomDAWriterWithNonCustomDAMessage(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	validator := NewMockValidator()
	writer := NewWriter(validator)

	// Create a message without CustomDA header flag
	message := []byte("test batch data " + time.Now().String())

	// Store the batch - should pass through unchanged
	returnedMessage, err := writer.Store(ctx, message, uint64(time.Now().Add(time.Hour).Unix()), false) // #nosec G115
	Require(t, err, "Error storing batch")

	// Verify the message is returned unchanged
	if !bytes.Equal(message, returnedMessage) {
		Fail(t, "Message should be returned unchanged")
	}
}

func TestCustomDAValidatorIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	validator := NewMockValidator()
	writer := NewWriter(validator)

	// Create test batch data with random content
	batchSize := 1024
	batchData := make([]byte, batchSize)
	_, err := rand.Read(batchData)
	Require(t, err, "Error generating random batch data")

	// Prepend CustomDA header flag
	message := append([]byte{daprovider.CustomDAMessageHeaderFlag}, batchData...)

	// Store the batch
	certificate, err := writer.Store(ctx, message, uint64(time.Now().Add(time.Hour).Unix()), false) // #nosec G115
	Require(t, err, "Error storing batch")

	// Extract hashes from certificate
	keccak256Hash := common.BytesToHash(certificate[1:33])
	sha256Hash := common.BytesToHash(certificate[33:65])

	// Verify keccak256 hash
	expectedKeccakHash := crypto.Keccak256Hash(batchData)
	if keccak256Hash != expectedKeccakHash {
		Fail(t, "Keccak256 hash in certificate doesn't match expected")
	}

	// Verify sha256 hash
	expectedSha256Hash := sha256.Sum256(batchData)
	if !bytes.Equal(sha256Hash.Bytes(), expectedSha256Hash[:]) {
		Fail(t, "SHA256 hash in certificate doesn't match expected")
	}

	// Check that the validator recorded the preimage
	preimages := validator.preimages
	if _, exists := preimages[keccak256Hash]; !exists {
		Fail(t, "Validator did not record the preimage")
	}

	// Generate proof for the preimage
	proof, err := validator.GenerateProof(ctx, arbutil.CustomDAPreimageType, keccak256Hash, 0)
	Require(t, err, "Error generating proof")

	// Verify the proof matches the original data
	if !bytes.Equal(proof, batchData) {
		Fail(t, "Generated proof doesn't match original data")
	}
}

func TestCustomDAWriterCertificateFormat(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	validator := NewMockValidator()
	writer := NewWriter(validator)

	// Create test batch data
	batchData := []byte("test batch data " + time.Now().String())

	// Prepend CustomDA header flag
	message := append([]byte{daprovider.CustomDAMessageHeaderFlag}, batchData...)

	// Store the batch
	certificate, err := writer.Store(ctx, message, uint64(time.Now().Add(time.Hour).Unix()), false) // #nosec G115
	Require(t, err, "Error storing batch")

	// Verify certificate format:
	// - 1 byte header (CustomDAMessageHeaderFlag)
	// - 32 bytes Keccak256 hash
	// - 32 bytes SHA256 hash
	// - 16 bytes nonce
	// - 4 bytes length of batch data in big-endian
	if len(certificate) != 85 {
		Fail(t, "Certificate has unexpected length")
	}

	if certificate[0] != daprovider.CustomDAMessageHeaderFlag {
		Fail(t, "Certificate header byte incorrect")
	}
}
