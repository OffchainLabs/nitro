// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package referenceda

import (
	flag "github.com/spf13/pflag"
)

type Config struct {
	Enable                        bool             `koanf:"enable"`
	SigningKey                    SigningKeyConfig `koanf:"signing-key"`
	ValidatorContract             string           `koanf:"validator-contract"`
	ParentChainNodeURL            string           `koanf:"parent-chain-node-url"`
	ParentChainConnectionAttempts int              `koanf:"parent-chain-connection-attempts"`
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
	Enable:                        false,
	SigningKey:                    DefaultSigningKeyConfig,
	ValidatorContract:             "",
	ParentChainNodeURL:            "",
	ParentChainConnectionAttempts: 15,
}

func SigningKeyConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.String(prefix+".private-key", DefaultSigningKeyConfig.PrivateKey, "hex-encoded private key for signing certificates")
	f.String(prefix+".key-file", DefaultSigningKeyConfig.KeyFile, "path to file containing private key")
}

func ConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".enable", DefaultConfig.Enable, "enable reference DA provider implementation")
	SigningKeyConfigAddOptions(prefix+".signing-key", f)
	f.String(prefix+".validator-contract", DefaultConfig.ValidatorContract, "address of the ReferenceDAProofValidator contract")
	f.String(prefix+".parent-chain-node-url", DefaultConfig.ParentChainNodeURL, "URL for parent chain node, only used in standalone daprovider; when running as part of a node that node's L1 configuration is used")
	f.Int(prefix+".parent-chain-connection-attempts", DefaultConfig.ParentChainConnectionAttempts, "parent chain RPC connection attempts (spaced out at least 1 second per attempt, 0 to retry infinitely), only used in standalone daserver; when running as part of a node that node's parent chain configuration is used")
}
