// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
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
	MarkerCustomDAReadPreimage        = 0xDA
	MarkerCustomDAValidateCertificate = 0xDB

	// SequencerMessageHeaderSize is the size of the sequencer message header
	// (MinTimestamp + MaxTimestamp + MinL1Block + MaxL1Block + AfterDelayedMessages = 8+8+8+8+8)
	SequencerMessageHeaderSize = 40

	// Sizes for proof enhancement marker data
	CertificateHashSize      = 32 // Size of keccak256 hash of the certificate
	OffsetSize               = 8  // Size of uint64 offset
	MarkerSize               = 1  // Size of marker byte
	CertificateSizeFieldSize = 8  // Size of uint64 certificate size field

	// MinCertificateSize is the minimum size of a certificate (just the header byte).
	// Real certificates will have more data, but the proof enhancer system doesn't
	// put any further constraints on certificate structure.
	MinCertificateSize = 1
)

// ProofMarker identifies the type of proof enhancement needed
type ProofMarker byte

// ProofEnhancer enhances one-step proofs with additional data.
// For proving certain opcodes, like for CustomDA, Arbitrator doesn't have enough information
// to generate the full proofs. In the case of CustomDA, daprovider implementations usually
// need network access to talk to external DA systems to get full proof details. To indicate
// that a proof needs enhancement, Arbitrator sets the ProofEnhancementFlag on the machine
// status byte of the proof that it returns, and also appends one of the Marker bytes to
// indicate which ProofEnhancer is required.
type ProofEnhancer interface {
	// EnhanceProof checks if enhancement is needed and applies it
	// Returns the enhanced proof or the original if no enhancement needed
	EnhanceProof(ctx context.Context, messageNum arbutil.MessageIndex, proof []byte) ([]byte, error)
}

// ProofEnhancementManager allows registration of ProofEnhancers and provides forwarding of EnhanceProof
// requests to the appropriate ProofEnhancer.
type ProofEnhancementManager struct {
	enhancers map[ProofMarker]ProofEnhancer
}

// NewProofEnhancementManager creates a new proof enhancement manager
func NewProofEnhancementManager() *ProofEnhancementManager {
	return &ProofEnhancementManager{
		enhancers: make(map[ProofMarker]ProofEnhancer),
	}
}

// RegisterEnhancer registers an enhancer for a specific marker byte
func (m *ProofEnhancementManager) RegisterEnhancer(marker ProofMarker, enhancer ProofEnhancer) {
	m.enhancers[marker] = enhancer
}

// EnhanceProof implements ProofEnhancer interface to forward EnhanceProof requests
// to the appropriate registered ProofEnhancer implementation, if there is any
// and proof enhancement was requested.
func (m *ProofEnhancementManager) EnhanceProof(ctx context.Context, messageNum arbutil.MessageIndex, proof []byte) ([]byte, error) {
	if len(proof) == 0 {
		return proof, nil
	}

	// Check if enhancement flag is set
	if proof[0]&ProofEnhancementFlag == 0 {
		return proof, nil // No enhancement needed
	}

	// Find marker at end of proof
	if len(proof) < 2 { // Need at least the marker byte after the enhancement flag
		return nil, fmt.Errorf("proof too short for enhancement: %d bytes", len(proof))
	}
	marker := ProofMarker(proof[len(proof)-1])
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
