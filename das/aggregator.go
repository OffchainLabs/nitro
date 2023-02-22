// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/bits"
	"os"
	"time"

	flag "github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"

	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/blsSignatures"
	"github.com/offchainlabs/nitro/das/dastree"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/util/contracts"
	"github.com/offchainlabs/nitro/util/pretty"
)

type AggregatorConfig struct {
	Enable        bool   `koanf:"enable"`
	AssumedHonest int    `koanf:"assumed-honest"`
	Backends      string `koanf:"backends"`
	DumpKeyset    bool   `koanf:"dump-keyset"`
}

var DefaultAggregatorConfig = AggregatorConfig{
	AssumedHonest: 0,
	Backends:      "",
	DumpKeyset:    false,
}

var BatchToDasFailed = errors.New("unable to batch to DAS")

func AggregatorConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".enable", DefaultAggregatorConfig.Enable, "enable storage/retrieval of sequencer batch data from a list of RPC endpoints; this should only be used by the batch poster and not in combination with other DAS storage types")
	f.Int(prefix+".assumed-honest", DefaultAggregatorConfig.AssumedHonest, "Number of assumed honest backends (H). If there are N backends, K=N+1-H valid responses are required to consider an Store request to be successful.")
	f.String(prefix+".backends", DefaultAggregatorConfig.Backends, "JSON RPC backend configuration")
	f.Bool(prefix+".dump-keyset", DefaultAggregatorConfig.DumpKeyset, "Dump the keyset encoded in hexadecimal for the backends string")
}

type Aggregator struct {
	config         AggregatorConfig
	services       []ServiceDetails
	requestTimeout time.Duration

	// calculated fields
	requiredServicesForStore       int
	maxAllowedServiceStoreFailures int
	keysetHash                     [32]byte
	keysetBytes                    []byte
	bpVerifier                     *contracts.BatchPosterVerifier
}

type ServiceDetails struct {
	service     DataAvailabilityServiceWriter
	pubKey      blsSignatures.PublicKey
	signersMask uint64
	metricName  string
}

func (s *ServiceDetails) String() string {
	return fmt.Sprintf("ServiceDetails{service: %v, signersMask %d}", s.service, s.signersMask)
}

func NewServiceDetails(service DataAvailabilityServiceWriter, pubKey blsSignatures.PublicKey, signersMask uint64, metricName string) (*ServiceDetails, error) {
	if bits.OnesCount64(signersMask) != 1 {
		return nil, fmt.Errorf("tried to configure backend DAS %v with invalid signersMask %X", service, signersMask)
	}
	return &ServiceDetails{
		service:     service,
		pubKey:      pubKey,
		signersMask: signersMask,
		metricName:  metricName,
	}, nil
}

func NewAggregator(ctx context.Context, config DataAvailabilityConfig, services []ServiceDetails) (*Aggregator, error) {
	if config.L1NodeURL == "none" {
		return NewAggregatorWithSeqInboxCaller(config, services, nil)
	}
	l1client, err := GetL1Client(ctx, config.L1ConnectionAttempts, config.L1NodeURL)
	if err != nil {
		return nil, err
	}
	seqInboxAddress, err := OptionalAddressFromString(config.SequencerInboxAddress)
	if err != nil {
		return nil, err
	}
	if seqInboxAddress == nil {
		return NewAggregatorWithSeqInboxCaller(config, services, nil)
	}
	return NewAggregatorWithL1Info(config, services, l1client, *seqInboxAddress)
}

func NewAggregatorWithL1Info(
	config DataAvailabilityConfig,
	services []ServiceDetails,
	l1client arbutil.L1Interface,
	seqInboxAddress common.Address,
) (*Aggregator, error) {
	seqInboxCaller, err := bridgegen.NewSequencerInboxCaller(seqInboxAddress, l1client)
	if err != nil {
		return nil, err
	}
	return NewAggregatorWithSeqInboxCaller(config, services, seqInboxCaller)
}

