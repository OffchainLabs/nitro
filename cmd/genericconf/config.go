// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package genericconf

import (
	"errors"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
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

type FileLoggingConfig struct {
	Enable     bool   `koanf:"enable"`
	File       string `koanf:"file"`
	MaxSize    int    `koanf:"max-size"`
	MaxAge     int    `koanf:"max-age"`
	MaxBackups int    `koanf:"max-backups"`
	LocalTime  bool   `koanf:"local-time"`
	Compress   bool   `koanf:"compress"`
	BufSize    int    `koanf:"buf-size"`
}

var DefaultFileLoggingConfig = FileLoggingConfig{
	Enable:     true,
	File:       "nitro.log",
	MaxSize:    5,     // 5Mb
	MaxAge:     0,     // don't remove old files based on age
	MaxBackups: 20,    // keep 20 files
	LocalTime:  false, // use UTC time
	Compress:   true,
	BufSize:    512,
}

func FileLoggingConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".enable", DefaultFileLoggingConfig.Enable, "enable logging to file")
	f.String(prefix+".file", DefaultFileLoggingConfig.File, "path to log file")
	f.Int(prefix+".max-size", DefaultFileLoggingConfig.MaxSize, "log file size in Mb that will trigger log file rotation (0 = trigger disabled)")
	f.Int(prefix+".max-age", DefaultFileLoggingConfig.MaxAge, "maximum number of days to retain old log files based on the timestamp encoded in their filename (0 = no limit)")
	f.Int(prefix+".max-backups", DefaultFileLoggingConfig.MaxBackups, "maximum number of old log files to retain (0 = no limit)")
	f.Bool(prefix+".local-time", DefaultFileLoggingConfig.LocalTime, "if true: local time will be used in old log filename timestamps")
	f.Bool(prefix+".compress", DefaultFileLoggingConfig.Compress, "enable compression of old log files")
	f.Int(prefix+".buf-size", DefaultFileLoggingConfig.BufSize, "size of intermediate log records buffer")
}

type RpcConfig struct {
	MaxBatchResponseSize int `koanf:"max-batch-response-size"`
}

var DefaultRpcConfig = RpcConfig{
	MaxBatchResponseSize: 10_000_000, // 10MB
}

func (c *RpcConfig) Apply() {
	rpc.MaxBatchResponseSize = c.MaxBatchResponseSize
}

func RpcConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Int(prefix+".max-batch-response-size", DefaultRpcConfig.MaxBatchResponseSize, "the maximum response size for a JSON-RPC request measured in bytes (-1 means no limit)")
}
