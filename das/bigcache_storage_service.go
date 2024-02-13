// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package das

import (
	"context"
	"fmt"
	"time"

	"github.com/allegro/bigcache"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/das/dastree"
	"github.com/offchainlabs/nitro/util/pretty"
	flag "github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

type BigCacheConfig struct {
	// TODO add other config information like HardMaxCacheSize
	Enable             bool          `koanf:"enable"`
	Expiration         time.Duration `koanf:"expiration"`
	MaxEntriesInWindow int
}

var DefaultBigCacheConfig = BigCacheConfig{
	Expiration: time.Hour,
}

var TestBigCacheConfig = BigCacheConfig{
	Enable:             true,
	Expiration:         time.Hour,
	MaxEntriesInWindow: 1000,
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
	conf := bigcache.DefaultConfig(bigCacheConfig.Expiration)
	if bigCacheConfig.MaxEntriesInWindow > 0 {
		conf.MaxEntriesInWindow = bigCacheConfig.MaxEntriesInWindow
	}
	bigCache, err := bigcache.NewBigCache(conf)
	if err != nil {
		return nil, err
	}
	return &BigCacheStorageService{
		baseStorageService: baseStorageService,
		bigCacheConfig:     bigCacheConfig,
		bigCache:           bigCache,
	}, nil
}

func (bcs *BigCacheStorageService) GetByHash(ctx context.Context, key common.Hash) ([]byte, error) {
	log.Trace("das.BigCacheStorageService.GetByHash", "key", pretty.PrettyHash(key), "this", bcs)

	ret, err := bcs.bigCache.Get(string(key.Bytes()))
	if err != nil {
		ret, err = bcs.baseStorageService.GetByHash(ctx, key)
		if err != nil {
			return nil, err
		}

		err = bcs.bigCache.Set(string(key.Bytes()), ret)
		if err != nil {
			return nil, err
		}
		return ret, err
	}

	return ret, err
}

func (bcs *BigCacheStorageService) Put(ctx context.Context, value []byte, timeout uint64) error {
	logPut("das.BigCacheStorageService.Put", value, timeout, bcs)
	err := bcs.baseStorageService.Put(ctx, value, timeout)
	if err != nil {
		return err
	}
	return bcs.bigCache.Set(string(dastree.HashBytes(value)), value)
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
