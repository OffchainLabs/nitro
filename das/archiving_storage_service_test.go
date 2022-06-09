// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"bytes"
	"context"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/offchainlabs/nitro/arbstate"
	"testing"
	"time"
)

func TestArchivingStorageService(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	futureTime := uint64(time.Now().Add(time.Hour).Unix())
	val1 := []byte("First value")
	hash1 := crypto.Keccak256(val1)
	val2 := []byte("Second value")
	hash2 := crypto.Keccak256(val2)

	firstStorage := NewMemoryBackedStorageService(ctx)
	archiveTo := NewMemoryBackedStorageService(ctx)

	err := firstStorage.Put(ctx, val1, futureTime)
	Require(t, err)

	archServ, err := NewArchivingStorageService(ctx, firstStorage, archiveTo, 60*60)
	Require(t, err)

	// verify that archServ is a StorageService
	var ss StorageService = archServ
	_ = ss

	result1, err1 := archServ.GetByHash(ctx, hash1)
	err2 := archServ.Put(ctx, val2, futureTime)
	// don't check results yet, in the hope that we call asdr.Close with some ops still in the archive queue
	err3 := archServ.Close(ctx)

	if !bytes.Equal(val1, result1) {
		t.Fatal()
	}
	Require(t, err1)
	Require(t, err2)
	Require(t, err3)

	result1, err1 = archiveTo.GetByHash(ctx, hash1)
	if !bytes.Equal(val1, result1) {
		t.Fatal()
	}
	Require(t, err1)
	result2, err2 := archiveTo.GetByHash(ctx, hash2)
	if !bytes.Equal(val2, result2) {
		t.Fatal()
	}
	Require(t, err2)

	// verify that an ArchivingSimpleDASReader is a DataAvailabilityReader
	var firstSDR arbstate.DataAvailabilityReader = firstStorage
	asdr, err := NewArchivingSimpleDASReader(ctx, firstSDR, archiveTo, 60*60)
	Require(t, err)
	var secondSDR arbstate.DataAvailabilityReader = asdr
	_ = secondSDR
}
