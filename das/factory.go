// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"context"
)

// Create any storage services that persist to files, database, cloud storage,
// and group them together into a RedundantStorage instance if there is more than one.
func CreatePersistentStorageService(
	ctx context.Context,
	config *DataAvailabilityConfig,
) (StorageService, error) {
	storageServices := make([]StorageService, 0, 10)
	if config.LocalDBStorageConfig.Enable {
		s, err := NewDBStorageService(ctx, config.LocalDBStorageConfig.DataDir, false /* TODO plumb this from config  */)
		if err != nil {
			return nil, err
		}
		go func() {
			<-ctx.Done()
			_ = s.Close(context.Background())
		}()
		storageServices = append(storageServices, s)
	}

	if config.LocalFileStorageConfig.Enable {
		s, err := NewLocalFileStorageService(config.LocalFileStorageConfig.DataDir)
		if err != nil {
			return nil, err
		}
		storageServices = append(storageServices, s)
	}

	if config.S3StorageServiceConfig.Enable {
		s, err := NewS3StorageService(config.S3StorageServiceConfig)
		if err != nil {
			return nil, err
		}
		storageServices = append(storageServices, s)
	}

	if len(storageServices) > 1 {
		return NewRedundantStorageService(ctx, storageServices)
	}
	if len(storageServices) == 1 {
		return storageServices[0], nil
	}
	return nil, nil
}
