// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbos

import (
	"testing"

	"github.com/offchainlabs/nitro/util/testhelpers"
)

func Require(t *testing.T, err error, printables ...interface{}) {
	t.Helper()
	testhelpers.RequireImpl(t, err, printables...)
}

func Fail(t *testing.T, printables ...interface{}) {
	t.Helper()
	testhelpers.FailImpl(t, printables...)
}
