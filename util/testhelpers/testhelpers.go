//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package testhelpers

import (
	"testing"

	"github.com/offchainlabs/arbstate/util/colors"
)

// Fail a test should an error occur
func RequireImpl(t *testing.T, err error, text ...string) {
	t.Helper()
	if err != nil {
		t.Fatal(colors.Red, text, err, colors.Clear)
	}
}

func FailImpl(t *testing.T, printables ...interface{}) {
	t.Helper()
	t.Fatal(colors.Red, printables, colors.Clear)
}
