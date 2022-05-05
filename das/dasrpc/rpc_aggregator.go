// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package dasrpc

import "github.com/offchainlabs/nitro/das"

func NewRPCAggregator(config das.AggregatorConfig) (*das.Aggregator, error) {
	var services []das.ServiceDetails
	for _, b := range config.Backends {
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

	return das.NewAggregator(config, services)
}
