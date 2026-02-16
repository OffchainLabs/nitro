// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package proofenhancement

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/staker"
)

// ReadPreimageProofEnhancer enhances proofs that involve CustomDA preimage operations
type ReadPreimageProofEnhancer struct {
	dapRegistry  *daprovider.DAProviderRegistry
	inboxTracker staker.InboxTrackerInterface
	inboxReader  staker.InboxReaderInterface
}

// NewReadPreimageProofEnhancer creates a new CustomDA proof enhancer
func NewReadPreimageProofEnhancer(
	dapRegistry *daprovider.DAProviderRegistry,
	inboxTracker staker.InboxTrackerInterface,
	inboxReader staker.InboxReaderInterface,
) *ReadPreimageProofEnhancer {
	return &ReadPreimageProofEnhancer{
		dapRegistry:  dapRegistry,
		inboxTracker: inboxTracker,
		inboxReader:  inboxReader,
	}
}

// EnhanceProof implements ProofEnhancer for CustomDA
func (e *ReadPreimageProofEnhancer) EnhanceProof(ctx context.Context, messageNum arbutil.MessageIndex, proof []byte) ([]byte, error) {
	// Extract keccak256 of the certificate and offset from end of proof
	// Format: [...proof..., certKeccak256(32), offset(8), marker(1)]
	minProofSize := CertificateHashSize + OffsetSize + MarkerSize
	if len(proof) < minProofSize {
		return nil, fmt.Errorf("proof too short for ReadPreimage enhancement: expected at least %d bytes, got %d", minProofSize, len(proof))
	}

	// The entire proof is of variable length, so we work backwards from
	// final marker byte to find all the marker data added by serialize_proof() for CustomDA ReadPreImage.
	markerPos := len(proof) - MarkerSize
	offsetPos := markerPos - OffsetSize
	certKeccak256Pos := offsetPos - CertificateHashSize

	// Verify marker
	if proof[markerPos] != MarkerCustomDAReadPreimage {
		return nil, fmt.Errorf("invalid marker for ReadPreimage enhancer: 0x%02x", proof[markerPos])
	}

	// Extract certKeccak256 and offset
	var certKeccak256 [32]byte
	copy(certKeccak256[:], proof[certKeccak256Pos:offsetPos])

	certificate, err := retrieveCertificateFromInboxMessage(ctx, messageNum, e.inboxTracker, e.inboxReader)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve certificate from inbox message %d: %w", messageNum, err)
	}

	// Verify the certificate hash matches what's in the proof
	certHash := crypto.Keccak256Hash(certificate)
	if !bytes.Equal(certHash[:], certKeccak256[:]) {
		return nil, fmt.Errorf("certificate hash mismatch: expected %x, got %x", certKeccak256, certHash)
	}

	// Get validator for this certificate type
	validator := e.dapRegistry.GetValidator(certificate[0])
	if validator == nil {
		return nil, fmt.Errorf("no validator registered for certificate type 0x%02x", certificate[0])
	}

	// Generate custom proof with certificate
	offset := binary.BigEndian.Uint64(proof[offsetPos:markerPos])
	promise := validator.GenerateReadPreimageProof(offset, certificate)
	result, err := promise.Await(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to generate custom DA proof: %w", err)
	}

	// We're dropping the CustomDA marker data (certKeccak256, offset, marker byte) from the original proof.
	// It was only needed here to call GenerateReadPreimageProof above, the same information is
	// available to the OSP in the instruction arguments.
	markerDataStart := certKeccak256Pos // Start of CustomDA marker data that we'll drop
	return constructEnhancedProof(proof[:markerDataStart], certificate, result.Proof), nil
}
