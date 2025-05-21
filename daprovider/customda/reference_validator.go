// Copyright 2021-2025, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package customda

import (
	"context"
	"crypto/sha256"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/daprovider"
)

// DefaultValidator is a basic implementation of the Validator interface
// that provides simple SHA-256 based hash verification for CustomDA type preimages
type DefaultValidator struct {
	storage PreimageStorage
}

// PreimageStorage defines an interface for storing and retrieving preimages
// This is a simplification of the Storage interface to avoid dependencies
type PreimageStorage interface {
	Store(ctx context.Context, data []byte) error
	GetByHash(ctx context.Context, hash common.Hash) ([]byte, error)
}

// NewDefaultValidator creates a new DefaultValidator with the given storage
func NewDefaultValidator(storage PreimageStorage) *DefaultValidator {
	return &DefaultValidator{
		storage: storage,
	}
}

// RecordPreimages extracts CustomDA preimages from a batch
// In this default implementation, we hash the entire batch with SHA-256
// and record it as a CustomDA preimage
func (v *DefaultValidator) RecordPreimages(ctx context.Context, batch []byte) ([]daprovider.PreimageWithType, error) {
	if len(batch) == 0 {
		return nil, fmt.Errorf("empty batch data")
	}

	// Store the batch data
	err := v.storage.Store(ctx, batch)
	if err != nil {
		return nil, fmt.Errorf("failed to store batch data: %w", err)
	}

	// Hash the batch data with SHA-256
	hashBytes := sha256.Sum256(batch)
	hash := common.BytesToHash(hashBytes[:])

	log.Debug("DefaultCustomDAValidator: Recording CustomDA preimage",
		"hash", hash.Hex(),
		"dataSize", len(batch))

	// Return a single CustomDA preimage
	return []daprovider.PreimageWithType{
		{
			Hash:         hash,
			Data:         batch,
			PreimageType: arbutil.CustomDAPreimageType,
		},
	}, nil
}

// Storage returns the underlying storage implementation
// This method is mainly for testing and debugging
func (v *DefaultValidator) Storage() PreimageStorage {
	return v.storage
}

// GenerateProof generates a proof for a specific preimage
// In this default implementation, we use a simple proof format:
// [proof_type (1 byte) | preimage_data (variable length)]
// where proof_type is 0 for a simple hash-based proof
func (v *DefaultValidator) GenerateProof(ctx context.Context, preimageType arbutil.PreimageType, hash common.Hash, offset uint64) ([]byte, error) {
	if preimageType != arbutil.CustomDAPreimageType {
		return nil, fmt.Errorf("unsupported preimage type: %d", preimageType)
	}

	// Retrieve the preimage data
	data, err := v.storage.GetByHash(ctx, hash)
	if err != nil {
		return nil, fmt.Errorf("failed to get preimage data for hash %s: %w", hash.Hex(), err)
	}

	// For the default implementation, the proof is simply:
	// - A type byte (0 for simple hash proof)
	// - Followed by the entire preimage data
	proofType := byte(0) // Simple hash-based proof
	proof := append([]byte{proofType}, data...)

	log.Debug("DefaultCustomDAValidator: Generated proof",
		"hash", hash.Hex(),
		"offset", offset,
		"proofSize", len(proof))

	return proof, nil
}
