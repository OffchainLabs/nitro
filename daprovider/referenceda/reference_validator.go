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
// Format: [Version(1), CertificateSize(8), Certificate, PreimageSize(8), PreimageData]
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

	// Build proof: [Version(1), CertificateSize(8), Certificate, PreimageSize(8), PreimageData]
	certLen := len(certificate)
	proof := make([]byte, 1+8+certLen+8+len(preimage))
	proof[0] = 1 // Version
	binary.BigEndian.PutUint64(proof[1:9], uint64(certLen))
	copy(proof[9:9+certLen], certificate)
	binary.BigEndian.PutUint64(proof[9+certLen:9+certLen+8], uint64(len(preimage)))
	copy(proof[9+certLen+8:], preimage)

	return proof, nil
}
