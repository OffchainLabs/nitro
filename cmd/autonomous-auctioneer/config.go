package main

import (
	"fmt"

	"reflect"
	"time"

	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/offchainlabs/nitro/cmd/conf"
	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/timeboost"
	"github.com/offchainlabs/nitro/util/colors"
	flag "github.com/spf13/pflag"
)

type AuctioneerConfig struct {
	Conf          genericconf.ConfConfig          `koanf:"conf" reload:"hot"`
	LogLevel      string                          `koanf:"log-level" reload:"hot"`
	LogType       string                          `koanf:"log-type" reload:"hot"`
	FileLogging   genericconf.FileLoggingConfig   `koanf:"file-logging" reload:"hot"`
	HTTP          genericconf.HTTPConfig          `koanf:"http"`
	WS            genericconf.WSConfig            `koanf:"ws"`
	IPC           genericconf.IPCConfig           `koanf:"ipc"`
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

var AuctioneerConfigDefault = AuctioneerConfig{
	Conf:          genericconf.ConfConfigDefault,
	LogLevel:      "INFO",
	LogType:       "plaintext",
	HTTP:          HTTPConfigDefault,
	WS:            WSConfigDefault,
	IPC:           IPCConfigDefault,
	Metrics:       false,
	MetricsServer: genericconf.MetricsServerConfigDefault,
	PProf:         false,
	PprofCfg:      genericconf.PProfDefault,
}

func ValidationNodeConfigAddOptions(f *flag.FlagSet) {
	genericconf.ConfConfigAddOptions("conf", f)
	f.String("log-level", AuctioneerConfigDefault.LogLevel, "log level, valid values are CRIT, ERROR, WARN, INFO, DEBUG, TRACE")
	f.String("log-type", AuctioneerConfigDefault.LogType, "log type (plaintext or json)")
	genericconf.FileLoggingConfigAddOptions("file-logging", f)
	conf.PersistentConfigAddOptions("persistent", f)
	genericconf.HTTPConfigAddOptions("http", f)
	genericconf.WSConfigAddOptions("ws", f)
	genericconf.IPCConfigAddOptions("ipc", f)
	genericconf.AuthRPCConfigAddOptions("auth", f)
	f.Bool("metrics", AuctioneerConfigDefault.Metrics, "enable metrics")
	genericconf.MetricsServerAddOptions("metrics-server", f)
	f.Bool("pprof", AuctioneerConfigDefault.PProf, "enable pprof")
	genericconf.PProfAddOptions("pprof-cfg", f)
}

func (c *AuctioneerConfig) ShallowClone() *AuctioneerConfig {
	config := &AuctioneerConfig{}
	*config = *c
	return config
}

func (c *AuctioneerConfig) CanReload(new *AuctioneerConfig) error {
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

func (c *AuctioneerConfig) GetReloadInterval() time.Duration {
	return c.Conf.ReloadInterval
}

func (c *AuctioneerConfig) Validate() error {
	return nil
}

var DefaultAuctioneerStackConfig = node.Config{
	DataDir:             node.DefaultDataDir(),
	HTTPPort:            node.DefaultHTTPPort,
	AuthAddr:            node.DefaultAuthHost,
	AuthPort:            node.DefaultAuthPort,
	AuthVirtualHosts:    node.DefaultAuthVhosts,
	HTTPModules:         []string{timeboost.AuctioneerNamespace},
	HTTPVirtualHosts:    []string{"localhost"},
	HTTPTimeouts:        rpc.DefaultHTTPTimeouts,
	WSPort:              node.DefaultWSPort,
	WSModules:           []string{timeboost.AuctioneerNamespace},
	GraphQLVirtualHosts: []string{"localhost"},
	P2P: p2p.Config{
		ListenAddr:  "",
		NoDiscovery: true,
		NoDial:      true,
	},
}
