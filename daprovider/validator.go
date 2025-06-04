// Copyright 2024-2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package daprovider

import (
	"context"
	"errors"

	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/arbutil"
)

// Common errors
var (
	ErrNoSuchPreimage = errors.New("no such preimage")
)

// PreimageWithType represents a preimage along with its type
type PreimageWithType struct {
	Hash         common.Hash
	Data         []byte
	PreimageType arbutil.PreimageType
}

// Validator defines the interface for custom data availability systems
// This interface is used to validate and generate proofs for CustomDA preimages
type Validator interface {
	// RecordPreimages extracts and records preimages from a batch
	// Returns a list of preimages with their types that were extracted
	RecordPreimages(ctx context.Context, batch []byte) ([]PreimageWithType, error)

	// GenerateProof generates a proof for a specific preimage at a given offset
	// The proof format depends on the implementation and must be compatible with
	// the Solidity ICustomDAValidator contract
	GenerateProof(ctx context.Context, preimageType arbutil.PreimageType, hash common.Hash, offset uint64) ([]byte, error)
}
