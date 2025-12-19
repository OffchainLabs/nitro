package arbtest

import (
	"context"
	"encoding/binary"
	"fmt"

	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/validator/proofenhancement"
)

// EvilCustomDAProofEnhancer wraps the standard ReadPreimageProofEnhancer to inject evil certificates
type EvilCustomDAProofEnhancer struct {
	*proofenhancement.ReadPreimageProofEnhancer
	evilMappings map[common.Hash][]byte // goodCertKeccak -> evil certificate
}

func NewEvilCustomDAProofEnhancer(
	standardEnhancer *proofenhancement.ReadPreimageProofEnhancer,
) *EvilCustomDAProofEnhancer {
	return &EvilCustomDAProofEnhancer{
		ReadPreimageProofEnhancer: standardEnhancer,
		evilMappings:              make(map[common.Hash][]byte),
	}
}

func (e *EvilCustomDAProofEnhancer) SetMapping(goodCertKeccak common.Hash, evilCertificate []byte) {
	e.evilMappings[goodCertKeccak] = evilCertificate
}

func (e *EvilCustomDAProofEnhancer) EnhanceProof(ctx context.Context, messageNum arbutil.MessageIndex, proof []byte) ([]byte, error) {
	// Extract keccak256 of the certificate and offset from end of proof
	// Format: [...proof..., certKeccak256(32), offset(8), marker(1)]
	if len(proof) < 41 {
		return nil, fmt.Errorf("proof too short for CustomDA enhancement: %d bytes", len(proof))
	}

	// Work backwards from marker
	markerPos := len(proof) - 1
	offsetPos := markerPos - 8
	certKeccak256Pos := offsetPos - 32

	// Verify marker
	if proof[markerPos] != proofenhancement.MarkerCustomDAReadPreimage {
		return nil, fmt.Errorf("invalid marker for CustomDA enhancer: 0x%02x", proof[markerPos])
	}

	// Extract certKeccak256
	var certKeccak256 [32]byte
	copy(certKeccak256[:], proof[certKeccak256Pos:offsetPos])

	// Check if we have an evil mapping for this certificate
	if evilCert, ok := e.evilMappings[certKeccak256]; ok {
		// We need to get the custom proof data
		// Let the standard enhancer do its work to get the custom proof
		standardEnhanced, err := e.ReadPreimageProofEnhancer.EnhanceProof(ctx, messageNum, proof)
		if err != nil {
			return nil, err
		}

		// Extract the custom proof from the standard enhanced proof
		// Standard format: [...proof..., certSize(8), certificate, customProof]
		// We need to find where customProof starts

		// Read certSize from standard enhanced proof at certKeccak256Pos
		certSize := binary.BigEndian.Uint64(standardEnhanced[certKeccak256Pos : certKeccak256Pos+8])
		customProofStart := certKeccak256Pos + 8 + int(certSize) //nolint:gosec
		customProof := standardEnhanced[customProofStart:]

		// Build evil enhanced proof with evil certificate
		evilCertSize := uint64(len(evilCert))
		markerDataStart := certKeccak256Pos
		enhancedProof := make([]byte, markerDataStart+8+len(evilCert)+len(customProof))

		// Copy original proof up to the CustomDA marker data
		copy(enhancedProof, proof[:markerDataStart])

		// Add evil cert size
		binary.BigEndian.PutUint64(enhancedProof[markerDataStart:], evilCertSize)

		// Add evil certificate
		copy(enhancedProof[markerDataStart+8:], evilCert)

		// Add custom proof
		copy(enhancedProof[markerDataStart+8+len(evilCert):], customProof)

		return enhancedProof, nil
	}

	// No evil mapping, use standard behavior
	return e.ReadPreimageProofEnhancer.EnhanceProof(ctx, messageNum, proof)
}
