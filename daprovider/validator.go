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
// This interface is used to validate and generate proofs for DACertificate preimages
type Validator interface {
	// GenerateProof generates a proof for a specific preimage at a given offset
	// The proof format depends on the implementation and must be compatible with
	// the Solidity IDACertificateValidator contract
	// For DACertificate preimages, certificate contains the DA certificate
	// certHash is the keccak256 hash of the certificate
	GenerateProof(ctx context.Context, preimageType arbutil.PreimageType, certHash common.Hash, offset uint64, certificate []byte) ([]byte, error)

	// GenerateCertificateValidityProof generates a proof of certificate validity
	// Returns a proof that includes whether the certificate is valid according to the DA system rules
	GenerateCertificateValidityProof(ctx context.Context, preimageType arbutil.PreimageType, certificate []byte) ([]byte, error)
}
