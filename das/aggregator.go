//
// Copyright 2022, Offchain Labs, Inc. All rights reserved.
//

package das

import (
	"context"
	"errors"
	"fmt"
	"math/bits"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/blsSignatures"
)

type AggregatorConfig struct {
	assumedHonest   int
	retentionPeriod time.Duration
}

type Aggregator struct {
	config   AggregatorConfig
	services []serviceDetails

	/// calculated fields
	requiredServicesForStore       int
	maxAllowedServiceStoreFailures int
}

type serviceDetails struct {
	service    DataAvailabilityService
	pubKey     blsSignatures.PublicKey
	signerMask uint64
}

func NewAggregator(config AggregatorConfig, services []serviceDetails) *Aggregator {
	return &Aggregator{
		config:                         config,
		services:                       services,
		requiredServicesForStore:       len(services) + 1 - config.assumedHonest,
		maxAllowedServiceStoreFailures: config.assumedHonest - 1,
	}
}

// Retrieve calls  on each backend DAS in parallel and returns immediately on the
// first successful response where the data matches the requested hash. Otherwise
// if all requests fail or if its context is canceled (eg via DeadlineWrapper) then
// it returns an error.
func (a *Aggregator) Retrieve(ctx context.Context, cert []byte) ([]byte, error) {
	requestedCert, _, err := arbstate.DeserializeDASCertFrom(cert)
	if err != nil {
		return nil, err
	}

	// Cert is the aggregate cert, validate it against DAS public keys
	var servicesThatSignedCert []serviceDetails
	var pubKeys []blsSignatures.PublicKey
	for _, d := range a.services {
		if requestedCert.SignersMask&d.signerMask != 0 {
			servicesThatSignedCert = append(servicesThatSignedCert, d)
			pubKeys = append(pubKeys, d.pubKey)
		}
	}
	signedBlob := serializeSignableFields(*requestedCert)
	sigMatch, err := blsSignatures.VerifySignature(requestedCert.Sig, signedBlob, blsSignatures.AggregatePublicKeys(pubKeys))
	if err != nil {
		return nil, err
	}
	if !sigMatch {
		return nil, errors.New("Signature of data in cert passed in doesn't match")
	}

	blobChan := make(chan []byte)
	errorChan := make(chan error)
	subCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	for _, d := range servicesThatSignedCert {
		go func(ctx context.Context, d serviceDetails) {
			blob, err := d.service.Retrieve(ctx, cert)
			if err != nil {
				errorChan <- err
				return
			}
			var blobHash [32]byte
			copy(blobHash[:], crypto.Keccak256(blob))
			if blobHash == requestedCert.DataHash {
				blobChan <- blob
			} else {
				errorChan <- fmt.Errorf("DAS (mask %X) returned data that doesn't match requested hash!", d.signerMask)
			}
		}(subCtx, d)
	}

	errorCount := 0
	var errorCollection []error
	for errorCount < len(servicesThatSignedCert) {
		select {
		case blob := <-blobChan:
			return blob, nil
		case err = <-errorChan:
			errorCollection = append(errorCollection, err)
			log.Warn("Couldn't retrieve message from DAS", "err", err)
			errorCount++
		case <-ctx.Done():
			break
		}
	}

	return nil, fmt.Errorf("Data wasn't able to be retrieved from any DAS: %v", errorCollection)
}

type storeResponse struct {
	cert *arbstate.DataAvailabilityCertificate
	err  error

	details serviceDetails
}

// Store calls Store on each backend DAS in parallel and collects responses; if
// there were enough (at least K) responses then it aggregates the signatures and
// signerMasks from each DAS together to put in the DataAvailabilityCertificate
// that it returns; if it gets enough errors that K successes is impossible, then
// it stops early and returns an error, and if it gets not enough successful
// responses by the time its context is canceled (eg via DeadlineWrapper) then it
// also returns an error.
func (a *Aggregator) Store(ctx context.Context, message []byte, timeout uint64) (*arbstate.DataAvailabilityCertificate, error) {
	var aggSignersMask uint64
	var pubKeys []blsSignatures.PublicKey
	var sigs []blsSignatures.Signature
	var aggCert arbstate.DataAvailabilityCertificate

	var initialStoreSucceeded bool
	storeFailures := 0
	if timeout == CALLEE_PICKS_TIMEOUT {
		timeout = uint64(time.Now().Add(a.config.retentionPeriod).Unix())
	}

	responses := make(chan storeResponse)
	subCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	for _, d := range a.services {
		go func(ctx context.Context, d serviceDetails) {
			cert, err := d.service.Store(ctx, message, timeout)
			responses <- storeResponse{cert, err, d}
		}(subCtx, d)
	}

	successfullyStoredCount := 0
	for i := 0; i < len(a.services); i++ {
		select {
		case <-ctx.Done():
			if successfullyStoredCount < a.requiredServicesForStore {
				return nil, errors.New("Terminated das.Aggregator.Store() with requests outstanding before enough responses received")
			} else {
				break
			}
		case r := <-responses:
			if r.err != nil {
				storeFailures++
				log.Warn("Failed to store message to DAS", "err", r.err)
				if storeFailures <= a.maxAllowedServiceStoreFailures {
					continue
				} else {
					return nil, fmt.Errorf("Aggregator failed to store message to at least %d out of %d DASes (assuming %d are honest)", a.requiredServicesForStore, len(a.services), a.config.assumedHonest)
				}
			}
			verified, err := blsSignatures.VerifySignature(r.cert.Sig, serializeSignableFields(*r.cert), r.details.pubKey)
			if err != nil {
				return nil, err
			}
			if !verified {
				return nil, errors.New("Failed signature check")
			}

			prevPopCount := bits.OnesCount64(aggSignersMask)
			certPopCount := bits.OnesCount64(r.cert.SignersMask)
			aggSignersMask |= r.cert.SignersMask
			newPopCount := bits.OnesCount64(aggSignersMask)
			if prevPopCount+certPopCount != newPopCount {
				return nil, errors.New("Duplicate signers error.")
			}
			pubKeys = append(pubKeys, r.details.pubKey)
			sigs = append(sigs, r.cert.Sig)
			if !initialStoreSucceeded {
				initialStoreSucceeded = true
				aggCert.DataHash = r.cert.DataHash
				aggCert.Timeout = r.cert.Timeout
			} else {
				if aggCert.DataHash != r.cert.DataHash {
					return nil, fmt.Errorf("Mismatched DataHash from DAS %v", r.details)
				}
				if aggCert.Timeout != r.cert.Timeout {
					return nil, fmt.Errorf("Mismatched Timeout from DAS %v", r.details)
				}
			}
			successfullyStoredCount++
		}
	}

	aggCert.Sig = blsSignatures.AggregateSignatures(sigs)
	aggPubKey := blsSignatures.AggregatePublicKeys(pubKeys)
	aggCert.SignersMask = aggSignersMask

	verified, err := blsSignatures.VerifySignature(aggCert.Sig, serializeSignableFields(aggCert), aggPubKey)
	if err != nil {
		return nil, err
	}
	if !verified {
		return nil, errors.New("Failed aggregate signature check")
	}
	return &aggCert, nil
}
