// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"context"
	"fmt"

	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/das/dastree"
	"github.com/offchainlabs/nitro/util/pretty"
	flag "github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/lru"
	"github.com/ethereum/go-ethereum/log"
)

type BigCacheConfig struct {
	Enable   bool `koanf:"enable"`
	Capacity int  `koanf:"capacity"`
}

var DefaultBigCacheConfig = BigCacheConfig{
	Capacity: 20_000,
}

var TestBigCacheConfig = BigCacheConfig{
	Capacity: 1_000,
}

func BigCacheConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".enable", DefaultBigCacheConfig.Enable, "Enable local in-memory caching of sequencer batch data")
	f.Int(prefix+".capacity", DefaultBigCacheConfig.Capacity, "Maximum number of entries (up to 64KB each) to store in the cache.")
}

type BigCacheStorageService struct {
	baseStorageService StorageService
	cache              *lru.Cache[common.Hash, []byte]
}

func NewBigCacheStorageService(bigCacheConfig BigCacheConfig, baseStorageService StorageService) *BigCacheStorageService {
	return &BigCacheStorageService{
		baseStorageService: baseStorageService,
		cache:              lru.NewCache[common.Hash, []byte](bigCacheConfig.Capacity),
	}
}

func (bcs *BigCacheStorageService) GetByHash(ctx context.Context, key common.Hash) ([]byte, error) {
	log.Trace("das.BigCacheStorageService.GetByHash", "key", pretty.PrettyHash(key), "this", bcs)

	if val, wasCached := bcs.cache.Get(key); wasCached {
		return val, nil
	}

	val, err := bcs.baseStorageService.GetByHash(ctx, key)
	if err != nil {
		return nil, err
	}

	bcs.cache.Add(key, val)

	return val, nil
}

func (bcs *BigCacheStorageService) Put(ctx context.Context, value []byte, timeout uint64) error {
	logPut("das.BigCacheStorageService.Put", value, timeout, bcs)
	err := bcs.baseStorageService.Put(ctx, value, timeout)
	if err != nil {
		return err
	}
	bcs.cache.Add(common.Hash(dastree.Hash(value)), value)
	return nil
}

func (bcs *BigCacheStorageService) Sync(ctx context.Context) error {
	return bcs.baseStorageService.Sync(ctx)
}

func (bcs *BigCacheStorageService) Close(ctx context.Context) error {
	return bcs.baseStorageService.Close(ctx)
}

func (bcs *BigCacheStorageService) ExpirationPolicy(ctx context.Context) (arbstate.ExpirationPolicy, error) {
	return bcs.baseStorageService.ExpirationPolicy(ctx)
}

func (bcs *BigCacheStorageService) String() string {
	return fmt.Sprintf("BigCacheStorageService(size:%+v)", len(bcs.cache.Keys()))
}

func (bcs *BigCacheStorageService) HealthCheck(ctx context.Context) error {
	return bcs.baseStorageService.HealthCheck(ctx)
}
