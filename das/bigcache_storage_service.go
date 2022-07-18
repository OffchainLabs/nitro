// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"context"
	"fmt"
	"time"

	"github.com/allegro/bigcache"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/util/pretty"
	flag "github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
)

type BigCacheConfig struct {
	// TODO add other config information like HardMaxCacheSize
	Enable     bool          `koanf:"enable"`
	Expiration time.Duration `koanf:"expiration"`
}

var DefaultBigCacheConfig = BigCacheConfig{
	Expiration: time.Hour,
}

func BigCacheConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".enable", DefaultBigCacheConfig.Enable, "Enable local in-memory caching of sequencer batch data")
	f.Duration(prefix+".expiration", DefaultBigCacheConfig.Expiration, "Expiration time for in-memory cached sequencer batches")
}

type BigCacheStorageService struct {
	baseStorageService StorageService
	bigCacheConfig     BigCacheConfig
	bigCache           *bigcache.BigCache
}

func NewBigCacheStorageService(bigCacheConfig BigCacheConfig, baseStorageService StorageService) (StorageService, error) {
	bigCache, err := bigcache.NewBigCache(bigcache.DefaultConfig(bigCacheConfig.Expiration))
	if err != nil {
		return nil, err
	}
	return &BigCacheStorageService{
		baseStorageService: baseStorageService,
		bigCacheConfig:     bigCacheConfig,
		bigCache:           bigCache,
	}, nil
}

func (bcs *BigCacheStorageService) GetByHash(ctx context.Context, key []byte) ([]byte, error) {
	log.Trace("das.BigCacheStorageService.GetByHash", "key", pretty.FirstFewBytes(key), "this", bcs)

	ret, err := bcs.bigCache.Get(string(key))
	if err != nil {
		ret, err = bcs.baseStorageService.GetByHash(ctx, key)
		if err != nil {
			return nil, err
		}

		err = bcs.bigCache.Set(string(key), ret)
		if err != nil {
			return nil, err
		}
		return ret, err
	}

	return ret, err
}

func (bcs *BigCacheStorageService) Put(ctx context.Context, value []byte, timeout uint64) error {
	log.Trace("das.BigCacheStorageService.Put", "message", pretty.FirstFewBytes(value), "timeout", time.Unix(int64(timeout), 0), "this", bcs)

	err := bcs.baseStorageService.Put(ctx, value, timeout)
	if err != nil {
		return err
	}
	err = bcs.bigCache.Set(string(crypto.Keccak256(value)), value)
	return err
}

func (bcs *BigCacheStorageService) Sync(ctx context.Context) error {
	return bcs.baseStorageService.Sync(ctx)
}

func (bcs *BigCacheStorageService) Close(ctx context.Context) error {
	err := bcs.bigCache.Close()
	if err != nil {
		return err
	}
	return bcs.baseStorageService.Close(ctx)
}

func (bcs *BigCacheStorageService) ExpirationPolicy(ctx context.Context) (arbstate.ExpirationPolicy, error) {
	return bcs.baseStorageService.ExpirationPolicy(ctx)
}

func (bcs *BigCacheStorageService) String() string {
	return fmt.Sprintf("BigCacheStorageService(%+v)", bcs.bigCacheConfig)
}

func (bcs *BigCacheStorageService) HealthCheck(ctx context.Context) error {
	return bcs.baseStorageService.HealthCheck(ctx)
}
