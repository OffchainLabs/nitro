// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package main

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"os/signal"
	"syscall"

	"github.com/knadh/koanf/parsers/json"
	"github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/cmd/conf"
	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/cmd/transaction-filterer/api"
	"github.com/offchainlabs/nitro/cmd/util"
	"github.com/offchainlabs/nitro/cmd/util/confighelpers"
	"github.com/offchainlabs/nitro/util/rpcclient"
)

type TransactionFiltererConfig struct {
	Conf       genericconf.ConfConfig `koanf:"conf"`
	Persistent conf.PersistentConfig  `koanf:"persistent"`

	FileLogging genericconf.FileLoggingConfig `koanf:"file-logging" reload:"hot"`
	LogLevel    string                        `koanf:"log-level"`
	LogType     string                        `koanf:"log-type"`

	Metrics       bool                            `koanf:"metrics"`
	MetricsServer genericconf.MetricsServerConfig `koanf:"metrics-server"`

	PProf    bool              `koanf:"pprof"`
	PprofCfg genericconf.PProf `koanf:"pprof-cfg"`

	HTTP genericconf.HTTPConfig    `koanf:"http"`
	WS   genericconf.WSConfig      `koanf:"ws"`
	IPC  genericconf.IPCConfig     `koanf:"ipc"`
	Auth genericconf.AuthRPCConfig `koanf:"auth"`

	ChainId     int64                    `koanf:"chain-id"`
	Wallet      genericconf.WalletConfig `koanf:"wallet"`
	Sequencer   rpcclient.ClientConfig   `koanf:"sequencer"`
	ParentChain ParentChainConfig        `koanf:"parent-chain"`
	Pruning     api.PruneConfig          `koanf:"pruning"`
}

type ParentChainConfig struct {
	Connection rpcclient.ClientConfig `koanf:"connection"`
	Bridge     string                 `koanf:"bridge"`
}

var DefaultParentChainConfig = ParentChainConfig{
	Connection: rpcclient.DefaultClientConfig,
}

func ParentChainConfigAddOptions(prefix string, f *pflag.FlagSet) {
	rpcclient.RPCClientAddOptions(prefix+".connection", f, &DefaultParentChainConfig.Connection)
	f.String(prefix+".bridge", DefaultParentChainConfig.Bridge, "parent chain bridge contract address (hex); required when pruning is enabled")
}

var HTTPConfigDefault = genericconf.HTTPConfig{
	Addr:           "",
	Port:           genericconf.HTTPConfigDefault.Port,
	API:            []string{},
	RPCPrefix:      genericconf.HTTPConfigDefault.RPCPrefix,
	CORSDomain:     genericconf.HTTPConfigDefault.CORSDomain,
	VHosts:         genericconf.HTTPConfigDefault.VHosts,
	ServerTimeouts: genericconf.HTTPConfigDefault.ServerTimeouts,
}

var WSConfigDefault = genericconf.WSConfig{
	Addr:      "",
	Port:      genericconf.WSConfigDefault.Port,
	API:       []string{},
	RPCPrefix: genericconf.WSConfigDefault.RPCPrefix,
	Origins:   genericconf.WSConfigDefault.Origins,
	ExposeAll: genericconf.WSConfigDefault.ExposeAll,
}

var IPCConfigDefault = genericconf.IPCConfig{
	Path: "",
}

var DefaultTransactionFiltererConfig = TransactionFiltererConfig{
	Conf:          genericconf.ConfConfigDefault,
	LogLevel:      "INFO",
	LogType:       "plaintext",
	Metrics:       false,
	MetricsServer: genericconf.MetricsServerConfigDefault,
	PProf:         false,
	PprofCfg:      genericconf.PProfDefault,
	HTTP:          HTTPConfigDefault,
	WS:            WSConfigDefault,
	IPC:           IPCConfigDefault,
	Auth:          genericconf.AuthRPCConfigDefault,
	ChainId:       412346, // nitro-testnode chainid
	Sequencer:     rpcclient.DefaultClientConfig,
	ParentChain:   DefaultParentChainConfig,
	Pruning:       api.DefaultPruneConfig,
}

func addFlags(f *pflag.FlagSet) {
	genericconf.ConfConfigAddOptions("conf", f)
	conf.PersistentConfigAddOptions("persistent", f)

	genericconf.FileLoggingConfigAddOptions("file-logging", f)
	f.String("log-level", DefaultTransactionFiltererConfig.LogLevel, "log level, valid values are CRIT, ERROR, WARN, INFO, DEBUG, TRACE")
	f.String("log-type", DefaultTransactionFiltererConfig.LogType, "log type (plaintext or json)")

	f.Bool("metrics", DefaultTransactionFiltererConfig.Metrics, "enable metrics")
	genericconf.MetricsServerAddOptions("metrics-server", f)

	f.Bool("pprof", DefaultTransactionFiltererConfig.PProf, "enable pprof")
	genericconf.PProfAddOptions("pprof-cfg", f)

	genericconf.HTTPConfigAddOptions("http", f)
	genericconf.WSConfigAddOptions("ws", f)
	genericconf.IPCConfigAddOptions("ipc", f)

	f.Int64("chain-id", DefaultTransactionFiltererConfig.ChainId, "chain ID of the chain being filtered")
	genericconf.WalletConfigAddOptions("wallet", f, "")
	rpcclient.RPCClientAddOptions("sequencer", f, &DefaultTransactionFiltererConfig.Sequencer)

	ParentChainConfigAddOptions("parent-chain", f)
	api.PruneConfigAddOptions("pruning", f)
}

