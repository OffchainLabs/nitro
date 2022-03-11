package util

import (
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
	f.Bool(prefix+".dump", DefaultConfConfig.Dump, "print out currently active configuration file")
	f.String(prefix+".env-prefix", DefaultConfConfig.EnvPrefix, "environment variables with given prefix will be loaded as configuration values")
	f.String(prefix+".file", DefaultConfConfig.File, "name of configuration file")
	S3ConfigAddOptions(prefix+".s3", f)
	f.String(prefix+".string", DefaultConfConfig.String, "configuration as JSON string")
}

var DefaultConfConfig = ConfConfig{
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
	ChainID uint64 `koanf:"chain-id"`
	URL     string `koanf:"url"`
}

var DefaultL1Config = L1Config{
	ChainID: 0,
	URL:     "",
}

func L1ConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Uint64(prefix+".chain-id", DefaultL1Config.ChainID, "if set other than 0, will be used to validate database and L1 connection")
	f.String(prefix+".url", DefaultL1Config.URL, "layer 1 ethereum node RPC URL")
}

type WalletConfig struct {
	Pathname     string `koanf:"pathname"`
	PasswordImpl string `koanf:"password"`
	PrivateKey   string `koanf:"private-key"`
}

func (w WalletConfig) Password() *string {
	if w.PasswordImpl == PASSWORD_NOT_SET {
		return nil
	}
	return &w.PasswordImpl
}

var DefaultWalletConfig = WalletConfig{
	Pathname:     "",
	PasswordImpl: "",
	PrivateKey:   "",
}

func WalletConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.String(prefix+".pathname", DefaultWalletConfig.Pathname, "pathname for wallet")
	f.String(prefix+".password", DefaultWalletConfig.PasswordImpl, "wallet passphrase")
	f.String(prefix+".private-key", DefaultWalletConfig.PasswordImpl, "private key for wallet")
}
