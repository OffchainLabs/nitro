// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package dasrpc

import (
	"encoding/json"

	"github.com/offchainlabs/nitro/solgen/go/bridgegen"

	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/arbutil"

	"github.com/offchainlabs/nitro/das"
)

type BackendConfig struct {
	URL                 string `json:"url"`
	PubKeyBase64Encoded string `json:"pubkey"`
	SignerMask          uint64 `json:"signermask"`
}

func NewRPCAggregator(config das.AggregatorConfig) (*das.Aggregator, error) {
	services, err := setUpServices(config)
	if err != nil {
		return nil, err
	}
	return das.NewAggregator(config, services)
}

func NewRPCAggregatorWithL1Info(config das.AggregatorConfig, l1client arbutil.L1Interface, seqInboxAddress common.Address) (*das.Aggregator, error) {
	services, err := setUpServices(config)
	if err != nil {
		return nil, err
	}
	return das.NewAggregatorWithL1Info(config, services, l1client, seqInboxAddress)
}

func NewRPCAggregatorWithSeqInboxCaller(config das.AggregatorConfig, seqInboxCaller *bridgegen.SequencerInboxCaller) (*das.Aggregator, error) {
	services, err := setUpServices(config)
	if err != nil {
		return nil, err
	}
	return das.NewAggregatorWithSeqInboxCaller(config, services, seqInboxCaller)
}

func setUpServices(config das.AggregatorConfig) ([]das.ServiceDetails, error) {
	var cs []BackendConfig
	err := json.Unmarshal([]byte(config.Backends), &cs)
	if err != nil {
		return nil, err
	}

	var services []das.ServiceDetails

	for _, b := range cs {
		service, err := NewDASRPCClient(b.URL)
		if err != nil {
			return nil, err
		}

		pubKey, err := das.DecodeBase64BLSPublicKey([]byte(b.PubKeyBase64Encoded))
		if err != nil {
			return nil, err
		}

		d, err := das.NewServiceDetails(service, *pubKey, uint64(b.SignerMask))
		if err != nil {
			return nil, err
		}

		services = append(services, *d)
	}
	return services, nil
}