func parseConfig(args []string) (*TransactionFiltererConfig, error) {
	f := pflag.NewFlagSet("", pflag.ContinueOnError)

	addFlags(f)

	k, err := confighelpers.BeginCommonParse(f, args)
	if err != nil {
		return nil, err
	}

	var config TransactionFiltererConfig
	if err := confighelpers.EndCommonParse(k, &config); err != nil {
		return nil, err
	}
	if config.Conf.Dump {
		err = confighelpers.DumpConfig(k, map[string]interface{}{
			"wallet.password":    "",
			"wallet.private-key": "",
		})
		if err != nil {
			return nil, fmt.Errorf("error removing extra parameters before dump: %w", err)
		}

		c, err := k.Marshal(json.Parser())
		if err != nil {
			return nil, fmt.Errorf("unable to marshal config file to JSON: %w", err)
		}

		fmt.Println(string(c))
		os.Exit(0)
	}

	return &config, nil
}

func printSampleUsage(progname string) {
	fmt.Printf("\n")
	fmt.Printf("Sample usage:                  %s --help \n", progname)
}

func mainImpl() int {
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	config, err := parseConfig(os.Args[1:])
	if err != nil {
		confighelpers.PrintErrorAndExit(err, printSampleUsage)
	}

	stackConf := api.DefaultStackConfig
	config.HTTP.Apply(&stackConf)
	config.WS.Apply(&stackConf)
	config.IPC.Apply(&stackConf)
	config.Auth.Apply(&stackConf)
	_, strippedRevision, _ := confighelpers.GetVersion()
	stackConf.Version = strippedRevision

	if stackConf.JWTSecret == "" && stackConf.AuthAddr != "" {
		filename := genericconf.DefaultPathResolver(config.Persistent.GlobalConfig)("jwtsecret")
		if err := genericconf.TryCreatingJWTSecret(filename); err != nil {
			fmt.Fprintf(os.Stderr, "failed to prepare jwt secret file: %v\n", err)
			return 1
		}
		stackConf.JWTSecret = filename
	}

	err = genericconf.InitLog(config.LogType, config.LogLevel, &config.FileLogging, genericconf.DefaultPathResolver(config.Persistent.LogDir))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error initializing log: %v\n", err)
		return 1
	}

	if err := util.StartMetrics(config.Metrics, config.PProf, &config.MetricsServer, &config.PprofCfg); err != nil {
		fmt.Fprintf(os.Stderr, "error starting metrics server: %v\n", err)
		return 1
	}

	sequencerRPCConfigFetcher := func() *rpcclient.ClientConfig { return &config.Sequencer }
	sequencerRPCClient := rpcclient.NewRpcClient(sequencerRPCConfigFetcher, nil)
	err = sequencerRPCClient.Start(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error starting sequencer rpc client: %v\n", err)
		return 1
	}
	defer sequencerRPCClient.Close()
	sequencerClient := ethclient.NewClient(sequencerRPCClient)
	defer sequencerClient.Close()

	txOpts, _, err := util.OpenWallet("transaction-filterer", &config.Wallet, big.NewInt(config.ChainId))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error opening wallet: %v\n", err)
		return 1
	}

	var pruneOpts *api.PruneOptions
	if config.Pruning.Enable {
		if !common.IsHexAddress(config.ParentChain.Bridge) {
			fmt.Fprintf(os.Stderr, "parent-chain.bridge is required and must be a valid hex address when pruning is enabled; got %q\n", config.ParentChain.Bridge)
			return 1
		}
		bridgeAddr := common.HexToAddress(config.ParentChain.Bridge)
		if bridgeAddr == (common.Address{}) {
			fmt.Fprintf(os.Stderr, "parent-chain.bridge must not be the zero address\n")
			return 1
		}
		parentChainRPCConfigFetcher := func() *rpcclient.ClientConfig { return &config.ParentChain.Connection }
		parentChainRPCClient := rpcclient.NewRpcClient(parentChainRPCConfigFetcher, nil)
		if err := parentChainRPCClient.Start(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "error starting parent chain rpc client: %v\n", err)
			return 1
		}
		defer parentChainRPCClient.Close()
		parentChainClient := ethclient.NewClient(parentChainRPCClient)
		defer parentChainClient.Close()

		delayedBridge, err := arbnode.NewDelayedBridge(parentChainClient, bridgeAddr, 0)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error creating delayed bridge: %v\n", err)
			return 1
		}

		pruneOpts = &api.PruneOptions{
			Config:            config.Pruning,
			ChainId:           big.NewInt(config.ChainId),
			ParentChainClient: parentChainClient,
			ChildChainClient:  sequencerClient,
			DelayedBridge:     delayedBridge,
		}
	}

	stack, api, err := api.NewStack(&stackConf, txOpts, sequencerClient, pruneOpts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating stack: %v\n", err)
		return 1
	}
	err = api.Start(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error starting API: %v\n", err)
		return 1
	}
	defer api.StopAndWait()

	err = stack.Start()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error starting stack: %v\n", err)
		return 1
	}
	defer stack.Close()

	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)
	<-sigint

	return 0
}

func main() {
	os.Exit(mainImpl())
}
