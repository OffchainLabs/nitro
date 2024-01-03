package dbconv

import (
	"bytes"
	"context"
	"math/rand"
	"testing"

	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

func TestMiddleKey(t *testing.T) {
	triples := [][]byte{
		{0}, {0, 0}, {0, 0},
		{1}, {1, 1}, {1, 0, 128},
		{1}, {1, 0}, {1, 0},
		{1, 1}, {2}, {1, 128, 128},
		{1}, {2}, {1, 128},
		{1}, {2, 1}, {1, 128, 128},
		{0}, {255}, {127, 128},
		{0}, {}, {127, 128},
		{0, 0}, {}, {127, 255, 128},
	}

	for i := 0; i < len(triples)-2; i += 3 {
		start, end, expected := triples[i], triples[i+1], triples[i+2]
		if mid := middleKey(start, end); !bytes.Equal(mid, expected) {
			Fail(t, "Unexpected result for start:", start, "end:", end, "want:", expected, "have:", mid)
		}
	}

	//	for i := 0; i < 1000; i++ {
	//		for j := 0; j < 1000; j++ {
	//			start := new(big.Int.)
	//			m := moddleKey({i}, {j})
	//		}
	//	}
}

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
		for i := 0; i < 10000; i++ {
			size := 1 + rand.Uint64()%100
			randomBytes := testhelpers.RandomizeSlice(make([]byte, size))
			err = oldDb.Put(randomBytes, []byte{byte(i)})
			Require(t, err)
		}
	}()

	config := DefaultDBConvConfig
	config.Src = oldDBConfig
	config.Dst = newDBConfig
	config.Threads = 512
	config.IdealBatchSize = 100
	config.MinBatchesBeforeFork = 10
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
