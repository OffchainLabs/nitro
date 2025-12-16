// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package anytrust

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/bits"
	"sync/atomic"
	"time"

	"github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"

	"github.com/offchainlabs/nitro/blsSignatures"
	"github.com/offchainlabs/nitro/daprovider/anytrust/tree"
	anytrustutil "github.com/offchainlabs/nitro/daprovider/anytrust/util"
	"github.com/offchainlabs/nitro/daprovider/data_streaming"
	"github.com/offchainlabs/nitro/util/pretty"
	"github.com/offchainlabs/nitro/util/rpcclient"
)

// Metric path keeps "das" for backward compatibility with existing dashboards
const metricBase string = "arb/das/rpc/aggregator/store"

var (
	// This metric shows 1 if there was any error posting to the backends, until
	// there was a Store that had no backend failures.
	anyErrorGauge = metrics.GetOrRegisterGauge(metricBase+"/error/gauge", nil)

	// Other aggregator metrics are generated dynamically in the Store function.
)

type AggregatorConfig struct {
	Enable        bool              `koanf:"enable"`
	AssumedHonest int               `koanf:"assumed-honest"`
	Backends      BackendConfigList `koanf:"backends"`
	RPCClient     RPCClientConfig   `koanf:"rpc-client"`
}

var DefaultAggregatorConfig = AggregatorConfig{
	AssumedHonest: 0,
	Backends:      nil,
	RPCClient: RPCClientConfig{
		EnableChunkedStore: true,
		DataStream:         data_streaming.DefaultDataStreamerConfig(DefaultDataStreamRpcMethods),
		RPC:                rpcclient.DefaultClientConfig,
	},
}

var parsedBackendsConf BackendConfigList

func AggregatorConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Bool(prefix+".enable", DefaultAggregatorConfig.Enable, "enable storage of sequencer batch data from a list of RPC endpoints; this should only be used by the batch poster and not in combination with other AnyTrust storage types")
	f.Int(prefix+".assumed-honest", DefaultAggregatorConfig.AssumedHonest, "Number of assumed honest backends (H). If there are N backends, K=N+1-H valid responses are required to consider an Store request to be successful.")
	f.Var(&parsedBackendsConf, prefix+".backends", "JSON RPC backend configuration. This can be specified on the command line as a JSON array, eg: [{\"url\": \"...\", \"pubkey\": \"...\"},...], or as a JSON array in the config file.")
	RPCClientConfigAddOptions(prefix+".rpc-client", f)
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
	service     anytrustutil.Writer
	pubKey      blsSignatures.PublicKey
	signersMask uint64
	metricName  string
}

func (s *ServiceDetails) String() string {
	return fmt.Sprintf("ServiceDetails{service: %v, signersMask %d}", s.service, s.signersMask)
}

func NewServiceDetails(service anytrustutil.Writer, pubKey blsSignatures.PublicKey, signersMask uint64, metricName string) (*ServiceDetails, error) {
	if bits.OnesCount64(signersMask) != 1 {
		return nil, fmt.Errorf("tried to configure backend AnyTrust service %v with invalid signersMask %X", service, signersMask)
	}
	return &ServiceDetails{
		service:     service,
		pubKey:      pubKey,
		signersMask: signersMask,
		metricName:  metricName,
	}, nil
}

func newAggregator(
	config Config,
	services []ServiceDetails,
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

// Store calls Store on each backend AnyTrust service in parallel and collects responses.
// If there were at least K responses then it aggregates the signatures and
// signersMasks from each service together into the DataAvailabilityCertificate
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
func (a *Aggregator) Store(ctx context.Context, message []byte, timeout uint64) (*anytrustutil.DataAvailabilityCertificate, error) {
	// #nosec G115
	log.Trace("anytrust.Aggregator.Store", "message", pretty.FirstFewBytes(message), "timeout", time.Unix(int64(timeout), 0))

	allBackendsSucceeded := false
	defer func() {
		if allBackendsSucceeded {
			anyErrorGauge.Update(0)
		} else {
			anyErrorGauge.Update(1)
		}
	}()

	responses := make(chan storeResponse, len(a.services))

	expectedHash := tree.Hash(message)
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
				log.Warn("AnyTrust Aggregator failed to store batch to backend", "backend", d.metricName, "err", err)
				responses <- storeResponse{d, nil, err}
				return
			}

			verified, err := blsSignatures.VerifySignature(
				cert.Sig, cert.SerializeSignableFields(), d.pubKey,
			)
			if err != nil {
				incFailureMetric()
				log.Warn("AnyTrust Aggregator couldn't parse backend's store response signature", "backend", d.metricName, "err", err)
				responses <- storeResponse{d, nil, err}
				return
			}
			if !verified {
				incFailureMetric()
				log.Warn("AnyTrust Aggregator failed to verify backend's store response signature", "backend", d.metricName, "err", err)
				responses <- storeResponse{d, nil, errors.New("signature verification failed")}
				return
			}

			// SignersMask from backend AnyTrust is ignored.

			if cert.DataHash != expectedHash {
				incFailureMetric()
				log.Warn("AnyTrust Aggregator got a store response with a data hash not matching the expected hash", "backend", d.metricName, "dataHash", cert.DataHash, "expectedHash", expectedHash, "err", err)
				responses <- storeResponse{d, nil, errors.New("hash verification failed")}
				return
			}
			if cert.Timeout != timeout {
				incFailureMetric()
				log.Warn("AnyTrust Aggregator got a store response with an expiry time not matching the expected expiry time", "backend", d.metricName, "dataHash", cert.DataHash, "expectedHash", expectedHash, "err", err)
				responses <- storeResponse{d, nil, fmt.Errorf("timeout was %d, expected %d", cert.Timeout, timeout)}
				return
			}

			metrics.GetOrRegisterCounter(metricWithServiceName+"/success/total", nil).Inc(1)
			metrics.GetOrRegisterCounter(metricBase+"/success/all/total", nil).Inc(1)
			responses <- storeResponse{d, cert.Sig, nil}
		}(ctx, d)
	}

	var aggCert anytrustutil.DataAvailabilityCertificate

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
					log.Warn("anytrust.Aggregator: Error from backend", "backend", r.details.service, "signerMask", r.details.signersMask, "err", r.err)
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
					cd.err = fmt.Errorf("aggregator failed to store message to at least %d out of %d AnyTrust backends (assuming %d are honest). %w", a.requiredServicesForStore, len(a.services), a.config.AssumedHonest, anytrustutil.ErrBatchFailed)
					certDetailsChan <- cd
					returned = 2
				}
			}
		}
		if returned == 1 &&
			a.maxAllowedServiceStoreFailures > 0 && // Ignore the case where AssumedHonest = 1, probably a testnet
			int(storeFailures.Load())+1 > a.maxAllowedServiceStoreFailures {
			log.Error("anytrust.Aggregator: storing the batch data succeeded to enough AnyTrust committee members to generate the Data Availability Cert, but if one more had failed then the cert would not have been able to be generated. Look for preceding logs with \"Error from backend\"")
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
		return nil, fmt.Errorf("%s. %w", err.Error(), anytrustutil.ErrBatchFailed)
	}
	if !verified {
		return nil, fmt.Errorf("failed aggregate signature check. %w", anytrustutil.ErrBatchFailed)
	}

	if storeFailures.Load() == 0 {
		allBackendsSucceeded = true
	}

	return &aggCert, nil
}

func (a *Aggregator) String() string {
	var b bytes.Buffer
	b.WriteString("anytrust.Aggregator{")
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