func NewAggregatorWithSeqInboxCaller(
	config DataAvailabilityConfig,
	services []ServiceDetails,
	seqInboxCaller *bridgegen.SequencerInboxCaller,
) (*Aggregator, error) {
	var aggSignersMask uint64
	pubKeys := []blsSignatures.PublicKey{}
	for _, d := range services {
		if bits.OnesCount64(d.signersMask) != 1 {
			return nil, fmt.Errorf("tried to configure backend DAS %v with invalid signersMask %X", d.service, d.signersMask)
		}
		aggSignersMask |= d.signersMask
		pubKeys = append(pubKeys, d.pubKey)
	}
	if bits.OnesCount64(aggSignersMask) != len(services) {
		return nil, errors.New("at least two signers share a mask")
	}

	keyset := &arbstate.DataAvailabilityKeyset{
		AssumedHonest: uint64(config.AggregatorConfig.AssumedHonest),
		PubKeys:       pubKeys,
	}
	ksBuf := bytes.NewBuffer([]byte{})
	if err := keyset.Serialize(ksBuf); err != nil {
		return nil, err
	}
	keysetHash, err := keyset.Hash()
	if err != nil {
		return nil, err
	}
	if config.AggregatorConfig.DumpKeyset {
		fmt.Printf("Keyset: %s\n", hexutil.Encode(ksBuf.Bytes()))
		fmt.Printf("KeysetHash: %s\n", hexutil.Encode(keysetHash[:]))
		os.Exit(0)
	}

	var bpVerifier *contracts.BatchPosterVerifier
	if seqInboxCaller != nil {
		bpVerifier = contracts.NewBatchPosterVerifier(seqInboxCaller)
	}

	return &Aggregator{
		config:                         config.AggregatorConfig,
		services:                       services,
		requestTimeout:                 config.RequestTimeout,
		requiredServicesForStore:       len(services) + 1 - config.AggregatorConfig.AssumedHonest,
		maxAllowedServiceStoreFailures: config.AggregatorConfig.AssumedHonest - 1,
		keysetHash:                     keysetHash,
		keysetBytes:                    ksBuf.Bytes(),
		bpVerifier:                     bpVerifier,
	}, nil
}

type storeResponse struct {
	details ServiceDetails
	sig     blsSignatures.Signature
	err     error
}

