package conf

import (
	"errors"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/arbnode"
	flag "github.com/spf13/pflag"
)

const PASSWORD_NOT_SET = "PASSWORD_NOT_SET"

type ConfConfig struct {
	Dump      bool     `koanf:"dump"`
	EnvPrefix string   `koanf:"env-prefix"`
	File      string   `koanf:"file"`
	S3        S3Config `koanf:"s3"`
	String    string   `koanf:"string"`
}

func ConfConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".dump", ConfConfigDefault.Dump, "print out currently active configuration file")
	f.String(prefix+".env-prefix", ConfConfigDefault.EnvPrefix, "environment variables with given prefix will be loaded as configuration values")
	f.String(prefix+".file", ConfConfigDefault.File, "name of configuration file")
	S3ConfigAddOptions(prefix+".s3", f)
	f.String(prefix+".string", ConfConfigDefault.String, "configuration as JSON string")
}

var ConfConfigDefault = ConfConfig{
	Dump:      false,
	EnvPrefix: "",
	File:      "",
	S3:        DefaultS3Config,
	String:    "",
}

type S3Config struct {
	AccessKey string `koanf:"access-key"`
	Bucket    string `koanf:"bucket"`
	ObjectKey string `koanf:"object-key"`
	Region    string `koanf:"region"`
	SecretKey string `koanf:"secret-key"`
}

func S3ConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.String(prefix+".access-key", DefaultS3Config.AccessKey, "S3 access key")
	f.String(prefix+".bucket", DefaultS3Config.Bucket, "S3 bucket")
	f.String(prefix+".object-key", DefaultS3Config.ObjectKey, "S3 object key")
	f.String(prefix+".region", DefaultS3Config.Region, "S3 region")
	f.String(prefix+".secret-key", DefaultS3Config.SecretKey, "S3 secret key")
}

var DefaultS3Config = S3Config{
	AccessKey: "",
	Bucket:    "",
	ObjectKey: "",
	Region:    "",
	SecretKey: "",
}

type L1Config struct {
	ChainID            uint64                        `koanf:"chain-id"`
	Rollup             arbnode.RollupAddressesConfig `koanf:"rollup"`
	URL                string                        `koanf:"url"`
	ConnectionAttempts int                           `koanf:"connection-attempts"`
	Wallet             WalletConfig                  `koanf:"wallet"`
}

var L1ConfigDefault = L1Config{
	ChainID:            1337,
	Rollup:             arbnode.RollupAddressesConfig{},
	URL:                "",
	ConnectionAttempts: 15,
	Wallet:             WalletConfigDefault,
}

func L1ConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Uint64(prefix+".chain-id", L1ConfigDefault.ChainID, "if set other than 0, will be used to validate database and L1 connection")
	f.String(prefix+".url", L1ConfigDefault.URL, "layer 1 ethereum node RPC URL")
	arbnode.RollupAddressesConfigAddOptions(prefix+".rollup", f)
	f.Int(prefix+".connection-attempts", L1ConfigDefault.ConnectionAttempts, "layer 1 RPC connection attempts (spaced out at least 1 second per attempt, 0 to retry infinitely)")
	WalletConfigAddOptions(prefix+".wallet", f)
}

type L2Config struct {
	ChainID uint64       `koanf:"chain-id"`
	Wallet  WalletConfig `koanf:"wallet"`
}

var L2ConfigDefault = L2Config{
	ChainID: params.ArbitrumTestnetChainConfig().ChainID.Uint64(),
	Wallet:  WalletConfigDefault,
}

func L2ConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Uint64(prefix+".chain-id", L2ConfigDefault.ChainID, "L2 chain ID (determines Arbitrum network)")
	WalletConfigAddOptions(prefix+".wallet", f)
}

type WalletConfig struct {
	Pathname     string `koanf:"pathname"`
	PasswordImpl string `koanf:"password"`
	PrivateKey   string `koanf:"private-key"`
	Account      string `koanf:"account"`
}

func (w WalletConfig) Password() *string {
	if w.PasswordImpl == PASSWORD_NOT_SET {
		return nil
	}
	return &w.PasswordImpl
}

var WalletConfigDefault = WalletConfig{
	Pathname:     "",
	PasswordImpl: "",
	PrivateKey:   "",
	Account:      "",
}

func WalletConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.String(prefix+".pathname", WalletConfigDefault.Pathname, "pathname for wallet")
	f.String(prefix+".password", WalletConfigDefault.PasswordImpl, "wallet passphrase")
	f.String(prefix+".private-key", WalletConfigDefault.PasswordImpl, "private key for wallet")
	f.String(prefix+".account", WalletConfigDefault.Account, "account to use (default is first account in keystore)")
}

type PersistentConfig struct {
	GlobalConfig string `koanf:"global-config"`
	Chain        string `koanf:"chain"`
	Data         string `koanf:"data"`
}

var PersistentConfigDefault = PersistentConfig{
	GlobalConfig: "",
	Chain:        "",
	Data:         "",
}

func PersistentConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.String(prefix+".global-config", PersistentConfigDefault.GlobalConfig, "directory to store global config")
	f.String(prefix+".chain", PersistentConfigDefault.Chain, "directory to store chain state")
	f.String(prefix+".data", PersistentConfigDefault.Data, "directory for data storage requirements")
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
	Port:       7545,
	API:        append(node.DefaultConfig.HTTPModules, "eth"),
	RPCPrefix:  node.DefaultConfig.HTTPPathPrefix,
	CORSDomain: node.DefaultConfig.HTTPCors,
	VHosts:     node.DefaultConfig.HTTPVirtualHosts,
}

func HTTPConfigAddOptions(prefix string, f *flag.FlagSet) {
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
	Port:      7546,
	API:       append(node.DefaultConfig.WSModules, "eth"),
	RPCPrefix: node.DefaultConfig.WSPathPrefix,
	Origins:   node.DefaultConfig.WSOrigins,
	ExposeAll: node.DefaultConfig.WSExposeAll,
}

func WSConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.String(prefix+".addr", WSConfigDefault.Addr, "WS-RPC server listening interface")
	f.Int(prefix+".port", WSConfigDefault.Port, "WS-RPC server listening port")
	f.StringSlice(prefix+".api", WSConfigDefault.API, "APIs offered over the WS-RPC interface")
	f.String(prefix+".rpcprefix", WSConfigDefault.RPCPrefix, "WS path path prefix on which JSON-RPC is served. Use '/' to serve on all paths")
	f.StringSlice(prefix+".origins", WSConfigDefault.Origins, "Origins from which to accept websockets requests")
	f.Bool(prefix+".expose-all", WSConfigDefault.ExposeAll, "expose private api via websocket")
}

func ParseLogType(logType string) (log.Format, error) {
	if logType == "plaintext" {
		return log.TerminalFormat(false), nil
	} else if logType == "json" {
		return log.JSONFormat(), nil
	}
	return nil, errors.New("invalid log type")
}

type MetricsServerConfig struct {
	Addr string `koanf:"addr"`
	Port int    `koanf:"port"`
}

var MetricsServerConfigDefault = MetricsServerConfig{
	Addr: "127.0.0.1",
	Port: 6070,
}

func MetricsServerAddOptions(prefix string, f *flag.FlagSet) {
	f.String(prefix+".addr", MetricsServerConfigDefault.Addr, "metrics server address")
	f.Int(prefix+".port", MetricsServerConfigDefault.Port, "metrics server port")
}
