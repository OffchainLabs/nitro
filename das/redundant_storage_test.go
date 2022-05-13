// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"
)

const NumServices = 3

func TestRedundantStorageService(t *testing.T) {
	ctx := context.Background()
	timeout := uint64(time.Now().Add(time.Hour).Unix())
	services := []StorageService{}
	for i := 0; i < NumServices; i++ {
		services = append(services, NewMemoryBackedStorageService(ctx))
	}
	redundantService, err := NewRedundantStorageService(ctx, services)
	Require(t, err)

	key1 := []byte("The first key")
	key2 := []byte("The second key")
	val1 := []byte("The first value")

	_, err = redundantService.Read(ctx, key1)
	if !errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}

	err = redundantService.Write(ctx, key1, val1, timeout)
	Require(t, err)

	_, err = redundantService.Read(ctx, key2)
	if !errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}
	val, err := redundantService.Read(ctx, key1)
	Require(t, err)
	if !bytes.Equal(val, val1) {
		t.Fatal(val, val1)
	}

	err = redundantService.Close(ctx)
	Require(t, err)

	_, err = redundantService.Read(ctx, key1)
	if !errors.Is(err, ErrClosed) {
		t.Fatal(err)
	}
}
