// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package genericconf

import (
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/pkg/errors"
	flag "github.com/spf13/pflag"
)

type ConfConfig struct {
	Dump           bool          `koanf:"dump"`
	EnvPrefix      string        `koanf:"env-prefix"`
	File           []string      `koanf:"file"`
	S3             S3Config      `koanf:"s3"`
	String         string        `koanf:"string"`
	ReloadInterval time.Duration `koanf:"reload-interval" reload:"hot"`
}

func ConfConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".dump", ConfConfigDefault.Dump, "print out currently active configuration file")
	f.String(prefix+".env-prefix", ConfConfigDefault.EnvPrefix, "environment variables with given prefix will be loaded as configuration values")
	f.StringSlice(prefix+".file", ConfConfigDefault.File, "name of configuration file")
	S3ConfigAddOptions(prefix+".s3", f)
	f.String(prefix+".string", ConfConfigDefault.String, "configuration as JSON string")
	f.Duration(prefix+".reload-interval", ConfConfigDefault.ReloadInterval, "how often to reload configuration (0=disable periodic reloading)")
}

var ConfConfigDefault = ConfConfig{
	Dump:           false,
	EnvPrefix:      "",
	File:           nil,
	S3:             DefaultS3Config,
	String:         "",
	ReloadInterval: 0,
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

func ParseLogType(logType string) (log.Format, error) {
	if logType == "plaintext" {
		return log.TerminalFormat(false), nil
	} else if logType == "json" {
		return log.JSONFormat(), nil
	}
	return nil, errors.New("invalid log type")
}
