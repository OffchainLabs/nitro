// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package conf

import (
	"math/big"
	"time"

	"github.com/google/martian/v3/log"
	"github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/offchainlabs/nitro/util/rpcclient"
)

type ParentChainConfig struct {
	ID         uint64                        `koanf:"id"`
	Connection rpcclient.ClientConfig        `koanf:"connection" reload:"hot"`
	BlobClient headerreader.BlobClientConfig `koanf:"blob-client"`
}

var L1ConnectionConfigDefault = rpcclient.ClientConfig{
	URL:                       "",
	Retries:                   2,
	Timeout:                   time.Minute,
	ConnectionWait:            time.Minute,
	ArgLogLimit:               2048,
	WebsocketMessageSizeLimit: 256 * 1024 * 1024,
}

var L1ConfigDefault = ParentChainConfig{
	ID:         0,
	Connection: L1ConnectionConfigDefault,
	BlobClient: headerreader.DefaultBlobClientConfig,
}

var DefaultL1WalletConfig = genericconf.WalletConfig{
	Pathname:      "wallet",
	Password:      genericconf.WalletConfigDefault.Password,
	PrivateKey:    genericconf.WalletConfigDefault.PrivateKey,
	Account:       genericconf.WalletConfigDefault.Account,
	OnlyCreateKey: genericconf.WalletConfigDefault.OnlyCreateKey,
}

func L1ConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Uint64(prefix+".id", L1ConfigDefault.ID, "if set other than 0, will be used to validate database and L1 connection")
	rpcclient.RPCClientAddOptions(prefix+".connection", f, &L1ConfigDefault.Connection)
	headerreader.BlobClientAddOptions(prefix+".blob-client", f)
}

func (c *ParentChainConfig) Validate() error {
	return c.Connection.Validate()
}

// lint:require-exhaustive-initialization
type L2Config struct {
	ID               uint64                   `koanf:"id"`
	Name             string                   `koanf:"name"`
	InfoFiles        []string                 `koanf:"info-files"`
	InfoJson         string                   `koanf:"info-json"`
	InitialL1BaseFee string                   `koanf:"initial-l1base-fee"`
	DevWallet        genericconf.WalletConfig `koanf:"dev-wallet"`
}

func (c *L2Config) InitialL1BaseFeeParsed() *big.Int {
	if c.InitialL1BaseFee == "" {
		return params.DefaultInitialL1BaseFee
	}

	parsed, success := big.NewInt(0).SetString(c.InitialL1BaseFee, 10)
	if !success {
		log.Errorf("Failed to parse L1 BaseFee for L2 config: %v", c.InitialL1BaseFee)
		return params.DefaultInitialL1BaseFee
	}
	return parsed
}

var L2ConfigDefault = L2Config{
	ID:               0,
	Name:             "",
	InfoFiles:        []string{}, // Default file used is chaininfo/arbitrum_chain_info.json, stored in DefaultChainInfo in chain_info.go
	InfoJson:         "",
	InitialL1BaseFee: params.DefaultInitialL1BaseFee.String(),
	DevWallet:        genericconf.WalletConfigDefault,
}

func L2ConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Uint64(prefix+".id", L2ConfigDefault.ID, "L2 chain ID (determines Arbitrum network)")
	f.String(prefix+".name", L2ConfigDefault.Name, "L2 chain name (determines Arbitrum network)")
	f.StringSlice(prefix+".info-files", L2ConfigDefault.InfoFiles, "L2 chain info json files")
	f.String(prefix+".info-json", L2ConfigDefault.InfoJson, "L2 chain info in json string format")
	f.String(prefix+".initial-l1base-fee", L2ConfigDefault.InitialL1BaseFee, "Initial L1 base fee for the L2 chain")

	// Dev wallet does not exist unless specified
	genericconf.WalletConfigAddOptions(prefix+".dev-wallet", f, "")
}

func (c *L2Config) ResolveDirectoryNames(chain string) {
	c.DevWallet.ResolveDirectoryNames(chain)
}
