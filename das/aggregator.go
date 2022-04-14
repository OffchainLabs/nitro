//
// Copyright 2022, Offchain Labs, Inc. All rights reserved.
//

package das

import (
	"context"
	"errors"
	"fmt"
	"math/bits"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/blsSignatures"
)

type AggregatorConfig struct {
	assumedHonest int
}

type Aggregator struct {
	config   AggregatorConfig
	services []serviceDetails

	/// calculated fields
	requiredServicesForStore       int
	maxAllowedServiceStoreFailures int
}

type serviceDetails struct {
	service DataAvailabilityService
	pubKey  blsSignatures.PublicKey
}

func NewAggregator(config AggregatorConfig, services []serviceDetails) *Aggregator {
	return &Aggregator{
		config:                         config,
		services:                       services,
		requiredServicesForStore:       len(services) + 1 - config.assumedHonest,
		maxAllowedServiceStoreFailures: config.assumedHonest - 1,
	}
}

func (a *Aggregator) Retrieve(ctx context.Context, cert []byte) ([]byte, error) {
	requestedCert, _, err := arbstate.DeserializeDASCertFrom(cert)
	if err != nil {
		return nil, err
	}
	// Cert is the aggregate cert

	var blob []byte
	// TODO make this async
	for _, d := range a.services {
		blob, err = d.service.Retrieve(ctx, cert)
		if err != nil {
			log.Warn("Retrieve from backend DAS failed", "err", err)
			continue
		}
		var blobHash [32]byte
		copy(blobHash[:], crypto.Keccak256(blob))
		if blobHash == requestedCert.DataHash {
			return blob, nil
		}
	}

	// TODO better error reporting for each DAS that failed
	return nil, errors.New("Data wasn't able to be retrieved from any DAS")
}

func (a *Aggregator) Store(ctx context.Context, message []byte) (*arbstate.DataAvailabilityCertificate, error) {
	var aggSignersMask uint64
	var pubKeys []blsSignatures.PublicKey
	var sigs []blsSignatures.Signature
	var aggCert arbstate.DataAvailabilityCertificate

	var initialStoreSucceeded bool
	storeFailures := 0
	for i, d := range a.services {
		// TODO make this asnyc
		cert, err := d.service.Store(ctx, message)
		// TODO actually we will want to not bail if until we hit H failures
		if err != nil {
			storeFailures++
			log.Warn("Failed to store message to DAS", "err", err)
			if storeFailures <= a.maxAllowedServiceStoreFailures {
				continue
			} else {
				return nil, fmt.Errorf("Aggregator failed to store message to at least %d out of %d DASes (assuming %d are honest)", a.requiredServicesForStore, len(a.services), a.config.assumedHonest)
			}
		}
		verified, err := blsSignatures.VerifySignature(cert.Sig, serializeSignableFields(*cert), d.pubKey)
		if err != nil {
			return nil, err
		}
		if !verified {
			return nil, errors.New("Failed signature check")
		}

		// TODO need to think more about these bits
		// how to support downstream combining of signatures?
		prevPopCount := bits.OnesCount64(aggSignersMask)
		certPopCount := bits.OnesCount64(cert.SignersMask)
		aggSignersMask |= cert.SignersMask
		newPopCount := bits.OnesCount64(aggSignersMask)
		if prevPopCount+certPopCount != newPopCount {
			return nil, errors.New("Duplicate signers error.")
		}
		pubKeys = append(pubKeys, d.pubKey)
		sigs = append(sigs, cert.Sig)
		if !initialStoreSucceeded {
			initialStoreSucceeded = true
			aggCert.DataHash = cert.DataHash
			aggCert.Timeout = cert.Timeout
		} else {
			if aggCert.DataHash != cert.DataHash {
				return nil, fmt.Errorf("Mismatched DataHash from DAS %d", i)
			}
			if aggCert.Timeout != cert.Timeout {
				return nil, fmt.Errorf("Mismatched Timeout from DAS %d", i)
			}
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
