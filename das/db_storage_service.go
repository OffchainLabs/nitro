// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"context"
	badger "github.com/dgraph-io/badger/v3"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/offchainlabs/nitro/util/stopwaiter"
	flag "github.com/spf13/pflag"
	"time"
)

type LocalDBStorageConfig struct {
	Enable  bool   `koanf:"enable"`
	DataDir string `koanf:"data-dir"`
}

var DefaultLocalDBStorageConfig = LocalDBStorageConfig{}

func LocalDBStorageConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".enable", DefaultLocalDBStorageConfig.Enable, "Enable storage/retrieval of sequencer batch data from a database on the local filesystem")
	f.String(prefix+".data-dir", DefaultLocalDBStorageConfig.DataDir, "Directory in which to store the database")
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
	if err := ret.stopWaiter.Start(ctx); err != nil {
		return nil, err
	}
	err = ret.stopWaiter.LaunchThread(func(myCtx context.Context) {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		defer func() {
			_ = ret.db.Close()
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

func (dbs *DBStorageService) GetByHash(ctx context.Context, key []byte) ([]byte, error) {
	var ret []byte
	err := dbs.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			ret = append([]byte{}, val...)
			return nil
		})
	})
	return ret, err
}

func (dbs *DBStorageService) Put(ctx context.Context, data []byte, timeout uint64) error {
	return dbs.db.Update(func(txn *badger.Txn) error {
		e := badger.NewEntry(crypto.Keccak256(data), data)
		if dbs.discardAfterTimeout {
			e = e.WithTTL(time.Until(time.Unix(int64(timeout), 0)))
		}
		return txn.SetEntry(e)
	})
}

func (dbs *DBStorageService) Sync(ctx context.Context) error {
	return dbs.db.Sync()
}

func (dbs *DBStorageService) Close(ctx context.Context) error {
	dbs.stopWaiter.StopAndWait()
	return nil
}

func (dbs *DBStorageService) String() string {
	return "BadgerDB(" + dbs.dirPath + ")"
}
