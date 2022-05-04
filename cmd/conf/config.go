package conf

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/offchainlabs/nitro/das"
	"github.com/pkg/errors"
	flag "github.com/spf13/pflag"
)

const PASSWORD_NOT_SET = "PASSWORD_NOT_SET"

type ConfConfig struct {
	Dump      bool         `koanf:"dump"`
	EnvPrefix string       `koanf:"env-prefix"`
	File      []string     `koanf:"file"`
	S3        das.S3Config `koanf:"s3"`
	String    string       `koanf:"string"`
}

func ConfConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".dump", ConfConfigDefault.Dump, "print out currently active configuration file")
	f.String(prefix+".env-prefix", ConfConfigDefault.EnvPrefix, "environment variables with given prefix will be loaded as configuration values")
	f.StringSlice(prefix+".file", ConfConfigDefault.File, "name of configuration file")
	das.S3ConfigAddOptions(prefix+".s3", f)
	f.String(prefix+".string", ConfConfigDefault.String, "configuration as JSON string")
}

var ConfConfigDefault = ConfConfig{
	Dump:      false,
	EnvPrefix: "",
	File:      nil,
	S3:        das.DefaultS3Config,
	String:    "",
}

type RollupAddresses struct {
	Bridge                 common.Address `json:"bridge"`
	Inbox                  common.Address `json:"inbox"`
	SequencerInbox         common.Address `json:"sequencer-inbox"`
	Rollup                 common.Address `json:"rollup"`
	ValidatorUtils         common.Address `json:"validator-utils"`
	ValidatorWalletCreator common.Address `json:"validator-wallet-creator"`
	DeployedAt             uint64         `json:"deployed-at"`
}

type RollupAddressesConfig struct {
	Bridge                 string `koanf:"bridge"`
	Inbox                  string `koanf:"inbox"`
	SequencerInbox         string `koanf:"sequencer-inbox"`
	Rollup                 string `koanf:"rollup"`
	ValidatorUtils         string `koanf:"validator-utils"`
	ValidatorWalletCreator string `koanf:"validator-wallet-creator"`
	DeployedAt             uint64 `koanf:"deployed-at"`
}

var RollupAddressesConfigDefault = RollupAddressesConfig{}

func RollupAddressesConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.String(prefix+".bridge", "", "the bridge contract address")
	f.String(prefix+".inbox", "", "the inbox contract address")
	f.String(prefix+".sequencer-inbox", "", "the sequencer inbox contract address")
	f.String(prefix+".rollup", "", "the rollup contract address")
	f.String(prefix+".validator-utils", "", "the validator utils contract address")
	f.String(prefix+".validator-wallet-creator", "", "the validator wallet creator contract address")
	f.Uint64(prefix+".deployed-at", 0, "the block number at which the rollup was deployed")
}

func (c *RollupAddressesConfig) ParseAddresses() (RollupAddresses, error) {
	a := RollupAddresses{
		DeployedAt: c.DeployedAt,
	}
	strs := []string{
		c.Bridge,
		c.Inbox,
		c.SequencerInbox,
		c.Rollup,
		c.ValidatorUtils,
		c.ValidatorWalletCreator,
	}
	addrs := []*common.Address{
		&a.Bridge,
		&a.Inbox,
		&a.SequencerInbox,
		&a.Rollup,
		&a.ValidatorUtils,
		&a.ValidatorWalletCreator,
	}
	names := []string{
		"Bridge",
		"Inbox",
		"SequencerInbox",
		"Rollup",
		"ValidatorUtils",
		"ValidatorWalletCreator",
	}
	if len(strs) != len(addrs) {
		return RollupAddresses{}, fmt.Errorf("internal error: attempting to parse %v strings into %v addresses", len(strs), len(addrs))
	}
	complete := true
	for i, s := range strs {
		if !common.IsHexAddress(s) {
			log.Error("invalid address", "name", names[i], "value", s)
			complete = false
		}
		*addrs[i] = common.HexToAddress(s)
	}
	if !complete {
		return RollupAddresses{}, fmt.Errorf("invalid addresses")
	}
	return a, nil
}

type L1Config struct {
	ChainID            uint64                `koanf:"chain-id"`
	Rollup             RollupAddressesConfig `koanf:"rollup"`
	URL                string                `koanf:"url"`
	ConnectionAttempts int                   `koanf:"connection-attempts"`
	Wallet             WalletConfig          `koanf:"wallet"`
}

