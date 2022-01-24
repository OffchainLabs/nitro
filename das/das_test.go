//
// Copyright 2022, Offchain Labs, Inc. All rights reserved.
//

package das

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"

	"github.com/offchainlabs/arbstate/util/testhelpers"
)

func TestConstruction(t *testing.T) {
	dbPath, err := ioutil.TempDir("/tmp", "das_test")
	defer os.RemoveAll(dbPath)

	Require(t, err)
	das, err := NewLocalDiskDataAvailabilityService(dbPath)
	Require(t, err, "no das")

	messageSaved := []byte("hello world")
	h, _, err := das.Store(messageSaved)
	Require(t, err, "Error storing message")

	messageRetrieved, err := das.Retrieve(h)
	if !bytes.Equal(messageSaved, messageRetrieved) {
		Fail(t, "Retrieved message is not the same as stored one.")
	}

	// 2nd das instance can read keys from disk
	das2, err := NewLocalDiskDataAvailabilityService(dbPath)
	Require(t, err, "no das")

	messageRetrieved2, err := das2.Retrieve(h)
	if !bytes.Equal(messageSaved, messageRetrieved2) {
		Fail(t, "Retrieved message is not the same as stored one.")
	}
}

func Require(t *testing.T, err error, text ...string) {
	t.Helper()
	testhelpers.RequireImpl(t, err, text...)
}

func Fail(t *testing.T, printables ...interface{}) {
	t.Helper()
	testhelpers.FailImpl(t, printables...)
}
