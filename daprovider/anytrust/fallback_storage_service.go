// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package anytrust

import (
	"context"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/daprovider/anytrust/tree"
	anytrustutil "github.com/offchainlabs/nitro/daprovider/anytrust/util"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/pretty"
)

type FallbackStorageService struct {
	StorageService
	backup                     anytrustutil.Reader
	backupHealthChecker        ServiceHealthChecker
	backupRetentionSeconds     uint64
	ignoreRetentionWriteErrors bool
	preventRecursiveGets       bool
	currentlyFetching          map[[32]byte]bool
	currentlyFetchingMutex     sync.RWMutex
}

// NewFallbackStorageService is a StorageService that relies on a "primary" StorageService and a "backup". Puts go to the primary.
// GetByHashes are tried first in the primary. If they aren't found in the primary, the backup is tried, and
// a successful GetByHash result from the backup is Put into the primary.
func NewFallbackStorageService(
	primary StorageService,
	backup anytrustutil.Reader,
	backupHealthChecker ServiceHealthChecker,
	backupRetentionSeconds uint64, // how long to retain data that we copy in from the backup (MaxUint64 means forever)
	ignoreRetentionWriteErrors bool, // if true, don't return error if write of retention data to primary fails
	preventRecursiveGets bool, // if true, return NotFound on simultaneous calls to Gets that miss in primary (prevents infinite recursion)
) *FallbackStorageService {
	return &FallbackStorageService{
		StorageService:             primary,
		backup:                     backup,
		backupHealthChecker:        backupHealthChecker,
		backupRetentionSeconds:     backupRetentionSeconds,
		ignoreRetentionWriteErrors: ignoreRetentionWriteErrors,
		preventRecursiveGets:       preventRecursiveGets,
		currentlyFetching:          make(map[[32]byte]bool),
		currentlyFetchingMutex:     sync.RWMutex{},
	}
}

func (f *FallbackStorageService) GetByHash(ctx context.Context, key common.Hash) ([]byte, error) {
	log.Trace("anytrust.FallbackStorageService.GetByHash", "key", pretty.PrettyHash(key), "this", f)
	if f.preventRecursiveGets {
		f.currentlyFetchingMutex.RLock()
		if f.currentlyFetching[key] {
			// This is a recursive call, so return not-found
			f.currentlyFetchingMutex.RUnlock()
			return nil, ErrNotFound
		}
		f.currentlyFetchingMutex.RUnlock()
	}

	data, err := f.StorageService.GetByHash(ctx, key)
	if err != nil {
		doDelete := false
		if f.preventRecursiveGets {
			f.currentlyFetchingMutex.Lock()
			if !f.currentlyFetching[key] {
				f.currentlyFetching[key] = true
				doDelete = true
			}
			f.currentlyFetchingMutex.Unlock()
		}
		log.Trace("anytrust.FallbackStorageService.GetByHash trying fallback")
		data, err = f.backup.GetByHash(ctx, key)
		if doDelete {
			f.currentlyFetchingMutex.Lock()
			delete(f.currentlyFetching, key)
			f.currentlyFetchingMutex.Unlock()
		}
		if err != nil {
			return nil, err
		}
		if tree.ValidHash(key, data) {
			putErr := f.StorageService.Put(
				// #nosec G115
				ctx, data, arbmath.SaturatingUAdd(uint64(time.Now().Unix()), f.backupRetentionSeconds),
			)
			if putErr != nil && !f.ignoreRetentionWriteErrors {
				return nil, err
			}
		}
	}
	return data, err
}

func (f *FallbackStorageService) String() string {
	return "FallbackStorageService(storageService:" + f.StorageService.String() + ")"
}

func (f *FallbackStorageService) HealthCheck(ctx context.Context) error {
	err := f.StorageService.HealthCheck(ctx)
	if err != nil {
		return err
	}
	return f.backupHealthChecker.HealthCheck(ctx)
}
