// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

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

type RegularlySyncStorage struct {
	stopwaiter.StopWaiter
	iterableStorageServices            []*IterableStorageService
	lastSyncedHashOfEachStorageService map[*IterableStorageService]common.Hash
	syncInterval                       time.Duration
}

func NewRegularlySyncStorage(iterableStorageServices []*IterableStorageService, conf RegularSyncStorageConfig) *RegularlySyncStorage {
	lastSyncedHashOfEachStorageService := make(map[*IterableStorageService]common.Hash)
	for _, services := range iterableStorageServices {
		lastSyncedHashOfEachStorageService[services] = services.DefaultBegin()
	}
	return &RegularlySyncStorage{
		iterableStorageServices:            iterableStorageServices,
		lastSyncedHashOfEachStorageService: lastSyncedHashOfEachStorageService,
		syncInterval:                       conf.SyncInterval,
	}
}

func (r *RegularlySyncStorage) Start(ctx context.Context) {
	// Start thread for regular sync
	r.StopWaiter.Start(ctx, r)
	r.StopWaiter.LaunchThread(func(ctx context.Context) {
		regularSyncTicker := time.NewTicker(r.syncInterval)
		for {
			select {
			case <-ctx.Done():
				return
			case <-regularSyncTicker.C:
				r.syncAllStorages(ctx)
			}
		}
	})
}

func (r *RegularlySyncStorage) syncAllStorages(ctx context.Context) {
	for syncFrom, lastSyncedHash := range r.lastSyncedHashOfEachStorageService {
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
			for _, syncTo := range r.iterableStorageServices {
				if syncFrom == syncTo {
					continue
				}

				_, err = syncTo.GetByHash(ctx, syncHash)
				if err == nil {
					continue
				}

				if err = syncTo.Put(ctx, data, expirationTime); err != nil {
					log.Error("Error while running regular storage sync", "err", err)
				}
			}
		}
		r.lastSyncedHashOfEachStorageService[syncFrom] = end
	}
}
