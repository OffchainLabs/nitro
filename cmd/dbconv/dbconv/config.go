package dbconv

import (
	"errors"
	"fmt"

	flag "github.com/spf13/pflag"

	"github.com/offchainlabs/nitro/cmd/conf"
	"github.com/offchainlabs/nitro/cmd/genericconf"
)

var DBConfigDefaultDst = conf.DBConfig{
	Ancient:   "", // not supported
	DBEngine:  "pebble",
	Handles:   conf.PersistentConfigDefault.Handles,
	Cache:     2048, // 2048 MB
	Namespace: "dstdb/",
	Pebble:    conf.PebbleConfigDefault,
}

var DBConfigDefaultSrc = conf.DBConfig{
	Ancient:   "", // not supported
	DBEngine:  "leveldb",
	Handles:   conf.PersistentConfigDefault.Handles,
	Cache:     2048, // 2048 MB
	Namespace: "srcdb/",
}

type DBConvConfig struct {
	Src            conf.DBConfig                   `koanf:"src"`
	Dst            conf.DBConfig                   `koanf:"dst"`
	IdealBatchSize int                             `koanf:"ideal-batch-size"`
	Convert        bool                            `koanf:"convert"`
	Compact        bool                            `koanf:"compact"`
	Verify         string                          `koanf:"verify"`
	LogLevel       string                          `koanf:"log-level"`
	LogType        string                          `koanf:"log-type"`
	Metrics        bool                            `koanf:"metrics"`
	MetricsServer  genericconf.MetricsServerConfig `koanf:"metrics-server"`
}

var DefaultDBConvConfig = DBConvConfig{
	Src:            DBConfigDefaultSrc,
	Dst:            DBConfigDefaultDst,
	IdealBatchSize: 100 * 1024 * 1024, // 100 MB
	Convert:        false,
	Compact:        false,
	Verify:         "",
	LogLevel:       "INFO",
	LogType:        "plaintext",
	Metrics:        false,
	MetricsServer:  genericconf.MetricsServerConfigDefault,
}

func DBConvConfigAddOptions(f *flag.FlagSet) {
	conf.DBConfigAddOptions("src", f, &DefaultDBConvConfig.Src)
	conf.DBConfigAddOptions("dst", f, &DefaultDBConvConfig.Dst)
	f.Int("ideal-batch-size", DefaultDBConvConfig.IdealBatchSize, "ideal write batch size in bytes")
	f.Bool("convert", DefaultDBConvConfig.Convert, "enables conversion step")
	f.Bool("compact", DefaultDBConvConfig.Compact, "enables compaction step")
	f.String("verify", DefaultDBConvConfig.Verify, "enables verification step (\"\" = disabled, \"keys\" = only keys, \"full\" = keys and values)")
	f.String("log-level", DefaultDBConvConfig.LogLevel, "log level, valid values are CRIT, ERROR, WARN, INFO, DEBUG, TRACE")
	f.String("log-type", DefaultDBConvConfig.LogType, "log type (plaintext or json)")
	f.Bool("metrics", DefaultDBConvConfig.Metrics, "enable metrics")
	genericconf.MetricsServerAddOptions("metrics-server", f)
}

func (c *DBConvConfig) Validate() error {
	if c.Verify != "keys" && c.Verify != "full" && c.Verify != "" {
		return fmt.Errorf("Invalid verify mode: %v", c.Verify)
	}
	if !c.Convert && c.Verify == "" && !c.Compact {
		return errors.New("nothing to be done, conversion, verification and compaction disabled")
	}
	if c.IdealBatchSize <= 0 {
		return fmt.Errorf("Invalid ideal batch size: %d, has to be greater then 0", c.IdealBatchSize)
	}
	if c.Src.Ancient != "" || c.Dst.Ancient != "" {
		return errors.New("copying source database ancients is not supported and has to be done manually")
	}
	return nil
}
