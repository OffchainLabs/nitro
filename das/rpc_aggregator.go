// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"context"
	"encoding/json"

	"github.com/offchainlabs/nitro/solgen/go/bridgegen"

	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/arbutil"
)

type BackendConfig struct {
	URL                 string `json:"url"`
	PubKeyBase64Encoded string `json:"pubkey"`
	SignerMask          uint64 `json:"signermask"`
}

func NewRPCAggregator(ctx context.Context, config DataAvailabilityConfig) (*Aggregator, error) {
	services, err := setUpServices(config.AggregatorConfig)
	if err != nil {
		return nil, err
	}
	return NewAggregator(ctx, config, services)
}

func NewRPCAggregatorWithL1Info(config AggregatorConfig, l1client arbutil.L1Interface, seqInboxAddress common.Address) (*Aggregator, error) {
	services, err := setUpServices(config)
	if err != nil {
		return nil, err
	}
	return NewAggregatorWithL1Info(config, services, l1client, seqInboxAddress)
}

func NewRPCAggregatorWithSeqInboxCaller(config AggregatorConfig, seqInboxCaller *bridgegen.SequencerInboxCaller) (*Aggregator, error) {
	services, err := setUpServices(config)
	if err != nil {
		return nil, err
	}
	return NewAggregatorWithSeqInboxCaller(config, services, seqInboxCaller)
}

func setUpServices(config AggregatorConfig) ([]ServiceDetails, error) {
	var cs []BackendConfig
	err := json.Unmarshal([]byte(config.Backends), &cs)
	if err != nil {
		return nil, err
	}

	var services []ServiceDetails

	for _, b := range cs {
		service, err := NewDASRPCClient(b.URL)
		if err != nil {
			return nil, err
		}

		pubKey, err := DecodeBase64BLSPublicKey([]byte(b.PubKeyBase64Encoded))
		if err != nil {
			return nil, err
		}

		d, err := NewServiceDetails(service, *pubKey, uint64(b.SignerMask))
		if err != nil {
			return nil, err
		}

		services = append(services, *d)
	}

	return services, nil
}
