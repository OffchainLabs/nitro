// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/common/math"
	"github.com/offchainlabs/nitro/cmd/daprovider/das/dastree"
)

func TestFallbackStorageService(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	val1 := []byte("First value")
	hash1 := dastree.Hash(val1)
	val2 := []byte("Second value")
	hash2 := dastree.Hash(val2)

	primary := NewMemoryBackedStorageService(ctx)
	err := primary.Put(ctx, val1, math.MaxUint64)
	Require(t, err)
	fallback := NewMemoryBackedStorageService(ctx)
	err = fallback.Put(ctx, val2, math.MaxUint64)
	Require(t, err)

	fss := NewFallbackStorageService(primary, fallback, fallback, 60*60, true, true)

	res1, err := fss.GetByHash(ctx, hash1)
	Require(t, err)
	if !bytes.Equal(res1, val1) {
		t.Fatal()
	}
	res2, err := fss.GetByHash(ctx, hash2)
	Require(t, err)
	if !bytes.Equal(res2, val2) {
		t.Fatal()
	}

	res2, err = primary.GetByHash(ctx, hash2)
	Require(t, err)
	if !bytes.Equal(res2, val2) {
		t.Fatal()
	}
}

func TestFallbackStorageServiceRecursive(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	val1 := []byte("First value")
	hash1 := dastree.Hash(val1)

	ss := NewMemoryBackedStorageService(ctx)
	fss := NewFallbackStorageService(ss, ss, ss, 60*60, true, true)

	// artificially make fss recursive
	fss.backup = fss

	// try a recursive read of a non-existent item -- should give ErrNotFound
	_, err := fss.GetByHash(ctx, hash1)
	if !errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}
}
