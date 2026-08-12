// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package addressfilter

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/util/s3syncer"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

// fileSync bundles one configured hash-list file: its config, its own hash
// store and its syncer.
type fileSync struct {
	config    FileConfig
	hashStore *HashStore
	syncMgr   *S3SyncManager
}

// FilterService manages the address-filter synchronization pipeline.
// It periodically polls S3 for updates of each configured hash-list file,
// each at its own interval, and maintains in-memory copies for efficient
// address filtering. An address is restricted if it appears in any file.
type FilterService struct {
	stopwaiter.StopWaiter
	config         *Config
	files          []*fileSync
	storeSet       *HashStoreSet
	addressChecker *HashedAddressChecker
}

// NewFilterService creates a new address-filter service.
func NewFilterService(config *Config) (*FilterService, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	files := make([]*fileSync, 0, len(config.Files))
	stores := make([]*HashStore, 0, len(config.Files))
	for i := range config.Files {
		fileConfig := config.Files[i].withDefaults()
		maxHashes := fileConfig.numPreallocatedHashes()
		if maxHashes > 0 {
			log.Info("address-filter preallocating memory for hash list",
				"bucket", fileConfig.Bucket, "key", fileConfig.ObjectKey, "maxHashes", maxHashes)
		}
		hashStore := newHashStore(config.CacheSize, maxHashes)
		log.Info("address-filter file configured",
			"index", i, "bucket", fileConfig.Bucket, "key", fileConfig.ObjectKey, "poll_interval", fileConfig.PollInterval)
		fs := &fileSync{
			config:    fileConfig,
			hashStore: hashStore,
		}
		fs.syncMgr = NewS3SyncManager(&fs.config, hashStore, newFileSizeGauge(&fs.config))
		files = append(files, fs)
		stores = append(stores, hashStore)
	}
	if config.StaticList != "" {
		staticStore := newHashStore(config.CacheSize, 0)
		var listMeta *ListMeta
		fill := func(addHash func(common.Hash)) (*ListMeta, error) {
			var err error
			listMeta, err = parseHashListStream(strings.NewReader(config.StaticList), addHash)
			return listMeta, err
		}
		if err := staticStore.Store("static-list", estimateHashCount(int64(len(config.StaticList)), minBytesPerHashEntry), fill); err != nil {
			return nil, fmt.Errorf("failed to parse address-filter static-list: %w", err)
		}
		log.Info("address-filter static list loaded",
			"filterSetID", listMeta.ID, "hash_count", staticStore.Size(), "scheme", listMeta.Scheme)
		stores = append(stores, staticStore)
	}
	storeSet := NewHashStoreSet(stores)

	return &FilterService{
		config:         config,
		files:          files,
		storeSet:       storeSet,
		addressChecker: NewHashedAddressChecker(storeSet, config.AddressCheckerWorkerCount, config.AddressCheckerQueueSize),
	}, nil
}

func (f *fileSync) recordSyncFailure(err error) {
	syncFailureCounter.Inc(1)
	if errors.Is(err, s3syncer.ErrObjectTooLarge) {
		fileTooLargeCounter.Inc(1)
	}
}

// Initialize downloads the initial hash list of every configured file, one at
// a time so at most one temporary download file occupies download-dir.
// This method blocks until every hash list is successfully loaded.
// If this fails, the node should not start.
func (s *FilterService) Initialize(ctx context.Context) error {
	for _, fs := range s.files {
		log.Info("initializing address-filter file, downloading initial hash list",
			"bucket", fs.config.Bucket,
			"key", fs.config.ObjectKey,
		)

		if err := fs.syncMgr.Initialize(ctx); err != nil {
			return fmt.Errorf("failed to init S3 syncer for s3://%s/%s: %w", fs.config.Bucket, fs.config.ObjectKey, err)
		}

		// Force download (ignore ETag check for initial load)
		if err := fs.syncMgr.Syncer.DownloadAndLoad(ctx); err != nil {
			fs.recordSyncFailure(err)
			return fmt.Errorf("failed to load initial hash list from s3://%s/%s: %w", fs.config.Bucket, fs.config.ObjectKey, err)
		}

		log.Info("address-filter file loaded",
			"bucket", fs.config.Bucket,
			"key", fs.config.ObjectKey,
			"hash_count", fs.hashStore.Size(),
			"etag-digest", fs.hashStore.Digest(),
		)
	}

	log.Info("address-filter service initialized", "file_count", len(s.files))
	return nil
}

// Start begins one background polling goroutine per configured file.
// This should be called after Initialize() succeeds.
func (s *FilterService) Start(ctx context.Context) {
	s.StopWaiter.Start(ctx, s)

	// Start one periodic polling goroutine per file, each at its own interval
	for _, fs := range s.files {
		s.CallIteratively(func(ctx context.Context) time.Duration {
			if err := fs.syncMgr.Syncer.CheckAndSync(ctx); err != nil {
				fs.recordSyncFailure(err)
				if errors.Is(err, s3syncer.ErrObjectTooLarge) {
					log.Error("address-filter S3 file exceeds max-file-size, skipping download; keeping previously loaded list",
						"bucket", fs.config.Bucket, "key", fs.config.ObjectKey, "err", err)
				} else {
					log.Error("failed to sync address-filter list; keeping previously loaded list",
						"bucket", fs.config.Bucket, "key", fs.config.ObjectKey, "err", err)
				}
			}
			return fs.config.PollInterval
		})
	}

	s.StartAndTrackChild(s.addressChecker)

	log.Info("address-filter service started",
		"file_count", len(s.files),
	)
}

func (s *FilterService) TriggerSyncForTest(_ *testing.T, ctx context.Context) error {
	var errs []error
	for _, fs := range s.files {
		errs = append(errs, fs.syncMgr.Syncer.CheckAndSync(ctx))
	}
	return errors.Join(errs...)
}

func (s *FilterService) numFiles() int {
	return len(s.files)
}

func (s *FilterService) getHashCount(i int) int {
	return s.files[i].hashStore.Size()
}

// getHashStoreDigest returns the S3 ETag Digest of the hash list currently loaded for file i.
func (s *FilterService) getHashStoreDigest(i int) string {
	return s.files[i].hashStore.Digest()
}

// AllFilesLoaded reports whether every configured file has loaded a hash list.
func (s *FilterService) AllFilesLoaded() bool {
	return s.storeSet.AllLoaded()
}

func (s *FilterService) GetHashStore(i int) *HashStore {
	return s.files[i].hashStore
}

func (s *FilterService) CurrentFilterSetIDs() []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(s.storeSet.stores))
	for _, store := range s.storeSet.stores {
		ids = append(ids, store.Id())
	}
	return ids
}

func (s *FilterService) GetAddressChecker() *HashedAddressChecker {
	return s.addressChecker
}
