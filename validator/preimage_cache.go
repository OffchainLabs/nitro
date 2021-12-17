package validator

/*
#cgo CFLAGS: -g -Wall -I../arbitrator/target/env/include/
#include "arbitrator.h"
#include <stdlib.h>
*/
import "C"
import (
	"sync"
	"unsafe"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/pkg/errors"
)

type preimageCache struct {
	cacheMap    sync.Map
	maintenance sync.RWMutex
}

type preimageEntry struct {
	Mutex    sync.Mutex
	Refcount int
	Data     C.CByteArray
}

func (p *preimageCache) PourToCache(preimages map[common.Hash][]byte) []common.Hash {
	p.maintenance.RLock()
	defer p.maintenance.RUnlock()
	var newEntry *preimageEntry = nil
	hashlist := make([]common.Hash, 0, len(preimages))
	for hash, val := range preimages {
		if newEntry == nil {
			newEntry = new(preimageEntry)
		}
		actual, found := p.cacheMap.LoadOrStore(hash, newEntry)
		var curEntry *preimageEntry
		if found {
			ok := true
			curEntry, ok = actual.(*preimageEntry)
			if !ok {
				p.cacheMap.Store(hash, newEntry)
				curEntry = newEntry
				newEntry = nil
			}
		} else {
			curEntry = newEntry
			newEntry = nil
		}
		curEntry.Mutex.Lock()
		if curEntry.Refcount == 0 {
			curEntry.Data = CreateCByteArray(val)
		}
		curEntry.Refcount += 1
		curEntry.Mutex.Unlock()
		hashlist = append(hashlist, hash)
	}
	return hashlist
}

func (p *preimageCache) RemoveFromCache(hashlist []common.Hash) error {
	// don't need maintenance because we only decrease refcount
	for _, hash := range hashlist {
		actual, found := p.cacheMap.Load(hash)
		if !found {
			return errors.New("preimage not in cache")
		}
		curEntry, ok := actual.(*preimageEntry)
		if !ok {
			return errors.New("preimage cache entry invalid")
		}
		curEntry.Mutex.Lock()
		prevref := curEntry.Refcount
		curEntry.Refcount -= 1
		if curEntry.Refcount == 0 {
			DestroyCByteArray(curEntry.Data)
		}
		curEntry.Mutex.Unlock()
		if prevref <= 0 {
			return errors.New("preimage reference underflow")
		}
	}
	return nil
}

func (p *preimageCache) CacheMaintenance() {
	p.maintenance.Lock()
	defer p.maintenance.Unlock()
	p.cacheMap.Range(func(key, val interface{}) bool {
		entry, ok := val.(*preimageEntry)
		if !ok {
			log.Error("preimage map: invalid entry")
			return false
		}
		refc := entry.Refcount
		if refc == 0 {
			p.cacheMap.Delete(key)
		}
		return true
	})
}

// The top-level CMultipleByteArrays returned must be freed, but the inner byte arrays must **not** be freed.
func (p *preimageCache) PrepareMultByteArrays(hashlist []common.Hash) (C.CMultipleByteArrays, error) {
	length := len(hashlist)
	array := AllocateMultipleCByteArrays(length)
	for i, hash := range hashlist {
		actual, found := p.cacheMap.Load(hash)
		if !found {
			C.free(unsafe.Pointer(array.ptr))
			return C.CMultipleByteArrays{}, errors.New("preimage not in cache")
		}
		curEntry, ok := actual.(*preimageEntry)
		if !ok {
			C.free(unsafe.Pointer(array.ptr))
			return C.CMultipleByteArrays{}, errors.New("preimage malformed in cache")
		}

		curEntry.Mutex.Lock()
		curData := curEntry.Data
		curRefCount := curEntry.Refcount
		curEntry.Mutex.Unlock()
		if curRefCount <= 0 {
			C.free(unsafe.Pointer(array.ptr))
			return C.CMultipleByteArrays{}, errors.New("preimage cache in bad state")
		}
		UpdateCByteArrayInMultiple(array, i, curData)
	}
	return array, nil
}
