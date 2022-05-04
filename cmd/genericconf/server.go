// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package genericconf

import (
	"github.com/ethereum/go-ethereum/node"
	"github.com/spf13/pflag"
	"time"
)

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
	Port:       8547,
	API:        append(node.DefaultConfig.HTTPModules, "eth"),
	RPCPrefix:  node.DefaultConfig.HTTPPathPrefix,
	CORSDomain: node.DefaultConfig.HTTPCors,
	VHosts:     node.DefaultConfig.HTTPVirtualHosts,
}

func (c HTTPConfig) Apply(stackConf *node.Config) {
	stackConf.HTTPHost = c.Addr
	stackConf.HTTPPort = c.Port
	stackConf.HTTPModules = c.API
	stackConf.HTTPPathPrefix = c.RPCPrefix
	stackConf.HTTPCors = c.CORSDomain
	stackConf.HTTPVirtualHosts = c.VHosts
}

func HTTPConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.String(prefix+".addr", HTTPConfigDefault.Addr, "HTTP-RPC server listening interface")
	f.Int(prefix+".port", HTTPConfigDefault.Port, "HTTP-RPC server listening port")
	f.StringSlice(prefix+".api", HTTPConfigDefault.API, "APIs offered over the HTTP-RPC interface")
	f.String(prefix+".rpcprefix", HTTPConfigDefault.RPCPrefix, "HTTP path path prefix on which JSON-RPC is served. Use '/' to serve on all paths")
	f.StringSlice(prefix+".corsdomain", HTTPConfigDefault.CORSDomain, "Comma separated list of domains from which to accept cross origin requests (browser enforced)")
	f.StringSlice(prefix+".vhosts", HTTPConfigDefault.VHosts, "Comma separated list of virtual hostnames from which to accept requests (server enforced). Accepts '*' wildcard")
}

type WSConfig struct {
	Addr      string   `koanf:"addr"`
	Port      int      `koanf:"port"`
	API       []string `koanf:"api"`
	RPCPrefix string   `koanf:"rpcprefix"`
	Origins   []string `koanf:"origins"`
	ExposeAll bool     `koanf:"expose-all"`
}

var WSConfigDefault = WSConfig{
	Addr:      node.DefaultConfig.WSHost,
	Port:      8548,
	API:       append(node.DefaultConfig.WSModules, "eth"),
	RPCPrefix: node.DefaultConfig.WSPathPrefix,
	Origins:   node.DefaultConfig.WSOrigins,
	ExposeAll: node.DefaultConfig.WSExposeAll,
}

func (c WSConfig) Apply(stackConf *node.Config) {
	stackConf.WSHost = c.Addr
	stackConf.WSPort = c.Port
	stackConf.WSModules = c.API
	stackConf.WSPathPrefix = c.RPCPrefix
	stackConf.WSOrigins = c.Origins
	stackConf.WSExposeAll = c.ExposeAll
}

func WSConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.String(prefix+".addr", WSConfigDefault.Addr, "WS-RPC server listening interface")
	f.Int(prefix+".port", WSConfigDefault.Port, "WS-RPC server listening port")
	f.StringSlice(prefix+".api", WSConfigDefault.API, "APIs offered over the WS-RPC interface")
	f.String(prefix+".rpcprefix", WSConfigDefault.RPCPrefix, "WS path path prefix on which JSON-RPC is served. Use '/' to serve on all paths")
	f.StringSlice(prefix+".origins", WSConfigDefault.Origins, "Origins from which to accept websockets requests")
	f.Bool(prefix+".expose-all", WSConfigDefault.ExposeAll, "expose private api via websocket")
}

type MetricsServerConfig struct {
	Addr           string        `koanf:"addr"`
	Port           int           `koanf:"port"`
	UpdateInterval time.Duration `koanf:"update-interval"`
}

var MetricsServerConfigDefault = MetricsServerConfig{
	Addr:           "127.0.0.1",
	Port:           6070,
	UpdateInterval: 3 * time.Second,
}

func MetricsServerAddOptions(prefix string, f *pflag.FlagSet) {
	f.String(prefix+".addr", MetricsServerConfigDefault.Addr, "metrics server address")
	f.Int(prefix+".port", MetricsServerConfigDefault.Port, "metrics server port")
	f.Duration(prefix+".update-interval", MetricsServerConfigDefault.UpdateInterval, "metrics server update interval")
}
