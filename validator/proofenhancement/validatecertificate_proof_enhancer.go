// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package proofenhancement

import (
	"context"
	"encoding/binary"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/staker"
)

type ValidateCertificateProofEnhancer struct {
	dapRegistry  *daprovider.DAProviderRegistry
	inboxTracker staker.InboxTrackerInterface
	inboxReader  staker.InboxReaderInterface
}

func NewValidateCertificateProofEnhancer(
	dapRegistry *daprovider.DAProviderRegistry,
	inboxTracker staker.InboxTrackerInterface,
	inboxReader staker.InboxReaderInterface,
) *ValidateCertificateProofEnhancer {
	return &ValidateCertificateProofEnhancer{
		dapRegistry:  dapRegistry,
		inboxTracker: inboxTracker,
		inboxReader:  inboxReader,
	}
}

func (e *ValidateCertificateProofEnhancer) EnhanceProof(ctx context.Context, messageNum arbutil.MessageIndex, proof []byte) ([]byte, error) {
	// Extract the hash and marker from the proof
	// Format: [...proof..., certHash(32), marker(1)]
	minProofSize := CertificateHashSize + MarkerSize
	if len(proof) < minProofSize {
		return nil, fmt.Errorf("proof too short for ValidateCertificate enhancement: expected at least %d bytes, got %d", minProofSize, len(proof))
	}

	markerPos := len(proof) - MarkerSize
	hashPos := markerPos - CertificateHashSize

	// Verify marker
	if proof[markerPos] != MarkerCustomDAValidateCertificate {
		return nil, fmt.Errorf("invalid marker for ValidateCertificate enhancer: 0x%02x", proof[markerPos])
	}

	// Extract certificate hash
	var certHash [32]byte
	copy(certHash[:], proof[hashPos:markerPos])

	// Find the batch containing this message
	batchContainingMessage, found, err := e.inboxTracker.FindInboxBatchContainingMessage(messageNum)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("couldn't find batch for message #%d to enhance proof", messageNum)
	}

	// Get the sequencer message
	sequencerMessage, _, err := e.inboxReader.GetSequencerMessageBytes(ctx, batchContainingMessage)
	if err != nil {
		return nil, fmt.Errorf("failed to get sequencer message for batch %d: %w", batchContainingMessage, err)
	}

	// Extract certificate from sequencer message (skip sequencer message header)
	if len(sequencerMessage) < SequencerMessageHeaderSize+1 {
		return nil, fmt.Errorf("sequencer message too short: expected at least %d bytes, got %d", SequencerMessageHeaderSize+1, len(sequencerMessage))
	}
	certificate := sequencerMessage[SequencerMessageHeaderSize:]

	// Verify the certificate hash matches what's requested
	actualHash := crypto.Keccak256Hash(certificate)
	if actualHash != common.BytesToHash(certHash[:]) {
		return nil, fmt.Errorf("certificate hash mismatch: expected %x, got %x", certHash, actualHash)
	}

	// Get validator for this certificate type
	if len(certificate) == 0 {
		return nil, fmt.Errorf("empty certificate")
	}
	validator := e.dapRegistry.GetValidator(certificate[0])
	if validator == nil {
		return nil, fmt.Errorf("no validator registered for certificate type 0x%02x", certificate[0])
	}

	// Generate certificate validity proof
	promise := validator.GenerateCertificateValidityProof(certificate)
	result, err := promise.Await(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to generate certificate validity proof: %w", err)
	}
	validityProof := result.Proof

	// Build enhanced proof: [...originalProof..., certSize(8), certificate, validityProof]
	// Remove the marker data (hash + marker) from original proof
	originalProofLen := hashPos
	certSize := uint64(len(certificate))
	enhancedProof := make([]byte, originalProofLen+CertificateSizeFieldSize+len(certificate)+len(validityProof))

	// Copy original proof (without marker data)
	copy(enhancedProof, proof[:originalProofLen])

	// Add certSize
	offset := originalProofLen
	binary.BigEndian.PutUint64(enhancedProof[offset:], certSize)
	offset += CertificateSizeFieldSize

	// Add certificate
	copy(enhancedProof[offset:], certificate)
	offset += len(certificate)

	// Add validity proof
	copy(enhancedProof[offset:], validityProof)

	return enhancedProof, nil
}
