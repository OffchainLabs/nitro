// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/util/pretty"
)

type CacheStorageToDASAdapter struct {
	DataAvailabilityService
	cache StorageService
}

func NewCacheStorageToDASAdapter(
	das DataAvailabilityService,
	cache StorageService,
) *CacheStorageToDASAdapter {
	return &CacheStorageToDASAdapter{
		DataAvailabilityService: das,
		cache:                   cache,
	}
}

func (a *CacheStorageToDASAdapter) GetByHash(ctx context.Context, hash common.Hash) ([]byte, error) {
	log.Trace("das.CacheStorageToDASAdapter.GetByHash", "key", pretty.PrettyHash(hash), "this", a)
	ret, err := a.cache.GetByHash(ctx, hash)
	if err != nil {
		ret, err = a.DataAvailabilityService.GetByHash(ctx, hash)
		if err != nil {
			return nil, err
		}

		err = a.cache.Put(ctx, ret, 0 /* this value is ignored for the cache */)
		if err != nil {
			log.Warn("Error caching retrieved DAS batch data, returning anyway", "err", err)
		}
	}

	return ret, nil
}

func (a *CacheStorageToDASAdapter) Store(
	ctx context.Context, message []byte, timeout uint64, sig []byte,
) (*arbstate.DataAvailabilityCertificate, error) {
	log.Trace("das.CacheStorageToDASAdapter.Store", "message", pretty.FirstFewBytes(message), "timeout", time.Unix(int64(timeout), 0), "sig", pretty.FirstFewBytes(sig), "this", a)
	cert, err := a.DataAvailabilityService.Store(ctx, message, timeout, sig)
	if err != nil {
		return nil, err
	}

	err = a.cache.Put(ctx, message, 0 /* this value is ignored for the cache */)
	if err != nil {
		log.Warn("Error caching stored DAS batch data, returning anyway", "err", err)
	}

	return cert, nil
}

func (a *CacheStorageToDASAdapter) String() string {
	return fmt.Sprintf("CacheStorageToDASAdapter{inner: %v, cache: %v}", a.DataAvailabilityService, a.cache)
}

type emptyStorageService struct {
}

func NewEmptyStorageService() *emptyStorageService {
	return &emptyStorageService{}
}

func (s *emptyStorageService) GetByHash(ctx context.Context, hash common.Hash) ([]byte, error) {
	return nil, ErrNotFound
}

func (s *emptyStorageService) Put(ctx context.Context, data []byte, expiration uint64) error {
	return nil
}

func (s *emptyStorageService) Sync(ctx context.Context) error {
	return nil
}

func (s *emptyStorageService) Close(ctx context.Context) error {
	return nil
}

func (s *emptyStorageService) ExpirationPolicy(ctx context.Context) (arbstate.ExpirationPolicy, error) {
	return arbstate.DiscardImmediately, nil
}

func (s *emptyStorageService) String() string {
	return "emptyStorageService"
}

func (s *emptyStorageService) HealthCheck(ctx context.Context) error {
	return nil
}
