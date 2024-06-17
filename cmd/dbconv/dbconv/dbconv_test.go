package dbconv

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

func TestConversion(t *testing.T) {
	_ = testhelpers.InitTestLog(t, log.LvlTrace)
	oldDBConfig := DBConfigDefault
	oldDBConfig.Data = t.TempDir()
	oldDBConfig.DBEngine = "leveldb"

	newDBConfig := DBConfigDefault
	newDBConfig.Data = t.TempDir()
	newDBConfig.DBEngine = "pebble"

	func() {
		oldDb, err := openDB(&oldDBConfig, "", false)
		defer oldDb.Close()
		Require(t, err)
		err = oldDb.Put([]byte{}, []byte{0xde, 0xed, 0xbe, 0xef})
		Require(t, err)
		for i := 0; i < 20; i++ {
			err = oldDb.Put([]byte{byte(i)}, []byte{byte(i + 1)})
			Require(t, err)
		}
	}()

	config := DefaultDBConvConfig
	config.Src = oldDBConfig
	config.Dst = newDBConfig
	config.IdealBatchSize = 5
	config.Verify = "full"
	conv := NewDBConverter(&config)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := conv.Convert(ctx)
	Require(t, err)

	err = conv.Verify(ctx)
	Require(t, err)

	// check if new database doesn't have any extra keys
	oldDb, err := openDB(&oldDBConfig, "", true)
	Require(t, err)
	defer oldDb.Close()
	newDb, err := openDB(&newDBConfig, "", true)
	Require(t, err)
	defer newDb.Close()
	it := newDb.NewIterator(nil, nil)
	defer it.Release()
	for it.Next() {
		has, err := oldDb.Has(it.Key())
		Require(t, err)
		if !has {
			Fail(t, "Unexpected key in the converted db, key:", it.Key())
		}
	}
}

func Require(t *testing.T, err error, printables ...interface{}) {
	t.Helper()
	testhelpers.RequireImpl(t, err, printables...)
}

func Fail(t *testing.T, printables ...interface{}) {
	t.Helper()
	testhelpers.FailImpl(t, printables...)
}
