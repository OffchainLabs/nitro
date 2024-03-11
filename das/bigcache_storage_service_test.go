// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/offchainlabs/nitro/das/dastree"
)

func TestBigCacheStorageService(t *testing.T) {
	ctx := context.Background()
	baseStorageService := NewMemoryBackedStorageService(ctx)
	bigCacheService := NewBigCacheStorageService(TestBigCacheConfig, baseStorageService)

	val1 := []byte("The first value")
	val1CorrectKey := dastree.Hash(val1)
	val1IncorrectKey := dastree.Hash(append(val1, 0))

	_, err := bigCacheService.GetByHash(ctx, val1CorrectKey)
	if !errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}

	err = bigCacheService.Put(ctx, val1, 1)
	Require(t, err)

	_, err = bigCacheService.GetByHash(ctx, val1IncorrectKey)
	if !errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}
	val, err := bigCacheService.GetByHash(ctx, val1CorrectKey)
	Require(t, err)
	if !bytes.Equal(val, val1) {
		t.Fatal(val, val1)
	}

	// For Case where the value is present in the base storage but not present in the cache.
	val2 := []byte("The Second value")
	val2CorrectKey := dastree.Hash(val2)
	val2IncorrectKey := dastree.Hash(append(val2, 0))

	err = baseStorageService.Put(ctx, val2, 1)
	Require(t, err)

	_, err = bigCacheService.GetByHash(ctx, val2IncorrectKey)
	if !errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}
	val, err = bigCacheService.GetByHash(ctx, val2CorrectKey)
	Require(t, err)
	if !bytes.Equal(val, val2) {
		t.Fatal(val, val2)
	}

	// For Case where the value is present in the cache storage but not present in the base.
	emptyBaseStorageService := NewMemoryBackedStorageService(ctx)
	bigCacheServiceWithEmptyBaseStorage := &BigCacheStorageService{
		baseStorageService: emptyBaseStorageService,
		cache:              bigCacheService.cache,
	}
	val, err = bigCacheServiceWithEmptyBaseStorage.GetByHash(ctx, val1CorrectKey)
	Require(t, err)
	if !bytes.Equal(val, val1) {
		t.Fatal(val, val1)
	}

	// Closes the base storage properly.
	err = bigCacheService.Close(ctx)
	Require(t, err)
	_, err = baseStorageService.GetByHash(ctx, val1CorrectKey)
	if !errors.Is(err, ErrClosed) {
		t.Fatal(err)
	}
}
