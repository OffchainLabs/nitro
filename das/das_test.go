// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/offchainlabs/nitro/util/testhelpers"
)

func testDASStoreRetrieveMultipleInstances(t *testing.T, storageType string) {
	firstCtx, firstCancel := context.WithCancel(context.Background())

	dbPath := t.TempDir()
	_, _, err := GenerateAndStoreKeys(dbPath)
	Require(t, err)

	enableFileStorage, enableDbStorage := false, false
	switch storageType {
	case "db":
		enableDbStorage = true
	case "files":
		enableFileStorage = true
	default:
		Fail(t, "unknown storage type")
	}

	dbConfig := DefaultLocalDBStorageConfig
	dbConfig.Enable = enableDbStorage
	dbConfig.DataDir = dbPath

	config := DataAvailabilityConfig{
		Enable: true,
		Key: KeyConfig{
			KeyDir: dbPath,
		},
		LocalFileStorage: LocalFileStorageConfig{
			Enable:       enableFileStorage,
			DataDir:      dbPath,
			MaxRetention: DefaultLocalFileStorageConfig.MaxRetention,
		},
		LocalDBStorage:     dbConfig,
		ParentChainNodeURL: "none",
	}

	storageService, lifecycleManager, err := CreatePersistentStorageService(firstCtx, &config)
	Require(t, err)
	defer lifecycleManager.StopAndWaitUntil(time.Second)
	daWriter, err := NewSignAfterStoreDASWriter(firstCtx, config, storageService)
	Require(t, err, "no das")
	var daReader DataAvailabilityServiceReader = storageService

	// #nosec G115
	timeout := uint64(time.Now().Add(time.Hour * 24).Unix())
	messageSaved := []byte("hello world")
	cert, err := daWriter.Store(firstCtx, messageSaved, timeout)
	Require(t, err, "Error storing message")
	if cert.Timeout != timeout {
		Fail(t, fmt.Sprintf("Expected timeout of %d in cert, was %d", timeout, cert.Timeout))
	}

	messageRetrieved, err := daReader.GetByHash(firstCtx, cert.DataHash)
	Require(t, err, "Failed to retrieve message")
	if !bytes.Equal(messageSaved, messageRetrieved) {
		Fail(t, "Retrieved message is not the same as stored one.")
	}

	firstCancel()
	time.Sleep(500 * time.Millisecond)

	// 2nd das instance can read keys from disk
	secondCtx, secondCancel := context.WithCancel(context.Background())
	defer secondCancel()

	storageService2, lifecycleManager, err := CreatePersistentStorageService(secondCtx, &config)
	Require(t, err)
	defer lifecycleManager.StopAndWaitUntil(time.Second)
	var daReader2 DataAvailabilityServiceReader = storageService2

	messageRetrieved2, err := daReader2.GetByHash(secondCtx, cert.DataHash)
	Require(t, err, "Failed to retrieve message")
	if !bytes.Equal(messageSaved, messageRetrieved2) {
		Fail(t, "Retrieved message is not the same as stored one.")
	}

	messageRetrieved2, err = daReader2.GetByHash(secondCtx, cert.DataHash)
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

	dbPath := t.TempDir()
	_, _, err := GenerateAndStoreKeys(dbPath)
	Require(t, err)

	enableFileStorage, enableDbStorage := false, false
	switch storageType {
	case "db":
		enableDbStorage = true
	case "files":
		enableFileStorage = true
	default:
		Fail(t, "unknown storage type")
	}

	dbConfig := DefaultLocalDBStorageConfig
	dbConfig.Enable = enableDbStorage
	dbConfig.DataDir = dbPath

	config := DataAvailabilityConfig{
		Enable: true,
		Key: KeyConfig{
			KeyDir: dbPath,
		},
		LocalFileStorage: LocalFileStorageConfig{
			Enable:       enableFileStorage,
			DataDir:      dbPath,
			MaxRetention: DefaultLocalFileStorageConfig.MaxRetention,
		},
		LocalDBStorage:     dbConfig,
		ParentChainNodeURL: "none",
	}

	storageService, lifecycleManager, err := CreatePersistentStorageService(ctx, &config)
	Require(t, err)
	defer lifecycleManager.StopAndWaitUntil(time.Second)
	daWriter, err := NewSignAfterStoreDASWriter(ctx, config, storageService)
	Require(t, err, "no das")
	var daReader DataAvailabilityServiceReader = storageService

	messageSaved := []byte("hello world")
	// #nosec G115
	timeout := uint64(time.Now().Add(time.Hour * 24).Unix())
	cert, err := daWriter.Store(ctx, messageSaved, timeout)
	Require(t, err, "Error storing message")
	if cert.Timeout != timeout {
		Fail(t, fmt.Sprintf("Expected timeout of %d in cert, was %d", timeout, cert.Timeout))
	}

	// Change the hash to look up
	cert.DataHash[0] += 1

	_, err = daReader.GetByHash(ctx, cert.DataHash)
	if err == nil {
		Fail(t, "Expected an error when retrieving message that is not in the store.")
	}

	_, err = daReader.GetByHash(ctx, cert.DataHash)
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
