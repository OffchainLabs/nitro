// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"context"
	"github.com/dgraph-io/badger"
	"time"
)

type DBStorageService struct {
	db                  *badger.DB
	discardAfterTimeout bool
	dirPath             string
	shutdownFunc        func()
	closeOnShutdown     chan interface{}
}

func NewDBStorageService(ctx context.Context, dirPath string, discardAfterTimeout bool) (StorageService, error) {
	db, err := badger.Open(badger.DefaultOptions(dirPath))
	if err != nil {
		return nil, err
	}

	shutdownCtx, cancel := context.WithCancel(ctx)

	ret := &DBStorageService{
		db:                  db,
		discardAfterTimeout: discardAfterTimeout,
		dirPath:             dirPath,
		shutdownFunc:        cancel,
		closeOnShutdown:     make(chan interface{}),
	}

	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		defer func() {
			_ = ret.db.Close()
			close(ret.closeOnShutdown)
		}()
		for {
			select {
			case <-ticker.C:
				for db.RunValueLogGC(0.7) == nil {
					select {
					case <-shutdownCtx.Done():
						return
					default:
					}
				}
			case <-shutdownCtx.Done():
				return
			}
		}
	}()

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

func (dbs *DBStorageService) PutByHash(ctx context.Context, key []byte, value []byte, timeout uint64) error {
	return dbs.db.Update(func(txn *badger.Txn) error {
		e := badger.NewEntry(key, value)
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
	dbs.shutdownFunc()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-dbs.closeOnShutdown:
		return nil
	}
}

func (dbs *DBStorageService) String() string {
	return "BadgerDB(" + dbs.dirPath + ")"
}
