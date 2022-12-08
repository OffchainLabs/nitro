// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/das/dastree"
)

func TestRegularSyncStorage(t *testing.T) {
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	syncFromStorageService := []*IterableStorageService{
		NewIterableStorageService(ConvertStorageServiceToIterationCompatibleStorageService(NewMemoryBackedStorageService(ctx))),
		NewIterableStorageService(ConvertStorageServiceToIterationCompatibleStorageService(NewMemoryBackedStorageService(ctx))),
	}
	syncToStorageService := []StorageService{
		NewMemoryBackedStorageService(ctx),
		NewMemoryBackedStorageService(ctx),
	}

	regularSyncStorage := NewRegularlySyncStorage(
		syncFromStorageService,
		syncToStorageService, RegularSyncStorageConfig{
			Enable:       true,
			SyncInterval: 100 * time.Millisecond,
		})

	val := [][]byte{
		[]byte("The first value"),
		[]byte("The second value"),
		[]byte("The third value"),
		[]byte("The forth value"),
	}
	valKey := []common.Hash{
		dastree.Hash(val[0]),
		dastree.Hash(val[1]),
		dastree.Hash(val[2]),
		dastree.Hash(val[3]),
	}

	reqCtx := context.Background()
	timeout := uint64(time.Now().Add(time.Hour).Unix())
	for i := 0; i < 2; i++ {
		for j := 0; j < 2; j++ {
			err := syncFromStorageService[i].Put(reqCtx, val[j], timeout)
			Require(t, err)
		}
	}

	regularSyncStorage.Start(ctx)
	time.Sleep(300 * time.Millisecond)

	for i := 0; i < 2; i++ {
		for j := 2; j < 4; j++ {
			err := syncFromStorageService[i].Put(reqCtx, val[j], timeout)
			Require(t, err)
		}
	}

	time.Sleep(300 * time.Millisecond)

	for i := 0; i < 2; i++ {
		for j := 0; j < 4; j++ {
			v, err := syncToStorageService[i].GetByHash(reqCtx, valKey[j])
			Require(t, err)
			if !bytes.Equal(v, val[j]) {
				t.Fatal(v, val[j])
			}
		}
	}
}
