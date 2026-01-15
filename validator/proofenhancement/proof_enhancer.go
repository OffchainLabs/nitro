// Copyright 20255-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package proofenhancement

import (
	"context"
	"encoding/binary"
	"fmt"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/staker"
)

const (
	// Enhancement flag in machine status byte (first byte in proof)
	ProofEnhancementFlag = 0x80

	// Marker bytes for different enhancement types (last byte in an un-enhanced proof)
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

// NewCustomDAProofEnhancer creates a ProofEnhancementManager pre-configured with both
// CustomDA proof enhancers (ReadPreimage and ValidateCertificate). This is the recommended
// constructor for production use with CustomDA systems.
//
// For testing or custom configurations, use NewProofEnhancementManager and RegisterEnhancer directly.
func NewCustomDAProofEnhancer(
	dapRegistry *daprovider.DAProviderRegistry,
	inboxTracker staker.InboxTrackerInterface,
	inboxReader staker.InboxReaderInterface,
) *ProofEnhancementManager {
	manager := NewProofEnhancementManager()

	// Register both CustomDA enhancers
	manager.RegisterEnhancer(
		MarkerCustomDAReadPreimage,
		NewReadPreimageProofEnhancer(dapRegistry, inboxTracker, inboxReader),
	)
	manager.RegisterEnhancer(
		MarkerCustomDAValidateCertificate,
		NewValidateCertificateProofEnhancer(dapRegistry, inboxTracker, inboxReader),
	)

	return manager
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

func retrieveCertificateFromInboxMessage(
	ctx context.Context,
	messageNum arbutil.MessageIndex,
	tracker staker.InboxTrackerInterface,
	reader staker.InboxReaderInterface,
) ([]byte, error) {
	batchContainingMessage, found, err := tracker.FindInboxBatchContainingMessage(messageNum)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("couldn't find batch for message #%d to enhance proof", messageNum)
	}

	sequencerMessage, _, err := reader.GetSequencerMessageBytes(ctx, batchContainingMessage)
	if err != nil {
		return nil, fmt.Errorf("failed to get sequencer message for batch %d: %w", batchContainingMessage, err)
	}

	// Extract and validate certificate from sequencer message
	if len(sequencerMessage) < SequencerMessageHeaderSize+1 {
		return nil, fmt.Errorf("sequencer message too short: expected at least %d bytes, got %d", SequencerMessageHeaderSize+1, len(sequencerMessage))
	}

	// Extract certificate (skip sequencer message header)
	certificate := sequencerMessage[SequencerMessageHeaderSize:]

	// Validate certificate format
	if len(certificate) < MinCertificateSize {
		return nil, fmt.Errorf("certificate too short: expected at least %d bytes, got %d", MinCertificateSize, len(certificate))
	}

	if certificate[0] != daprovider.DACertificateMessageHeaderFlag {
		return nil, fmt.Errorf("invalid certificate header: expected 0x%02x, got 0x%02x",
			daprovider.DACertificateMessageHeaderFlag, certificate[0])
	}

	return certificate, nil
}

// Build standard CustomDA proof
// [...proof..., certSize(8), certificate, customProof]
func constructEnhancedProof(
	originalProof []byte,
	certificate []byte,
	customProof []byte,
) []byte {
	enhancedProof := make([]byte, len(originalProof)+CertificateSizeFieldSize+len(certificate)+len(customProof))

	// Copy the raw original proof
	copy(enhancedProof, originalProof)
	offset := len(originalProof)

	// Add certSize
	binary.BigEndian.PutUint64(enhancedProof[offset:], uint64(len(certificate)))
	offset += CertificateSizeFieldSize

	// Add certificate
	copy(enhancedProof[offset:], certificate)
	offset += len(certificate)

	// Add custom proof
	copy(enhancedProof[offset:], customProof)

	return enhancedProof
}
