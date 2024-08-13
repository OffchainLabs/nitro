// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

//go:build ipfs
// +build ipfs

package das

import (
	"bytes"
	"context"
	"math"
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/das/dastree"
)

func runAddAndGetTest(t *testing.T, ctx context.Context, svc *IpfsStorageService, size int) {

	data := make([]byte, size)
	_, err := rand.Read(data)
	Require(t, err)

	err = svc.Put(ctx, data, 0)
	Require(t, err)

	hash := dastree.Hash(data).Bytes()
	returnedData, err := svc.GetByHash(ctx, common.BytesToHash(hash))
	Require(t, err)
	if !bytes.Equal(data, returnedData) {
		Fail(t, "Returned data didn't match!")
	}

}

func TestIpfsStorageServiceAddAndGet(t *testing.T) {
	enableLogging()
	ctx := context.Background()
	svc, err := NewIpfsStorageService(ctx,
		IpfsStorageServiceConfig{
			Enable:      true,
			RepoDir:     t.TempDir(),
			ReadTimeout: time.Minute,
			Profiles:    "test",
		})
	defer svc.Close(ctx)
	Require(t, err)

	pow2Size := 1 << 16 // 64kB
	for i := 1; i < 8; i++ {
		runAddAndGetTest(t, ctx, svc, int(math.Pow10(i)))
		runAddAndGetTest(t, ctx, svc, pow2Size)
		runAddAndGetTest(t, ctx, svc, pow2Size-1)
		runAddAndGetTest(t, ctx, svc, pow2Size+1)
		pow2Size = pow2Size << 1
	}
}
