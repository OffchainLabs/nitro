// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package redisutil

import (
	"context"
	"fmt"
	"testing"

	"github.com/alicebob/miniredis/v2"

	"github.com/offchainlabs/nitro/util/testhelpers"
)

// CreateTestRedis Provides external redis url, this is only done with -test_redis flag,
// else creates a new miniredis and returns its url.
func CreateTestRedis(ctx context.Context, t testing.TB) string {
	testhelpers.ParseFlag()
	if *testhelpers.RedisFlag != "" {
		return *testhelpers.RedisFlag
	}
	redisServer, err := miniredis.Run()
	testhelpers.RequireImpl(t, err)
	go func() {
		<-ctx.Done()
		redisServer.Close()
	}()

	return fmt.Sprintf("redis://%s/0", redisServer.Addr())
}
