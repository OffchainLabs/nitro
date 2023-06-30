// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/nitro/blob/master/LICENSE

package das

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/offchainlabs/nitro/das/dastree"
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

	val1 := []byte("The first value")
	key1 := dastree.Hash(val1)
	key2 := dastree.Hash(append(val1, 0))

	_, err = redundantService.GetByHash(ctx, key1)
	if !errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}

	err = redundantService.Put(ctx, val1, timeout)
	Require(t, err)

	_, err = redundantService.GetByHash(ctx, key2)
	if !errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}
	val, err := redundantService.GetByHash(ctx, key1)
	Require(t, err)
	if !bytes.Equal(val, val1) {
		t.Fatal(val, val1)
	}

	for _, serv := range services {
		val, err = serv.GetByHash(ctx, key1)
		Require(t, err)
		if !bytes.Equal(val, val1) {
			t.Fatal(val, val1)
		}
	}

	err = redundantService.Close(ctx)
	Require(t, err)

	_, err = redundantService.GetByHash(ctx, key1)
	if !errors.Is(err, ErrClosed) {
		t.Fatal(err)
	}
}
