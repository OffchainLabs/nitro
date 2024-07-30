// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	badger "github.com/dgraph-io/badger/v4"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/cmd/daprovider/das/dastree"
	"github.com/offchainlabs/nitro/cmd/daprovider/das/dasutil"
	"github.com/offchainlabs/nitro/util/pretty"
	"github.com/offchainlabs/nitro/util/stopwaiter"
	flag "github.com/spf13/pflag"
)

type LocalDBStorageConfig struct {
	Enable              bool   `koanf:"enable"`
	DataDir             string `koanf:"data-dir"`
	DiscardAfterTimeout bool   `koanf:"discard-after-timeout"`

	// BadgerDB options
	NumMemtables            int   `koanf:"num-memtables"`
	NumLevelZeroTables      int   `koanf:"num-level-zero-tables"`
	NumLevelZeroTablesStall int   `koanf:"num-level-zero-tables-stall"`
	NumCompactors           int   `koanf:"num-compactors"`
	BaseTableSize           int64 `koanf:"base-table-size"`
	ValueLogFileSize        int64 `koanf:"value-log-file-size"`
}

var badgerDefaultOptions = badger.DefaultOptions("")

const migratedMarker = "MIGRATED"

var DefaultLocalDBStorageConfig = LocalDBStorageConfig{
	Enable:              false,
	DataDir:             "",
	DiscardAfterTimeout: false,

	NumMemtables:            badgerDefaultOptions.NumMemtables,
	NumLevelZeroTables:      badgerDefaultOptions.NumLevelZeroTables,
	NumLevelZeroTablesStall: badgerDefaultOptions.NumLevelZeroTablesStall,
	NumCompactors:           badgerDefaultOptions.NumCompactors,
	BaseTableSize:           badgerDefaultOptions.BaseTableSize,
	ValueLogFileSize:        badgerDefaultOptions.ValueLogFileSize,
}

func LocalDBStorageConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".enable", DefaultLocalDBStorageConfig.Enable, "!!!DEPRECATED, USE local-file-storage!!! enable storage/retrieval of sequencer batch data from a database on the local filesystem")
	f.String(prefix+".data-dir", DefaultLocalDBStorageConfig.DataDir, "directory in which to store the database")
	f.Bool(prefix+".discard-after-timeout", DefaultLocalDBStorageConfig.DiscardAfterTimeout, "discard data after its expiry timeout")

	f.Int(prefix+".num-memtables", DefaultLocalDBStorageConfig.NumMemtables, "BadgerDB option: sets the maximum number of tables to keep in memory before stalling")
	f.Int(prefix+".num-level-zero-tables", DefaultLocalDBStorageConfig.NumLevelZeroTables, "BadgerDB option: sets the maximum number of Level 0 tables before compaction starts")
	f.Int(prefix+".num-level-zero-tables-stall", DefaultLocalDBStorageConfig.NumLevelZeroTablesStall, "BadgerDB option: sets the number of Level 0 tables that once reached causes the DB to stall until compaction succeeds")
	f.Int(prefix+".num-compactors", DefaultLocalDBStorageConfig.NumCompactors, "BadgerDB option: Sets the number of compaction workers to run concurrently")
	f.Int64(prefix+".base-table-size", DefaultLocalDBStorageConfig.BaseTableSize, "BadgerDB option: sets the maximum size in bytes for LSM table or file in the base level")
	f.Int64(prefix+".value-log-file-size", DefaultLocalDBStorageConfig.ValueLogFileSize, "BadgerDB option: sets the maximum size of a single log file")

}

type DBStorageService struct {
	db                  *badger.DB
	discardAfterTimeout bool
	dirPath             string
	stopWaiter          stopwaiter.StopWaiterSafe
}

