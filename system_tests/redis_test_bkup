// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

//go:build redistest
// +build redistest

package arbtest

import (
	"os"
	"testing"

	"github.com/offchainlabs/nitro/arbnode"
)

func getTestRedisUrl() string {
	redisUrl := os.Getenv("TEST_REDIS")
	if redisUrl == "" {
		redisUrl = arbnode.TestSeqCoordinatorConfig.RedisUrl
	}
	return redisUrl
}

func TestRedisBatchPosterParallel(t *testing.T) {
	TestBatchPosterParallel(t)
}
