// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/nitro/blob/master/LICENSE

package conf

import (
	"time"

	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/util/rpcclient"
	flag "github.com/spf13/pflag"
)

type L1Config struct {
	ChainID    uint64                   `koanf:"id"`
	Connection rpcclient.ClientConfig   `koanf:"connection" reload:"hot"`
	Wallet     genericconf.WalletConfig `koanf:"wallet"`
}

var L1ConnectionConfigDefault = rpcclient.ClientConfig{
	URL:            "",
	Retries:        2,
	Timeout:        time.Minute,
	ConnectionWait: time.Minute,
}

var L1ConfigDefault = L1Config{
	ChainID:    0,
	Connection: L1ConnectionConfigDefault,
	Wallet:     DefaultL1WalletConfig,
}

var DefaultL1WalletConfig = genericconf.WalletConfig{
	Pathname:      "wallet",
	PasswordImpl:  genericconf.WalletConfigDefault.PasswordImpl,
	PrivateKey:    genericconf.WalletConfigDefault.PrivateKey,
	Account:       genericconf.WalletConfigDefault.Account,
	OnlyCreateKey: genericconf.WalletConfigDefault.OnlyCreateKey,
}

func L1ConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Uint64(prefix+".id", L1ConfigDefault.ChainID, "if set other than 0, will be used to validate database and L1 connection")
	rpcclient.RPCClientAddOptions(prefix+".connection", f, &L1ConfigDefault.Connection)
	genericconf.WalletConfigAddOptions(prefix+".wallet", f, L1ConfigDefault.Wallet.Pathname)
}

func (c *L1Config) ResolveDirectoryNames(chain string) {
	c.Wallet.ResolveDirectoryNames(chain)
}

func (c *L1Config) Validate() error {
	return c.Connection.Validate()
}

type L2Config struct {
	ChainID                   uint64                   `koanf:"id"`
	ChainName                 string                   `koanf:"name"`
	ChainInfoFiles            []string                 `koanf:"info-files"`
	ChainInfoJson             string                   `koanf:"info-json"`
	DevWallet                 genericconf.WalletConfig `koanf:"dev-wallet"`
	ChainInfoIpfsUrl          string                   `koanf:"info-ipfs-url"`
	ChainInfoIpfsDownloadPath string                   `koanf:"info-ipfs-download-path"`
}

var L2ConfigDefault = L2Config{
	ChainID:                   0,
	ChainName:                 "",
	ChainInfoFiles:            []string{}, // Default file used is chaininfo/arbitrum_chain_info.json, stored in DefaultChainInfo in chain_info.go
	ChainInfoJson:             "",
	DevWallet:                 genericconf.WalletConfigDefault,
	ChainInfoIpfsUrl:          "",
	ChainInfoIpfsDownloadPath: "/tmp/",
}

func L2ConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Uint64(prefix+".id", L2ConfigDefault.ChainID, "L2 chain ID (determines Arbitrum network)")
	f.String(prefix+".name", L2ConfigDefault.ChainName, "L2 chain name (determines Arbitrum network)")
	f.StringSlice(prefix+".info-files", L2ConfigDefault.ChainInfoFiles, "L2 chain info json files")
	f.String(prefix+".info-json", L2ConfigDefault.ChainInfoJson, "L2 chain info in json string format")

	// Dev wallet does not exist unless specified
	genericconf.WalletConfigAddOptions(prefix+".dev-wallet", f, "")
	f.String(prefix+".info-ipfs-url", L2ConfigDefault.ChainInfoIpfsUrl, "url to download chain info file")
	f.String(prefix+".info-ipfs-download-path", L2ConfigDefault.ChainInfoIpfsDownloadPath, "path to save temp downloaded file")

}

func (c *L2Config) ResolveDirectoryNames(chain string) {
	c.DevWallet.ResolveDirectoryNames(chain)
}
