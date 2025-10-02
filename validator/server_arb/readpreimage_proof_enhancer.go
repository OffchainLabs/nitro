package server_arb

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/staker"
)

// ReadPreimageProofEnhancer enhances proofs that involve CustomDA preimage operations
type ReadPreimageProofEnhancer struct {
	daValidator  daprovider.Validator
	inboxTracker staker.InboxTrackerInterface
	inboxReader  staker.InboxReaderInterface
}

// NewReadPreimageProofEnhancer creates a new CustomDA proof enhancer
func NewReadPreimageProofEnhancer(
	validator daprovider.Validator,
	inboxTracker staker.InboxTrackerInterface,
	inboxReader staker.InboxReaderInterface,
) *ReadPreimageProofEnhancer {
	return &ReadPreimageProofEnhancer{
		daValidator:  validator,
		inboxTracker: inboxTracker,
		inboxReader:  inboxReader,
	}
}

// EnhanceProof implements ProofEnhancer for CustomDA
func (e *ReadPreimageProofEnhancer) EnhanceProof(ctx context.Context, messageNum arbutil.MessageIndex, proof []byte) ([]byte, error) {
	batchContainingMessage, found, err := e.inboxTracker.FindInboxBatchContainingMessage(messageNum)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("Couldn't find batch for message #%d to enhance proof", messageNum)
	}

	sequencerMessage, _, err := e.inboxReader.GetSequencerMessageBytes(ctx, batchContainingMessage)
	if err != nil {
		return nil, fmt.Errorf("failed to get sequencer message for batch %d: %w", batchContainingMessage, err)
	}

	// Extract and validate certificate from sequencer message
	if len(sequencerMessage) < 41 {
		return nil, fmt.Errorf("sequencer message too short: expected at least 41 bytes, got %d", len(sequencerMessage))
	}

	// Extract certificate (skip 40-byte header)
	certificate := sequencerMessage[40:]

	// Validate certificate format
	if len(certificate) < 33 {
		return nil, fmt.Errorf("certificate too short: expected at least 33 bytes, got %d", len(certificate))
	}

	if certificate[0] != daprovider.DACertificateMessageHeaderFlag {
		return nil, fmt.Errorf("invalid certificate header: expected 0x%02x, got 0x%02x",
			daprovider.DACertificateMessageHeaderFlag, certificate[0])
	}

	// Extract keccak256 of the certificate and offset from end of proof
	// Format: [...proof..., certKeccak256(32), offset(8), marker(1)]
	if len(proof) < 41 {
		return nil, fmt.Errorf("proof too short for CustomDA enhancement: %d bytes", len(proof))
	}

	// The entire proof is of variable length, so we work backwards from
	// final marker byte to find all the marker data added by serialize_proof() for CustomDA ReadPreImage.
	markerPos := len(proof) - 1
	offsetPos := markerPos - 8
	certKeccak256Pos := offsetPos - 32

	// Verify marker
	if proof[markerPos] != MarkerCustomDAReadPreimage {
		return nil, fmt.Errorf("invalid marker for CustomDA enhancer: 0x%02x", proof[markerPos])
	}

	// Extract certKeccak256 and offset
	var certKeccak256 [32]byte
	copy(certKeccak256[:], proof[certKeccak256Pos:offsetPos])
	offset := binary.BigEndian.Uint64(proof[offsetPos:markerPos])

	// Verify the certificate hash matches what's in the proof
	certHash := crypto.Keccak256Hash(certificate)
	if !bytes.Equal(certHash[:], certKeccak256[:]) {
		return nil, fmt.Errorf("certificate hash mismatch: expected %x, got %x", certKeccak256, certHash)
	}

	// Generate custom proof with certificate
	promise := e.daValidator.GenerateReadPreimageProof(common.BytesToHash(certKeccak256[:]), offset, certificate)
	result, err := promise.Await(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to generate custom DA proof: %w", err)
	}
	customProof := result.Proof

	// Build standard CustomDA proof preamble:
	// [...proof..., certSize(8), certificate, customProof]
	// We're dropping the CustomDA marker data (certKeccak256, offset, marker byte) from the original proof.
	// It was only needed here to call GenerateReadPreimageProof above, the same information is
	// available to the OSP in the instruction arguments.
	certSize := uint64(len(certificate))
	markerDataStart := certKeccak256Pos // Start of CustomDA marker data that we'll drop
	enhancedProof := make([]byte, markerDataStart+8+len(certificate)+len(customProof))

	// Copy original proof up to the CustomDA marker data
	copy(enhancedProof, proof[:markerDataStart])

	// Add certSize
	binary.BigEndian.PutUint64(enhancedProof[markerDataStart:], certSize)

	// Add certificate
	copy(enhancedProof[markerDataStart+8:], certificate)

	// Add custom proof
	copy(enhancedProof[markerDataStart+8+len(certificate):], customProof)

	return enhancedProof, nil
}
