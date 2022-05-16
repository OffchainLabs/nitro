// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"bytes"
	"context"
	"github.com/ethereum/go-ethereum/crypto"
	"testing"
	"time"
)

func TestArchivingSimpleDASReader(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	futureTime := uint64(time.Now().Add(time.Hour).Unix())
	val1 := []byte("First value")
	hash1 := crypto.Keccak256(val1)
	val2 := []byte("Second value")
	hash2 := crypto.Keccak256(val2)

	firstStorage := NewMemoryBackedStorageService(ctx)
	archiveTo := NewMemoryBackedStorageService(ctx)

	err := firstStorage.PutByHash(ctx, hash1, val1, futureTime)
	Require(t, err)
	err = firstStorage.PutByHash(ctx, hash2, val2, futureTime)
	Require(t, err)

	asdr, err := NewArchivingSimpleDASReader(ctx, firstStorage, archiveTo, 60*60)
	Require(t, err)

	result1, err1 := asdr.GetByHash(ctx, hash1)
	result2, err2 := asdr.GetByHash(ctx, hash2)
	// don't check results yet, in the hope that we call asdr.Close with some ops still in the archive queue
	err3 := asdr.Close(ctx)

	if !bytes.Equal(val1, result1) {
		t.Fatal()
	}
	Require(t, err1)
	if !bytes.Equal(val2, result2) {
		t.Fatal()
	}
	Require(t, err2)
	Require(t, err3)

	result1, err1 = archiveTo.GetByHash(ctx, hash1)
	if !bytes.Equal(val1, result1) {
		t.Fatal()
	}
	Require(t, err1)
	result2, err2 = archiveTo.GetByHash(ctx, hash2)
	if !bytes.Equal(val2, result2) {
		t.Fatal()
	}
	Require(t, err2)
}
