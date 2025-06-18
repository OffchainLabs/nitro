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
// Format: [Version(1), PreimageSize(8), PreimageData(variable)]
func (v *Validator) GenerateProof(ctx context.Context, preimageType arbutil.PreimageType, hash common.Hash, offset uint64) ([]byte, error) {
	if preimageType != arbutil.CustomDAPreimageType {
		return nil, fmt.Errorf("unsupported preimage type: %v", preimageType)
	}

	// Convert common.Hash to [32]byte for storage lookup
	var hash32 [32]byte
	copy(hash32[:], hash[:])

	// Get preimage from storage
	preimage, err := v.storage.GetByHash(ctx, hash)
	if err != nil {
		return nil, fmt.Errorf("failed to get preimage: %w", err)
	}
	if preimage == nil {
		return nil, fmt.Errorf("preimage not found for hash %x", hash)
	}

	// Build proof: [Version(1), PreimageSize(8), PreimageData]
	proof := make([]byte, 1+8+len(preimage))
	proof[0] = 1 // Version
	binary.BigEndian.PutUint64(proof[1:9], uint64(len(preimage)))
	copy(proof[9:], preimage)

	return proof, nil
}
