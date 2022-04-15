// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"bytes"
	"context"
	"io/ioutil"
	"os"
	"testing"

	"github.com/offchainlabs/nitro/util/testhelpers"
)

func TestDASStoreRetrieveMultipleInstances(t *testing.T) {
	dbPath, err := ioutil.TempDir("/tmp", "das_test")
	defer os.RemoveAll(dbPath)

	Require(t, err)
	das, err := NewLocalDiskDataAvailabilityService(dbPath, 1)
	Require(t, err, "no das")

	ctx := context.Background()

	messageSaved := []byte("hello world")
	cert, err := das.Store(ctx, messageSaved)
	Require(t, err, "Error storing message")

	certBytes := Serialize(*cert)

	messageRetrieved, err := das.Retrieve(ctx, certBytes)
	Require(t, err, "Failed to retrieve message")
	if !bytes.Equal(messageSaved, messageRetrieved) {
		Fail(t, "Retrieved message is not the same as stored one.")
	}

	// 2nd das instance can read keys from disk
	das2, err := NewLocalDiskDataAvailabilityService(dbPath, 1)
	Require(t, err, "no das")

	messageRetrieved2, err := das2.Retrieve(ctx, certBytes)
	Require(t, err, "Failed to retrieve message")
	if !bytes.Equal(messageSaved, messageRetrieved2) {
		Fail(t, "Retrieved message is not the same as stored one.")
	}
}

func TestDASMissingMessage(t *testing.T) {
	dbPath, err := ioutil.TempDir("/tmp", "das_test")
	defer os.RemoveAll(dbPath)

	Require(t, err)
	das, err := NewLocalDiskDataAvailabilityService(dbPath, 1)
	Require(t, err, "no das")

	ctx := context.Background()

	messageSaved := []byte("hello world")
	cert, err := das.Store(ctx, messageSaved)
	Require(t, err, "Error storing message")

	// Change the hash to look up
	cert.DataHash[0] += 1

	certBytes := Serialize(*cert)

	_, err = das.Retrieve(ctx, certBytes)
	if err == nil {
		Fail(t, "Expected an error when retrieving message that is not in the store.")
	}
}

func Require(t *testing.T, err error, printables ...interface{}) {
	t.Helper()
	testhelpers.RequireImpl(t, err, printables...)
}

func Fail(t *testing.T, printables ...interface{}) {
	t.Helper()
	testhelpers.FailImpl(t, printables...)
}
