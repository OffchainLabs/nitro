// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"context"
	"encoding/binary"
	"errors"

	flag "github.com/spf13/pflag"

	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/blsSignatures"
)

type DataAvailabilityServiceWriter interface {
	Store(ctx context.Context, message []byte) (*arbstate.DataAvailabilityCertificate, error)
}

type DataAvailabilityService interface {
	arbstate.DataAvailabilityServiceReader
	DataAvailabilityServiceWriter
}

type DataAvailabilityMode uint64

const (
	OnchainDataAvailability DataAvailabilityMode = iota
	LocalDataAvailability
)

type DataAvailabilityConfig struct {
	ModeImpl         string `koanf:"mode"`
	LocalDiskDataDir string `koanf:"local-disk-data-dir"`
}

var DefaultDataAvailabilityConfig = DataAvailabilityConfig{
	ModeImpl:         "onchain",
	LocalDiskDataDir: "",
}

func (c *DataAvailabilityConfig) Mode() (DataAvailabilityMode, error) {
	if c.ModeImpl == "" {
		return 0, errors.New("--data-availability.mode missing")
	}

	if c.ModeImpl == "onchain" {
		return OnchainDataAvailability, nil
	}

	if c.ModeImpl == "local" {
		if c.LocalDiskDataDir == "" {
			flag.Usage()
			return 0, errors.New("--data-availability.local-disk-data-dir must be specified if mode is set to local")
		}
		return LocalDataAvailability, nil
	}

	flag.Usage()
	return 0, errors.New("--data-availability.mode " + c.ModeImpl + " not recognized")
}

func DataAvailabilityConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.String(prefix+".mode", DefaultDataAvailabilityConfig.ModeImpl, "mode (onchain or local)")
	f.String(prefix+".local-disk-data-dir", DefaultDataAvailabilityConfig.LocalDiskDataDir, "For local mode, the directory of the data store")
}

func serializeSignableFields(c arbstate.DataAvailabilityCertificate) []byte {
	buf := make([]byte, 0, 32+8+8)
	buf = append(buf, c.DataHash[:]...)

	var intData [8]byte
	binary.BigEndian.PutUint64(intData[:], c.Timeout)
	buf = append(buf, intData[:]...)

	binary.BigEndian.PutUint64(intData[:], c.SignersMask)
	buf = append(buf, intData[:]...)
	return buf
}

func Serialize(c arbstate.DataAvailabilityCertificate) []byte {
	buf := make([]byte, 0)

	buf = append(buf, arbstate.DASMessageHeaderFlag)

	buf = append(buf, serializeSignableFields(c)...)

	return append(buf, blsSignatures.SignatureToBytes(c.Sig)...)
}
