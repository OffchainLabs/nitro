// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/bits"
	"sync/atomic"
	"time"

	flag "github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"

	"github.com/offchainlabs/nitro/arbstate/daprovider"
	"github.com/offchainlabs/nitro/blsSignatures"
	"github.com/offchainlabs/nitro/das/dastree"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/util/pretty"
)

const metricBase string = "arb/das/rpc/aggregator/store"

var (
	// This metric shows 1 if there was any error posting to the backends, until
	// there was a Store that had no backend failures.
	anyErrorGauge = metrics.GetOrRegisterGauge(metricBase+"/error/gauge", nil)

// Other aggregator metrics are generated dynamically in the Store function.
)

type AggregatorConfig struct {
	Enable                bool              `koanf:"enable"`
	AssumedHonest         int               `koanf:"assumed-honest"`
	Backends              BackendConfigList `koanf:"backends"`
	MaxStoreChunkBodySize int               `koanf:"max-store-chunk-body-size"`
	EnableChunkedStore    bool              `koanf:"enable-chunked-store"`
}

var DefaultAggregatorConfig = AggregatorConfig{
	AssumedHonest:         0,
	Backends:              nil,
	MaxStoreChunkBodySize: 512 * 1024,
	EnableChunkedStore:    true,
}

var parsedBackendsConf BackendConfigList

func AggregatorConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".enable", DefaultAggregatorConfig.Enable, "enable storage of sequencer batch data from a list of RPC endpoints; this should only be used by the batch poster and not in combination with other DAS storage types")
	f.Int(prefix+".assumed-honest", DefaultAggregatorConfig.AssumedHonest, "Number of assumed honest backends (H). If there are N backends, K=N+1-H valid responses are required to consider an Store request to be successful.")
	f.Var(&parsedBackendsConf, prefix+".backends", "JSON RPC backend configuration. This can be specified on the command line as a JSON array, eg: [{\"url\": \"...\", \"pubkey\": \"...\"},...], or as a JSON array in the config file.")
	f.Int(prefix+".max-store-chunk-body-size", DefaultAggregatorConfig.MaxStoreChunkBodySize, "maximum HTTP POST body size to use for individual batch chunks, including JSON RPC overhead and an estimated overhead of 512B of headers")
	f.Bool(prefix+".enable-chunked-store", DefaultAggregatorConfig.EnableChunkedStore, "enable data to be sent to DAS in chunks instead of all at once")
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
	if config.ParentChainNodeURL == "none" {
		return NewAggregatorWithSeqInboxCaller(config, services, nil)
	}
	l1client, err := GetL1Client(ctx, config.ParentChainConnectionAttempts, config.ParentChainNodeURL)
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
	l1client *ethclient.Client,
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

	// #nosec G115
	keysetHash, keysetBytes, err := KeysetHashFromServices(services, uint64(config.RPCAggregator.AssumedHonest))
	if err != nil {
		return nil, err
	}

	return &Aggregator{
		config:                         config.RPCAggregator,
		services:                       services,
		requestTimeout:                 config.RequestTimeout,
		requiredServicesForStore:       len(services) + 1 - config.RPCAggregator.AssumedHonest,
		maxAllowedServiceStoreFailures: config.RPCAggregator.AssumedHonest - 1,
		keysetHash:                     keysetHash,
		keysetBytes:                    keysetBytes,
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
func (a *Aggregator) Store(ctx context.Context, message []byte, timeout uint64) (*daprovider.DataAvailabilityCertificate, error) {
	// #nosec G115
	log.Trace("das.Aggregator.Store", "message", pretty.FirstFewBytes(message), "timeout", time.Unix(int64(timeout), 0))

	allBackendsSucceeded := false
	defer func() {
		if allBackendsSucceeded {
			anyErrorGauge.Update(0)
		} else {
			anyErrorGauge.Update(1)
		}
	}()

	responses := make(chan storeResponse, len(a.services))

	expectedHash := dastree.Hash(message)
	for _, d := range a.services {
		go func(ctx context.Context, d ServiceDetails) {
			storeCtx, cancel := context.WithTimeout(ctx, a.requestTimeout)
			var metricWithServiceName = metricBase + "/" + d.metricName
			defer cancel()
			incFailureMetric := func() {
				metrics.GetOrRegisterCounter(metricWithServiceName+"/error/total", nil).Inc(1)
				metrics.GetOrRegisterCounter(metricBase+"/error/all/total", nil).Inc(1)
			}

			cert, err := d.service.Store(storeCtx, message, timeout)
			if err != nil {
				incFailureMetric()
				log.Warn("DAS Aggregator failed to store batch to backend", "backend", d.metricName, "err", err)
				responses <- storeResponse{d, nil, err}
				return
			}

			verified, err := blsSignatures.VerifySignature(
				cert.Sig, cert.SerializeSignableFields(), d.pubKey,
			)
			if err != nil {
				incFailureMetric()
				log.Warn("DAS Aggregator couldn't parse backend's store response signature", "backend", d.metricName, "err", err)
				responses <- storeResponse{d, nil, err}
				return
			}
			if !verified {
				incFailureMetric()
				log.Warn("DAS Aggregator failed to verify backend's store response signature", "backend", d.metricName, "err", err)
				responses <- storeResponse{d, nil, errors.New("signature verification failed")}
				return
			}

			// SignersMask from backend DAS is ignored.

			if cert.DataHash != expectedHash {
				incFailureMetric()
				log.Warn("DAS Aggregator got a store response with a data hash not matching the expected hash", "backend", d.metricName, "dataHash", cert.DataHash, "expectedHash", expectedHash, "err", err)
				responses <- storeResponse{d, nil, errors.New("hash verification failed")}
				return
			}
			if cert.Timeout != timeout {
				incFailureMetric()
				log.Warn("DAS Aggregator got a store response with any expiry time not matching the expected expiry time", "backend", d.metricName, "dataHash", cert.DataHash, "expectedHash", expectedHash, "err", err)
				responses <- storeResponse{d, nil, fmt.Errorf("timeout was %d, expected %d", cert.Timeout, timeout)}
				return
			}

			metrics.GetOrRegisterCounter(metricWithServiceName+"/success/total", nil).Inc(1)
			metrics.GetOrRegisterCounter(metricBase+"/success/all/total", nil).Inc(1)
			responses <- storeResponse{d, cert.Sig, nil}
		}(ctx, d)
	}

	var aggCert daprovider.DataAvailabilityCertificate

	type certDetails struct {
		pubKeys        []blsSignatures.PublicKey
		sigs           []blsSignatures.Signature
		aggSignersMask uint64
		err            error
	}

	var storeFailures atomic.Int64
	// Collect responses from backends.
	certDetailsChan := make(chan certDetails)
	go func() {
		var pubKeys []blsSignatures.PublicKey
		var sigs []blsSignatures.Signature
		var aggSignersMask uint64
		var successfullyStoredCount int
		var returned int // 0-no status, 1-succeeded, 2-failed
		for i := 0; i < len(a.services); i++ {
			select {
			case <-ctx.Done():
				break
			case r := <-responses:
				if r.err != nil {
					_ = storeFailures.Add(1)
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
			if returned == 0 {
				if successfullyStoredCount >= a.requiredServicesForStore {
					cd := certDetails{}
					cd.pubKeys = append(cd.pubKeys, pubKeys...)
					cd.sigs = append(cd.sigs, sigs...)
					cd.aggSignersMask = aggSignersMask
					certDetailsChan <- cd
					returned = 1
				} else if int(storeFailures.Load()) > a.maxAllowedServiceStoreFailures {
					cd := certDetails{}
					cd.err = fmt.Errorf("aggregator failed to store message to at least %d out of %d DASes (assuming %d are honest). %w", a.requiredServicesForStore, len(a.services), a.config.AssumedHonest, daprovider.ErrBatchToDasFailed)
					certDetailsChan <- cd
					returned = 2
				}
			}
		}
		if returned == 1 &&
			a.maxAllowedServiceStoreFailures > 0 && // Ignore the case where AssumedHonest = 1, probably a testnet
			int(storeFailures.Load())+1 > a.maxAllowedServiceStoreFailures {
			log.Error("das.Aggregator: storing the batch data succeeded to enough DAS commitee members to generate the Data Availability Cert, but if one more had failed then the cert would not have been able to be generated. Look for preceding logs with \"Error from backend\"")
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
		//nolint:errorlint
		return nil, fmt.Errorf("%s. %w", err.Error(), daprovider.ErrBatchToDasFailed)
	}
	if !verified {
		return nil, fmt.Errorf("failed aggregate signature check. %w", daprovider.ErrBatchToDasFailed)
	}

	if storeFailures.Load() == 0 {
		allBackendsSucceeded = true
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
