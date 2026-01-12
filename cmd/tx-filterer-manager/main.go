// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/knadh/koanf/parsers/json"
	"github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/cmd/tx-filterer-manager/api"
	"github.com/offchainlabs/nitro/cmd/util"
	"github.com/offchainlabs/nitro/cmd/util/confighelpers"
)

type TxFiltererManagerConfig struct {
	Conf genericconf.ConfConfig `koanf:"conf"`

	FileLogging genericconf.FileLoggingConfig `koanf:"file-logging" reload:"hot"`
	LogLevel    string                        `koanf:"log-level"`
	LogType     string                        `koanf:"log-type"`

	Metrics       bool                            `koanf:"metrics"`
	MetricsServer genericconf.MetricsServerConfig `koanf:"metrics-server"`

	PProf    bool              `koanf:"pprof"`
	PprofCfg genericconf.PProf `koanf:"pprof-cfg"`

	HTTP genericconf.HTTPConfig `koanf:"http"`
	WS   genericconf.WSConfig   `koanf:"ws"`
	IPC  genericconf.IPCConfig  `koanf:"ipc"`

	Wallet genericconf.WalletConfig `koanf:"wallet"`
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

var DefaultTxFiltererManagerConfig = TxFiltererManagerConfig{
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
}

var DefaultStackConfig = node.Config{
	DataDir:             node.DefaultDataDir(),
	HTTPPort:            node.DefaultHTTPPort,
	AuthAddr:            node.DefaultAuthHost,
	AuthPort:            node.DefaultAuthPort,
	AuthVirtualHosts:    node.DefaultAuthVhosts,
	HTTPModules:         []string{"txfilterermanager"},
	HTTPHost:            "localhost",
	HTTPVirtualHosts:    []string{"localhost"},
	HTTPTimeouts:        rpc.DefaultHTTPTimeouts,
	WSHost:              "localhost",
	WSPort:              node.DefaultWSPort,
	WSModules:           []string{"txfilterermanager"},
	GraphQLVirtualHosts: []string{"localhost"},
	P2P: p2p.Config{
		ListenAddr:  "",
		NoDiscovery: true,
		NoDial:      true,
	},
}

func addFlags(f *pflag.FlagSet) {
	genericconf.ConfConfigAddOptions("conf", f)

	genericconf.FileLoggingConfigAddOptions("file-logging", f)
	f.String("log-level", DefaultTxFiltererManagerConfig.LogLevel, "log level, valid values are CRIT, ERROR, WARN, INFO, DEBUG, TRACE")
	f.String("log-type", DefaultTxFiltererManagerConfig.LogType, "log type (plaintext or json)")

	f.Bool("metrics", DefaultTxFiltererManagerConfig.Metrics, "enable metrics")
	genericconf.MetricsServerAddOptions("metrics-server", f)

	f.Bool("pprof", DefaultTxFiltererManagerConfig.PProf, "enable pprof")
	genericconf.PProfAddOptions("pprof-cfg", f)

	genericconf.HTTPConfigAddOptions("http", f)
	genericconf.WSConfigAddOptions("ws", f)
	genericconf.IPCConfigAddOptions("ipc", f)

	genericconf.WalletConfigAddOptions("wallet", f, "")
}

func parseConfig(args []string) (*TxFiltererManagerConfig, error) {
	f := pflag.NewFlagSet("", pflag.ContinueOnError)

	addFlags(f)

	k, err := confighelpers.BeginCommonParse(f, args)
	if err != nil {
		return nil, err
	}

	var config TxFiltererManagerConfig
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

func startup() error {
	config, err := parseConfig(os.Args[1:])
	if err != nil {
		confighelpers.PrintErrorAndExit(err, printSampleUsage)
	}

	stackConf := DefaultStackConfig
	stackConf.DataDir = "" // ephemeral
	config.HTTP.Apply(&stackConf)
	config.WS.Apply(&stackConf)
	config.IPC.Apply(&stackConf)
	stackConf.P2P.ListenAddr = ""
	stackConf.P2P.NoDial = true
	stackConf.P2P.NoDiscovery = true
	_, strippedRevision, _ := confighelpers.GetVersion()
	stackConf.Version = strippedRevision

	err = util.SetLogger(config.LogLevel, config.LogType)
	if err != nil {
		return err
	}

	if err := util.StartMetrics(config.Metrics, config.PProf, &config.MetricsServer, &config.PprofCfg); err != nil {
		return err
	}

	stack, err := node.New(&stackConf)
	if err != nil {
		return err
	}
	api.RegisterAPI(stack)

	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)

	err = stack.Start()
	if err != nil {
		return err
	}
	defer stack.Close()

	<-sigint

	return nil
}

func main() {
	if err := startup(); err != nil {
		log.Error("Error running tx-filterer-manager", "err", err)
	}
}
