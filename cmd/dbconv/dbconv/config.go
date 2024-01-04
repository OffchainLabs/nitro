package dbconv

import (
	"fmt"

	"github.com/ethereum/go-ethereum/log"
	flag "github.com/spf13/pflag"
)

type DBConfig struct {
	Data     string `koanf:"data"`
	DBEngine string `koanf:"db-engine"`
	Handles  int    `koanf:"handles"`
	Cache    int    `koanf:"cache"`
}

// TODO
var DBConfigDefault = DBConfig{}

func DBConfigAddOptions(prefix string, f *flag.FlagSet) {
	// TODO
	f.String(prefix+".data", DBConfigDefault.Data, "directory of stored chain state")
	f.String(prefix+".db-engine", DBConfigDefault.DBEngine, "backing database implementation to use ('leveldb' or 'pebble')")
	f.Int(prefix+".handles", DBConfigDefault.Handles, "number of file descriptor handles to use for the database")
	f.Int(prefix+".cache", DBConfigDefault.Cache, "the capacity(in megabytes) of the data caching")
}

type DBConvConfig struct {
	Src                  DBConfig `koanf:"src"`
	Dst                  DBConfig `koanf:"dst"`
	Threads              int      `koanf:"threads"`
	IdealBatchSize       int      `koanf:"ideal-batch-size"`
	MinBatchesBeforeFork int      `koanf:"min-batches-before-fork"`
	Verify               int      `koanf:"verify"`
	VerifyOnly           bool     `koanf:"verify-only"`
}

var DefaultDBConvConfig = DBConvConfig{
	IdealBatchSize:       100 * 1024 * 1024, // 100 MB
	MinBatchesBeforeFork: 10,
	Threads:              0,
	Verify:               1,
	VerifyOnly:           false,
}

func DBConvConfigAddOptions(f *flag.FlagSet) {
	DBConfigAddOptions("src", f)
	DBConfigAddOptions("dst", f)
	f.Int("threads", DefaultDBConvConfig.Threads, "number of threads to use (0 = auto)")
	f.Int("ideal-batch-size", DefaultDBConvConfig.IdealBatchSize, "ideal write batch size")                                         // TODO
	f.Int("min-batches-before-fork", DefaultDBConvConfig.MinBatchesBeforeFork, "minimal number of batches before forking a thread") // TODO
	f.Int("verify", DefaultDBConvConfig.Verify, "enables verification (0 = disabled, 1 = only keys, 2 = keys and values)")          // TODO
	f.Bool("verify-only", DefaultDBConvConfig.VerifyOnly, "skips conversion, runs verification only")                               // TODO
}

func (c *DBConvConfig) Validate() error {
	if c.Verify < 0 || c.Verify > 2 {
		return fmt.Errorf("Invalid verify config value: %v", c.Verify)
	}
	if c.VerifyOnly && c.Verify == 0 {
		log.Info("enabling keys verification as --verify-only flag is set")
		c.Verify = 1
	}
	return nil
}
