// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

//go:build !redistest
// +build !redistest

package redisutil

import (
	"context"
	"fmt"
	"github.com/alicebob/miniredis/v2"
	"github.com/offchainlabs/nitro/util/testhelpers"
	"testing"
)

// CreateTestRedis Creates a new miniredis and returns its url.
// t param is used to make sure this is only called in tests
func CreateTestRedis(ctx context.Context, t *testing.T) string {
	redisServer, err := miniredis.Run()
	testhelpers.RequireImpl(t, err)
	go func() {
		<-ctx.Done()
		redisServer.Close()
	}()

	return fmt.Sprintf("redis://%s/0", redisServer.Addr())
}
