// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package redisutil

import (
	"context"
	"fmt"
	"testing"

	"github.com/alicebob/miniredis/v2"

	"github.com/offchainlabs/nitro/util/testhelpers"
	testflag "github.com/offchainlabs/nitro/util/testhelpers/flag"
)

// CreateTestRedis Provides external redis url, this is only done with --test_redis flag,
// else creates a new miniredis and returns its url.
func CreateTestRedis(ctx context.Context, t testing.TB) string {
	_, url := CreateTestRedisAdvanced(ctx, t)
	return url
}

// CreateTestRedisAdvanced returns either (nil, test_redis url) or (miniredis, local url)
func CreateTestRedisAdvanced(ctx context.Context, t testing.TB) (*miniredis.Miniredis, string) {
	if *testflag.RedisFlag != "" {
		return nil, *testflag.RedisFlag
	}
	redisServer, err := miniredis.Run()
	testhelpers.RequireImpl(t, err)
	go func() {
		<-ctx.Done()
		redisServer.Close()
	}()
	return redisServer, fmt.Sprintf("redis://%s/0", redisServer.Addr())
}