var L1ConfigDefault = L1Config{
	ChainID:            0,
	Rollup:             RollupAddressesConfigDefault,
	URL:                "",
	ConnectionAttempts: 15,
	Wallet:             WalletConfigDefault,
}

func L1ConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Uint64(prefix+".chain-id", L1ConfigDefault.ChainID, "if set other than 0, will be used to validate database and L1 connection")
	f.String(prefix+".url", L1ConfigDefault.URL, "layer 1 ethereum node RPC URL")
	RollupAddressesConfigAddOptions(prefix+".rollup", f)
	f.Int(prefix+".connection-attempts", L1ConfigDefault.ConnectionAttempts, "layer 1 RPC connection attempts (spaced out at least 1 second per attempt, 0 to retry infinitely)")
	WalletConfigAddOptions(prefix+".wallet", f, "wallet")
}

func (c *L1Config) ResolveDirectoryNames(chain string) {
	c.Wallet.ResolveDirectoryNames(chain)
}

type L2Config struct {
	ChainID   uint64       `koanf:"chain-id"`
	DevWallet WalletConfig `koanf:"dev-wallet"`
}

var L2ConfigDefault = L2Config{
	ChainID:   0,
	DevWallet: WalletConfigDefault,
}

func L2ConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Uint64(prefix+".chain-id", L2ConfigDefault.ChainID, "L2 chain ID (determines Arbitrum network)")
	// Dev wallet does not exist unless specified
	WalletConfigAddOptions(prefix+".dev-wallet", f, "")
}

func (c *L2Config) ResolveDirectoryNames(chain string) {
	c.DevWallet.ResolveDirectoryNames(chain)
}

type WalletConfig struct {
	Pathname     string `koanf:"pathname"`
	PasswordImpl string `koanf:"password"`
	PrivateKey   string `koanf:"private-key"`
	Account      string `koanf:"account"`
}

func (w *WalletConfig) Password() *string {
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

func WalletConfigAddOptions(prefix string, f *flag.FlagSet, defaultPathname string) {
	f.String(prefix+".pathname", defaultPathname, "pathname for wallet")
	f.String(prefix+".password", WalletConfigDefault.PasswordImpl, "wallet passphrase")
	f.String(prefix+".private-key", WalletConfigDefault.PasswordImpl, "private key for wallet")
	f.String(prefix+".account", WalletConfigDefault.Account, "account to use (default is first account in keystore)")
}

func (w *WalletConfig) ResolveDirectoryNames(chain string) {
	// Make wallet directories relative to chain directory if specified and not already absolute
	if len(w.Pathname) != 0 && !filepath.IsAbs(w.Pathname) {
		w.Pathname = path.Join(chain, w.Pathname)
	}
}

type PersistentConfig struct {
	GlobalConfig string `koanf:"global-config"`
	Chain        string `koanf:"chain"`
}

var PersistentConfigDefault = PersistentConfig{
	GlobalConfig: ".arbitrum",
	Chain:        "",
}

func PersistentConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.String(prefix+".global-config", PersistentConfigDefault.GlobalConfig, "directory to store global config")
	f.String(prefix+".chain", PersistentConfigDefault.Chain, "directory to store chain state")
}

func (c *PersistentConfig) ResolveDirectoryNames() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return errors.Wrap(err, "Unable to read users home directory")
	}

	// Make persistent storage directory relative to home directory if not already absolute
	if !filepath.IsAbs(c.GlobalConfig) {
		c.GlobalConfig = path.Join(homeDir, c.GlobalConfig)
	}
	err = os.MkdirAll(c.GlobalConfig, os.ModePerm)
	if err != nil {
		return errors.Wrap(err, "Unable to create global configuration directory")
	}

	// Make chain directory relative to persistent storage directory if not already absolute
	if !filepath.IsAbs(c.Chain) {
		c.Chain = path.Join(c.GlobalConfig, c.Chain)
	}
	err = os.MkdirAll(c.Chain, os.ModePerm)
	if err != nil {
		return errors.Wrap(err, "Unable to create chain directory")
	}
	if DatabaseInDirectory(c.Chain) {
		return errors.Errorf("Database in --persistent.chain (%s) directory, try specifying parent directory", c.Chain)
	}

	return nil
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
	Port:       8547,
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
	Port:      8548,
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

func DatabaseInDirectory(path string) bool {
	// Consider database present if file `CURRENT` in directory
	_, err := os.Stat(path + "/CURRENT")

	return err == nil
}
