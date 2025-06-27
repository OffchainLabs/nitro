// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

//go:build challengetest && !race

package arbtest

import (
	"context"
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
		certHash := crypto.Keccak256Hash(certificate)

		e.mu.RLock()
		if evilData, exists := e.evilMappings[certHash]; exists {
			e.mu.RUnlock()

			// Record preimages with evil data
			if preimages != nil {
				preimageRecorder := daprovider.RecordPreimagesTo(preimages)
				preimageRecorder(certHash, evilData, arbutil.CustomDAPreimageType)
			}

			log.Debug("EvilDAProvider returning evil data",
				"certHash", certHash.Hex(),
				"evilDataSize", len(evilData))

			return evilData, preimages, nil
		}
		e.mu.RUnlock()
	}

	// Fall back to underlying reader for non-evil certificates
	return e.reader.RecoverPayloadFromBatch(ctx, batchNum, batchBlockHash, sequencerMsg, preimages, validateSeqMsg)
}

// GenerateProof delegates to underlying validator
// TODO: Implement evil proof generation for certificates with evil data mappings
func (e *EvilDAProvider) GenerateProof(
	ctx context.Context,
	preimageType arbutil.PreimageType,
	hash common.Hash,
	offset uint64,
) ([]byte, error) {
	// For now, just delegate to underlying validator
	// If needed, we could intercept here too for evil data proofs
	return e.validator.GenerateProof(ctx, preimageType, hash, offset)
}
