// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"context"
	"errors"

	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/util/signature"
)

// Create any storage services that persist to files, database, cloud storage,
// and group them together into a RedundantStorage instance if there is more than one.
func CreatePersistentStorageService(
	ctx context.Context,
	config *DataAvailabilityConfig,
) (StorageService, *LifecycleManager, error) {
	storageServices := make([]StorageService, 0, 10)
	var lifecycleManager LifecycleManager
	if config.LocalDBStorageConfig.Enable {
		s, err := NewDBStorageService(ctx, config.LocalDBStorageConfig.DataDir, config.LocalDBStorageConfig.DiscardAfterTimeout)
		if err != nil {
			return nil, nil, err
		}
		lifecycleManager.Register(s)
		storageServices = append(storageServices, s)
	}

	if config.LocalFileStorageConfig.Enable {
		s, err := NewLocalFileStorageService(config.LocalFileStorageConfig.DataDir)
		if err != nil {
			return nil, nil, err
		}
		lifecycleManager.Register(s)
		storageServices = append(storageServices, s)
	}

	if config.S3StorageServiceConfig.Enable {
		s, err := NewS3StorageService(config.S3StorageServiceConfig)
		if err != nil {
			return nil, nil, err
		}
		lifecycleManager.Register(s)
		storageServices = append(storageServices, s)
	}

	if len(storageServices) > 1 {
		s, err := NewRedundantStorageService(ctx, storageServices)
		if err != nil {
			return nil, nil, err
		}
		lifecycleManager.Register(s)
		return s, &lifecycleManager, nil
	}
	if len(storageServices) == 1 {
		return storageServices[0], &lifecycleManager, nil
	}
	return nil, &lifecycleManager, nil
}

func CreateBatchPosterDAS(
	ctx context.Context,
	config *DataAvailabilityConfig,
	dataSigner signature.DataSignerFunc,
	l1Reader arbutil.L1Interface,
	sequencerInboxAddr common.Address,
) (DataAvailabilityServiceWriter, DataAvailabilityServiceReader, *LifecycleManager, error) {
	if !config.Enable {
		return nil, nil, nil, nil
	}

	if !config.AggregatorConfig.Enable || !config.RestfulClientAggregatorConfig.Enable {
		return nil, nil, nil, errors.New("--node.data-availabilty.rpc-aggregator.enable and rest-aggregator.enable must be set when running a Batch Poster in AnyTrust mode.")
	}

	if config.LocalDBStorageConfig.Enable || config.LocalFileStorageConfig.Enable || config.S3StorageServiceConfig.Enable {
		return nil, nil, nil, errors.New("--node.data-availability.local-db-storage.enable, local-file-storage.enable, s3-storage.enable may not be set when running a Batch Poster in AnyTrust mode.")
	}

	if config.KeyConfig.KeyDir != "" || config.KeyConfig.PrivKey != "" {
		return nil, nil, nil, errors.New("--node.data-availability.key.key-dir, priv-key may not be set when running a Batch Poster in AnyTrust mode.")
	}

	var daWriter DataAvailabilityServiceWriter
	daWriter, err := NewRPCAggregator(ctx, *config)
	if err != nil {
		return nil, nil, nil, err
	}
	if dataSigner != nil {
		// In some tests the batch poster does not sign Store requests
		daWriter, err = NewStoreSigningDAS(daWriter, dataSigner)
		if err != nil {
			return nil, nil, nil, err
		}
	}

	restAgg, err := NewRestfulClientAggregator(ctx, &config.RestfulClientAggregatorConfig)
	if err != nil {
		return nil, nil, nil, err
	}
	restAgg.Start(ctx)
	var lifecycleManager LifecycleManager
	lifecycleManager.Register(restAgg)
	var daReader DataAvailabilityServiceReader = restAgg
	daReader, err = NewChainFetchReader(daReader, l1Reader, sequencerInboxAddr)
	if err != nil {
		return nil, nil, nil, err
	}

	return daWriter, daReader, &lifecycleManager, nil
}
