// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package das

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/offchainlabs/nitro/das/dastree"
)

func TestRedisStorageService(t *testing.T) {
	ctx := context.Background()
	timeout := uint64(time.Now().Add(time.Hour).Unix())
	baseStorageService := NewMemoryBackedStorageService(ctx)
	server, err := miniredis.Run()
	Require(t, err)
	redisService, err := NewRedisStorageService(
		RedisConfig{
			Enable:     true,
			Url:        "redis://" + server.Addr(),
			Expiration: time.Hour,
			KeyConfig:  "b561f5d5d98debc783aa8a1472d67ec3bcd532a1c8d95e5cb23caa70c649f7c9",
		}, baseStorageService)

	Require(t, err)

	val1 := []byte("The first value")
	val1CorrectKey := dastree.Hash(val1)
	val1IncorrectKey := dastree.Hash(append(val1, 0))

	_, err = redisService.GetByHash(ctx, val1CorrectKey)
	if !errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}

	err = redisService.Put(ctx, val1, timeout)
	Require(t, err)

	_, err = redisService.GetByHash(ctx, val1IncorrectKey)
	if !errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}
	val, err := redisService.GetByHash(ctx, val1CorrectKey)
	Require(t, err)
	if !bytes.Equal(val, val1) {
		t.Fatal(val, val1)
	}

	// For Case where the value is present in the base storage but not present in the cache.
	val2 := []byte("The Second value")
	val2CorrectKey := dastree.Hash(val2)
	val2IncorrectKey := dastree.Hash(append(val2, 0))

	err = baseStorageService.Put(ctx, val2, timeout)
	Require(t, err)

	_, err = redisService.GetByHash(ctx, val2IncorrectKey)
	if !errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}
	val, err = redisService.GetByHash(ctx, val2CorrectKey)
	Require(t, err)
	if !bytes.Equal(val, val2) {
		t.Fatal(val, val2)
	}

	// For Case where the value is present in the cache storage but not present in the base.
	emptyBaseStorageService := NewMemoryBackedStorageService(ctx)
	redisServiceWithEmptyBaseStorage, err := NewRedisStorageService(
		RedisConfig{
			Enable:     true,
			Url:        "redis://" + server.Addr(),
			Expiration: time.Hour,
			KeyConfig:  "b561f5d5d98debc783aa8a1472d67ec3bcd532a1c8d95e5cb23caa70c649f7c9",
		}, emptyBaseStorageService)
	Require(t, err)
	val, err = redisServiceWithEmptyBaseStorage.GetByHash(ctx, val1CorrectKey)
	Require(t, err)
	if !bytes.Equal(val, val1) {
		t.Fatal(val, val1)
	}

	err = redisService.Close(ctx)
	Require(t, err)
	_, err = redisService.GetByHash(ctx, val1CorrectKey)
	if !errors.Is(err, ErrClosed) {
		t.Fatal(err)
	}
	// Closes the base storage properly.
	_, err = baseStorageService.GetByHash(ctx, val1CorrectKey)
	if !errors.Is(err, ErrClosed) {
		t.Fatal(err)
	}
}
