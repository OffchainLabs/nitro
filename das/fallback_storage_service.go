// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"bytes"
	"context"
	"errors"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/util/arbmath"
	"sync"
	"time"
)

type FallbackStorageService struct {
	StorageService
	backup                     arbstate.SimpleDASReader
	backupRetentionSeconds     uint64
	ignoreRetentionWriteErrors bool
	preventRecursiveGets       bool
	currentlyFetching          map[[32]byte]bool
	currentlyFetchingMutex     sync.RWMutex
}

// This is a StorageService that relies on a "primary" StorageService and a "backup". Puts go to the primary.
// GetByHashes are tried first in the primary. If they aren't found in the primary, the backup is tried, and
//     a successful GetByHash result from the backup is Put into the primary.
func NewFallbackStorageService(
	primary StorageService,
	backup arbstate.SimpleDASReader,
	backupRetentionSeconds uint64, // how long to retain data that we copy in from the backup (MaxUint64 means forever)
	ignoreRetentionWriteErrors bool, // if true, don't return error if write of retention data to primary fails
	preventRecursiveGets bool, // if true, return NotFound on simultaneous calls to Gets that miss in primary (prevents infinite recursion)
) *FallbackStorageService {
	return &FallbackStorageService{
		primary,
		backup,
		backupRetentionSeconds,
		ignoreRetentionWriteErrors,
		preventRecursiveGets,
		make(map[[32]byte]bool),
		sync.RWMutex{},
	}
}

func (f *FallbackStorageService) GetByHash(ctx context.Context, key []byte) ([]byte, error) {
	var key32 [32]byte
	if f.preventRecursiveGets {
		f.currentlyFetchingMutex.RLock()
		copy(key32[:], key)
		if f.currentlyFetching[key32] {
			// This is a recursive call, so return not-found
			f.currentlyFetchingMutex.RUnlock()
			return nil, ErrNotFound
		}
		f.currentlyFetchingMutex.RUnlock()
	}

	data, err := f.StorageService.GetByHash(ctx, key)
	if errors.Is(err, ErrNotFound) {
		doDelete := false
		if f.preventRecursiveGets {
			f.currentlyFetchingMutex.Lock()
			if !f.currentlyFetching[key32] {
				f.currentlyFetching[key32] = true
				doDelete = true
			}
			f.currentlyFetchingMutex.Unlock()
		}
		data, err = f.backup.GetByHash(ctx, key)
		if doDelete {
			f.currentlyFetchingMutex.Lock()
			delete(f.currentlyFetching, key32)
			f.currentlyFetchingMutex.Unlock()
		}
		if err != nil {
			return nil, err
		}
		if bytes.Equal(key, crypto.Keccak256(data)) {
			putErr := f.StorageService.Put(ctx, data, arbmath.SaturatingUAdd(uint64(time.Now().Unix()), f.backupRetentionSeconds))
			if putErr != nil && !f.ignoreRetentionWriteErrors {
				return nil, err
			}
		}
	}
	return data, err
}

func (f *FallbackStorageService) String() string {
	return "FallbackStorageService(" + f.StorageService.String() + ")"
}

func (f *FallbackStorageService) HealthCheck(ctx context.Context) error {
	err := f.StorageService.HealthCheck(ctx)
	if err != nil {
		return err
	}
	return f.backup.HealthCheck(ctx)
}
