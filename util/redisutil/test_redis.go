// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package redisutil

import (
	"fmt"
	"testing"

	"github.com/alicebob/miniredis/v2"

	"github.com/offchainlabs/nitro/util/testhelpers"
)

// t param is used to make sure this is only called in tests
func GetTestRedisURL(t *testing.T) string {
	redisServer, err := miniredis.Run()
	testhelpers.RequireImpl(t, err)

	return fmt.Sprintf("redis://%s/0", redisServer.Addr())

}
