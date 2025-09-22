// Copyright 2021-2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package referenceda

import (
	"context"
	"crypto/sha256"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/daprovider"
)

// Reader implements the daprovider.Reader interface for ReferenceDA
type Reader struct {
	storage       *InMemoryStorage
	l1Client      *ethclient.Client
	validatorAddr common.Address
}

func NewReader(l1Client *ethclient.Client, validatorAddr common.Address) *Reader {
	return &Reader{
		storage:       GetInMemoryStorage(),
		l1Client:      l1Client,
		validatorAddr: validatorAddr,
	}
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
	certBytes := sequencerMsg[40:]

	// Deserialize certificate
	cert, err := Deserialize(certBytes)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to deserialize certificate: %w", err)
	}

	// Validate certificate if requested
	// TODO: Uncomment the following once we have merged customda contracts changes.
	/*
		if validateSeqMsg {
				// Create contract binding
				validator, err := ospgen.NewReferenceDAProofValidator(r.validatorAddr, r.l1Client)
				if err != nil {
					return nil, nil, fmt.Errorf("failed to create validator binding: %w", err)
				}

				// Validate using contract
				callOpts := &bind.CallOpts{Context: ctx}
				err = cert.ValidateWithContract(validator, callOpts)
				if err != nil {
					return nil, nil, fmt.Errorf("certificate validation failed: %w", err)
				}
		}
	*/

	log.Debug("ReferenceDA reader extracting hash",
		"certificateLen", len(certBytes),
		"sha256Hash", common.Hash(cert.DataHash).Hex(),
		"certificateHex", fmt.Sprintf("0x%x", certBytes))

	// Retrieve the data from storage using the hash
	payload, err := r.storage.GetByHash(ctx, cert.DataHash)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to retrieve data from storage: %w", err)
	}
	if payload == nil {
		return nil, nil, fmt.Errorf("data not found in storage for hash %s", common.Hash(cert.DataHash).Hex())
	}

	// Verify data matches certificate hash (SHA256)
	actualHash := sha256.Sum256(payload)
	if actualHash != cert.DataHash {
		return nil, nil, fmt.Errorf("data hash mismatch: expected %s, got %s", common.Hash(cert.DataHash).Hex(), common.Hash(actualHash).Hex())
	}

	// Record preimages if needed
	if preimages != nil {
		preimageRecorder := daprovider.RecordPreimagesTo(preimages)

		// Record the mapping from certificate hash to actual payload data
		// This is what the replay binary expects: keccak256(certificate) -> payload
		certHash := crypto.Keccak256Hash(certBytes)
		preimageRecorder(certHash, payload, arbutil.DACertificatePreimageType)
	}

	log.Debug("ReferenceDA batch recovery completed",
		"batchNum", batchNum,
		"blockHash", batchBlockHash,
		"sha256", common.Hash(cert.DataHash).Hex(),
		"payloadSize", len(payload))

	return payload, preimages, nil
}
