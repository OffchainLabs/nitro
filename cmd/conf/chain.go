// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package conf

import (
	"time"

	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/offchainlabs/nitro/util/rpcclient"
	flag "github.com/spf13/pflag"
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

func L1ConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Uint64(prefix+".id", L1ConfigDefault.ID, "if set other than 0, will be used to validate database and L1 connection")
	rpcclient.RPCClientAddOptions(prefix+".connection", f, &L1ConfigDefault.Connection)
	headerreader.BlobClientAddOptions(prefix+".blob-client", f)
}

func (c *ParentChainConfig) Validate() error {
	return c.Connection.Validate()
}

type L2Config struct {
	ID                   uint64                   `koanf:"id"`
	Name                 string                   `koanf:"name"`
	InfoFiles            []string                 `koanf:"info-files"`
	InfoJson             string                   `koanf:"info-json"`
	DevWallet            genericconf.WalletConfig `koanf:"dev-wallet"`
	InfoIpfsUrl          string                   `koanf:"info-ipfs-url"`
	InfoIpfsDownloadPath string                   `koanf:"info-ipfs-download-path"`
}

var L2ConfigDefault = L2Config{
	ID:                   0,
	Name:                 "",
	InfoFiles:            []string{}, // Default file used is chaininfo/arbitrum_chain_info.json, stored in DefaultChainInfo in chain_info.go
	InfoJson:             "",
	DevWallet:            genericconf.WalletConfigDefault,
	InfoIpfsUrl:          "",
	InfoIpfsDownloadPath: "/tmp/",
}

func L2ConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Uint64(prefix+".id", L2ConfigDefault.ID, "L2 chain ID (determines Arbitrum network)")
	f.String(prefix+".name", L2ConfigDefault.Name, "L2 chain name (determines Arbitrum network)")
	f.StringSlice(prefix+".info-files", L2ConfigDefault.InfoFiles, "L2 chain info json files")
	f.String(prefix+".info-json", L2ConfigDefault.InfoJson, "L2 chain info in json string format")

	// Dev wallet does not exist unless specified
	genericconf.WalletConfigAddOptions(prefix+".dev-wallet", f, "")
	f.String(prefix+".info-ipfs-url", L2ConfigDefault.InfoIpfsUrl, "url to download chain info file")
	f.String(prefix+".info-ipfs-download-path", L2ConfigDefault.InfoIpfsDownloadPath, "path to save temp downloaded file")

}

func (c *L2Config) ResolveDirectoryNames(chain string) {
	c.DevWallet.ResolveDirectoryNames(chain)
}
