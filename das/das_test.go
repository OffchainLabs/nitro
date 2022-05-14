// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/offchainlabs/nitro/util/testhelpers"
)

func testDASStoreRetrieveMultipleInstances(t *testing.T, storageType string) {
	firstCtx, firstCancel := context.WithCancel(context.Background())

	dbPath, err := ioutil.TempDir("/tmp", "das_test")
	defer os.RemoveAll(dbPath)
	Require(t, err)

	config := LocalDiskDASConfig{
		KeyDir:            dbPath,
		DataDir:           dbPath,
		AllowGenerateKeys: true,
		L1NodeURL:         "none",
		StorageType:       storageType,
	}
	das, err := NewLocalDiskDAS(firstCtx, config)
	Require(t, err, "no das")

	timeout := uint64(time.Now().Add(time.Hour * 24).Unix())
	messageSaved := []byte("hello world")
	cert, err := das.Store(firstCtx, messageSaved, timeout, []byte{})
	Require(t, err, "Error storing message")
	if cert.Timeout != timeout {
		Fail(t, fmt.Sprintf("Expected timeout of %d in cert, was %d", timeout, cert.Timeout))
	}

	messageRetrieved, err := das.Retrieve(firstCtx, cert)
	Require(t, err, "Failed to retrieve message")
	if !bytes.Equal(messageSaved, messageRetrieved) {
		Fail(t, "Retrieved message is not the same as stored one.")
	}

	messageRetrieved, err = das.GetByHash(firstCtx, cert.DataHash[:])
	Require(t, err, "Failed to getByHash message")
	if !bytes.Equal(messageSaved, messageRetrieved) {
		Fail(t, "Retrieved message is not the same as stored one.")
	}

	firstCancel()
	time.Sleep(100 * time.Millisecond)

	// 2nd das instance can read keys from disk
	secondCtx, secondCancel := context.WithCancel(context.Background())
	defer secondCancel()

	das2, err := NewLocalDiskDAS(secondCtx, config)
	Require(t, err, "no das")

	messageRetrieved2, err := das2.Retrieve(secondCtx, cert)
	Require(t, err, "Failed to retrieve message")
	if !bytes.Equal(messageSaved, messageRetrieved2) {
		Fail(t, "Retrieved message is not the same as stored one.")
	}

	messageRetrieved2, err = das2.GetByHash(secondCtx, cert.DataHash[:])
	Require(t, err, "Failed to getByHash message")
	if !bytes.Equal(messageSaved, messageRetrieved2) {
		Fail(t, "Retrieved message is not the same as stored one.")
	}
}

func TestDASStoreRetrieveMultipleInstancesFiles(t *testing.T) {
	testDASStoreRetrieveMultipleInstances(t, "files")
}

func TestDASStoreRetrieveMultipleInstancesDB(t *testing.T) {
	testDASStoreRetrieveMultipleInstances(t, "db")
}

func testDASMissingMessage(t *testing.T, storageType string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dbPath, err := ioutil.TempDir("/tmp", "das_test")
	defer os.RemoveAll(dbPath)
	Require(t, err)

	config := LocalDiskDASConfig{
		KeyDir:            dbPath,
		DataDir:           dbPath,
		AllowGenerateKeys: true,
		L1NodeURL:         "none",
		StorageType:       storageType,
	}
	das, err := NewLocalDiskDAS(ctx, config)
	Require(t, err, "no das")

	messageSaved := []byte("hello world")
	timeout := uint64(time.Now().Add(time.Hour * 24).Unix())
	cert, err := das.Store(ctx, messageSaved, timeout, []byte{})
	Require(t, err, "Error storing message")
	if cert.Timeout != timeout {
		Fail(t, fmt.Sprintf("Expected timeout of %d in cert, was %d", timeout, cert.Timeout))
	}

	// Change the hash to look up
	cert.DataHash[0] += 1

	_, err = das.Retrieve(ctx, cert)
	if err == nil {
		Fail(t, "Expected an error when retrieving message that is not in the store.")
	}

	_, err = das.GetByHash(ctx, cert.DataHash[:])
	if err == nil {
		Fail(t, "Expected an error when getting by hash a message that is not in the store.")
	}
}

func TestDASMissingMessageFiles(t *testing.T) {
	testDASMissingMessage(t, "files")
}

func TestDASMissingMessageDB(t *testing.T) {
	testDASMissingMessage(t, "db")
}

func Require(t *testing.T, err error, printables ...interface{}) {
	t.Helper()
	testhelpers.RequireImpl(t, err, printables...)
}

func Fail(t *testing.T, printables ...interface{}) {
	t.Helper()
	testhelpers.FailImpl(t, printables...)
}
