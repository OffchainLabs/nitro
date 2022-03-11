package main

import (
	"context"
	"fmt"
	"github.com/offchainlabs/nitro/broadcastclient"
	"os"

	"github.com/ethereum/go-ethereum/log"
	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/pkg/errors"
	flag "github.com/spf13/pflag"

	"github.com/offchainlabs/nitro/cmd/util"
)

type NodeConfig struct {
	Conf         util.ConfConfig            `koanf:"conf"`
	Feed         broadcastclient.FeedConfig `koanf:"feed"`
	L1           util.L1Config              `koanf:"l1"`
	LogLevelImpl string                     `koanf:"log-level"`
	Wallet       util.WalletConfig          `koanf:"wallet"`
}

func (c *NodeConfig) LogLevel() (log.Lvl, error) {
	return log.LvlFromString(c.LogLevelImpl)
}

var DefaultNodeConfig = NodeConfig{
	Conf:         util.DefaultConfConfig,
	Feed:         broadcastclient.DefaultFeedConfig,
	L1:           util.DefaultL1Config,
	LogLevelImpl: "info",
	Wallet:       util.DefaultWalletConfig,
}

func NodeConfigAddOptions(f *flag.FlagSet) {
	util.ConfConfigAddOptions("conf", f)
	broadcastclient.FeedConfigAddOptions("feed", f)
	util.L1ConfigAddOptions("l1", f)
	f.String("log-level", DefaultNodeConfig.LogLevelImpl, "log level")
	util.WalletConfigAddOptions("wallet", f)
}

func ParseNode(ctx context.Context) (*NodeConfig, *util.WalletConfig, error) {
	f := flag.NewFlagSet("", flag.ContinueOnError)

	NodeConfigAddOptions(f)

	k, err := util.BeginCommonParse(f)
	if err != nil {
		return nil, nil, err
	}

	var nodeConfig NodeConfig
	if err := util.EndCommonParse(k, &nodeConfig); err != nil {
		return nil, nil, err
	}

	if nodeConfig.Conf.Dump {
		// Print out current configuration

		// Don't keep printing configuration file and don't print wallet passwords
		err := k.Load(confmap.Provider(map[string]interface{}{
			"conf.dump":          false,
			"wallet.password":    "",
			"wallet.private-key": "",
		}, "."), nil)
		if err != nil {
			return nil, nil, errors.Wrap(err, "error removing extra parameters before dump")
		}

		c, err := k.Marshal(json.Parser())
		if err != nil {
			return nil, nil, errors.Wrap(err, "unable to marshal config file to JSON")
		}

		fmt.Println(string(c))
		os.Exit(1)
	}

	// Don't pass around wallet contents with normal configuration
	wallet := nodeConfig.Wallet
	nodeConfig.Wallet = util.WalletConfig{}

	return &nodeConfig, &wallet, nil
}
