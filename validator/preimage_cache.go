package validator

/*
#cgo CFLAGS: -g -Wall -I../arbitrator/target/env/include/
#include "arbitrator.h"
#include <stdlib.h>
*/
import "C"
import (
	"sync"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/pkg/errors"
)

// deletion without maitenance leaves a preimageEntry in memory
// A few MBs of those should be o.k.
const maintenanceEvery int32 = 100000

type preimageCache struct {
	cacheMap                  sync.Map
	maintenance               sync.RWMutex
	deletionsSinceMaintenance int32
}

type preimageEntry struct {
	Mutex    sync.Mutex
	Refcount int
	Data     []byte
}

func (p *preimageCache) PourToCache(preimages map[common.Hash][]byte) []common.Hash {
	// multiple can be done in parallel, but cannot be done during maintenance
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
			curEntry.Data = val
		}
		curEntry.Refcount += 1
		curEntry.Mutex.Unlock()
		hashlist = append(hashlist, hash)
	}
	return hashlist
}

func (p *preimageCache) RemoveFromCache(hashlist []common.Hash) error {
	// don't need maintenance lock because we only decrease refcount
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
			curEntry.Data = nil
			deletionsNum := atomic.AddInt32(&p.deletionsSinceMaintenance, 1)
			if deletionsNum%maintenanceEvery == 0 {
				go p.CacheMaintenance()
			}
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
func (p *preimageCache) FillHashedValues(hashlist []common.Hash) (map[common.Hash][]byte, error) {
	length := len(hashlist)
	res := make(map[common.Hash][]byte, length)
	for _, hash := range hashlist {
		actual, found := p.cacheMap.Load(hash)
		if !found {
			return nil, errors.New("preimage not in cache")
		}
		curEntry, ok := actual.(*preimageEntry)
		if !ok {
			return nil, errors.New("preimage malformed in cache")
		}

		curEntry.Mutex.Lock()
		curData := curEntry.Data
		curRefCount := curEntry.Refcount
		curEntry.Mutex.Unlock()
		if curRefCount <= 0 {
			return nil, errors.New("preimage cache in bad state")
		}
		res[hash] = curData
	}
	return res, nil
}
