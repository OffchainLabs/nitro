// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"context"
	"fmt"

	"github.com/offchainlabs/nitro/arbstate/daprovider"
	"github.com/offchainlabs/nitro/das/dastree"
	"github.com/offchainlabs/nitro/util/pretty"
	flag "github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/lru"
	"github.com/ethereum/go-ethereum/log"
)

type CacheConfig struct {
	Enable   bool `koanf:"enable"`
	Capacity int  `koanf:"capacity"`
}

var DefaultCacheConfig = CacheConfig{
	Capacity: 20_000,
}

var TestCacheConfig = CacheConfig{
	Capacity: 1_000,
}

func CacheConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".enable", DefaultCacheConfig.Enable, "Enable local in-memory caching of sequencer batch data")
	f.Int(prefix+".capacity", DefaultCacheConfig.Capacity, "Maximum number of entries (up to 64KB each) to store in the cache.")
}

type CacheStorageService struct {
	baseStorageService StorageService
	cache              *lru.Cache[common.Hash, []byte]
}

func NewCacheStorageService(cacheConfig CacheConfig, baseStorageService StorageService) *CacheStorageService {
	return &CacheStorageService{
		baseStorageService: baseStorageService,
		cache:              lru.NewCache[common.Hash, []byte](cacheConfig.Capacity),
	}
}

func (c *CacheStorageService) GetByHash(ctx context.Context, key common.Hash) ([]byte, error) {
	log.Trace("das.CacheStorageService.GetByHash", "key", pretty.PrettyHash(key), "this", c)

	if val, wasCached := c.cache.Get(key); wasCached {
		return val, nil
	}

	val, err := c.baseStorageService.GetByHash(ctx, key)
	if err != nil {
		return nil, err
	}

	c.cache.Add(key, val)

	return val, nil
}

func (c *CacheStorageService) Put(ctx context.Context, value []byte, timeout uint64) error {
	logPut("das.CacheStorageService.Put", value, timeout, c)
	err := c.baseStorageService.Put(ctx, value, timeout)
	if err != nil {
		return err
	}
	c.cache.Add(common.Hash(dastree.Hash(value)), value)
	return nil
}

func (c *CacheStorageService) Sync(ctx context.Context) error {
	return c.baseStorageService.Sync(ctx)
}

func (c *CacheStorageService) Close(ctx context.Context) error {
	return c.baseStorageService.Close(ctx)
}

func (c *CacheStorageService) ExpirationPolicy(ctx context.Context) (daprovider.ExpirationPolicy, error) {
	return c.baseStorageService.ExpirationPolicy(ctx)
}

func (c *CacheStorageService) String() string {
	return fmt.Sprintf("CacheStorageService(size:%+v)", len(c.cache.Keys()))
}

func (c *CacheStorageService) HealthCheck(ctx context.Context) error {
	return c.baseStorageService.HealthCheck(ctx)
}
