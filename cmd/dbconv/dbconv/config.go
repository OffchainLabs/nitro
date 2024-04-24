package dbconv

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/cmd/conf"
	"github.com/offchainlabs/nitro/cmd/genericconf"
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

// TODO
var DBConfigDefault = DBConfig{}

func DBConfigAddOptions(prefix string, f *flag.FlagSet, defaultNamespace string) {
	// TODO
	f.String(prefix+".data", DBConfigDefault.Data, "directory of stored chain state")
	f.String(prefix+".db-engine", DBConfigDefault.DBEngine, "backing database implementation to use ('leveldb' or 'pebble')")
	f.Int(prefix+".handles", DBConfigDefault.Handles, "number of file descriptor handles to use for the database")
	f.Int(prefix+".cache", DBConfigDefault.Cache, "the capacity(in megabytes) of the data caching")
	f.String(prefix+".namespace", defaultNamespace, "metrics namespace")
	conf.PebbleConfigAddOptions(prefix+".pebble", f)
}

type DBConvConfig struct {
	Src                  DBConfig                        `koanf:"src"`
	Dst                  DBConfig                        `koanf:"dst"`
	Threads              int                             `koanf:"threads"`
	IdealBatchSize       int                             `koanf:"ideal-batch-size"`
	MinBatchesBeforeFork int                             `koanf:"min-batches-before-fork"`
	Convert              bool                            `koanf:"convert"`
	Compact              bool                            `koanf:"compact"`
	Verify               int                             `koanf:"verify"`
	LogLevel             int                             `koanf:"log-level"`
	Metrics              bool                            `koanf:"metrics"`
	MetricsServer        genericconf.MetricsServerConfig `koanf:"metrics-server"`
}

var DefaultDBConvConfig = DBConvConfig{
	IdealBatchSize:       100 * 1024 * 1024, // 100 MB
	MinBatchesBeforeFork: 2,
	Threads:              1,
	Convert:              false,
	Compact:              false,
	Verify:               0,
	LogLevel:             int(log.LvlDebug),
	Metrics:              false,
	MetricsServer:        genericconf.MetricsServerConfigDefault,
}

func DBConvConfigAddOptions(f *flag.FlagSet) {
	DBConfigAddOptions("src", f, "srcdb/")
	DBConfigAddOptions("dst", f, "destdb/")
	f.Int("threads", DefaultDBConvConfig.Threads, "number of threads to use")
	f.Int("ideal-batch-size", DefaultDBConvConfig.IdealBatchSize, "ideal write batch size")
	f.Int("min-batches-before-fork", DefaultDBConvConfig.MinBatchesBeforeFork, "minimal number of batches before forking a thread")
	f.Bool("convert", DefaultDBConvConfig.Convert, "enables conversion step")
	f.Bool("compact", DefaultDBConvConfig.Compact, "enables compaction step")
	f.Int("verify", DefaultDBConvConfig.Verify, "enables verification step (0 = disabled, 1 = only keys, 2 = keys and values)")
	f.Int("log-level", DefaultDBConvConfig.LogLevel, "log level (0 crit - 5 trace)")
	f.Bool("metrics", DefaultDBConvConfig.Metrics, "enable metrics")
	genericconf.MetricsServerAddOptions("metrics-server", f)
}

func (c *DBConvConfig) Validate() error {
	if c.Threads < 0 {
		return fmt.Errorf("Invalid threads number: %v", c.Threads)
	}
	if c.Verify < 0 || c.Verify > 2 {
		return fmt.Errorf("Invalid verify config value: %v", c.Verify)
	}
	if !c.Convert && c.Verify == 0 && !c.Compact {
		return errors.New("nothing to be done, conversion, verification and compaction disabled")
	}
	return nil
}
