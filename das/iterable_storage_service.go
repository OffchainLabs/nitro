// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"context"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/das/dastree"
)

const iteratorStorageKeyPrefix = "iterator_key_prefix_"
const iteratorBegin = "iterator_begin"
const iteratorEnd = "iterator_end"
const expirationTimeKeyPrefix = "expiration_time_key_prefix_"

// An IterableStorageService is used as a wrapper on top of a storage service,
// to add the capability of iterating over the stored date in a sequential manner.
type IterableStorageService struct {
	// Local copy of iterator end. End can also be accessed by getByHash for iteratorEnd.
	end atomic.Value // atomic access to common.Hash
	StorageService

	mutex sync.Mutex
}

func NewIterableStorageService(storageService StorageService) *IterableStorageService {
	i := &IterableStorageService{StorageService: storageService}
	i.end.Store(common.Hash{})
	return i
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

	dataHash := dastree.Hash(data)
	endHash := i.End(ctx)
	if (endHash == common.Hash{}) {
		// First element being inserted in the chain.
		if err := i.putKeyValue(ctx, dastree.Hash([]byte(iteratorBegin)), dataHash.Bytes()); err != nil {
			return err
		}
	} else {
		if err := i.putKeyValue(ctx, dastree.Hash([]byte(iteratorStorageKeyPrefix+EncodeStorageServiceKey(endHash))), dataHash.Bytes()); err != nil {
			return err
		}
	}

	if err := i.putKeyValue(ctx, dastree.Hash([]byte(iteratorEnd)), dataHash.Bytes()); err != nil {
		return err
	}
	i.end.Store(dataHash)

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
	endHash, ok := i.end.Load().(common.Hash)
	if !ok {
		return common.Hash{}
	}
	if (endHash != common.Hash{}) {
		return endHash
	}
	value, err := i.GetByHash(ctx, dastree.Hash([]byte(iteratorEnd)))
	if err != nil {
		return common.Hash{}
	}
	endHash = common.BytesToHash(value)
	i.end.Store(endHash)
	return endHash
}

func (i *IterableStorageService) Next(ctx context.Context, hash common.Hash) common.Hash {
	if hash != i.DefaultBegin() {
		hash = dastree.Hash([]byte(iteratorStorageKeyPrefix + EncodeStorageServiceKey(hash)))
	}
	value, err := i.GetByHash(ctx, hash)
	if err != nil {
		return common.Hash{}
	}
	return common.BytesToHash(value)
}
