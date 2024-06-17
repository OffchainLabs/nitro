package dbconv

import (
	"errors"
	"fmt"

	"github.com/offchainlabs/nitro/cmd/conf"
	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/execution/gethexec"
	flag "github.com/spf13/pflag"
)

type DBConfig struct {
	Data      string            `koanf:"data"`
	DBEngine  string            `koanf:"db-engine"`
	Handles   int               `koanf:"handles"`
	Cache     int               `koanf:"cache"`
	Namespace string            `koanf:"namespace"`
	Pebble    conf.PebbleConfig `koanf:"pebble"`
}

var DBConfigDefault = DBConfig{
	Handles: conf.PersistentConfigDefault.Handles,
	Cache:   gethexec.DefaultCachingConfig.DatabaseCache,
	Pebble:  conf.PebbleConfigDefault,
}

func DBConfigAddOptions(prefix string, f *flag.FlagSet, defaultNamespace string) {
	f.String(prefix+".data", DBConfigDefault.Data, "directory of stored chain state")
	f.String(prefix+".db-engine", DBConfigDefault.DBEngine, "backing database implementation to use ('leveldb' or 'pebble')")
	f.Int(prefix+".handles", DBConfigDefault.Handles, "number of files to be open simultaneously")
	f.Int(prefix+".cache", DBConfigDefault.Cache, "the capacity(in megabytes) of the data caching")
	f.String(prefix+".namespace", defaultNamespace, "metrics namespace")
	conf.PebbleConfigAddOptions(prefix+".pebble", f)
}

type DBConvConfig struct {
	Src            DBConfig                        `koanf:"src"`
	Dst            DBConfig                        `koanf:"dst"`
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
	DBConfigAddOptions("src", f, "srcdb/")
	DBConfigAddOptions("dst", f, "destdb/")
	f.Int("ideal-batch-size", DefaultDBConvConfig.IdealBatchSize, "ideal write batch size")
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
	return nil
}
