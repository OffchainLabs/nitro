// Copyright 2024, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package util

import (
	"bytes"
	"slices"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/google/go-cmp/cmp"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

func TestStorageCache(t *testing.T) {
	keys := make([]common.Hash, 3)
	values := make([]common.Hash, len(keys))
	for i := range keys {
		keys[i] = testhelpers.RandomHash()
		values[i] = testhelpers.RandomHash()
	}

	cache := newStorageCache()

	t.Run("load then load", func(t *testing.T) {
		emitLog := cache.Load(keys[0], values[0])
		if !emitLog {
			t.Fatal("unexpected value in cache")
		}
		emitLog = cache.Load(keys[0], values[0])
		if emitLog {
			t.Fatal("expected value in cache")
		}
	})

	t.Run("load another value", func(t *testing.T) {
		emitLog := cache.Load(keys[1], values[1])
		if !emitLog {
			t.Fatal("unexpected value in cache")
		}
	})

	t.Run("load then store", func(t *testing.T) {
		_ = cache.Load(keys[2], values[0])
		cache.Store(keys[2], values[2])
		if !cache.cache[keys[2]].dirty() {
			t.Fatal("expected value to be dirty")
		}
		if cache.cache[keys[2]].Value != values[2] {
			t.Fatal("wrong value in cache")
		}
	})

	t.Run("clear", func(t *testing.T) {
		cache.Clear()
		if len(cache.cache) != 0 {
			t.Fatal("expected to be empty")
		}
	})

	t.Run("store then load", func(t *testing.T) {
		cache.Store(keys[0], values[0])
		emitLog := cache.Load(keys[0], values[0])
		if emitLog {
			t.Fatal("expected value in cache")
		}
	})

	t.Run("flush only stored", func(t *testing.T) {
		_ = cache.Load(keys[1], values[1])
		cache.Store(keys[2], values[2])
		stores := cache.Flush()
		expected := []storageCacheStores{
			{Key: keys[0], Value: values[0]},
			{Key: keys[2], Value: values[2]},
		}
		sortFunc := func(a, b storageCacheStores) int {
			return bytes.Compare(a.Key.Bytes(), b.Key.Bytes())
		}
		slices.SortFunc(stores, sortFunc)
		slices.SortFunc(expected, sortFunc)
		if diff := cmp.Diff(stores, expected); diff != "" {
			t.Fatalf("wrong flush: %s", diff)
		}
		// everything should still be in cache
		for i := range keys {
			entry, ok := cache.cache[keys[i]]
			if !ok {
				t.Fatal("entry missing from cache")
			}
			if entry.dirty() {
				t.Fatal("dirty entry after flush")
			}
			if entry.Value != values[i] {
				t.Fatal("wrong value in entry")
			}
		}
	})

	t.Run("do not flush known values", func(t *testing.T) {
		cache.Clear()
		_ = cache.Load(keys[0], values[0])
		cache.Store(keys[0], values[0])
		stores := cache.Flush()
		if len(stores) != 0 {
			t.Fatal("unexpected store")
		}
	})
}
