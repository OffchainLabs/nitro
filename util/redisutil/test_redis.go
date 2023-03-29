// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

//go:build redistest
// +build redistest

package redisutil

import (
	"context"
	"os"
	"testing"
)

// CreateTestRedis Provides external redis url, this is only used when redistest tag is added.
// t param is used to make sure this is only called in tests
func CreateTestRedis(ctx context.Context, t *testing.T) string {
	redisUrl := os.Getenv("TEST_REDIS")
	if redisUrl == "" {
		redisUrl = DefaultTestRedisURL
	}
	return redisUrl
}
