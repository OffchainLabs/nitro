package dbconv

import (
	"bytes"
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
		oldDb, err := openDB(&oldDBConfig, false)
		Require(t, err)
		defer oldDb.Close()
		for i := 0; i < 0xfe; i++ {
			data := []byte{byte(i)}
			err = oldDb.Put(data, data)
			Require(t, err)
			for j := 0; j < 0xf; j++ {
				data := []byte{byte(i), byte(j)}
				err = oldDb.Put(data, data)
				Require(t, err)
			}
		}
		err = oldDb.Put([]byte{}, []byte{0xde, 0xed, 0xbe, 0xef})
		Require(t, err)
	}()

	config := DefaultDBConvConfig
	config.Src = oldDBConfig
	config.Dst = newDBConfig
	config.Threads = 32
	conv := NewDBConverter(&config)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err := conv.Convert(ctx)
	Require(t, err)
	conv.Close()

	oldDb, err := openDB(&oldDBConfig, true)
	Require(t, err)
	defer oldDb.Close()
	newDb, err := openDB(&newDBConfig, true)
	Require(t, err)
	defer newDb.Close()

	func() {
		it := oldDb.NewIterator(nil, nil)
		defer it.Release()
		for it.Next() {
			if has, _ := newDb.Has(it.Key()); !has {
				t.Log("Missing key in the converted db, key:", it.Key())
			}
			newValue, err := newDb.Get(it.Key())
			Require(t, err)
			if !bytes.Equal(newValue, it.Value()) {
				Fail(t, "Value mismatch, old:", it.Value(), "new:", newValue)
			}
		}
	}()
	func() {
		it := newDb.NewIterator(nil, nil)
		defer it.Release()
		for it.Next() {
			if has, _ := oldDb.Has(it.Key()); !has {
				Fail(t, "Unexpected key in the converted db, key:", it.Key())
			}
		}
	}()
}

func Require(t *testing.T, err error, printables ...interface{}) {
	t.Helper()
	testhelpers.RequireImpl(t, err, printables...)
}

func Fail(t *testing.T, printables ...interface{}) {
	t.Helper()
	testhelpers.FailImpl(t, printables...)
}
