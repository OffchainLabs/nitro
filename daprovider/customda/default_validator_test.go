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

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

func TestDefaultValidatorRecordPreimages(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	validator := DefaultValidator{}

	// Create a random batch
	batch := make([]byte, 1024)
	_, err := rand.Read(batch)
	testhelpers.RequireImpl(t, err, "Failed to generate random batch data")

	// Record preimages
	preimages, err := validator.RecordPreimages(ctx, batch)
	testhelpers.RequireImpl(t, err, "Failed to record preimages")

	// Verify we got at least one preimage
	if len(preimages) == 0 {
		testhelpers.FailImpl(t, "No preimages recorded")
	}

	// Verify at least one preimage is of CustomPreimageType
	foundCustom := false
	for _, p := range preimages {
		if p.PreimageType == arbutil.CustomDAPreimageType {
			foundCustom = true
			break
		}
	}

	if !foundCustom {
		testhelpers.FailImpl(t, "No CustomPreimageType preimages recorded")
	}
}

func TestDefaultValidatorGenerateProof(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	validator := DefaultValidator{}

	// Create a random batch
	batch := make([]byte, 1024)
	_, err := rand.Read(batch)
	testhelpers.RequireImpl(t, err, "Failed to generate random batch data")

	// Record preimages
	preimages, err := validator.RecordPreimages(ctx, batch)
	testhelpers.RequireImpl(t, err, "Failed to record preimages")

	// Find a CustomPreimageType preimage
	var customPreimage struct {
		hash    common.Hash
		content []byte
	}
	for _, p := range preimages {
		if p.PreimageType == arbutil.CustomDAPreimageType {
			customPreimage.hash = p.Hash
			customPreimage.content = p.Data
			break
		}
	}

	if customPreimage.hash == (common.Hash{}) {
		testhelpers.FailImpl(t, "No CustomPreimageType preimages found")
	}

	// Generate proof for the preimage
	proof, err := validator.GenerateProof(ctx, arbutil.CustomDAPreimageType, customPreimage.hash, 0)
	testhelpers.RequireImpl(t, err, "Failed to generate proof")

	// Verify proof content includes expected data
	if len(proof) == 0 {
		testhelpers.FailImpl(t, "Empty proof generated")
	}

	// Verify the proof matches the original content
	if !bytes.Equal(proof, customPreimage.content) {
		testhelpers.FailImpl(t, "Proof does not match expected data")
	}
}

func TestDefaultValidatorNonExistentPreimage(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	validator := DefaultValidator{}

	// Try to generate proof for a hash that doesn't exist
	nonExistentHash := common.HexToHash("0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef")
	_, err := validator.GenerateProof(ctx, arbutil.CustomDAPreimageType, nonExistentHash, 0)

	// Should return an error
	if err == nil {
		testhelpers.FailImpl(t, "Expected error when generating proof for non-existent hash")
	}
}
