package util

import (
	"github.com/ethereum/go-ethereum/params"
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
	ChainID    uint64       `koanf:"chain-id"`
	Deployment string       `koanf:"deployment"`
	URL        string       `koanf:"url"`
	Wallet     WalletConfig `koanf:"wallet"`
}

var L1ConfigDefault = L1Config{
	ChainID:    1337,
	Deployment: "",
	URL:        "",
	Wallet:     WalletConfigDefault,
}

func L1ConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Uint64(prefix+".chain-id", L1ConfigDefault.ChainID, "if set other than 0, will be used to validate database and L1 connection")
	f.String(prefix+".deployment", L1ConfigDefault.Deployment, "json file including the existing deployment information")
	f.String(prefix+".url", L1ConfigDefault.URL, "layer 1 ethereum node RPC URL")
	WalletConfigAddOptions("wallet", f)
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
	f.Uint64(prefix+".chain-id", L1ConfigDefault.ChainID, "L2 chain ID (determines Arbitrum network)")
	WalletConfigAddOptions("wallet", f)
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
