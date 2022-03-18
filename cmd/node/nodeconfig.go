package main

import (
	"context"
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/pkg/errors"
	flag "github.com/spf13/pflag"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/cmd/util"
)

type NodeConfig struct {
	Conf       util.ConfConfig  `koanf:"conf"`
	Node       arbnode.Config   `koanf:"node"`
	L1         util.L1Config    `koanf:"l1"`
	L2         util.L2Config    `koanf:"l2"`
	LogLevel   int              `koanf:"log-level"`
	DataDir    string           `koanf:"data-dir"`
	Persistent PersistentConfig `koanf:"persistent"`
	HTTP       HTTPConfig       `koanf:"http"`
	WS         WSConfig         `koanf:"ws"`
	DevInit    bool             `koanf:"dev-init"`
	ImportFile string           `koanf:"import-file"`
}

var NodeConfigDefault = NodeConfig{
	Conf:       util.ConfConfigDefault,
	Node:       arbnode.ConfigDefault,
	L1:         util.L1ConfigDefault,
	L2:         util.L2ConfigDefault,
	LogLevel:   int(log.LvlInfo),
	Persistent: PersistentConfigDefault,
	HTTP:       HTTPConfigDefault,
	WS:         WSConfigDefault,
	DevInit:    false,
	ImportFile: "",
}

func NodeConfigAddOptions(f *flag.FlagSet) {
	util.ConfConfigAddOptions("conf", f)
	arbnode.ConfigAddOptions("node", f, true, true)
	util.L1ConfigAddOptions("l1", f)
	util.L2ConfigAddOptions("l2", f)
	f.Int("log-level", NodeConfigDefault.LogLevel, "log level")
	PersistentConfigAddOptions("persistent", f)
	HTTPConfigAddOptions("http", f)
	WSConfigAddOptions("ws", f)
	f.Bool("dev-init", NodeConfigDefault.DevInit, "init with dev data (1 account with balance) instead of file import")
	f.String("import-file", NodeConfigDefault.ImportFile, "path for json data to import")
}

type PersistentConfig struct {
	Chain        string `koanf:"chain"`
	GlobalConfig string `koanf:"global-config"`
}

var PersistentConfigDefault = PersistentConfig{
	Chain:        "",
	GlobalConfig: "",
}

func PersistentConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.String(prefix+".chain", PersistentConfigDefault.Chain, "directory to store chain state")
	f.String(prefix+".global-config", PersistentConfigDefault.GlobalConfig, "directory to store global config")
}

type HTTPConfig struct {
	Addr       string   `koanf:"addr"`
	Port       int      `koanf:"port"`
	API        []string `koanf:"api"`
	RPCPrefix  string   `koanf:"rpcprefix"`
	CORSDomain []string `koanf:"corsdomain"`
	VHosts     []string `koanf:"vhosts"`
}

var HTTPConfigDefault = HTTPConfig{
	Addr:       node.DefaultConfig.HTTPHost,
	Port:       node.DefaultConfig.HTTPPort,
	API:        append(node.DefaultConfig.HTTPModules, "eth"),
	RPCPrefix:  node.DefaultConfig.HTTPPathPrefix,
	CORSDomain: node.DefaultConfig.HTTPCors,
	VHosts:     node.DefaultConfig.HTTPVirtualHosts,
}

func HTTPConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.String(prefix+".addr", HTTPConfigDefault.Addr, "HTTP-RPC server listening interface")
	f.Int(prefix+".port", HTTPConfigDefault.Port, "HTTP-RPC server listening port")
	f.StringSlice(prefix+".api", HTTPConfigDefault.API, "APIs offered over the HTTP-RPC interface")
	f.String(prefix+".rpc-prefix", HTTPConfigDefault.RPCPrefix, "HTTP path path prefix on which JSON-RPC is served. Use '/' to serve on all paths")
	f.StringSlice(prefix+".cors-domain", HTTPConfigDefault.CORSDomain, "Comma separated list of domains from which to accept cross origin requests (browser enforced)")
	f.StringSlice(prefix+".vhosts", HTTPConfigDefault.VHosts, "Comma separated list of virtual hostnames from which to accept requests (server enforced). Accepts '*' wildcard")
}

type WSConfig struct {
	Addr      string   `koanf:"addr"`
	Port      int      `koanf:"port"`
	API       []string `koanf:"api"`
	RPCPrefix string   `koanf:"rpc-prefix"`
	Origins   []string `koanf:"origins"`
	ExposeAll bool     `koanf:"expose-all"`
}

var WSConfigDefault = WSConfig{
	Addr:      node.DefaultConfig.WSHost,
	Port:      node.DefaultConfig.WSPort,
	API:       append(node.DefaultConfig.WSModules, "eth"),
	RPCPrefix: node.DefaultConfig.WSPathPrefix,
	Origins:   node.DefaultConfig.WSOrigins,
	ExposeAll: node.DefaultConfig.WSExposeAll,
}

func WSConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.String(prefix+".addr", WSConfigDefault.Addr, "WS-RPC server listening interface")
	f.Int(prefix+".port", WSConfigDefault.Port, "WS-RPC server listening port")
	f.StringSlice(prefix+".api", WSConfigDefault.API, "APIs offered over the WS-RPC interface")
	f.String(prefix+".rpc-prefix", WSConfigDefault.RPCPrefix, "WS path path prefix on which JSON-RPC is served. Use '/' to serve on all paths")
	f.StringSlice(prefix+".origins", WSConfigDefault.Origins, "Origins from which to accept websockets requests")
	f.Bool(prefix+".expose-all", WSConfigDefault.ExposeAll, "expose private api via websocket")
}

func ParseNode(_ context.Context) (*NodeConfig, *util.WalletConfig, *util.WalletConfig, error) {
	f := flag.NewFlagSet("", flag.ContinueOnError)

	NodeConfigAddOptions(f)

	k, err := util.BeginCommonParse(f)
	if err != nil {
		return nil, nil, nil, err
	}

	var nodeConfig NodeConfig
	if err := util.EndCommonParse(k, &nodeConfig); err != nil {
		return nil, nil, nil, err
	}

	if nodeConfig.Conf.Dump {
		// Print out current configuration

		// Don't keep printing configuration file and don't print wallet passwords
		err := k.Load(confmap.Provider(map[string]interface{}{
			"conf.dump":             false,
			"wallet.l1.password":    "",
			"wallet.l1.private-key": "",
			"wallet.l2.password":    "",
			"wallet.l2.private-key": "",
		}, "."), nil)
		if err != nil {
			return nil, nil, nil, errors.Wrap(err, "error removing extra parameters before dump")
		}

		c, err := k.Marshal(json.Parser())
		if err != nil {
			return nil, nil, nil, errors.Wrap(err, "unable to marshal config file to JSON")
		}

		fmt.Println(string(c))
		os.Exit(0)
	}

	// Don't pass around wallet contents with normal configuration
	l1wallet := nodeConfig.L1.Wallet
	l2wallet := nodeConfig.L2.Wallet
	nodeConfig.L1.Wallet = util.WalletConfig{}
	nodeConfig.L2.Wallet = util.WalletConfig{}

	return &nodeConfig, &l1wallet, &l2wallet, nil
}
