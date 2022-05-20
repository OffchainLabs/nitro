// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
)

func TestBigCacheStorageService(t *testing.T) {
	ctx := context.Background()
	timeout := uint64(time.Now().Add(time.Hour).Unix())
	baseStorageService := NewMemoryBackedStorageService(ctx)
	bigCacheService, err := NewBigCacheStorageService(BigCacheConfig{time.Hour}, baseStorageService)
	Require(t, err)

	val1 := []byte("The first value")
	val1CorrectKey := crypto.Keccak256(val1)
	val1IncorrectKey := crypto.Keccak256(append(val1, 0))

	_, err = bigCacheService.GetByHash(ctx, val1CorrectKey)
	if !errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}

	err = bigCacheService.Put(ctx, val1, timeout)
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
	val2CorrectKey := crypto.Keccak256(val2)
	val2IncorrectKey := crypto.Keccak256(append(val2, 0))

	err = baseStorageService.Put(ctx, val2, timeout)
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

	// Closes the base storage properly.
	err = bigCacheService.Close(ctx)
	Require(t, err)
	_, err = baseStorageService.GetByHash(ctx, val1CorrectKey)
	if !errors.Is(err, ErrClosed) {
		t.Fatal(err)
	}
}
