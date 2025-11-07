// Copyright 2021-2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package referenceda

import (
	"context"
	"crypto/sha256"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/solgen/go/localgen"
	"github.com/offchainlabs/nitro/util/containers"
)

// Reader implements the daprovider.Reader interface for ReferenceDA
type Reader struct {
	storage       *InMemoryStorage
	l1Client      *ethclient.Client
	validatorAddr common.Address
}

// NewReader creates a new ReferenceDA reader
func NewReader(storage *InMemoryStorage, l1Client *ethclient.Client, validatorAddr common.Address) *Reader {
	return &Reader{
		storage:       storage,
		l1Client:      l1Client,
		validatorAddr: validatorAddr,
	}
}

// recoverInternal is the shared implementation for both RecoverPayload and CollectPreimages
func (r *Reader) recoverInternal(
	ctx context.Context,
	batchNum uint64,
	batchBlockHash common.Hash,
	sequencerMsg []byte,
	needPayload bool,
	needPreimages bool,
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

	// Validate certificate - always validate for ReferenceDA
	// Create contract binding
	validator, err := localgen.NewReferenceDAProofValidator(r.validatorAddr, r.l1Client)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create validator binding: %w", err)
	}

	// Validate using contract
	callOpts := &bind.CallOpts{Context: ctx}
	err = cert.ValidateWithContract(validator, callOpts)
	if err != nil {
		return nil, nil, fmt.Errorf("certificate validation failed: %w", err)
	}

	log.Debug("ReferenceDA reader extracting hash",
		"certificateLen", len(certBytes),
		"sha256Hash", common.Hash(cert.DataHash).Hex(),
		"certificateHex", fmt.Sprintf("0x%x", certBytes))

	// Retrieve the data from storage using the hash
	var payload []byte
	if needPayload || needPreimages {
		payload, err = r.storage.GetByHash(common.BytesToHash(cert.DataHash[:]))
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
	}

	// Record preimages if needed
	var preimages daprovider.PreimagesMap
	if needPreimages {
		preimages = make(daprovider.PreimagesMap)
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

// RecoverPayload fetches the underlying payload from the DA provider
func (r *Reader) RecoverPayload(
	batchNum uint64,
	batchBlockHash common.Hash,
	sequencerMsg []byte,
) containers.PromiseInterface[daprovider.PayloadResult] {
	return containers.DoPromise(context.Background(), func(ctx context.Context) (daprovider.PayloadResult, error) {
		payload, _, err := r.recoverInternal(ctx, batchNum, batchBlockHash, sequencerMsg, true, false)
		return daprovider.PayloadResult{Payload: payload}, err
	})
}

// CollectPreimages collects preimages from the DA provider
func (r *Reader) CollectPreimages(
	batchNum uint64,
	batchBlockHash common.Hash,
	sequencerMsg []byte,
) containers.PromiseInterface[daprovider.PreimagesResult] {
	return containers.DoPromise(context.Background(), func(ctx context.Context) (daprovider.PreimagesResult, error) {
		_, preimages, err := r.recoverInternal(ctx, batchNum, batchBlockHash, sequencerMsg, false, true)
		return daprovider.PreimagesResult{Preimages: preimages}, err
	})
}
