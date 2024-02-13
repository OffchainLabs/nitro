// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package das

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/util/stopwaiter"

	flag "github.com/spf13/pflag"
)

type RegularSyncStorageConfig struct {
	Enable       bool          `koanf:"enable"`
	SyncInterval time.Duration `koanf:"sync-interval"`
}

var DefaultRegularSyncStorageConfig = RegularSyncStorageConfig{
	Enable:       false,
	SyncInterval: 5 * time.Minute,
}

func RegularSyncStorageConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".enable", DefaultRegularSyncStorageConfig.Enable, "enable regular storage syncing")
	f.Duration(prefix+".sync-interval", DefaultRegularSyncStorageConfig.SyncInterval, "interval for running regular storage sync")
}

// A RegularlySyncStorage is used to sync data from syncFromStorageServices to
// all the syncToStorageServices at regular intervals.
// (Only newly added data since the last sync is copied over.)
type RegularlySyncStorage struct {
	stopwaiter.StopWaiter
	syncFromStorageServices                    []*IterableStorageService
	syncToStorageServices                      []StorageService
	lastSyncedHashOfEachSyncFromStorageService map[*IterableStorageService]common.Hash
	syncInterval                               time.Duration
}

func NewRegularlySyncStorage(syncFromStorageServices []*IterableStorageService, syncToStorageServices []StorageService, conf RegularSyncStorageConfig) *RegularlySyncStorage {
	lastSyncedHashOfEachSyncFromStorageService := make(map[*IterableStorageService]common.Hash)
	for _, syncFrom := range syncFromStorageServices {
		lastSyncedHashOfEachSyncFromStorageService[syncFrom] = syncFrom.DefaultBegin()
	}
	return &RegularlySyncStorage{
		syncFromStorageServices:                    syncFromStorageServices,
		syncToStorageServices:                      syncToStorageServices,
		lastSyncedHashOfEachSyncFromStorageService: lastSyncedHashOfEachSyncFromStorageService,
		syncInterval:                               conf.SyncInterval,
	}
}

func (r *RegularlySyncStorage) Start(ctx context.Context) {
	// Start thread for regular sync
	r.StopWaiter.Start(ctx, r)
	r.CallIteratively(r.syncAllStorages)
}

func (r *RegularlySyncStorage) syncAllStorages(ctx context.Context) time.Duration {
	for syncFrom, lastSyncedHash := range r.lastSyncedHashOfEachSyncFromStorageService {
		end := syncFrom.End(ctx)
		if (end == common.Hash{}) {
			continue
		}

		syncHash := lastSyncedHash
		for syncHash != end {
			syncHash = syncFrom.Next(ctx, syncHash)
			data, err := syncFrom.GetByHash(ctx, syncHash)
			if err != nil {
				continue
			}
			expirationTime, err := syncFrom.GetExpirationTime(ctx, syncHash)
			if err != nil {
				continue
			}
			for _, syncTo := range r.syncToStorageServices {
				_, err = syncTo.GetByHash(ctx, syncHash)
				if err == nil {
					continue
				}

				if err = syncTo.Put(ctx, data, expirationTime); err != nil {
					log.Error("Error while running regular storage sync", "err", err)
				}
			}
		}
		r.lastSyncedHashOfEachSyncFromStorageService[syncFrom] = end
	}
	return r.syncInterval
}
