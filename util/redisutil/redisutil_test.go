// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package redisutil

import (
	"reflect"
	"testing"
)

func TestParseFailoverRedisUrlDefaultsPortPerAddress(t *testing.T) {
	options, err := parseFailoverRedisUrl("redis+sentinel://sentinel-1,sentinel-2:26379/master")
	if err != nil {
		t.Fatal(err)
	}

	want := []string{"sentinel-1:6379", "sentinel-2:26379"}
	if !reflect.DeepEqual(options.SentinelAddrs, want) {
		t.Fatalf("unexpected sentinel addrs: have %v, want %v", options.SentinelAddrs, want)
	}
}