// Store calls Store on each backend DAS in parallel and collects responses.
// If there were at least K responses then it aggregates the signatures and
// signersMasks from each DAS together into the DataAvailabilityCertificate
// then Store returns immediately. If there were any backend Store subroutines
// that were still running when Aggregator.Store returns, they are allowed to
// continue running until the context is canceled (eg via TimeoutWrapper),
// with their results discarded.
//
// If Store gets enough errors that K successes is impossible, then it stops early
// and returns an error.
//
// If Store gets not enough successful responses by the time its context is canceled
// (eg via TimeoutWrapper) then it also returns an error.
//
// If Sequencer Inbox contract details are provided when a das.Aggregator is
// constructed, calls to Store(...) will try to verify the passed-in data's signature
// is from the batch poster. If the contract details are not provided, then the
// signature is not checked, which is useful for testing.
func (a *Aggregator) Store(ctx context.Context, message []byte, timeout uint64, sig []byte) (*arbstate.DataAvailabilityCertificate, error) {
	log.Trace("das.Aggregator.Store", "message", pretty.FirstFewBytes(message), "timeout", time.Unix(int64(timeout), 0), "sig", pretty.FirstFewBytes(sig))
	if a.bpVerifier != nil {
		actualSigner, err := DasRecoverSigner(message, timeout, sig)
		if err != nil {
			return nil, err
		}
		isBatchPoster, err := a.bpVerifier.IsBatchPoster(ctx, actualSigner)
		if err != nil {
			return nil, err
		}
		if !isBatchPoster {
			return nil, errors.New("store request not properly signed")
		}
	}

	responses := make(chan storeResponse, len(a.services))

	expectedHash := dastree.Hash(message)
	for _, d := range a.services {
		go func(ctx context.Context, d ServiceDetails) {
			storeCtx, cancel := context.WithTimeout(ctx, a.requestTimeout)
			const metricBase string = "arb/das/rpc/aggregator/store"
			var metricWithServiceName = metricBase + "/" + d.metricName
			defer cancel()
			incFailureMetric := func() {
				metrics.GetOrRegisterCounter(metricWithServiceName+"/error/total", nil).Inc(1)
				metrics.GetOrRegisterCounter(metricBase+"/error/all/total", nil).Inc(1)
			}

			cert, err := d.service.Store(storeCtx, message, timeout, sig)
			if err != nil {
				incFailureMetric()
				if errors.Is(err, context.DeadlineExceeded) {
					metrics.GetOrRegisterCounter(metricWithServiceName+"/error/timeout/total", nil).Inc(1)
				} else {
					metrics.GetOrRegisterCounter(metricWithServiceName+"/error/client/total", nil).Inc(1)
				}
				responses <- storeResponse{d, nil, err}
				return
			}

			verified, err := blsSignatures.VerifySignature(
				cert.Sig, cert.SerializeSignableFields(), d.pubKey,
			)
			if err != nil {
				incFailureMetric()
				metrics.GetOrRegisterCounter(metricWithServiceName+"/error/bad_response/total", nil).Inc(1)
				responses <- storeResponse{d, nil, err}
				return
			}
			if !verified {
				incFailureMetric()
				metrics.GetOrRegisterCounter(metricWithServiceName+"/error/bad_response/total", nil).Inc(1)
				responses <- storeResponse{d, nil, errors.New("signature verification failed")}
				return
			}

			// SignersMask from backend DAS is ignored.

			if cert.DataHash != expectedHash {
				incFailureMetric()
				metrics.GetOrRegisterCounter(metricWithServiceName+"/error/bad_response/total", nil).Inc(1)
				responses <- storeResponse{d, nil, errors.New("hash verification failed")}
				return
			}
			if cert.Timeout != timeout {
				incFailureMetric()
				metrics.GetOrRegisterCounter(metricWithServiceName+"/error/bad_response/total", nil).Inc(1)
				responses <- storeResponse{d, nil, fmt.Errorf("timeout was %d, expected %d", cert.Timeout, timeout)}
				return
			}

			metrics.GetOrRegisterCounter(metricWithServiceName+"/success/total", nil).Inc(1)
			metrics.GetOrRegisterCounter(metricBase+"/success/all/total", nil).Inc(1)
			responses <- storeResponse{d, cert.Sig, nil}
		}(ctx, d)
	}

	var aggCert arbstate.DataAvailabilityCertificate

	type certDetails struct {
		pubKeys        []blsSignatures.PublicKey
		sigs           []blsSignatures.Signature
		aggSignersMask uint64
		err            error
	}

	// Collect responses from backends.
	certDetailsChan := make(chan certDetails)
	go func() {
		var pubKeys []blsSignatures.PublicKey
		var sigs []blsSignatures.Signature
		var aggSignersMask uint64
		var storeFailures, successfullyStoredCount int
		var returned bool
		for i := 0; i < len(a.services); i++ {

			select {
			case <-ctx.Done():
				break
			case r := <-responses:
				if r.err != nil {
					storeFailures++
					log.Warn("das.Aggregator: Error from backend", "backend", r.details.service, "signerMask", r.details.signersMask, "err", r.err)
				} else {
					pubKeys = append(pubKeys, r.details.pubKey)
					sigs = append(sigs, r.sig)
					aggSignersMask |= r.details.signersMask

					successfullyStoredCount++
				}
			}

			// As soon as enough responses are returned, pass the response to
			// certDetailsChan, so the Store function can return, but also continue
			// running until all responses are received (or the context is canceled)
			// in order to produce accurate logs/metrics.
			if !returned {
				if successfullyStoredCount >= a.requiredServicesForStore {
					cd := certDetails{}
					cd.pubKeys = append(cd.pubKeys, pubKeys...)
					cd.sigs = append(cd.sigs, sigs...)
					cd.aggSignersMask = aggSignersMask
					certDetailsChan <- cd
					returned = true
				} else if storeFailures > a.maxAllowedServiceStoreFailures {
					cd := certDetails{}
					cd.err = fmt.Errorf("aggregator failed to store message to at least %d out of %d DASes (assuming %d are honest). %w", a.requiredServicesForStore, len(a.services), a.config.AssumedHonest, BatchToDasFailed)
					certDetailsChan <- cd
					returned = true
				}
			}

		}
	}()

	cd := <-certDetailsChan

	if cd.err != nil {
		return nil, cd.err
	}

	aggCert.Sig = blsSignatures.AggregateSignatures(cd.sigs)
	aggPubKey := blsSignatures.AggregatePublicKeys(cd.pubKeys)
	aggCert.SignersMask = cd.aggSignersMask

	aggCert.DataHash = expectedHash
	aggCert.Timeout = timeout
	aggCert.KeysetHash = a.keysetHash
	aggCert.Version = 1

	verified, err := blsSignatures.VerifySignature(aggCert.Sig, aggCert.SerializeSignableFields(), aggPubKey)
	if err != nil {
		return nil, fmt.Errorf("%w. %w", err, BatchToDasFailed)
	}
	if !verified {
		return nil, fmt.Errorf("failed aggregate signature check. %w", BatchToDasFailed)
	}
	return &aggCert, nil
}

func (a *Aggregator) String() string {
	var b bytes.Buffer
	b.WriteString("das.Aggregator{")
	first := true
	for _, d := range a.services {
		if !first {
			b.WriteString(",")
		}
		b.WriteString(fmt.Sprintf("signersMask(aggregator):%d,", d.signersMask))
		b.WriteString(d.service.String())
	}
	b.WriteString("}")
	return b.String()
}