// The DBStorageService is deprecated. This function will migrate data to the target
// LocalFileStorageService if it is provided and migration hasn't already happened.
func NewDBStorageService(ctx context.Context, config *LocalDBStorageConfig, target *LocalFileStorageService) (*DBStorageService, error) {
	if alreadyMigrated(config.DataDir) {
		log.Warn("local-db-storage already migrated, please remove it from the daserver configuration and restart. data-dir can be cleaned up manually now")
		return nil, nil
	}
	if target == nil {
		log.Error("local-db-storage is DEPRECATED, please use use the local-file-storage and migrate-local-db-to-file-storage options. This error will be made fatal in future, continuing for now...")
	}

	options := badger.DefaultOptions(config.DataDir).
		WithNumMemtables(config.NumMemtables).
		WithNumLevelZeroTables(config.NumLevelZeroTables).
		WithNumLevelZeroTablesStall(config.NumLevelZeroTablesStall).
		WithNumCompactors(config.NumCompactors).
		WithBaseTableSize(config.BaseTableSize).
		WithValueLogFileSize(config.ValueLogFileSize)
	db, err := badger.Open(options)
	if err != nil {
		return nil, err
	}

	ret := &DBStorageService{
		db:                  db,
		discardAfterTimeout: config.DiscardAfterTimeout,
		dirPath:             config.DataDir,
	}

	if target != nil {
		if err = ret.migrateTo(ctx, target); err != nil {
			return nil, fmt.Errorf("error migrating local-db-storage to %s: %w", target, err)
		}
		if err = ret.setMigrated(); err != nil {
			return nil, fmt.Errorf("error finalizing migration of local-db-storage to %s: %w", target, err)
		}
		return nil, nil
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

func (dbs *DBStorageService) migrateTo(ctx context.Context, s StorageService) error {
	originExpirationPolicy, err := dbs.ExpirationPolicy(ctx)
	if err != nil {
		return err
	}
	targetExpirationPolicy, err := s.ExpirationPolicy(ctx)
	if err != nil {
		return err
	}

	if originExpirationPolicy == dasutil.KeepForever && targetExpirationPolicy == dasutil.DiscardAfterDataTimeout {
		return errors.New("can't migrate from DBStorageService to target, incompatible expiration policies - can't migrate from non-expiring to expiring since non-expiring DB lacks expiry time metadata")
	}

	return dbs.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		it := txn.NewIterator(opts)
		defer it.Close()
		log.Info("Migrating from DBStorageService", "target", s)
		migrationStart := time.Now()
		count := 0
		for it.Rewind(); it.Valid(); it.Next() {
			if count%1000 == 0 {
				log.Info("Migration in progress", "migrated", count)
			}
			item := it.Item()
			k := item.Key()
			expiry := item.ExpiresAt()
			err := item.Value(func(v []byte) error {
				log.Trace("migrated", "key", pretty.FirstFewBytes(k), "value", pretty.FirstFewBytes(v), "expiry", expiry)
				return s.Put(ctx, v, expiry)
			})
			if err != nil {
				return err
			}
			count++
		}
		log.Info("Migration from DBStorageService complete", "target", s, "migrated", count, "duration", time.Since(migrationStart))
		return nil
	})
}

func (dbs *DBStorageService) Sync(ctx context.Context) error {
	return dbs.db.Sync()
}

func (dbs *DBStorageService) Close(ctx context.Context) error {
	return dbs.stopWaiter.StopAndWait()
}

func alreadyMigrated(dirPath string) bool {
	migratedMarkerFile := filepath.Join(dirPath, migratedMarker)
	_, err := os.Stat(migratedMarkerFile)
	if os.IsNotExist(err) {
		return false
	}
	if err != nil {
		log.Error("error checking if local-db-storage is already migrated", "err", err)
		return false
	}
	return true
}

func (dbs *DBStorageService) setMigrated() error {
	migratedMarkerFile := filepath.Join(dbs.dirPath, migratedMarker)
	file, err := os.OpenFile(migratedMarkerFile, os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	file.Close()
	return nil
}

func (dbs *DBStorageService) ExpirationPolicy(ctx context.Context) (dasutil.ExpirationPolicy, error) {
	if dbs.discardAfterTimeout {
		return dasutil.DiscardAfterDataTimeout, nil
	}
	return dasutil.KeepForever, nil
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
