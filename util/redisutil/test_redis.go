// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package redisutil

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

// CreateTestRedis Provides external redis url, this is only done in TEST_REDIS env,
// else creates a new miniredis and returns its url.
func CreateTestRedis(ctx context.Context, t *testing.T) string {
	redisUrl := os.Getenv("TEST_REDIS")
	if redisUrl != "" {
		return redisUrl
	}
	redisServer, err := miniredis.Run()
	testhelpers.RequireImpl(t, err)
	go func() {
		<-ctx.Done()
		redisServer.Close()
	}()

	return fmt.Sprintf("redis://%s/0", redisServer.Addr())
}
