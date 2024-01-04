package dbconv

import (
	"github.com/ethereum/go-ethereum/ethdb"
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
}

var DefaultDBConvConfig = DBConvConfig{
	IdealBatchSize:       ethdb.IdealBatchSize,
	MinBatchesBeforeFork: 10,
	Threads:              0,
}

func DBConvConfigAddOptions(f *flag.FlagSet) {
	DBConfigAddOptions("src", f)
	DBConfigAddOptions("dst", f)
	f.Int("threads", DefaultDBConvConfig.Threads, "number of threads to use (0 = auto)")
	f.Int("ideal-batch-size", DefaultDBConvConfig.IdealBatchSize, "ideal write batch size")                                         // TODO
	f.Int("min-batches-before-fork", DefaultDBConvConfig.MinBatchesBeforeFork, "minimal number of batches before forking a thread") // TODO
}
