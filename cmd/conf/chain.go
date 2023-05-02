// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package conf

import (
	"time"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/util/rpcclient"
	flag "github.com/spf13/pflag"
)

type L1Config struct {
	ChainID    uint64                        `koanf:"chain-id"`
	Rollup     arbnode.RollupAddressesConfig `koanf:"rollup"`
	Connection rpcclient.ClientConfig        `koanf:"connection"`
	Wallet     genericconf.WalletConfig      `koanf:"wallet"`
}

var L1ConnectionConfigDefault = rpcclient.ClientConfig{
	URL:            "",
	Retries:        2,
	Timeout:        time.Minute * 5,
	ConnectionWait: time.Minute,
}

var L1ConfigDefault = L1Config{
	ChainID:    0,
	Rollup:     arbnode.RollupAddressesConfigDefault,
	Connection: L1ConnectionConfigDefault,
	Wallet:     genericconf.WalletConfigDefault,
}

func L1ConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Uint64(prefix+".chain-id", L1ConfigDefault.ChainID, "if set other than 0, will be used to validate database and L1 connection")
	arbnode.RollupAddressesConfigAddOptions(prefix+".rollup", f)
	rpcclient.RPCClientAddOptions(prefix+".connection", f, &L1ConfigDefault.Connection)
	genericconf.WalletConfigAddOptions(prefix+".wallet", f, "wallet")
}

func (c *L1Config) ResolveDirectoryNames(chain string) {
	c.Wallet.ResolveDirectoryNames(chain)
}

type L2Config struct {
	ChainID   uint64                   `koanf:"chain-id"`
	DevWallet genericconf.WalletConfig `koanf:"dev-wallet"`
}

var L2ConfigDefault = L2Config{
	ChainID:   0,
	DevWallet: genericconf.WalletConfigDefault,
}

func L2ConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Uint64(prefix+".chain-id", L2ConfigDefault.ChainID, "L2 chain ID (determines Arbitrum network)")
	// Dev wallet does not exist unless specified
	genericconf.WalletConfigAddOptions(prefix+".dev-wallet", f, "")
}

func (c *L2Config) ResolveDirectoryNames(chain string) {
	c.DevWallet.ResolveDirectoryNames(chain)
}
