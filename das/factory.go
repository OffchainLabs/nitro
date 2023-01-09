// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/offchainlabs/nitro/util/signature"
)

// CreatePersistentStorageService creates any storage services that persist to files, database, cloud storage,
// and group them together into a RedundantStorage instance if there is more than one.
func CreatePersistentStorageService(
	ctx context.Context,
	config *DataAvailabilityConfig,
	syncFromStorageServices *[]*IterableStorageService,
	syncToStorageServices *[]StorageService,
) (StorageService, *LifecycleManager, error) {
	storageServices := make([]StorageService, 0, 10)
	var lifecycleManager LifecycleManager
	if config.LocalDBStorageConfig.Enable {
		s, err := NewDBStorageService(ctx, config.LocalDBStorageConfig.DataDir, config.LocalDBStorageConfig.DiscardAfterTimeout)
		if err != nil {
			return nil, nil, err
		}
		if config.LocalDBStorageConfig.SyncFromStorageServices {
			iterableStorageService := NewIterableStorageService(ConvertStorageServiceToIterationCompatibleStorageService(s))
			*syncFromStorageServices = append(*syncFromStorageServices, iterableStorageService)
			s = iterableStorageService
		}
		if config.LocalDBStorageConfig.SyncToStorageServices {
			*syncToStorageServices = append(*syncToStorageServices, s)
		}
		lifecycleManager.Register(s)
		storageServices = append(storageServices, s)
	}

	if config.LocalFileStorageConfig.Enable {
		s, err := NewLocalFileStorageService(config.LocalFileStorageConfig.DataDir)
		if err != nil {
			return nil, nil, err
		}
		if config.LocalFileStorageConfig.SyncFromStorageServices {
			iterableStorageService := NewIterableStorageService(ConvertStorageServiceToIterationCompatibleStorageService(s))
			*syncFromStorageServices = append(*syncFromStorageServices, iterableStorageService)
			s = iterableStorageService
		}
		if config.LocalFileStorageConfig.SyncToStorageServices {
			*syncToStorageServices = append(*syncToStorageServices, s)
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
		if config.S3StorageServiceConfig.SyncFromStorageServices {
			iterableStorageService := NewIterableStorageService(ConvertStorageServiceToIterationCompatibleStorageService(s))
			*syncFromStorageServices = append(*syncFromStorageServices, iterableStorageService)
			s = iterableStorageService
		}
		if config.S3StorageServiceConfig.SyncToStorageServices {
			*syncToStorageServices = append(*syncToStorageServices, s)
		}
		storageServices = append(storageServices, s)
	}

	if config.IpfsStorageServiceConfig.Enable {
		s, err := NewIpfsStorageService(ctx, config.IpfsStorageServiceConfig)
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

func WrapStorageWithCache(
	ctx context.Context,
	config *DataAvailabilityConfig,
	storageService StorageService,
	syncFromStorageServices *[]*IterableStorageService,
	syncToStorageServices *[]StorageService,
	lifecycleManager *LifecycleManager) (StorageService, error) {
	if storageService == nil {
		return nil, nil
	}

	// Enable caches, Redis and (local) BigCache. Local is the outermost, so it will be tried first.
	var err error
	if config.RedisCacheConfig.Enable {
		storageService, err = NewRedisStorageService(config.RedisCacheConfig, storageService)
		lifecycleManager.Register(storageService)
		if err != nil {
			return nil, err
		}
		if config.RedisCacheConfig.SyncFromStorageServices {
			iterableStorageService := NewIterableStorageService(ConvertStorageServiceToIterationCompatibleStorageService(storageService))
			*syncFromStorageServices = append(*syncFromStorageServices, iterableStorageService)
			storageService = iterableStorageService
		}
		if config.RedisCacheConfig.SyncToStorageServices {
			*syncToStorageServices = append(*syncToStorageServices, storageService)
		}
	}
	if config.LocalCacheConfig.Enable {
		storageService, err = NewBigCacheStorageService(config.LocalCacheConfig, storageService)
		lifecycleManager.Register(storageService)
		if err != nil {
			return nil, err
		}
	}
	return storageService, nil
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
		return nil, nil, nil, errors.New("--node.data-availabilty.rpc-aggregator.enable and rest-aggregator.enable must be set when running a Batch Poster in AnyTrust mode")
	}

	if config.LocalDBStorageConfig.Enable || config.LocalFileStorageConfig.Enable || config.S3StorageServiceConfig.Enable {
		return nil, nil, nil, errors.New("--node.data-availability.local-db-storage.enable, local-file-storage.enable, s3-storage.enable may not be set when running a Batch Poster in AnyTrust mode")
	}

	if config.KeyConfig.KeyDir != "" || config.KeyConfig.PrivKey != "" {
		return nil, nil, nil, errors.New("--node.data-availability.key.key-dir, priv-key may not be set when running a Batch Poster in AnyTrust mode")
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

func CreateDAReaderWriterForStorage(
	ctx context.Context,
	config *DataAvailabilityConfig,
	l1Reader *headerreader.HeaderReader,
	seqInboxAddress *common.Address,
) (DataAvailabilityServiceReader, DataAvailabilityServiceWriter, *LifecycleManager, error) {
	if !config.Enable {
		return nil, nil, nil, nil
	}

	// TODO move invariant checks to another fn
	if config.AggregatorConfig.Enable {
		panic("Tried to make an aggregator using wrong factory method")
	}
	if !config.LocalDBStorageConfig.Enable &&
		!config.LocalFileStorageConfig.Enable &&
		!config.S3StorageServiceConfig.Enable &&
		!config.IpfsStorageServiceConfig.Enable {
		return nil, nil, nil, errors.New("At least one of --data-availability.(local-db-storage|local-file-storage|s3-storage|ipfs-storage) must be enabled.")
	}
	// TODO more invariants

	var syncFromStorageServices []*IterableStorageService
	var syncToStorageServices []StorageService

	// TODO rewrite
	// This function builds up the DataAvailabilityService with the following topology, starting from the leaves.
	/*
			      ChainFetchDAS → Bigcache → Redis →
				       SignAfterStoreDAS →
				              FallbackStorageService (if the REST client aggregator was specified)
				              (primary) → RedundantStorage (if multiple persistent backing stores were specified)
				                            → S3
				                            → DiskStorage
				                            → Database
				         (fallback only)→ RESTful client aggregator

		          → : X--delegates to-->Y
	*/

	storageService, dasLifecycleManager, err := CreatePersistentStorageService(ctx, config, &syncFromStorageServices, &syncToStorageServices)
	if err != nil {
		return nil, nil, nil, err
	}

	storageService, err = WrapStorageWithCache(ctx, config, storageService, &syncFromStorageServices, &syncToStorageServices, dasLifecycleManager)
	if err != nil {
		return nil, nil, nil, err
	}

	// Create the REST aggregator if one was requested. If other storage types were enabled above, then
	// the REST aggregator is used as the fallback to them.
	if config.RestfulClientAggregatorConfig.Enable {
		restAgg, err := NewRestfulClientAggregator(ctx, &config.RestfulClientAggregatorConfig)
		if err != nil {
			return nil, nil, nil, err
		}
		restAgg.Start(ctx)
		dasLifecycleManager.Register(restAgg)

		// Wrap the primary storage service with the fallback to the restful aggregator
		syncConf := &config.RestfulClientAggregatorConfig.SyncToStorageConfig
		var retentionPeriodSeconds uint64
		if uint64(syncConf.RetentionPeriod) == math.MaxUint64 {
			retentionPeriodSeconds = math.MaxUint64
		} else {
			retentionPeriodSeconds = uint64(syncConf.RetentionPeriod.Seconds())
		}
		if syncConf.Eager {
			if l1Reader == nil || seqInboxAddress == nil {
				return nil, nil, nil, errors.New("l1-node-url and sequencer-inbox-address must be specified along with sync-to-storage.eager")
			}
			storageService, err = NewSyncingFallbackStorageService(
				ctx,
				storageService,
				restAgg,
				l1Reader,
				*seqInboxAddress,
				syncConf)
			dasLifecycleManager.Register(storageService)
			if err != nil {
				return nil, nil, nil, err
			}
		} else {
			storageService = NewFallbackStorageService(storageService, restAgg,
				retentionPeriodSeconds, syncConf.IgnoreWriteErrors, true)
			dasLifecycleManager.Register(storageService)
		}

	}

	var daWriter DataAvailabilityServiceWriter
	var daReader DataAvailabilityServiceReader = storageService

	if config.KeyConfig.KeyDir != "" || config.KeyConfig.PrivKey != "" {
		var seqInboxCaller *bridgegen.SequencerInboxCaller
		if seqInboxAddress != nil {
			seqInbox, err := bridgegen.NewSequencerInbox(*seqInboxAddress, (*l1Reader).Client())
			if err != nil {
				return nil, nil, nil, err
			}

			seqInboxCaller = &seqInbox.SequencerInboxCaller
		}
		if config.DisableSignatureChecking {
			seqInboxCaller = nil
		}

		privKey, err := config.KeyConfig.BLSPrivKey()
		if err != nil {
			return nil, nil, nil, err
		}

		// TODO rename
		daWriter, err = NewSignAfterStoreDASWithSeqInboxCaller(
			privKey,
			seqInboxCaller,
			storageService,
			config.ExtraSignatureCheckingPublicKey,
		)
		if err != nil {
			return nil, nil, nil, err
		}
	}

	if config.RegularSyncStorageConfig.Enable && len(syncFromStorageServices) != 0 && len(syncToStorageServices) != 0 {
		regularlySyncStorage := NewRegularlySyncStorage(syncFromStorageServices, syncToStorageServices, config.RegularSyncStorageConfig)
		regularlySyncStorage.Start(ctx)
	}

	if seqInboxAddress != nil {
		daReader, err = NewChainFetchReader(daReader, (*l1Reader).Client(), *seqInboxAddress)
		if err != nil {
			return nil, nil, nil, err
		}
	}

	return daReader, daWriter, dasLifecycleManager, nil
}

func CreateDAReaderForNode(
	ctx context.Context,
	config *DataAvailabilityConfig,
	l1Reader *headerreader.HeaderReader,
	seqInboxAddress *common.Address,
) (DataAvailabilityServiceReader, *LifecycleManager, error) {
	if !config.Enable {
		return nil, nil, nil
	}

	topLevelStorageService, dasLifecycleManager, err := CreatePersistentStorageService(ctx, config, nil, nil)
	if err != nil {
		return nil, nil, err
	}
	hasPersistentStorage := topLevelStorageService != nil

	// Create the REST aggregator if one was requested. If other storage types were enabled above, then
	// the REST aggregator is used as the fallback to them.
	var restAgg *SimpleDASReaderAggregator
	if config.RestfulClientAggregatorConfig.Enable {
		restAgg, err = NewRestfulClientAggregator(ctx, &config.RestfulClientAggregatorConfig)
		if err != nil {
			return nil, nil, err
		}
		restAgg.Start(ctx)
		dasLifecycleManager.Register(restAgg)

		// Wrap the primary storage service with the fallback to the restful aggregator
		if hasPersistentStorage {
			syncConf := &config.RestfulClientAggregatorConfig.SyncToStorageConfig
			var retentionPeriodSeconds uint64
			if uint64(syncConf.RetentionPeriod) == math.MaxUint64 {
				retentionPeriodSeconds = math.MaxUint64
			} else {
				retentionPeriodSeconds = uint64(syncConf.RetentionPeriod.Seconds())
			}
			if syncConf.Eager {
				return nil, nil, errors.New("sync-to-storage.eager can't be used with a Nitro node")
			} else {
				// TODO fallback doesn't make sense for nodes
				topLevelStorageService = NewFallbackStorageService(topLevelStorageService, restAgg,
					retentionPeriodSeconds, syncConf.IgnoreWriteErrors, true)
			}
		}
		// TODO cleanup
		if topLevelStorageService != nil {
			dasLifecycleManager.Register(topLevelStorageService)
		}
	}

	if config.AggregatorConfig.Enable {
		return nil, nil, errors.New("node.data-availability.rpc-aggregator is only for Batch Poster mode")
	}
	if config.KeyConfig.KeyDir != "" || config.KeyConfig.PrivKey != "" {
		return nil, nil, errors.New("node.data-availability.key options are only for daserver committee members")
	}

	if config.RedisCacheConfig.Enable || config.LocalCacheConfig.Enable {
		return nil, nil, errors.New("node.data-availbility.*-cache options are only for daserver")
	}

	if config.RegularSyncStorageConfig.Enable {
		return nil, nil, errors.New("node.data-availability.regular-sync-store options are only for daserver")
	}

	if topLevelStorageService == nil && restAgg == nil {
		return nil, nil, fmt.Errorf("data-availability.enable was specified but no Data Availability server types were enabled, %+v", config)
	}

	var daReader DataAvailabilityServiceReader = restAgg
	if topLevelStorageService != nil {
		// TODO only doing this because of fallback being a storage iface
		daReader = topLevelStorageService
	}
	if seqInboxAddress != nil {
		seqInbox, err := bridgegen.NewSequencerInbox(*seqInboxAddress, (*l1Reader).Client())
		if err != nil {
			return nil, nil, err
		}
		daReader, err = NewChainFetchReaderWithSeqInbox(daReader, seqInbox)
		if err != nil {
			return nil, nil, err
		}
	}

	if dasLifecycleManager == nil {
		return nil, nil, errors.New("missing dasLifecycleManager")
	}

	return daReader, dasLifecycleManager, nil
}
