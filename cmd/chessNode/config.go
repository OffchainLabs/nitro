package main

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/providers/confmap"
	flag "github.com/spf13/pflag"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/cmd/conf"
	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/cmd/util/confighelpers"
	"github.com/offchainlabs/nitro/daprovider/das"
	"github.com/offchainlabs/nitro/util/colors"
)

type NodeConfig struct {
	Conf                   genericconf.ConfConfig          `koanf:"conf" reload:"hot"`
	Node                   arbnode.Config                  `koanf:"node" reload:"hot"`
	ParentChain            conf.ParentChainConfig          `koanf:"parent-chain" reload:"hot"`
	Chain                  conf.L2Config                   `koanf:"chain"`
	LogLevel               string                          `koanf:"log-level" reload:"hot"`
	LogType                string                          `koanf:"log-type" reload:"hot"`
	FileLogging            genericconf.FileLoggingConfig   `koanf:"file-logging" reload:"hot"`
	Persistent             conf.PersistentConfig           `koanf:"persistent"`
	HTTP                   genericconf.HTTPConfig          `koanf:"http"`
	WS                     genericconf.WSConfig            `koanf:"ws"`
	IPC                    genericconf.IPCConfig           `koanf:"ipc"`
	Auth                   genericconf.AuthRPCConfig       `koanf:"auth"`
	Metrics                bool                            `koanf:"metrics"`
	MetricsServer          genericconf.MetricsServerConfig `koanf:"metrics-server"`
	PProf                  bool                            `koanf:"pprof"`
	PprofCfg               genericconf.PProf               `koanf:"pprof-cfg"`
	Rpc                    genericconf.RpcConfig           `koanf:"rpc"`
	EnsureRollupDeployment bool                            `koanf:"ensure-rollup-deployment" reload:"hot"`
}

var NodeConfigDefault = NodeConfig{
	Conf:                   genericconf.ConfConfigDefault,
	Node:                   arbnode.ConfigDefault,
	ParentChain:            conf.L1ConfigDefault,
	Chain:                  conf.L2ConfigDefault,
	LogLevel:               "INFO",
	LogType:                "plaintext",
	FileLogging:            genericconf.DefaultFileLoggingConfig,
	Persistent:             conf.PersistentConfigDefault,
	HTTP:                   genericconf.HTTPConfigDefault,
	WS:                     genericconf.WSConfigDefault,
	IPC:                    genericconf.IPCConfigDefault,
	Auth:                   genericconf.AuthRPCConfigDefault,
	Metrics:                false,
	MetricsServer:          genericconf.MetricsServerConfigDefault,
	Rpc:                    genericconf.DefaultRpcConfig,
	PProf:                  false,
	PprofCfg:               genericconf.PProfDefault,
	EnsureRollupDeployment: true,
}

func NodeConfigAddOptions(f *flag.FlagSet) {
	genericconf.ConfConfigAddOptions("conf", f)
	arbnode.ConfigAddOptions("node", f, true, true)
	conf.L1ConfigAddOptions("parent-chain", f)
	conf.L2ConfigAddOptions("chain", f)
	f.String("log-level", NodeConfigDefault.LogLevel, "log level, valid values are CRIT, ERROR, WARN, INFO, DEBUG, TRACE")
	f.String("log-type", NodeConfigDefault.LogType, "log type (plaintext or json)")
	genericconf.FileLoggingConfigAddOptions("file-logging", f)
	conf.PersistentConfigAddOptions("persistent", f)
	genericconf.HTTPConfigAddOptions("http", f)
	genericconf.WSConfigAddOptions("ws", f)
	genericconf.IPCConfigAddOptions("ipc", f)
	genericconf.AuthRPCConfigAddOptions("auth", f)
	f.Bool("metrics", NodeConfigDefault.Metrics, "enable metrics")
	genericconf.MetricsServerAddOptions("metrics-server", f)
	f.Bool("pprof", NodeConfigDefault.PProf, "enable pprof")
	genericconf.PProfAddOptions("pprof-cfg", f)

	genericconf.RpcConfigAddOptions("rpc", f)
	f.Bool("ensure-rollup-deployment", NodeConfigDefault.EnsureRollupDeployment, "before starting the node, wait until the transaction that deployed rollup is finalized")
}

func (c *NodeConfig) ResolveDirectoryNames() error {
	err := c.Persistent.ResolveDirectoryNames()
	if err != nil {
		return err
	}
	c.Chain.ResolveDirectoryNames(c.Persistent.Chain)

	return nil
}

func (c *NodeConfig) ShallowClone() *NodeConfig {
	config := &NodeConfig{}
	*config = *c
	return config
}

