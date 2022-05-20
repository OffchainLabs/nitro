// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"context"
	"errors"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/util/arbmath"
	"time"
)

type FallbackStorageService struct {
	StorageService
	backup                     arbstate.SimpleDASReader
	backupRetentionSeconds     uint64
	ignoreRetentionWriteErrors bool
}

func NewFallbackStorageService(
	primary StorageService,
	backup arbstate.SimpleDASReader,
	backupRetentionSeconds uint64, // how long to retain data that we copy in from the backup (MaxUint64 means forever)
	ignoreRetentionWriteErrors bool, // if true, don't return error if write of retention data to primary fails
) *FallbackStorageService {
	return &FallbackStorageService{
		primary,
		backup,
		backupRetentionSeconds,
		ignoreRetentionWriteErrors,
	}
}

func (f *FallbackStorageService) GetByHash(ctx context.Context, key []byte) ([]byte, error) {
	data, err := f.StorageService.GetByHash(ctx, key)
	if errors.Is(err, ErrNotFound) {
		data, err = f.backup.GetByHash(ctx, key)
		if err != nil {
			return nil, err
		}

		// write data to the primary, ignore errors because nothing breaks if this doesn't succeed
		putErr := f.StorageService.Put(ctx, data, arbmath.SaturatingUAdd(uint64(time.Now().Unix()), f.backupRetentionSeconds))
		if putErr != nil && !f.ignoreRetentionWriteErrors {
			return nil, err
		}
	}
	return data, err
}

func (f *FallbackStorageService) String() string {
	return "FallbackStorageService(" + f.StorageService.String() + ")"
}
