// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/bits"
	"net/url"

	"github.com/offchainlabs/nitro/arbstate/daprovider"
	"github.com/offchainlabs/nitro/blsSignatures"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/util/metricsutil"
	"github.com/offchainlabs/nitro/util/signature"

	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/arbutil"
)

type BackendConfig struct {
	URL                 string `json:"url"`
	PubKeyBase64Encoded string `json:"pubkey"`
	SignerMask          uint64 `json:"signermask"`
}

func NewRPCAggregator(ctx context.Context, config DataAvailabilityConfig, signer signature.DataSignerFunc) (*Aggregator, error) {
	services, err := ParseServices(config.RPCAggregator, signer)
	if err != nil {
		return nil, err
	}
	return NewAggregator(ctx, config, services)
}

func NewRPCAggregatorWithL1Info(config DataAvailabilityConfig, l1client arbutil.L1Interface, seqInboxAddress common.Address, signer signature.DataSignerFunc) (*Aggregator, error) {
	services, err := ParseServices(config.RPCAggregator, signer)
	if err != nil {
		return nil, err
	}
	return NewAggregatorWithL1Info(config, services, l1client, seqInboxAddress)
}

func NewRPCAggregatorWithSeqInboxCaller(config DataAvailabilityConfig, seqInboxCaller *bridgegen.SequencerInboxCaller, signer signature.DataSignerFunc) (*Aggregator, error) {
	services, err := ParseServices(config.RPCAggregator, signer)
	if err != nil {
		return nil, err
	}
	return NewAggregatorWithSeqInboxCaller(config, services, seqInboxCaller)
}

func ParseServices(config AggregatorConfig, signer signature.DataSignerFunc) ([]ServiceDetails, error) {
	var cs []BackendConfig
	err := json.Unmarshal([]byte(config.Backends), &cs)
	if err != nil {
		return nil, err
	}

	var services []ServiceDetails

	for _, b := range cs {
		url, err := url.Parse(b.URL)
		if err != nil {
			return nil, err
		}
		metricName := metricsutil.CanonicalizeMetricName(url.Hostname())

		service, err := NewDASRPCClient(b.URL, signer, config.MaxStoreChunkBodySize)
		if err != nil {
			return nil, err
		}

		pubKey, err := DecodeBase64BLSPublicKey([]byte(b.PubKeyBase64Encoded))
		if err != nil {
			return nil, err
		}

		d, err := NewServiceDetails(service, *pubKey, b.SignerMask, metricName)
		if err != nil {
			return nil, err
		}

		services = append(services, *d)
	}

	return services, nil
}

func KeysetHashFromServices(services []ServiceDetails, assumedHonest uint64) ([32]byte, []byte, error) {
	var aggSignersMask uint64
	pubKeys := []blsSignatures.PublicKey{}
	for _, d := range services {
		if bits.OnesCount64(d.signersMask) != 1 {
			return [32]byte{}, nil, fmt.Errorf("tried to configure backend DAS %v with invalid signersMask %X", d.service, d.signersMask)
		}
		aggSignersMask |= d.signersMask
		pubKeys = append(pubKeys, d.pubKey)
	}
	if bits.OnesCount64(aggSignersMask) != len(services) {
		return [32]byte{}, nil, errors.New("at least two signers share a mask")
	}

	keyset := &daprovider.DataAvailabilityKeyset{
		AssumedHonest: uint64(assumedHonest),
		PubKeys:       pubKeys,
	}
	ksBuf := bytes.NewBuffer([]byte{})
	if err := keyset.Serialize(ksBuf); err != nil {
		return [32]byte{}, nil, err
	}
	keysetHash, err := keyset.Hash()
	if err != nil {
		return [32]byte{}, nil, err
	}

	return keysetHash, ksBuf.Bytes(), nil
}
