// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package customda

import (
	"bytes"
	"context"
	"crypto/rand"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

// TestFullCustomDAFlow tests the entire CustomDA flow from message creation to payload recovery
func TestFullCustomDAFlow(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create validator, writer, and reader
	validator := &DefaultValidator{}
	writer := NewWriter(validator)
	reader := NewReader(validator)

	// Create batch data with CustomDA header
	batchData := make([]byte, 4096)
	_, err := rand.Read(batchData)
	testhelpers.RequireImpl(t, err, "Failed to generate random batch data")

	// Original message with CustomDA header flag
	originalMessage := append([]byte{daprovider.CustomDAMessageHeaderFlag}, batchData...)

	// Store the message and get certificate
	certificate, err := writer.Store(ctx, originalMessage, uint64(time.Now().Add(time.Hour).Unix()), false) // #nosec G115 // #nosec G115
	testhelpers.RequireImpl(t, err, "Failed to store message")

	// Extract hash from certificate
	keccak256Hash := common.BytesToHash(certificate[1:33])
	expectedHash := crypto.Keccak256Hash(batchData)

	// Verify hash matches expected
	if keccak256Hash != expectedHash {
		testhelpers.FailImpl(t, "Certificate contains incorrect hash")
	}

	// Create preimages map for recovery
	preimagesMap := make(daprovider.PreimagesMap)

	// Recover payload from certificate
	recoveredPayload, newPreimages, err := reader.RecoverPayloadFromBatch(
		ctx,
		1,             // batchNum
		common.Hash{}, // batchBlockHash
		certificate,
		preimagesMap,
		true, // validateSeqMsg
	)
	testhelpers.RequireImpl(t, err, "Failed to recover payload")

	// Verify recovered payload matches original batch data
	if !bytes.Equal(recoveredPayload, batchData) {
		testhelpers.FailImpl(t, "Recovered payload doesn't match original batch data")
	}

	// Verify preimages were recorded
	customPreimages, exists := newPreimages[arbutil.CustomDAPreimageType]
	if !exists || len(customPreimages) == 0 {
		testhelpers.FailImpl(t, "No CustomPreimageType preimages recorded")
	}

	// Verify keccak256 hash preimage exists
	preimage, exists := customPreimages[keccak256Hash]
	if !exists {
		testhelpers.FailImpl(t, "Preimage for keccak256 hash not found")
	}

	// Verify preimage matches original batch data
	if !bytes.Equal(preimage, batchData) {
		testhelpers.FailImpl(t, "Preimage doesn't match original batch data")
	}

	// Generate and verify proof using the validator
	proof, err := validator.GenerateProof(ctx, arbutil.CustomDAPreimageType, keccak256Hash, 0)
	testhelpers.RequireImpl(t, err, "Failed to generate proof")

	// Proof should not be empty
	if len(proof) == 0 {
		testhelpers.FailImpl(t, "Generated proof is empty")
	}
}

// TestCustomDAMessageProcessing tests the processing of CustomDA messages in the complete flow
func TestCustomDAMessageProcessing(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create components
	validator := &DefaultValidator{}
	writer := NewWriter(validator)
	reader := NewReader(validator)

	// Test different message sizes
	testSizes := []int{
		100,   // Small message
		4096,  // Medium message
		65536, // Large message
	}

	for _, size := range testSizes {
		t.Run(testSizeName(size), func(t *testing.T) {
			// Create batch data
			batchData := make([]byte, size)
			_, err := rand.Read(batchData)
			testhelpers.RequireImpl(t, err, "Failed to generate random batch data")

			// Add CustomDA header
			message := append([]byte{daprovider.CustomDAMessageHeaderFlag}, batchData...)

			// Store and get certificate
			certificate, err := writer.Store(ctx, message, uint64(time.Now().Add(time.Hour).Unix()), false) // #nosec G115
			testhelpers.RequireImpl(t, err, "Failed to store message")

			// Create preimages map
			preimagesMap := make(daprovider.PreimagesMap)

			// Recover payload
			recoveredPayload, _, err := reader.RecoverPayloadFromBatch(
				ctx,
				1,             // batchNum
				common.Hash{}, // batchBlockHash
				certificate,
				preimagesMap,
				true, // validateSeqMsg
			)
			testhelpers.RequireImpl(t, err, "Failed to recover payload")

			// Verify recovered payload
			if !bytes.Equal(recoveredPayload, batchData) {
				testhelpers.FailImpl(t, "Recovered payload doesn't match original batch data")
			}
		})
	}
}

// Helper function to create test size name
func testSizeName(size int) string {
	switch {
	case size < 1024:
		return "small"
	case size < 10240:
		return "medium"
	default:
		return "large"
	}
}

// TestInvalidCustomDAMessages tests handling of invalid CustomDA messages
func TestInvalidCustomDAMessages(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create components
	validator := &DefaultValidator{}
	writer := NewWriter(validator)
	reader := NewReader(validator)

	// Test with empty message
	t.Run("EmptyMessage", func(t *testing.T) {
		// Empty message with just CustomDA header
		message := []byte{daprovider.CustomDAMessageHeaderFlag}

		// Store should still work but create minimal certificate
		certificate, err := writer.Store(ctx, message, uint64(time.Now().Add(time.Hour).Unix()), false) // #nosec G115
		testhelpers.RequireImpl(t, err, "Failed to store empty message")

		// Create preimages map
		preimagesMap := make(daprovider.PreimagesMap)

		// Recover payload (should be empty but valid)
		recoveredPayload, _, err := reader.RecoverPayloadFromBatch(
			ctx,
			1,             // batchNum
			common.Hash{}, // batchBlockHash
			certificate,
			preimagesMap,
			true, // validateSeqMsg
		)
		testhelpers.RequireImpl(t, err, "Failed to recover payload for empty message")

		// Verify recovered payload is empty
		if len(recoveredPayload) != 0 {
			testhelpers.FailImpl(t, "Recovered payload should be empty")
		}
	})

	// Test with non-CustomDA message
	t.Run("NonCustomDAMessage", func(t *testing.T) {
		// Message with BrotliMessageHeaderByte instead of CustomDA
		message := []byte{daprovider.BrotliMessageHeaderByte, 1, 2, 3, 4}

		// Store should return the message unchanged
		returnedMessage, err := writer.Store(ctx, message, uint64(time.Now().Add(time.Hour).Unix()), false) // #nosec G115
		testhelpers.RequireImpl(t, err, "Failed to process non-CustomDA message")

		// Verify message is unchanged
		if !bytes.Equal(returnedMessage, message) {
			testhelpers.FailImpl(t, "Non-CustomDA message should be returned unchanged")
		}

		// Create preimages map
		preimagesMap := make(daprovider.PreimagesMap)

		// Recover should fail because it's not a valid CustomDA certificate
		_, _, err = reader.RecoverPayloadFromBatch(
			ctx,
			1,             // batchNum
			common.Hash{}, // batchBlockHash
			returnedMessage,
			preimagesMap,
			true, // validateSeqMsg
		)

		// Should return an error
		if err == nil {
			testhelpers.FailImpl(t, "Expected error for invalid CustomDA message, but got none")
		}
	})
}
