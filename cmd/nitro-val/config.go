package main

import (
	"fmt"
	"reflect"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/nat"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/offchainlabs/nitro/cmd/conf"
	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/util/colors"
	"github.com/offchainlabs/nitro/validator/valnode"
	flag "github.com/spf13/pflag"
)

type ValidationNodeConfig struct {
	Conf          genericconf.ConfConfig          `koanf:"conf" reload:"hot"`
	Validation    valnode.Config                  `koanf:"validation" reload:"hot"`
	LogLevel      int                             `koanf:"log-level" reload:"hot"`
	LogType       string                          `koanf:"log-type" reload:"hot"`
	FileLogging   genericconf.FileLoggingConfig   `koanf:"file-logging" reload:"hot"`
	Persistent    conf.PersistentConfig           `koanf:"persistent"`
	HTTP          genericconf.HTTPConfig          `koanf:"http"`
	WS            genericconf.WSConfig            `koanf:"ws"`
	IPC           genericconf.IPCConfig           `koanf:"ipc"`
	AuthRPC       genericconf.AuthRPCConfig       `koanf:"auth"`
	Metrics       bool                            `koanf:"metrics"`
	MetricsServer genericconf.MetricsServerConfig `koanf:"metrics-server"`
}

var ValidationNodeConfigDefault = ValidationNodeConfig{
	Conf:          genericconf.ConfConfigDefault,
	LogLevel:      int(log.LvlInfo),
	LogType:       "plaintext",
	Persistent:    conf.PersistentConfigDefault,
	HTTP:          genericconf.HTTPConfigDefault,
	WS:            genericconf.WSConfigDefault,
	IPC:           genericconf.IPCConfigDefault,
	Metrics:       false,
	MetricsServer: genericconf.MetricsServerConfigDefault,
}

func ValidationNodeConfigAddOptions(f *flag.FlagSet) {
	genericconf.ConfConfigAddOptions("conf", f)
	valnode.ValidationConfigAddOptions("validation", f)
	f.Int("log-level", ValidationNodeConfigDefault.LogLevel, "log level")
	f.String("log-type", ValidationNodeConfigDefault.LogType, "log type (plaintext or json)")
	genericconf.FileLoggingConfigAddOptions("file-logging", f)
	conf.PersistentConfigAddOptions("persistent", f)
	genericconf.HTTPConfigAddOptions("http", f)
	genericconf.WSConfigAddOptions("ws", f)
	genericconf.IPCConfigAddOptions("ipc", f)
	genericconf.AuthRPCConfigAddOptions("auth", f)
	f.Bool("metrics", ValidationNodeConfigDefault.Metrics, "enable metrics")
	genericconf.MetricsServerAddOptions("metrics-server", f)
}

func (c *ValidationNodeConfig) ResolveDirectoryNames() error {
	err := c.Persistent.ResolveDirectoryNames()
	if err != nil {
		return err
	}

	return nil
}

func (c *ValidationNodeConfig) ShallowClone() *ValidationNodeConfig {
	config := &ValidationNodeConfig{}
	*config = *c
	return config
}

func (c *ValidationNodeConfig) CanReload(new *ValidationNodeConfig) error {
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

func (c *ValidationNodeConfig) GetReloadInterval() time.Duration {
	return c.Conf.ReloadInterval
}

func (c *ValidationNodeConfig) Validate() error {
	// TODO
	return nil
}

var DefaultValidationNodeStackConfig = node.Config{
	DataDir:             node.DefaultDataDir(),
	HTTPPort:            node.DefaultHTTPPort,
	AuthAddr:            node.DefaultAuthHost,
	AuthPort:            node.DefaultAuthPort,
	AuthVirtualHosts:    node.DefaultAuthVhosts,
	HTTPModules:         []string{""},
	HTTPVirtualHosts:    []string{"localhost"},
	HTTPTimeouts:        rpc.DefaultHTTPTimeouts,
	WSPort:              node.DefaultWSPort,
	WSModules:           []string{"validation"},
	GraphQLVirtualHosts: []string{"localhost"},
	P2P: p2p.Config{
		ListenAddr: ":30303",
		MaxPeers:   50,
		NAT:        nat.Any(),
	},
}
