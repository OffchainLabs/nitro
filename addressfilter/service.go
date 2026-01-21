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
type Service struct {
	stopwaiter.StopWaiter
	config  *Config
	store   *HashStore
	syncMgr *S3SyncManager
}

// NewService creates a new address-filteress service.
// Returns nil if the service is not enabled in the configuration.
func NewService(ctx context.Context, config *Config) (*Service, error) {
	if !config.Enable {
		return &Service{config: config}, nil
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	store := NewHashStore()
	syncMgr, err := NewS3SyncManager(ctx, config, store)
	if err != nil {
		return nil, fmt.Errorf("failed to create S3 syncer: %w", err)
	}

	return &Service{
		config:  config,
		store:   store,
		syncMgr: syncMgr,
	}, nil
}

// Initialize downloads the initial hash list from S3.
// This method blocks until the hash list is successfully loaded.
// If this fails, the node should not start.
func (s *Service) Initialize(ctx context.Context) error {
	if !s.config.Enable {
		log.Info("address-filter service disabled")
		return nil
	}
	log.Info("initializing address-filter service, downloading initial hash list",
		"bucket", s.config.S3.Bucket,
		"key", s.config.S3.ObjectKey,
	)

	// Force download (ignore ETag check for initial load)
	if err := s.syncMgr.Syncer.DownloadAndLoad(ctx); err != nil {
		return fmt.Errorf("failed to load initial hash list: %w", err)
	}

	log.Info("address-filter service initialized",
		"hash_count", s.store.Size(),
		"etag-digest", s.store.Digest(),
	)
	return nil
}

// Start begins the background polling goroutine.
// This should be called after Initialize() succeeds.
func (s *Service) Start(ctx context.Context) {
	if !s.config.Enable {
		return
	}
	s.StopWaiter.Start(ctx, s)

	// Start periodic polling goroutine
	s.CallIteratively(func(ctx context.Context) time.Duration {
		if err := s.syncMgr.Syncer.CheckAndSync(ctx); err != nil {
			log.Error("failed to sync address-filter list", "err", err)
		}
		return s.config.PollInterval
	})

	log.Info("address-filter service started",
		"poll_interval", s.config.PollInterval,
	)
}

func (s *Service) GetHashCount() int {
	if !s.config.Enable {
		return 0
	}
	return s.store.Size()
}

// GetHashStoreDigest GetETag returns the S3 ETag Digest of the currently loaded hash list.
func (s *Service) GetHashStoreDigest() string {
	if !s.config.Enable {
		return ""
	}
	return s.store.Digest()
}

func (s *Service) GetLoadedAt() time.Time {
	if !s.config.Enable {
		return time.Time{}
	}
	return s.store.LoadedAt()
}

func (s *Service) GetHashStore() *HashStore {
	if !s.config.Enable {
		return nil
	}
	return s.store
}
