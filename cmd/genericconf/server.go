// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package genericconf

import (
	"time"

	"github.com/offchainlabs/nitro/consensus"
	"github.com/offchainlabs/nitro/execution"
	flag "github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/node"
)

type HTTPConfig struct {
	Addr           string                  `koanf:"addr"`
	Port           int                     `koanf:"port"`
	API            []string                `koanf:"api"`
	RPCPrefix      string                  `koanf:"rpcprefix"`
	CORSDomain     []string                `koanf:"corsdomain"`
	VHosts         []string                `koanf:"vhosts"`
	ServerTimeouts HTTPServerTimeoutConfig `koanf:"server-timeouts"`
}

var HTTPConfigDefault = HTTPConfig{
	Addr:           node.DefaultConfig.HTTPHost,
	Port:           8547,
	API:            append(node.DefaultConfig.HTTPModules, "eth", "arb"),
	RPCPrefix:      node.DefaultConfig.HTTPPathPrefix,
	CORSDomain:     node.DefaultConfig.HTTPCors,
	VHosts:         node.DefaultConfig.HTTPVirtualHosts,
	ServerTimeouts: HTTPServerTimeoutConfigDefault,
}

type HTTPServerTimeoutConfig struct {
	ReadTimeout       time.Duration `koanf:"read-timeout"`
	ReadHeaderTimeout time.Duration `koanf:"read-header-timeout"`
	WriteTimeout      time.Duration `koanf:"write-timeout"`
	IdleTimeout       time.Duration `koanf:"idle-timeout"`
}

// HTTPServerTimeoutConfigDefault use geth defaults
var HTTPServerTimeoutConfigDefault = HTTPServerTimeoutConfig{
	ReadTimeout:       30 * time.Second,
	ReadHeaderTimeout: 30 * time.Second,
	WriteTimeout:      30 * time.Second,
	IdleTimeout:       120 * time.Second,
}

func (c HTTPConfig) Apply(stackConf *node.Config) {
	stackConf.HTTPHost = c.Addr
	stackConf.HTTPPort = c.Port
	stackConf.HTTPModules = c.API
	stackConf.HTTPPathPrefix = c.RPCPrefix
	stackConf.HTTPCors = c.CORSDomain
	stackConf.HTTPVirtualHosts = c.VHosts
	stackConf.HTTPTimeouts.ReadTimeout = c.ServerTimeouts.ReadTimeout
	// TODO ReadHeaderTimeout pending on https://github.com/ethereum/go-ethereum/pull/25338
	// stackConf.HTTPTimeouts.ReadHeaderTimeout = c.ServerTimeouts.ReadHeaderTimeout
	stackConf.HTTPTimeouts.WriteTimeout = c.ServerTimeouts.WriteTimeout
	stackConf.HTTPTimeouts.IdleTimeout = c.ServerTimeouts.IdleTimeout
}

func HTTPConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.String(prefix+".addr", HTTPConfigDefault.Addr, "HTTP-RPC server listening interface")
	f.Int(prefix+".port", HTTPConfigDefault.Port, "HTTP-RPC server listening port")
	f.StringSlice(prefix+".api", HTTPConfigDefault.API, "APIs offered over the HTTP-RPC interface")
	f.String(prefix+".rpcprefix", HTTPConfigDefault.RPCPrefix, "HTTP path path prefix on which JSON-RPC is served. Use '/' to serve on all paths")
	f.StringSlice(prefix+".corsdomain", HTTPConfigDefault.CORSDomain, "Comma separated list of domains from which to accept cross origin requests (browser enforced)")
	f.StringSlice(prefix+".vhosts", HTTPConfigDefault.VHosts, "Comma separated list of virtual hostnames from which to accept requests (server enforced). Accepts '*' wildcard")
	HTTPServerTimeoutConfigAddOptions(prefix+".server-timeouts", f)
}

func HTTPServerTimeoutConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Duration(prefix+".read-timeout", HTTPServerTimeoutConfigDefault.ReadTimeout, "the maximum duration for reading the entire request (http.Server.ReadTimeout)")
	f.Duration(prefix+".read-header-timeout", HTTPServerTimeoutConfigDefault.ReadHeaderTimeout, "the amount of time allowed to read the request headers (http.Server.ReadHeaderTimeout)")
	f.Duration(prefix+".write-timeout", HTTPServerTimeoutConfigDefault.WriteTimeout, "the maximum duration before timing out writes of the response (http.Server.WriteTimeout)")
	f.Duration(prefix+".idle-timeout", HTTPServerTimeoutConfigDefault.IdleTimeout, "the maximum amount of time to wait for the next request when keep-alives are enabled (http.Server.IdleTimeout)")
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
	API:       append(node.DefaultConfig.WSModules, "eth", "arb"),
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

func WSConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.String(prefix+".addr", WSConfigDefault.Addr, "WS-RPC server listening interface")
	f.Int(prefix+".port", WSConfigDefault.Port, "WS-RPC server listening port")
	f.StringSlice(prefix+".api", WSConfigDefault.API, "APIs offered over the WS-RPC interface")
	f.String(prefix+".rpcprefix", WSConfigDefault.RPCPrefix, "WS path path prefix on which JSON-RPC is served. Use '/' to serve on all paths")
	f.StringSlice(prefix+".origins", WSConfigDefault.Origins, "Origins from which to accept websockets requests")
	f.Bool(prefix+".expose-all", WSConfigDefault.ExposeAll, "expose private api via websocket")
}

type IPCConfig struct {
	Path string `koanf:"path"`
}

var IPCConfigDefault = IPCConfig{
	Path: "",
}

func (c *IPCConfig) Apply(stackConf *node.Config) {
	stackConf.IPCPath = c.Path
}

func IPCConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.String(prefix+".path", IPCConfigDefault.Path, "Requested location to place the IPC endpoint. An empty path disables IPC.")
}

type GraphQLConfig struct {
	Enable     bool     `koanf:"enable"`
	CORSDomain []string `koanf:"corsdomain"`
	VHosts     []string `koanf:"vhosts"`
}

var GraphQLConfigDefault = GraphQLConfig{
	Enable:     false,
	CORSDomain: node.DefaultConfig.GraphQLCors,
	VHosts:     node.DefaultConfig.GraphQLVirtualHosts,
}

func (c GraphQLConfig) Apply(stackConf *node.Config) {
	stackConf.GraphQLCors = c.CORSDomain
	stackConf.GraphQLVirtualHosts = c.VHosts
}

func GraphQLConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".enable", GraphQLConfigDefault.Enable, "Enable graphql endpoint on the rpc endpoint")
	f.StringSlice(prefix+".corsdomain", GraphQLConfigDefault.CORSDomain, "Comma separated list of domains from which to accept cross origin requests (browser enforced)")
	f.StringSlice(prefix+".vhosts", GraphQLConfigDefault.VHosts, "Comma separated list of virtual hostnames from which to accept requests (server enforced). Accepts '*' wildcard")
}

type AuthRPCConfig struct {
	Addr      string   `koanf:"addr"`
	Port      int      `koanf:"port"`
	API       []string `koanf:"api"`
	Origins   []string `koanf:"origins"`
	JwtSecret string   `koanf:"jwtsecret"`
}

func (a AuthRPCConfig) Apply(stackConf *node.Config) {
	stackConf.AuthAddr = a.Addr
	stackConf.AuthPort = a.Port
	stackConf.AuthVirtualHosts = []string{} // dont allow http access
	stackConf.JWTSecret = a.JwtSecret
	// a few settings are not available as stanard config, but we can change the default. sigh..
	node.DefaultAuthOrigins = make([]string, len(a.Origins))
	copy(node.DefaultAuthOrigins, a.Origins)
	node.DefaultAuthModules = make([]string, len(a.API))
	copy(node.DefaultAuthModules, a.API)
}

var AuthRPCConfigDefault = AuthRPCConfig{
	Addr:      "127.0.0.1",
	Port:      8549,
	API:       []string{"validation", execution.RPCNamespace, consensus.RPCNamespace},
	Origins:   []string{"localhost"},
	JwtSecret: "",
}

func AuthRPCConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.String(prefix+".addr", AuthRPCConfigDefault.Addr, "AUTH-RPC server listening interface")
	f.String(prefix+".jwtsecret", AuthRPCConfigDefault.JwtSecret, "Path to file holding JWT secret (32B hex)")
	f.Int(prefix+".port", AuthRPCConfigDefault.Port, "AUTH-RPC server listening port")
	f.StringSlice(prefix+".origins", AuthRPCConfigDefault.Origins, "Origins from which to accept AUTH requests")
	f.StringSlice(prefix+".api", AuthRPCConfigDefault.API, "APIs offered over the AUTH-RPC interface")
}

type MetricsServerConfig struct {
	Addr           string        `koanf:"addr"`
	Port           int           `koanf:"port"`
	Pprof          bool          `koanf:"pprof"`
	UpdateInterval time.Duration `koanf:"update-interval"`
}

var MetricsServerConfigDefault = MetricsServerConfig{
	Addr:           "127.0.0.1",
	Port:           6070,
	Pprof:          false,
	UpdateInterval: 3 * time.Second,
}

func MetricsServerAddOptions(prefix string, f *flag.FlagSet) {
	f.String(prefix+".addr", MetricsServerConfigDefault.Addr, "metrics server address")
	f.Int(prefix+".port", MetricsServerConfigDefault.Port, "metrics server port")
	f.Bool(prefix+".pprof", MetricsServerConfigDefault.Pprof, "enable profiling for Go")
	f.Duration(prefix+".update-interval", MetricsServerConfigDefault.UpdateInterval, "metrics server update interval")
}
