// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

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

// IterationCompatibleStorageService is a StorageService which is
// compatible to be used as a backend for IterableStorageService.
type IterationCompatibleStorageService interface {
	putKeyValue(ctx context.Context, key common.Hash, value []byte) error
	StorageService
}

// IterationCompatibleStorageServiceAdaptor is an adaptor used to covert iteration incompatible StorageService
// to IterationCompatibleStorageService (basically adds an empty putKeyValue to the StorageService)
type IterationCompatibleStorageServiceAdaptor struct {
	StorageService
}

func (i *IterationCompatibleStorageServiceAdaptor) putKeyValue(ctx context.Context, key common.Hash, value []byte) error {
	return nil
}

func ConvertStorageServiceToIterationCompatibleStorageService(storageService StorageService) IterationCompatibleStorageService {
	service, ok := storageService.(IterationCompatibleStorageService)
	if ok {
		return service
	}
	return &IterationCompatibleStorageServiceAdaptor{storageService}
}

// An IterableStorageService is used as a wrapper on top of a storage service,
// to add the capability of iterating over the stored date in a sequential manner.
type IterableStorageService struct {
	// Local copy of iterator end. End can also be accessed by getByHash for iteratorEnd.
	end atomic.Value // atomic access to common.Hash
	IterationCompatibleStorageService

	mutex sync.Mutex
}

func NewIterableStorageService(storageService IterationCompatibleStorageService) *IterableStorageService {
	i := &IterableStorageService{IterationCompatibleStorageService: storageService}
	i.end.Store(common.Hash{})
	return i
}

func (i *IterableStorageService) Put(ctx context.Context, data []byte, expiration uint64) error {
	dataHash := dastree.Hash(data)

	// Do not insert data if data is already present.
	// (This is being done to avoid redundant hash being added to the
	//  linked list ,since it can lead to loops in the linked list.)
	if _, err := i.IterationCompatibleStorageService.GetByHash(ctx, dataHash); err == nil {
		return nil
	}

	if err := i.IterationCompatibleStorageService.Put(ctx, data, expiration); err != nil {
		return err
	}

	if err := i.putKeyValue(ctx, dastree.Hash([]byte(expirationTimeKeyPrefix+EncodeStorageServiceKey(dastree.Hash(data)))), []byte(strconv.FormatUint(expiration, 10))); err != nil {
		return err
	}

	i.mutex.Lock()
	defer i.mutex.Unlock()

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
	value, err := i.IterationCompatibleStorageService.GetByHash(ctx, dastree.Hash([]byte(expirationTimeKeyPrefix+EncodeStorageServiceKey(hash))))
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