func (c *NodeConfig) CanReload(new *NodeConfig) error {
	var check func(node, other reflect.Value, path string)
	var err error

	check = func(node, value reflect.Value, path string) {
		if node.Kind() != reflect.Struct {
			return
		}

		for i := 0; i < node.NumField(); i++ {
			fieldTy := node.Type().Field(i)
			if !fieldTy.IsExported() {
				continue
			}
			hot := fieldTy.Tag.Get("reload") == "hot"
			dot := path + "." + fieldTy.Name

			first := node.Field(i).Interface()
			other := value.Field(i).Interface()

			if !hot && !reflect.DeepEqual(first, other) {
				err = fmt.Errorf("illegal change to %v%v%v", colors.Red, dot, colors.Clear)
			} else {
				check(node.Field(i), value.Field(i), dot)
			}
		}
	}

	check(reflect.ValueOf(c).Elem(), reflect.ValueOf(new).Elem(), "config")
	return err
}

func (c *NodeConfig) Validate() error {
	if err := c.ParentChain.Validate(); err != nil {
		return err
	}
	if err := c.Node.Validate(); err != nil {
		return err
	}
	return c.Persistent.Validate()
}

func (c *NodeConfig) GetReloadInterval() time.Duration {
	return c.Conf.ReloadInterval
}

func applyChainParameters(k *koanf.Koanf) error {
	// Arbitrum chains produce blocks more quickly, so the inbox reader should read more blocks at once.
	// Even if this is too large, on error the inbox reader will reset its query size down to the default.

	return k.Load(confmap.Provider(DefChainInfo, "."), nil)
}

func ParseNode(ctx context.Context, args []string) (*NodeConfig, error) {
	f := flag.NewFlagSet("", flag.ContinueOnError)

	NodeConfigAddOptions(f)

	k, err := confighelpers.BeginCommonParse(f, args)
	if err != nil {
		return nil, err
	}

	// #nosec G115
	if err != nil {
		return nil, err
	}

	err = confighelpers.ApplyOverrides(f, k)
	if err != nil {
		return nil, err
	}

	if err = das.FixKeysetCLIParsing("node.data-availability.rpc-aggregator.backends", k); err != nil {
		return nil, err
	}

	applyChainParameters(k)

	var nodeConfig NodeConfig
	if err := confighelpers.EndCommonParse(k, &nodeConfig); err != nil {
		return nil, err
	}

	// Don't print wallet passwords
	if nodeConfig.Conf.Dump {
		err = confighelpers.DumpConfig(k, map[string]interface{}{
			"node.staker.parent-chain-wallet.password":    "",
			"node.staker.parent-chain-wallet.private-key": "",
			"chain.dev-wallet.password":                   "",
			"chain.dev-wallet.private-key":                "",
		})
		if err != nil {
			return nil, err
		}
	}

	if nodeConfig.Persistent.Chain == "" {
		return nil, errors.New("--persistent.chain not specified")
	}

	err = nodeConfig.ResolveDirectoryNames()
	if err != nil {
		return nil, err
	}

	// Don't pass around wallet contents with normal configuration
	nodeConfig.Chain.DevWallet = genericconf.WalletConfigDefault

	err = nodeConfig.Validate()
	if err != nil {
		return nil, err
	}
	return &nodeConfig, nil
}

type NodeConfigFetcher struct {
	*genericconf.LiveConfig[*NodeConfig]
}

func (f *NodeConfigFetcher) Get() *arbnode.Config {
	return &f.LiveConfig.Get().Node
}

var DefChainInfo map[string]interface{}
var DefAddresses chaininfo.RollupAddresses

func init() {
	DefChainInfo = make(map[string]interface{})

	DefChainInfo["persistent.chain"] = "chess"
	DefChainInfo["chain.name"] = "chesschain"
	DefChainInfo["chain.id"] = uint64(0xc4355c4355)
	DefChainInfo["parent-chain.id"] = uint64(1337)
	DefChainInfo["node.inbox-reader.max-blocks-to-read"] = 100_000

	DefAddresses.Bridge = common.HexToAddress("0x12e93f005f4fe6490f9de27b72131992c4ac693d")
	DefAddresses.Inbox =
		common.HexToAddress("0xb811fA75EA2952112c12929f6d11A99C7726f67E")
	DefAddresses.UpgradeExecutor =
		common.HexToAddress("0x4D3c15601108d89E8A30053789BA7665B01Af645")
	DefAddresses.ValidatorWalletCreator =
		common.HexToAddress("0x2c37dCBCE3fbe32c9Ba62892F1E41DbB023BB62b")
	DefAddresses.DeployedAt = 152623308
	DefAddresses.Rollup = common.HexToAddress("0x71175FE7A66Dde4181C092F5a15cB567DFDa568e")         // RollupEventInbox?
	DefAddresses.SequencerInbox = common.HexToAddress("0x21d6CE46333b6af7C3554cf8FcC44560099754BE") //TODO: remove?
	// Rollup                 common.Address `json:"rollup"`
	// ValidatorUtils         common.Address `json:"validator-utils"`
	// DeployedAt             uint64         `json:"deployed-at"`
}
