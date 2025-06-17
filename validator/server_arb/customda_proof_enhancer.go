package server_arb

import (
	"context"
	"encoding/binary"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/daprovider"
)

// CustomDAProofEnhancer enhances proofs that involve CustomDA preimage operations
type CustomDAProofEnhancer struct {
	daValidator daprovider.Validator
}

// NewCustomDAProofEnhancer creates a new CustomDA proof enhancer
func NewCustomDAProofEnhancer(validator daprovider.Validator) *CustomDAProofEnhancer {
	return &CustomDAProofEnhancer{
		daValidator: validator,
	}
}

// EnhanceProof implements ProofEnhancer for CustomDA
func (e *CustomDAProofEnhancer) EnhanceProof(ctx context.Context, proof []byte) ([]byte, error) {
	// Extract hash and offset from end of proof
	// Format: [...proof..., hash(32), offset(8), marker(1)]
	if len(proof) < 41 {
		return nil, fmt.Errorf("proof too short for CustomDA enhancement: %d bytes", len(proof))
	}

	markerPos := len(proof) - 1
	offsetPos := markerPos - 8
	hashPos := offsetPos - 32

	// Verify marker
	if proof[markerPos] != MarkerCustomDARead {
		return nil, fmt.Errorf("invalid marker for CustomDA enhancer: 0x%02x", proof[markerPos])
	}

	// Extract hash and offset
	var hash [32]byte
	copy(hash[:], proof[hashPos:offsetPos])
	offset := binary.BigEndian.Uint64(proof[offsetPos:markerPos])

	// Generate custom proof
	customProof, err := e.daValidator.GenerateProof(ctx, arbutil.CustomDAPreimageType, common.BytesToHash(hash[:]), offset)
	if err != nil {
		return nil, fmt.Errorf("failed to generate custom DA proof: %w", err)
	}

	// Build enhanced proof: original proof up to hash position, then hash, offset, and custom proof
	enhancedProof := make([]byte, hashPos+32+8+len(customProof))
	copy(enhancedProof, proof[:hashPos])
	copy(enhancedProof[hashPos:], hash[:])
	binary.BigEndian.PutUint64(enhancedProof[hashPos+32:], offset)
	copy(enhancedProof[hashPos+40:], customProof)

	return enhancedProof, nil
}
