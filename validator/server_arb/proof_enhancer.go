package server_arb

import (
	"context"
	"fmt"

	"github.com/offchainlabs/nitro/arbutil"
)

const (
	// Enhancement flag in machine status byte
	ProofEnhancementFlag = 0x80

	// Marker bytes for different enhancement types
	MarkerCustomDARead     = 0xDA
	MarkerCustomDAValidate = 0xDB
)

// ProofEnhancer enhances one-step proofs with additional data
type ProofEnhancer interface {
	// EnhanceProof checks if enhancement is needed and applies it
	// Returns the enhanced proof or the original if no enhancement needed
	EnhanceProof(ctx context.Context, messageNum arbutil.MessageIndex, proof []byte) ([]byte, error)
}

// ProofEnhancementManager manages multiple proof enhancers by marker type
type ProofEnhancementManager struct {
	enhancers map[byte]ProofEnhancer
}

// NewProofEnhancementManager creates a new proof enhancement manager
func NewProofEnhancementManager() *ProofEnhancementManager {
	return &ProofEnhancementManager{
		enhancers: make(map[byte]ProofEnhancer),
	}
}

// RegisterEnhancer registers an enhancer for a specific marker byte
func (m *ProofEnhancementManager) RegisterEnhancer(marker byte, enhancer ProofEnhancer) {
	m.enhancers[marker] = enhancer
}

// EnhanceProof implements ProofEnhancer interface
func (m *ProofEnhancementManager) EnhanceProof(ctx context.Context, messageNum arbutil.MessageIndex, proof []byte) ([]byte, error) {
	if len(proof) == 0 {
		return proof, nil
	}

	// Check if enhancement flag is set
	if proof[0]&ProofEnhancementFlag == 0 {
		return proof, nil // No enhancement needed
	}

	// Find marker at end of proof
	if len(proof) < 1 { // Need at least the marker byte
		return nil, fmt.Errorf("proof too short for enhancement: %d bytes", len(proof))
	}

	marker := proof[len(proof)-1]
	enhancer, exists := m.enhancers[marker]
	if !exists {
		return nil, fmt.Errorf("unknown enhancement marker: 0x%02x", marker)
	}

	// Remove enhancement flag from machine status
	enhancedProof := make([]byte, len(proof))
	copy(enhancedProof, proof)
	enhancedProof[0] &= ^byte(ProofEnhancementFlag)

	// Let specific enhancer handle the proof
	return enhancer.EnhanceProof(ctx, messageNum, enhancedProof)
}
