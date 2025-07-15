// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package referenceda

import (
	"context"
	"encoding/binary"
	"fmt"

	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/arbutil"
)

type Validator struct {
	storage *InMemoryStorage
}

func NewValidator() *Validator {
	return &Validator{
		storage: GetInMemoryStorage(),
	}
}

// GenerateProof creates a proof for ReferenceDA
// The proof enhancer will prepend the standardized header [certKeccak256, offset, certSize, certificate]
// So we only need to return the custom data: [Version(1), PreimageSize(8), PreimageData]
func (v *Validator) GenerateProof(ctx context.Context, preimageType arbutil.PreimageType, certHash common.Hash, offset uint64, certificate []byte) ([]byte, error) {
	if preimageType != arbutil.CustomDAPreimageType {
		return nil, fmt.Errorf("unsupported preimage type: %v", preimageType)
	}

	// Extract SHA256 hash from certificate
	// Certificate format: [0x01 header byte][32 bytes SHA256]
	if len(certificate) != 33 || certificate[0] != 0x01 {
		return nil, fmt.Errorf("invalid certificate format, expected 33 bytes with 0x01 header")
	}

	// Extract data hash (SHA256) from certificate
	dataHash := common.BytesToHash(certificate[1:33])

	// Get preimage from storage using SHA256 hash
	preimage, err := v.storage.GetByHash(ctx, dataHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get preimage: %w", err)
	}
	if preimage == nil {
		return nil, fmt.Errorf("preimage not found for hash %x", dataHash)
	}

	// Build custom proof data: [Version(1), PreimageSize(8), PreimageData]
	// The certificate is NOT included here as it's already in the standardized header
	proof := make([]byte, 1+8+len(preimage))
	proof[0] = 1 // Version
	binary.BigEndian.PutUint64(proof[1:9], uint64(len(preimage)))
	copy(proof[9:], preimage)

	return proof, nil
}

// GenerateCertificateValidityProof implements Validator for ReferenceDA
func (v *Validator) GenerateCertificateValidityProof(ctx context.Context, preimageType arbutil.PreimageType, certificate []byte) ([]byte, error) {
	// ReferenceDA implementation returns:
	// - claimedValid (1 byte): 1 if valid, 0 if invalid
	// - version (1 byte): 0x01 for version 1
	//
	// This simple implementation only includes a version byte after the validity claim.
	// Other DA providers can return more complex validity proofs that include additional
	// verification data such as cryptographic signatures, merkle proofs, or other
	// authentication mechanisms. The OSP will pass this entire proof to validateCertificate.

	// Validate certificate format:
	// - Must be exactly 33 bytes (1 byte prefix + 32 bytes hash)
	// - First byte must be 0x01 (ReferenceDA marker)
	if len(certificate) != 33 {
		return []byte{0, 0x01}, nil // Invalid certificate, version 1
	}

	if certificate[0] != 0x01 {
		return []byte{0, 0x01}, nil // Invalid certificate, version 1
	}

	// Certificate is valid
	return []byte{1, 0x01}, nil // Valid certificate, version 1
}
