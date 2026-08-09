// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package addressfilter

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/util/s3syncer"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

// Service manages the address-filteress synchronization pipeline.
// It periodically polls S3 for hash list updates and maintains an in-memory
// copy for efficient address filtering.
type FilterService struct {
	stopwaiter.StopWaiter
	config         *Config
	hashStore      *HashStore
	syncMgr        *S3SyncManager
	addressChecker *HashedAddressChecker
}

// NewFilterService creates a new address-filteress service.
func NewFilterService(config *Config) (*FilterService, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	hashStore := NewHashStore(config.CacheSize)

	return &FilterService{
		config:         config,
		hashStore:      hashStore,
		syncMgr:        NewS3SyncManager(config, hashStore),
		addressChecker: NewHashedAddressChecker(hashStore, config.AddressCheckerWorkerCount, config.AddressCheckerQueueSize),
	}, nil
}

// Initialize downloads the initial hash list from S3.
// This method blocks until the hash list is successfully loaded.
// If this fails, the node should not start.
func (s *FilterService) Initialize(ctx context.Context) error {
	log.Info("initializing address-filter service, downloading initial hash list",
		"bucket", s.config.S3.Bucket,
		"key", s.config.S3.ObjectKey,
	)

	err := s.syncMgr.Initialize(ctx)
	if err != nil {
		return fmt.Errorf("failed to init S3 syncer: %w", err)
	}

	// Force download (ignore ETag check for initial load)
	if err := s.syncMgr.Syncer.DownloadAndLoad(ctx); err != nil {
		syncFailureCounter.Inc(1)
		if errors.Is(err, s3syncer.ErrObjectTooLarge) {
			fileTooLargeCounter.Inc(1)
		}
		return fmt.Errorf("failed to load initial hash list: %w", err)
	}

	log.Info("address-filter service initialized",
		"hash_count", s.hashStore.Size(),
		"etag-digest", s.hashStore.Digest(),
	)
	return nil
}

// Start begins the background polling goroutine.
// This should be called after Initialize() succeeds.
func (s *FilterService) Start(ctx context.Context) {
	s.StopWaiter.Start(ctx, s)

	// Start periodic polling goroutine
	s.CallIteratively(func(ctx context.Context) time.Duration {
		if err := s.syncMgr.Syncer.CheckAndSync(ctx); err != nil {
			syncFailureCounter.Inc(1)
			if errors.Is(err, s3syncer.ErrObjectTooLarge) {
				fileTooLargeCounter.Inc(1)
				log.Error("address-filter S3 file exceeds max-file-size, skipping download; keeping previously loaded list", "err", err)
			} else {
				log.Error("failed to sync address-filter list; keeping previously loaded list", "err", err)
			}
		}
		return s.config.PollInterval
	})

	s.StartAndTrackChild(s.addressChecker)

	log.Info("address-filter service started",
		"poll_interval", s.config.PollInterval,
	)
}

func (s *FilterService) GetHashCount() int {
	return s.hashStore.Size()
}

// GetHashStoreDigest GetETag returns the S3 ETag Digest of the currently loaded hash list.
func (s *FilterService) GetHashStoreDigest() string {
	return s.hashStore.Digest()
}

func (s *FilterService) GetLoadedAt() time.Time {
	return s.hashStore.LoadedAt()
}

func (s *FilterService) GetHashStore() *HashStore {
	return s.hashStore
}

func (s *FilterService) GetAddressChecker() *HashedAddressChecker {
	return s.addressChecker
}
