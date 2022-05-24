// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"reflect"

	"github.com/ethereum/go-ethereum/common"
	flag "github.com/spf13/pflag"

	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/blsSignatures"
)

type DataAvailabilityServiceWriter interface {
	// Requests that the message be stored until timeout (UTC time in unix epoch seconds).
	Store(ctx context.Context, message []byte, timeout uint64, sig []byte) (*arbstate.DataAvailabilityCertificate, error)
}

type DataAvailabilityService interface {
	arbstate.DataAvailabilityServiceReader
	DataAvailabilityServiceWriter
	fmt.Stringer
}

type DataAvailabilityMode uint64

const (
	OnchainDataAvailability DataAvailabilityMode = iota
	DASDataAvailability
	AggregatorDataAvailability
	// TODO RemoteDataAvailability
)

const (
	OnchainDataAvailabilityString    = "onchain"
	DASDataAvailabilityString        = "das"
	AggregatorDataAvailabilityString = "aggregator"
	// TODO RemoteDataAvailability
)

type DataAvailabilityConfig struct {
	ModeImpl         string           `koanf:"mode"`
	DASConfig        StorageConfig    `koanf:"das"`
	AggregatorConfig AggregatorConfig `koanf:"aggregator"`
}

var DefaultDataAvailabilityConfig = DataAvailabilityConfig{
	ModeImpl: OnchainDataAvailabilityString,
}

func (c *DataAvailabilityConfig) Mode() (DataAvailabilityMode, error) {
	if c.ModeImpl == "" {
		return 0, errors.New("--data-availability.mode missing")
	}

	if c.ModeImpl == OnchainDataAvailabilityString {
		return OnchainDataAvailability, nil
	}

	if c.ModeImpl == DASDataAvailabilityString {
		if c.DASConfig.LocalConfig.DataDir == "" || (c.DASConfig.KeyDir == "" && c.DASConfig.PrivKey == "") {
			flag.Usage()
			return 0, errors.New("--data-availability.das.local.data-dir and --data-availability.das.key-dir must be specified if mode is set to das")
		}
		return DASDataAvailability, nil
	}

	if c.ModeImpl == AggregatorDataAvailabilityString {
		if reflect.DeepEqual(c.AggregatorConfig, DefaultAggregatorConfig) {
			flag.Usage()
			return 0, errors.New("--data-availability.aggregator.X config options must be specified if mode is set to aggregator")
		}
		return AggregatorDataAvailability, nil
	}

	flag.Usage()
	return 0, errors.New("--data-availability.mode " + c.ModeImpl + " not recognized")
}

func OptionalAddressFromString(s string) (*common.Address, error) {
	if s == "none" {
		return nil, nil
	}
	if s == "" {
		return nil, errors.New("must provide address for signer or specify 'none'")
	}
	if !common.IsHexAddress(s) {
		return nil, fmt.Errorf("invalid address for signer: %v", s)
	}
	addr := common.HexToAddress(s)
	return &addr, nil
}

func DataAvailabilityConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.String(prefix+".mode", DefaultDataAvailabilityConfig.ModeImpl, "mode ('onchain', 'das', or 'aggregator')")
	StorageConfigAddOptions(prefix+".das", f)
	AggregatorConfigAddOptions(prefix+".aggregator", f)
}

func serializeSignableFields(c *arbstate.DataAvailabilityCertificate) []byte {
	buf := make([]byte, 0, 32+8)
	buf = append(buf, c.DataHash[:]...)

	var intData [8]byte
	binary.BigEndian.PutUint64(intData[:], c.Timeout)
	buf = append(buf, intData[:]...)

	return buf
}

func Serialize(c *arbstate.DataAvailabilityCertificate) []byte {
	buf := make([]byte, 0)

	buf = append(buf, arbstate.DASMessageHeaderFlag)

	buf = append(buf, c.KeysetHash[:]...)

	buf = append(buf, serializeSignableFields(c)...)

	var intData [8]byte
	binary.BigEndian.PutUint64(intData[:], c.SignersMask)
	buf = append(buf, intData[:]...)

	return append(buf, blsSignatures.SignatureToBytes(c.Sig)...)
}
