// Copyright 2021-2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package referenceda

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/daprovider"
)

// Reader implements the daprovider.Reader interface for ReferenceDA
type Reader struct {
	storage   *InMemoryStorage
	validator daprovider.Validator
}

// NewReader creates a new ReferenceDA reader with the provided validator
func NewReader(validator daprovider.Validator) *Reader {
	return &Reader{
		storage:   GetInMemoryStorage(),
		validator: validator,
	}
}

// IsValidHeaderByte returns true if the header byte indicates a CustomDA message
func (r *Reader) IsValidHeaderByte(ctx context.Context, headerByte byte) bool {
	return daprovider.IsCustomDAMessageHeaderByte(headerByte)
}

// RecoverPayloadFromBatch fetches the batch data from the ReferenceDA storage
func (r *Reader) RecoverPayloadFromBatch(
	ctx context.Context,
	batchNum uint64,
	batchBlockHash common.Hash,
	sequencerMsg []byte,
	preimages daprovider.PreimagesMap,
	validateSeqMsg bool,
) ([]byte, daprovider.PreimagesMap, error) {
	if len(sequencerMsg) <= 40 {
		return nil, nil, fmt.Errorf("sequencer message too small")
	}

	// Skip the 40-byte L1 header and get the certificate
	certificate := sequencerMsg[40:]

	// Verify the header byte
	if len(certificate) < 33 {
		return nil, nil, fmt.Errorf("certificate too small: expected at least 33 bytes, got %d", len(certificate))
	}

	headerByte := certificate[0]
	if !daprovider.IsCustomDAMessageHeaderByte(headerByte) {
		return nil, nil, fmt.Errorf("not a CustomDA message: header byte 0x%x", headerByte)
	}

	// Extract the SHA256 hash from the certificate
	var sha256Hash common.Hash
	copy(sha256Hash[:], certificate[1:33])

	// Retrieve the data from storage using the hash
	payload, err := r.storage.GetByHash(ctx, sha256Hash)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to retrieve data from storage: %w", err)
	}
	if payload == nil {
		return nil, nil, fmt.Errorf("data not found in storage for hash %s", sha256Hash.Hex())
	}

	// Record preimages if needed
	if preimages != nil {
		preimageRecorder := daprovider.RecordPreimagesTo(preimages)

		// Record the full sequencer message as a preimage
		batchHash := crypto.Keccak256Hash(sequencerMsg)
		preimageRecorder(batchHash, sequencerMsg, arbutil.CustomDAPreimageType)

		// If validator is provided, extract additional preimages
		if r.validator != nil {
			extractedPreimages, err := r.validator.RecordPreimages(ctx, payload)
			if err != nil {
				log.Warn("Failed to extract preimages from ReferenceDA batch",
					"batchNum", batchNum,
					"error", err)
			} else {
				for _, p := range extractedPreimages {
					preimageRecorder(p.Hash, p.Data, p.PreimageType)
				}
			}
		}
	}

	log.Debug("ReferenceDA batch recovery completed",
		"batchNum", batchNum,
		"blockHash", batchBlockHash,
		"sha256", sha256Hash.Hex(),
		"payloadSize", len(payload))

	return payload, preimages, nil
}
