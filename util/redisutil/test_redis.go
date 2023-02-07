// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

//go:build redistest
// +build redistest

package redisutil

import (
	"os"
	"testing"
)

// t param is used to make sure this is only called in tests
func GetTestRedisURL(t *testing.T) string {
	redisUrl := os.Getenv("TEST_REDIS")
	if redisUrl == "" {
		redisUrl = DefaultTestRedisURL
	}
	return redisUrl
}
