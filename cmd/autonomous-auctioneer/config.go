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
	"github.com/offchainlabs/nitro/timeboost"
	"github.com/offchainlabs/nitro/util/colors"
)

type AutonomousAuctioneerConfig struct {
	AuctioneerServer timeboost.AuctioneerServerConfig `koanf:"auctioneer-server"`
	BidValidator     timeboost.BidValidatorConfig     `koanf:"bid-validator"`
	Persistent       conf.PersistentConfig            `koanf:"persistent"`
	Conf             genericconf.ConfConfig           `koanf:"conf" reload:"hot"`
	LogLevel         string                           `koanf:"log-level" reload:"hot"`
	LogType          string                           `koanf:"log-type" reload:"hot"`
	FileLogging      genericconf.FileLoggingConfig    `koanf:"file-logging" reload:"hot"`
	HTTP             genericconf.HTTPConfig           `koanf:"http"`
	WS               genericconf.WSConfig             `koanf:"ws"`
	IPC              genericconf.IPCConfig            `koanf:"ipc"`
	Metrics          bool                             `koanf:"metrics"`
	MetricsServer    genericconf.MetricsServerConfig  `koanf:"metrics-server"`
	PProf            bool                             `koanf:"pprof"`
	PprofCfg         genericconf.PProf                `koanf:"pprof-cfg"`
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

var AutonomousAuctioneerConfigDefault = AutonomousAuctioneerConfig{
	Conf:          genericconf.ConfConfigDefault,
	LogLevel:      "INFO",
	LogType:       "plaintext",
	HTTP:          HTTPConfigDefault,
	WS:            WSConfigDefault,
	IPC:           IPCConfigDefault,
	Metrics:       false,
	MetricsServer: genericconf.MetricsServerConfigDefault,
	PProf:         false,
	Persistent:    conf.PersistentConfigDefault,
	PprofCfg:      genericconf.PProfDefault,
}

func AuctioneerConfigAddOptions(f *pflag.FlagSet) {
	timeboost.AuctioneerServerConfigAddOptions("auctioneer-server", f)
	timeboost.BidValidatorConfigAddOptions("bid-validator", f)
	conf.PersistentConfigAddOptions("persistent", f)
	genericconf.ConfConfigAddOptions("conf", f)
	f.String("log-level", AutonomousAuctioneerConfigDefault.LogLevel, "log level, valid values are CRIT, ERROR, WARN, INFO, DEBUG, TRACE")
	f.String("log-type", AutonomousAuctioneerConfigDefault.LogType, "log type (plaintext or json)")
	genericconf.FileLoggingConfigAddOptions("file-logging", f)
	genericconf.HTTPConfigAddOptions("http", f)
	genericconf.WSConfigAddOptions("ws", f)
	genericconf.IPCConfigAddOptions("ipc", f)
	f.Bool("metrics", AutonomousAuctioneerConfigDefault.Metrics, "enable metrics")
	genericconf.MetricsServerAddOptions("metrics-server", f)
	f.Bool("pprof", AutonomousAuctioneerConfigDefault.PProf, "enable pprof")
	genericconf.PProfAddOptions("pprof-cfg", f)
}

func (c *AutonomousAuctioneerConfig) ShallowClone() *AutonomousAuctioneerConfig {
	config := &AutonomousAuctioneerConfig{}
	*config = *c
	return config
}

func (c *AutonomousAuctioneerConfig) CanReload(new *AutonomousAuctioneerConfig) error {
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

func (c *AutonomousAuctioneerConfig) GetReloadInterval() time.Duration {
	return c.Conf.ReloadInterval
}

func (c *AutonomousAuctioneerConfig) Validate() error {
	if err := c.AuctioneerServer.S3Storage.Validate(); err != nil {
		return err
	}
	return nil
}

var DefaultAuctioneerStackConfig = node.Config{
	DataDir:             node.DefaultDataDir(),
	HTTPPort:            node.DefaultHTTPPort,
	AuthAddr:            node.DefaultAuthHost,
	AuthPort:            node.DefaultAuthPort,
	AuthVirtualHosts:    node.DefaultAuthVhosts,
	HTTPModules:         []string{timeboost.AuctioneerNamespace},
	HTTPHost:            "localhost",
	HTTPVirtualHosts:    []string{"localhost"},
	HTTPTimeouts:        rpc.DefaultHTTPTimeouts,
	WSHost:              "localhost",
	WSPort:              node.DefaultWSPort,
	WSModules:           []string{timeboost.AuctioneerNamespace},
	GraphQLVirtualHosts: []string{"localhost"},
	P2P: p2p.Config{
		ListenAddr:  "",
		NoDiscovery: true,
		NoDial:      true,
	},
}
