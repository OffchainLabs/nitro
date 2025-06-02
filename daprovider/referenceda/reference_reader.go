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

// Reader implements the daprovider.Reader interface for CustomDA
type Reader struct {
	// client interface for connecting to the CustomDA service
	validator daprovider.Validator
}

// NewReader creates a new CustomDA reader with the provided validator
func NewReader(validator daprovider.Validator) *Reader {
	return &Reader{
		validator: validator,
	}
}

// IsValidHeaderByte returns true if the header byte indicates a CustomDA message
func (r *Reader) IsValidHeaderByte(ctx context.Context, headerByte byte) bool {
	return daprovider.IsCustomDAMessageHeaderByte(headerByte)
}

// RecoverPayloadFromBatch fetches the batch data from the CustomDA service
func (r *Reader) RecoverPayloadFromBatch(
	ctx context.Context,
	batchNum uint64,
	batchBlockHash common.Hash,
	sequencerMsg []byte,
	preimages daprovider.PreimagesMap,
	validateSeqMsg bool,
) ([]byte, daprovider.PreimagesMap, error) {
	if len(sequencerMsg) == 0 {
		return nil, nil, fmt.Errorf("empty sequencer message")
	}

	headerByte := sequencerMsg[0]
	if !daprovider.IsCustomDAMessageHeaderByte(headerByte) {
		return nil, nil, fmt.Errorf("not a CustomDA message: header byte 0x%x", headerByte)
	}

	// Create a hash of the entire batch to use as a preimage key
	batchHash := crypto.Keccak256Hash(sequencerMsg)

	// In a real implementation, this would call into the CustomDA service
	// to retrieve and validate the actual batch data

	// For this reference implementation, we'll just return the sequencerMsg itself
	// minus the header byte, as the "payload" (this mimics what normal batch processing does)
	payload := sequencerMsg[40:] // Skip the HEADER_LENGTH (40) bytes

	// Record the preimages if needed
	if preimages != nil {
		preimageRecorder := daprovider.RecordPreimagesTo(preimages)

		// Record the full message as a CustomDA preimage
		preimageRecorder(batchHash, sequencerMsg, arbutil.CustomDAPreimageType)

		// For proper fraud proofs, the validator would extract all required preimages
		// and record them here
		extractedPreimages, err := r.validator.RecordPreimages(ctx, sequencerMsg)
		if err != nil {
			log.Warn("Failed to extract preimages from CustomDA batch",
				"batchNum", batchNum,
				"error", err)
		} else {
			for _, p := range extractedPreimages {
				preimageRecorder(p.Hash, p.Data, p.PreimageType)
			}
		}
	}

	log.Debug("CustomDA batch recovery completed",
		"batchNum", batchNum,
		"blockHash", batchBlockHash,
		"payloadSize", len(payload))

	return payload, preimages, nil
}
