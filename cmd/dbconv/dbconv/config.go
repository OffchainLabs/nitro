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
	Src            DBConfig `koanf:"src"`
	Dst            DBConfig `koanf:"dst"`
	Threads        uint8    `koanf:"threads"`
	IdealBatchSize int      `koanf:"ideal-batch"`
}

var DefaultDBConvConfig = DBConvConfig{IdealBatchSize: ethdb.IdealBatchSize}

func DBConvConfigAddOptions(f *flag.FlagSet) {
	DBConfigAddOptions("src", f)
	DBConfigAddOptions("dst", f)
	f.Uint8("threads", DefaultDBConvConfig.Threads, "number of threads to use (1-255, 0 = auto)")
	f.Uint8("ideal-batch", DefaultDBConvConfig.Threads, "ideal write batch size") // TODO
}
