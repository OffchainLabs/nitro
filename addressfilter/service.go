// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package addressfilter

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/log"

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
// Returns nil if the service is not enabled in the configuration.
func NewFilterService(ctx context.Context, config *Config) (*FilterService, error) {
	if !config.Enable {
		return nil, nil
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	hashStore := NewHashStore(config.CacheSize)
	syncMgr, err := NewS3SyncManager(ctx, config, hashStore)
	if err != nil {
		return nil, fmt.Errorf("failed to create S3 syncer: %w", err)
	}

	return &FilterService{
		config:         config,
		hashStore:      hashStore,
		syncMgr:        syncMgr,
		addressChecker: NewDefaultHashedAddressChecker(hashStore),
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

	// Force download (ignore ETag check for initial load)
	if err := s.syncMgr.Syncer.DownloadAndLoad(ctx); err != nil {
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
			log.Error("failed to sync address-filter list", "err", err)
		}
		return s.config.PollInterval
	})

	s.addressChecker.Start(ctx)

	log.Info("address-filter service started",
		"poll_interval", s.config.PollInterval,
	)
}

func (s *FilterService) GetHashCount() int {
	if !s.config.Enable {
		return 0
	}
	return s.hashStore.Size()
}

// GetHashStoreDigest GetETag returns the S3 ETag Digest of the currently loaded hash list.
func (s *FilterService) GetHashStoreDigest() string {
	if !s.config.Enable {
		return ""
	}
	return s.hashStore.Digest()
}

func (s *FilterService) GetLoadedAt() time.Time {
	if !s.config.Enable {
		return time.Time{}
	}
	return s.hashStore.LoadedAt()
}

func (s *FilterService) GetHashStore() *HashStore {
	if !s.config.Enable {
		return nil
	}
	return s.hashStore
}

func (s *FilterService) GetAddressChecker() *HashedAddressChecker {
	if !s.config.Enable {
		return nil
	}
	return s.addressChecker
}
