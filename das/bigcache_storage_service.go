// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"context"
	"encoding/base32"
	"fmt"
	"time"

	"github.com/allegro/bigcache"
	flag "github.com/spf13/pflag"
)

type BigCacheConfig struct {
	// TODO add other config information like HardMaxCacheSize
	Expiration time.Duration `koanf:"big-cache-expiration"`
}

var DefaultBigCacheConfig = BigCacheConfig{
	Expiration: time.Hour,
}

func BigCacheConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Duration(prefix+".big-cache-expiration", DefaultBigCacheConfig.Expiration, "Big cache expiration")
}

type BigCacheStorageService struct {
	baseStorageService StorageService
	bigCacheConfig     BigCacheConfig
	bigCache           *bigcache.BigCache
}

func NewBigCacheStorageService(bigCacheConfig BigCacheConfig, baseStorageService StorageService) (StorageService, error) {
	bigCache, err := bigcache.NewBigCache(bigcache.DefaultConfig(10 * time.Minute))
	if err != nil {
		return nil, err
	}
	return &BigCacheStorageService{
		baseStorageService: baseStorageService,
		bigCacheConfig:     bigCacheConfig,
		bigCache:           bigCache,
	}, nil
}

func (bcs *BigCacheStorageService) Read(ctx context.Context, key []byte) ([]byte, error) {
	ret, err := bcs.bigCache.Get(base32.StdEncoding.EncodeToString(key))
	if err != nil {
		ret, err = bcs.baseStorageService.Read(ctx, key)
		if err != nil {
			return nil, err
		}

		err = bcs.bigCache.Set(base32.StdEncoding.EncodeToString(key), ret)
		if err != nil {
			return nil, err
		}
		return ret, err
	}

	return ret, err
}

func (bcs *BigCacheStorageService) Write(ctx context.Context, key []byte, value []byte, timeout uint64) error {
	err := bcs.baseStorageService.Write(ctx, key, value, timeout)
	if err != nil {
		return err
	}
	err = bcs.bigCache.Set(base32.StdEncoding.EncodeToString(key), value)
	return err
}

func (bcs *BigCacheStorageService) Sync(ctx context.Context) error {
	return nil
}

func (bcs *BigCacheStorageService) Close(ctx context.Context) error {
	err := bcs.bigCache.Close()
	if err != nil {
		return err
	}
	return bcs.baseStorageService.Close(ctx)
}

func (bcs *BigCacheStorageService) String() string {
	return fmt.Sprintf("BigCahceStorageService(:%v)", bcs.bigCacheConfig)
}
