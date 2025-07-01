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

// CustomDAProofEnhancer enhances proofs that involve CustomDA preimage operations
type CustomDAProofEnhancer struct {
	daValidator  daprovider.Validator
	inboxTracker staker.InboxTrackerInterface
	inboxReader  staker.InboxReaderInterface
}

// NewCustomDAProofEnhancer creates a new CustomDA proof enhancer
func NewCustomDAProofEnhancer(
	validator daprovider.Validator,
	inboxTracker staker.InboxTrackerInterface,
	inboxReader staker.InboxReaderInterface,
) *CustomDAProofEnhancer {
	return &CustomDAProofEnhancer{
		daValidator:  validator,
		inboxTracker: inboxTracker,
		inboxReader:  inboxReader,
	}
}

// EnhanceProof implements ProofEnhancer for CustomDA
func (e *CustomDAProofEnhancer) EnhanceProof(ctx context.Context, messageNum arbutil.MessageIndex, proof []byte) ([]byte, error) {
	batchContainingMessage, found, err := e.inboxTracker.FindInboxBatchContainingMessage(messageNum)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("Couldn't find batch for message #%d to enhance proof", messageNum)
	}

	sequencerMessage, sequencerMessageHash, err := e.inboxReader.GetSequencerMessageBytes(ctx, batchContainingMessage)
	if err != nil {
		return nil, fmt.Errorf("failed to get sequencer message for batch %d: %w", batchContainingMessage, err)
	}
	_ = sequencerMessageHash // silence unused variable warning

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

	if certificate[0] != daprovider.CustomDAMessageHeaderFlag {
		return nil, fmt.Errorf("invalid certificate header: expected 0x%02x, got 0x%02x",
			daprovider.CustomDAMessageHeaderFlag, certificate[0])
	}

	// Extract keccak256 of the certificate and offset from end of proof
	// Format: [...proof..., certKeccak256(32), offset(8), marker(1)]
	if len(proof) < 41 {
		return nil, fmt.Errorf("proof too short for CustomDA enhancement: %d bytes", len(proof))
	}

	markerPos := len(proof) - 1
	offsetPos := markerPos - 8
	certKeccak256Pos := offsetPos - 32

	// Verify marker
	if proof[markerPos] != MarkerCustomDARead {
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
	customProof, err := e.daValidator.GenerateProof(ctx, arbutil.CustomDAPreimageType, common.BytesToHash(certKeccak256[:]), offset, certificate)
	if err != nil {
		return nil, fmt.Errorf("failed to generate custom DA proof: %w", err)
	}

	// Build enhanced proof format:
	// [original proof][certKeccak256][offset][customProof]
	// Note: certificate is now included inside customProof
	enhancedProof := make([]byte, certKeccak256Pos+32+8+len(customProof))

	// Copy original proof up to hash position
	copy(enhancedProof, proof[:certKeccak256Pos])

	// Add certKeccak256
	copy(enhancedProof[certKeccak256Pos:], certKeccak256[:])

	// Add offset
	binary.BigEndian.PutUint64(enhancedProof[certKeccak256Pos+32:], offset)

	// Add custom proof (which now contains the certificate)
	copy(enhancedProof[certKeccak256Pos+40:], customProof)

	return enhancedProof, nil
}
