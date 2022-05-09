// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math/bits"
	"strconv"

	flag "github.com/spf13/pflag"

	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/blsSignatures"
	"github.com/offchainlabs/nitro/cmd/conf"
)

type DataAvailabilityServiceWriter interface {
	// Requests that the message be stored until timeout (UTC time in unix epoch seconds).
	Store(ctx context.Context, message []byte, timeout uint64) (*arbstate.DataAvailabilityCertificate, error)
	PrivateKey() blsSignatures.PrivateKey
}

type DataAvailabilityService interface {
	arbstate.DataAvailabilityServiceReader
	DataAvailabilityServiceWriter
	fmt.Stringer
}

type DataAvailabilityMode uint64

const (
	OnchainDataAvailability DataAvailabilityMode = iota
	LocalDataAvailability
)

type DataAvailabilityConfig struct {
	ModeImpl         string        `koanf:"mode"`
	LocalDiskDataDir string        `koanf:"local-disk-data-dir"`
	S3Config         conf.S3Config `koanf:"s3"`
	RedisConfig      RedisConfig   `koanf:"redis"`
	SignerMask       SignerMask    `koanf:"signer-mask"`
}

var DefaultDataAvailabilityConfig = DataAvailabilityConfig{
	ModeImpl:         "onchain",
	LocalDiskDataDir: "",
	S3Config:         conf.DefaultS3Config,
	RedisConfig:      DefaultRedisConfig,
	SignerMask:       1,
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

type SignerMask uint64

func (m *SignerMask) String() string {
	return fmt.Sprintf("%X", *m)
}

func (m *SignerMask) Set(s string) error {
	res, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return err
	}
	if bits.OnesCount64(res) != 1 {
		return fmt.Errorf("Got invalid SignerMask %s (%X), must have only 1 bit set, had %d.", s, res, bits.OnesCount64(res))
	}
	return nil
}

func (m *SignerMask) Type() string {
	return "SignerMask"
}

func DataAvailabilityConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.String(prefix+".mode", DefaultDataAvailabilityConfig.ModeImpl, "mode (onchain or local)")
	f.String(prefix+".local-disk-data-dir", DefaultDataAvailabilityConfig.LocalDiskDataDir, "For local mode, the directory of the data store")
	f.Var(&DefaultDataAvailabilityConfig.SignerMask, prefix+".signer-mask", "Single bit uint64 unique for this DAS.")
}

func serializeSignableFields(c arbstate.DataAvailabilityCertificate) []byte {
	buf := make([]byte, 0, 32+8)
	buf = append(buf, c.DataHash[:]...)

	var intData [8]byte
	binary.BigEndian.PutUint64(intData[:], c.Timeout)
	buf = append(buf, intData[:]...)

	return buf
}

func Serialize(c arbstate.DataAvailabilityCertificate) []byte {
	buf := make([]byte, 0)

	buf = append(buf, arbstate.DASMessageHeaderFlag)

	buf = append(buf, serializeSignableFields(c)...)

	var intData [8]byte
	binary.BigEndian.PutUint64(intData[:], c.SignersMask)
	buf = append(buf, intData[:]...)

	return append(buf, blsSignatures.SignatureToBytes(c.Sig)...)
}
