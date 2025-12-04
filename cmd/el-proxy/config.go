package main

import (
	"fmt"
	"reflect"
	"time"

	"github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/cmd/conf"
	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/util/colors"
)

type ExpressLaneProxyConfig struct {
	ExpressLaneURL         string                   `koanf:"express-lane-url"`
	RPCURL                 string                   `koanf:"rpc-url"`
	ChainId                int64                    `koanf:"chain-id"`
	AuctionContractAddress string                   `koanf:"auction-contract-address"`
	Wallet                 genericconf.WalletConfig `koanf:"wallet"`

	Persistent    conf.PersistentConfig           `koanf:"persistent"`
	Conf          genericconf.ConfConfig          `koanf:"conf" reload:"hot"`
	LogLevel      string                          `koanf:"log-level" reload:"hot"`
	LogType       string                          `koanf:"log-type" reload:"hot"`
	FileLogging   genericconf.FileLoggingConfig   `koanf:"file-logging" reload:"hot"`
	HTTP          genericconf.HTTPConfig          `koanf:"http"`
	WS            genericconf.WSConfig            `koanf:"ws"`
	IPC           genericconf.IPCConfig           `koanf:"ipc"`
	MaxTxDataSize uint64                          `koanf:"max-tx-data-size" reload:"hot"`
	Metrics       bool                            `koanf:"metrics"`
	MetricsServer genericconf.MetricsServerConfig `koanf:"metrics-server"`
	PProf         bool                            `koanf:"pprof"`
	PprofCfg      genericconf.PProf               `koanf:"pprof-cfg"`
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

var ExpressLaneProxyConfigDefault = ExpressLaneProxyConfig{
	ExpressLaneURL:         "http://localhost:8547",
	RPCURL:                 "http://localhost:8547",
	ChainId:                412346, // nitro-testnode chainid
	AuctionContractAddress: "",

	Conf:          genericconf.ConfConfigDefault,
	LogLevel:      "INFO",
	LogType:       "plaintext",
	HTTP:          HTTPConfigDefault,
	WS:            WSConfigDefault,
	IPC:           IPCConfigDefault,
	MaxTxDataSize: uint64(gethexec.DefaultSequencerConfig.MaxTxDataSize),
	Metrics:       false,
	MetricsServer: genericconf.MetricsServerConfigDefault,
	PProf:         false,
	Persistent:    conf.PersistentConfigDefault,
	PprofCfg:      genericconf.PProfDefault,
}

func ExpressLaneProxyConfigAddOptions(f *pflag.FlagSet) {
	f.String("express-lane-url", ExpressLaneProxyConfigDefault.ExpressLaneURL, "URL to send timeboost_sendExpressLaneTransaction requests to")
	f.String("rpc-url", ExpressLaneProxyConfigDefault.RPCURL, "URL to proxy to all other RPC requests to")
	f.Int64("chain-id", ExpressLaneProxyConfigDefault.ChainId, "Chain ID of the chain being proxied to")
	f.String("auction-contract-address", ExpressLaneProxyConfigDefault.AuctionContractAddress, "Address of the proxy pointing to the ExpressLaneAuction contract")
	genericconf.WalletConfigAddOptions("wallet", f, "wallet with account for proxy to use to sign txs")

	conf.PersistentConfigAddOptions("persistent", f)
	genericconf.ConfConfigAddOptions("conf", f)
	f.String("log-level", ExpressLaneProxyConfigDefault.LogLevel, "log level, valid values are CRIT, ERROR, WARN, INFO, DEBUG, TRACE")
	f.String("log-type", ExpressLaneProxyConfigDefault.LogType, "log type (plaintext or json)")
	genericconf.FileLoggingConfigAddOptions("file-logging", f)
	genericconf.HTTPConfigAddOptions("http", f)
	genericconf.WSConfigAddOptions("ws", f)
	genericconf.IPCConfigAddOptions("ipc", f)
	f.Uint64("max-tx-data-size", ExpressLaneProxyConfigDefault.MaxTxDataSize, "maximum transaction size the sequencer will accept")
	f.Bool("metrics", ExpressLaneProxyConfigDefault.Metrics, "enable metrics")
	genericconf.MetricsServerAddOptions("metrics-server", f)
	f.Bool("pprof", ExpressLaneProxyConfigDefault.PProf, "enable pprof")
	genericconf.PProfAddOptions("pprof-cfg", f)
}

func (c *ExpressLaneProxyConfig) ShallowClone() *ExpressLaneProxyConfig {
	config := &ExpressLaneProxyConfig{}
	*config = *c
	return config
}

func (c *ExpressLaneProxyConfig) CanReload(new *ExpressLaneProxyConfig) error {
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

func (c *ExpressLaneProxyConfig) GetReloadInterval() time.Duration {
	return c.Conf.ReloadInterval
}

func (c *ExpressLaneProxyConfig) Validate() error {
	if err := gethexec.ValidateMaxTxDataSize(c.MaxTxDataSize); err != nil {
		return err
	}
	return nil
}

var DefaultExpressLaneProxyStackConfig = node.Config{
	DataDir:             node.DefaultDataDir(),
	HTTPPort:            node.DefaultHTTPPort,
	AuthAddr:            node.DefaultAuthHost,
	AuthPort:            node.DefaultAuthPort,
	AuthVirtualHosts:    node.DefaultAuthVhosts,
	HTTPModules:         []string{"eth"},
	HTTPHost:            "localhost",
	HTTPVirtualHosts:    []string{"localhost"},
	HTTPTimeouts:        rpc.DefaultHTTPTimeouts,
	WSHost:              "localhost",
	WSPort:              node.DefaultWSPort,
	WSModules:           []string{"eth"},
	GraphQLVirtualHosts: []string{"localhost"},
	P2P: p2p.Config{
		ListenAddr:  "",
		NoDiscovery: true,
		NoDial:      true,
	},
}
