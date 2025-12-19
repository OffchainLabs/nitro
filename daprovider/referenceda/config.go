// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package referenceda

import (
	flag "github.com/spf13/pflag"
)

type Config struct {
	Enable            bool             `koanf:"enable"`
	SigningKey        SigningKeyConfig `koanf:"signing-key"`
	ValidatorContract string           `koanf:"validator-contract"`
	MaxBatchSize      int              `koanf:"max-batch-size"`
}

type SigningKeyConfig struct {
	PrivateKey string `koanf:"private-key"`
	KeyFile    string `koanf:"key-file"`
}

var DefaultSigningKeyConfig = SigningKeyConfig{
	PrivateKey: "",
	KeyFile:    "",
}

var DefaultConfig = Config{
	Enable:            false,
	SigningKey:        DefaultSigningKeyConfig,
	ValidatorContract: "",
	MaxBatchSize:      1_000_000, // 1MB default
}

func SigningKeyConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.String(prefix+".private-key", DefaultSigningKeyConfig.PrivateKey, "hex-encoded private key for signing certificates")
	f.String(prefix+".key-file", DefaultSigningKeyConfig.KeyFile, "path to file containing private key")
}

func ConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".enable", DefaultConfig.Enable, "enable reference DA provider implementation")
	SigningKeyConfigAddOptions(prefix+".signing-key", f)
	f.String(prefix+".validator-contract", DefaultConfig.ValidatorContract, "address of the ReferenceDAProofValidator contract")
	f.Int(prefix+".max-batch-size", DefaultConfig.MaxBatchSize, "maximum batch size for reference DA")
}
