// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package referenceda

import (
	"context"
	"crypto/sha256"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/daprovider"
)

// DefaultValidator is a reference implementation of the Validator interface
// that provides simple SHA-256 based hash verification for CustomDA type preimages
type DefaultValidator struct {
	storage *InMemoryStorage
}

// NewDefaultValidator creates a new DefaultValidator with the given storage
func NewDefaultValidator() *DefaultValidator {
	return &DefaultValidator{
		storage: GetInMemoryStorage(),
	}
}

// RecordPreimages extracts CustomDA preimages from a batch
// In this reference implementation, we hash the entire batch with SHA-256
// and record it as a CustomDA preimage
func (v *DefaultValidator) RecordPreimages(ctx context.Context, batch []byte) ([]daprovider.PreimageWithType, error) {
	if len(batch) == 0 {
		return nil, fmt.Errorf("empty batch data")
	}

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

func (v *DefaultValidator) GenerateProof(ctx context.Context, preimageType arbutil.PreimageType, hash common.Hash, offset uint64) ([]byte, error) {
	panic("not implemented yet")
}
