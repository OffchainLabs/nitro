// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"context"
	"strconv"
	"sync"

	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/das/dastree"
)

const iteratorStorageKeyPrefix = "iterator_key_prefix_"
const iteratorBegin = "iterator_begin"
const iteratorEnd = "iterator_end"
const expirationTimeKeyPrefix = "expiration_time_key_prefix_"

type IterableStorageService struct {
	// Local copy of iterator end. End can also be accessing by getByHash for iteratorEnd.
	end common.Hash
	StorageService

	mutex sync.Mutex
}

func NewIterableStorageService(storageService StorageService) *IterableStorageService {
	return &IterableStorageService{end: common.Hash{}, StorageService: storageService}
}

func (i *IterableStorageService) Put(ctx context.Context, data []byte, expiration uint64) error {
	if err := i.StorageService.Put(ctx, data, expiration); err != nil {
		return err
	}

	if err := i.putKeyValue(ctx, dastree.Hash([]byte(expirationTimeKeyPrefix+EncodeStorageServiceKey(dastree.Hash(data)))), []byte(strconv.FormatUint(expiration, 10))); err != nil {
		return err
	}

	i.mutex.Lock()
	defer i.mutex.Unlock()

	if (i.End(ctx) == common.Hash{}) {
		// First element being inserted in the chain.
		if err := i.putKeyValue(ctx, dastree.Hash([]byte(iteratorBegin)), dastree.Hash(data).Bytes()); err != nil {
			return err
		}
	} else {
		if err := i.putKeyValue(ctx, dastree.Hash([]byte(iteratorStorageKeyPrefix+EncodeStorageServiceKey(i.End(ctx)))), dastree.Hash(data).Bytes()); err != nil {
			return err
		}
	}

	if err := i.putKeyValue(ctx, dastree.Hash([]byte(iteratorEnd)), dastree.Hash(data).Bytes()); err != nil {
		return err
	}
	i.end = dastree.Hash(data)

	return nil
}

func (i *IterableStorageService) GetExpirationTime(ctx context.Context, hash common.Hash) (uint64, error) {
	value, err := i.StorageService.GetByHash(ctx, dastree.Hash([]byte(expirationTimeKeyPrefix+EncodeStorageServiceKey(hash))))
	if err != nil {
		return 0, err
	}

	expirationTime, err := strconv.ParseUint(string(value), 10, 64)
	if err != nil {
		return 0, err
	}
	return expirationTime, nil
}

func (i *IterableStorageService) DefaultBegin() common.Hash {
	return dastree.Hash([]byte(iteratorBegin))
}

func (i *IterableStorageService) End(ctx context.Context) common.Hash {
	if (i.end != common.Hash{}) {
		return i.end
	}
	value, err := i.GetByHash(ctx, dastree.Hash([]byte(iteratorEnd)))
	if err != nil {
		return common.Hash{}
	}
	return common.BytesToHash(value)
}

func (i *IterableStorageService) Next(ctx context.Context, hash common.Hash) common.Hash {
	value, err := i.GetByHash(ctx, dastree.Hash([]byte(iteratorStorageKeyPrefix+EncodeStorageServiceKey(hash))))
	if err != nil {
		return common.Hash{}
	}
	return common.BytesToHash(value)
}
