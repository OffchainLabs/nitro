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

// Validator defines the interface for custom data availability systems
// This interface is used to validate and generate proofs for CustomDA preimages
type Validator interface {
	// GenerateProof generates a proof for a specific preimage at a given offset
	// The proof format depends on the implementation and must be compatible with
	// the Solidity ICustomDAValidator contract
	// For CustomDA preimages, certificate contains the DA certificate (e.g., [0x01][32-byte hash])
	// certHash is the keccak256 hash of the certificate
	GenerateProof(ctx context.Context, preimageType arbutil.PreimageType, certHash common.Hash, offset uint64, certificate []byte) ([]byte, error)
}
