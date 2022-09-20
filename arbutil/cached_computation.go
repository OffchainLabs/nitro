// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbutil

import (
	"sync"
	"sync/atomic"
)

type CachedComputation[T any] struct {
	complete int32
	mutex    sync.Mutex
	value    T
}

func (c *CachedComputation[T]) Get(compute func() (T, error)) (T, error) {
	if atomic.LoadInt32(&c.complete) == 1 {
		return c.value, nil
	}
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if atomic.LoadInt32(&c.complete) == 1 {
		return c.value, nil
	}
	computed, err := compute()
	if err != nil {
		return computed, err
	}
	c.value = computed
	atomic.StoreInt32(&c.complete, 1)
	return computed, err
}
