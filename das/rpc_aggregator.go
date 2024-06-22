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

	"github.com/knadh/koanf"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/offchainlabs/nitro/arbstate/daprovider"
	"github.com/offchainlabs/nitro/blsSignatures"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/util/metricsutil"
	"github.com/offchainlabs/nitro/util/signature"

	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/arbutil"
)

type BackendConfig struct {
	URL    string `koanf:"url" json:"url"`
	Pubkey string `koanf:"pubkey" json:"pubkey"`
}

type BackendConfigList []BackendConfig

func (l *BackendConfigList) String() string {
	b, _ := json.Marshal(*l)
	return string(b)
}

func (l *BackendConfigList) Set(value string) error {
	return l.UnmarshalJSON([]byte(value))
}

func (l *BackendConfigList) UnmarshalJSON(data []byte) error {
	var tmp []BackendConfig
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	*l = tmp
	return nil
}

func (l *BackendConfigList) Type() string {
	return "backendConfigList"
}

func FixKeysetCLIParsing(path string, k *koanf.Koanf) error {
	rawBackends := k.Get(path)
	if bk, ok := rawBackends.(string); ok {
		err := parsedBackendsConf.UnmarshalJSON([]byte(bk))
		if err != nil {
			return err
		}

		// Create a map with the parsed backend configurations
		tempMap := map[string]interface{}{
			path: parsedBackendsConf,
		}

		// Load the map into koanf
		if err = k.Load(confmap.Provider(tempMap, "."), nil); err != nil {
			return err
		}

	}
	return nil
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
	var services []ServiceDetails

	for i, b := range config.Backends {
		url, err := url.Parse(b.URL)
		if err != nil {
			return nil, err
		}
		metricName := metricsutil.CanonicalizeMetricName(url.Hostname())

		service, err := NewDASRPCClient(b.URL, signer, config.MaxStoreChunkBodySize)
		if err != nil {
			return nil, err
		}

		pubKey, err := DecodeBase64BLSPublicKey([]byte(b.Pubkey))
		if err != nil {
			return nil, err
		}

		d, err := NewServiceDetails(service, *pubKey, 1<<uint64(i), metricName)
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
