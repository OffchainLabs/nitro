// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"bytes"
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestIpfsStorageServiceAddAndGet(t *testing.T) {
	enableLogging()
	ctx := context.Background()
	svc, err := NewIpfsStorageService(ctx, "/tmp/ipfstest", "test")
	Require(t, err)
	data := []byte("hello world")

	err = svc.Put(ctx, data, 0)
	Require(t, err)

	hash := crypto.Keccak256(data)
	returnedData, err := svc.GetByHash(ctx, common.BytesToHash(hash))
	Require(t, err)
	if bytes.Compare(data, returnedData) != 0 {
		Fail(t, "Returned data didn't match!")
	}
}
