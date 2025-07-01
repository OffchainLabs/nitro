// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

//go:build challengetest && !race

package arbtest

import (
	"context"
	"encoding/binary"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/daprovider/referenceda"
)

// EvilDAProvider implements both Reader and Validator interfaces
// It wraps the regular ReferenceDA components and intercepts specific certificates
// Note: It's safe to create new underlying readers/validators because they all use
// the same singleton storage instance via GetInMemoryStorage()
type EvilDAProvider struct {
	reader       daprovider.Reader
	validator    daprovider.Validator
	evilMappings map[common.Hash][]byte // certHash -> evil data
	mu           sync.RWMutex
}

func NewEvilDAProvider() *EvilDAProvider {
	// Create fresh ReferenceDA components - they'll all share the singleton storage
	return &EvilDAProvider{
		reader:       referenceda.NewReader(),
		validator:    referenceda.NewValidator(),
		evilMappings: make(map[common.Hash][]byte),
	}
}

// SetMapping configures the provider to return evil data for a specific certificate
func (e *EvilDAProvider) SetMapping(certHash common.Hash, evilData []byte) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.evilMappings[certHash] = evilData
}

// IsValidHeaderByte delegates to underlying reader
func (e *EvilDAProvider) IsValidHeaderByte(ctx context.Context, headerByte byte) bool {
	return e.reader.IsValidHeaderByte(ctx, headerByte)
}

// RecoverPayloadFromBatch intercepts and returns evil data if configured
func (e *EvilDAProvider) RecoverPayloadFromBatch(
	ctx context.Context,
	batchNum uint64,
	batchBlockHash common.Hash,
	sequencerMsg []byte,
	preimages daprovider.PreimagesMap,
	validateSeqMsg bool,
) ([]byte, daprovider.PreimagesMap, error) {
	// Check if this is a CustomDA message and extract certificate
	if len(sequencerMsg) > 40 && daprovider.IsCustomDAMessageHeaderByte(sequencerMsg[40]) {
		certificate := sequencerMsg[40:]

		// Certificate format: [0x01 header byte][32 bytes SHA256]
		if len(certificate) >= 33 && certificate[0] == 0x01 {
			// Extract data hash (SHA256) from certificate
			dataHash := common.BytesToHash(certificate[1:33])

			e.mu.RLock()
			if evilData, exists := e.evilMappings[dataHash]; exists {
				e.mu.RUnlock()

				// Record preimages with evil data
				if preimages != nil {
					preimageRecorder := daprovider.RecordPreimagesTo(preimages)
					// Use keccak256 of certificate for preimage recording
					certKeccak := crypto.Keccak256Hash(certificate)
					preimageRecorder(certKeccak, evilData, arbutil.CustomDAPreimageType)
				}

				log.Info("EvilDAProvider returning evil data",
					"dataHash", dataHash.Hex(),
					"evilDataSize", len(evilData))

				return evilData, preimages, nil
			}
			e.mu.RUnlock()
		}
	}

	// Fall back to underlying reader for non-evil certificates
	return e.reader.RecoverPayloadFromBatch(ctx, batchNum, batchBlockHash, sequencerMsg, preimages, validateSeqMsg)
}

// GenerateProof generates proof for evil data if configured, otherwise delegates
func (e *EvilDAProvider) GenerateProof(
	ctx context.Context,
	preimageType arbutil.PreimageType,
	certHash common.Hash,
	offset uint64,
	certificate []byte,
) ([]byte, error) {
	if preimageType != arbutil.CustomDAPreimageType {
		return e.validator.GenerateProof(ctx, preimageType, certHash, offset, certificate)
	}

	// Extract SHA256 hash from certificate to check for evil mapping
	if len(certificate) == 33 && certificate[0] == 0x01 {
		// Extract data hash (SHA256) from certificate
		dataHash := common.BytesToHash(certificate[1:33])

		e.mu.RLock()
		evilData, hasEvil := e.evilMappings[dataHash]
		e.mu.RUnlock()

		if hasEvil {
			// Generate proof with evil data
			// Format: [Version(1), CertificateSize(8), Certificate, PreimageSize(8), PreimageData]
			certLen := len(certificate)
			proof := make([]byte, 1+8+certLen+8+len(evilData))
			proof[0] = 1 // Version
			binary.BigEndian.PutUint64(proof[1:9], uint64(certLen))
			copy(proof[9:9+certLen], certificate)
			binary.BigEndian.PutUint64(proof[9+certLen:9+certLen+8], uint64(len(evilData)))
			copy(proof[9+certLen+8:], evilData)

			log.Debug("EvilDAProvider generating evil proof",
				"certHash", certHash.Hex(),
				"dataHash", dataHash.Hex(),
				"evilDataSize", len(evilData))

			return proof, nil
		}
	}

	// No evil mapping, delegate to underlying validator
	return e.validator.GenerateProof(ctx, preimageType, certHash, offset, certificate)
}
