// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"bytes"
	"context"
	"errors"
	"time"

	badger "github.com/dgraph-io/badger/v3"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/das/dastree"
	"github.com/offchainlabs/nitro/util/pretty"
	"github.com/offchainlabs/nitro/util/stopwaiter"
	flag "github.com/spf13/pflag"
)

type LocalDBStorageConfig struct {
	Enable                  bool   `koanf:"enable"`
	DataDir                 string `koanf:"data-dir"`
	DiscardAfterTimeout     bool   `koanf:"discard-after-timeout"`
	SyncFromStorageServices bool   `koanf:"sync-from-storage-service"`
	SyncToStorageServices   bool   `koanf:"sync-to-storage-service"`
}

var DefaultLocalDBStorageConfig = LocalDBStorageConfig{}

func LocalDBStorageConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".enable", DefaultLocalDBStorageConfig.Enable, "enable storage/retrieval of sequencer batch data from a database on the local filesystem")
	f.String(prefix+".data-dir", DefaultLocalDBStorageConfig.DataDir, "directory in which to store the database")
	f.Bool(prefix+".discard-after-timeout", DefaultLocalDBStorageConfig.DiscardAfterTimeout, "discard data after its expiry timeout")
	f.Bool(prefix+".sync-from-storage-service", DefaultLocalDBStorageConfig.SyncFromStorageServices, "enable db storage to be used as a source for regular sync storage")
	f.Bool(prefix+".sync-to-storage-service", DefaultLocalDBStorageConfig.SyncToStorageServices, "enable db storage to be used as a sink for regular sync storage")
}

type DBStorageService struct {
	db                  *badger.DB
	discardAfterTimeout bool
	dirPath             string
	stopWaiter          stopwaiter.StopWaiterSafe
}

func NewDBStorageService(ctx context.Context, dirPath string, discardAfterTimeout bool) (StorageService, error) {
	db, err := badger.Open(badger.DefaultOptions(dirPath))
	if err != nil {
		return nil, err
	}

	ret := &DBStorageService{
		db:                  db,
		discardAfterTimeout: discardAfterTimeout,
		dirPath:             dirPath,
	}
	if err := ret.stopWaiter.Start(ctx, ret); err != nil {
		return nil, err
	}
	err = ret.stopWaiter.LaunchThreadSafe(func(myCtx context.Context) {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		defer func() {
			if err := ret.db.Close(); err != nil {
				log.Error("Failed to close DB", "err", err)
			}
		}()
		for {
			select {
			case <-ticker.C:
				for db.RunValueLogGC(0.7) == nil {
					select {
					case <-myCtx.Done():
						return
					default:
					}
				}
			case <-myCtx.Done():
				return
			}
		}
	})
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func (dbs *DBStorageService) GetByHash(ctx context.Context, key common.Hash) ([]byte, error) {
	log.Trace("das.DBStorageService.GetByHash", "key", pretty.PrettyHash(key), "this", dbs)

	var ret []byte
	err := dbs.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key.Bytes())
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			ret = append([]byte{}, val...)
			return nil
		})
	})
	if errors.Is(err, badger.ErrKeyNotFound) {
		return ret, ErrNotFound
	}
	return ret, err
}

func (dbs *DBStorageService) Put(ctx context.Context, data []byte, timeout uint64) error {
	logPut("das.DBStorageService.Put", data, timeout, dbs)

	return dbs.db.Update(func(txn *badger.Txn) error {
		e := badger.NewEntry(dastree.HashBytes(data), data)
		if dbs.discardAfterTimeout {
			e = e.WithTTL(time.Until(time.Unix(int64(timeout), 0)))
		}
		return txn.SetEntry(e)
	})
}

func (dbs *DBStorageService) putKeyValue(ctx context.Context, key common.Hash, value []byte) error {
	return dbs.db.Update(func(txn *badger.Txn) error {
		e := badger.NewEntry(key.Bytes(), value)
		return txn.SetEntry(e)
	})
}

func (dbs *DBStorageService) Sync(ctx context.Context) error {
	return dbs.db.Sync()
}

func (dbs *DBStorageService) Close(ctx context.Context) error {
	return dbs.stopWaiter.StopAndWait()
}

func (dbs *DBStorageService) ExpirationPolicy(ctx context.Context) (arbstate.ExpirationPolicy, error) {
	if dbs.discardAfterTimeout {
		return arbstate.DiscardAfterDataTimeout, nil
	}
	return arbstate.KeepForever, nil
}

func (dbs *DBStorageService) String() string {
	return "BadgerDB(" + dbs.dirPath + ")"
}

func (dbs *DBStorageService) HealthCheck(ctx context.Context) error {
	testData := []byte("Test-Data")
	err := dbs.Put(ctx, testData, uint64(time.Now().Add(time.Minute).Unix()))
	if err != nil {
		return err
	}
	res, err := dbs.GetByHash(ctx, dastree.Hash(testData))
	if err != nil {
		return err
	}
	if !bytes.Equal(res, testData) {
		return errors.New("invalid GetByHash result")
	}
	return nil
}
