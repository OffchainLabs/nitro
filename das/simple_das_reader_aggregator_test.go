// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
)

func TestSimpleDASReaderAggregator(t *testing.T) { //nolint
	initTest(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	storage1, storage2, storage3 := NewLocalDiskStorageService(t.TempDir()), NewLocalDiskStorageService(t.TempDir()), NewLocalDiskStorageService(t.TempDir())

	data1 := []byte("Testing a restful server now.")
	dataHash1 := crypto.Keccak256(data1)

	server1, server2, server3 :=
		NewRestfulDasServerHTTP(LocalServerAddressForTest, 9888, storage1),
		NewRestfulDasServerHTTP(LocalServerAddressForTest, 9889, storage2),
		NewRestfulDasServerHTTP(LocalServerAddressForTest, 9890, storage3)

	err := storage1.Put(ctx, data1, uint64(time.Now().Add(time.Hour).Unix()))
	Require(t, err)
	err = storage2.Put(ctx, data1, uint64(time.Now().Add(time.Hour).Unix()))
	Require(t, err)
	err = storage3.Put(ctx, data1, uint64(time.Now().Add(time.Hour).Unix()))
	Require(t, err)

	time.Sleep(100 * time.Millisecond)

	config := RestfulClientAggregatorConfig{
		Urls:                   []string{"http://localhost:9888", "http://localhost:9889", "http://localhost:9890"},
		Strategy:               "testing-sequential",
		StrategyUpdateInterval: time.Second,
		WaitBeforeTryNext:      500 * time.Millisecond,
		MaxPerEndpointStats:    10,
	}

	agg, err := NewRestfulClientAggregator(&config)
	Require(t, err)

	returnedData, err := agg.GetByHash(ctx, dataHash1)
	Require(t, err)
	if !bytes.Equal(data1, returnedData) {
		Fail(t, fmt.Sprintf("Returned data '%s' does not match expected '%s'", returnedData, data1))
	}

	_, err = agg.GetByHash(ctx, crypto.Keccak256([]byte("absent data")))
	if err == nil || !strings.Contains(err.Error(), "404") {
		Fail(t, "Expected a 404 error")
	}

	data2 := []byte("Testing data that is only on the last REST endpoint.")
	dataHash2 := crypto.Keccak256(data2)

	err = storage3.Put(ctx, data2, uint64(time.Now().Add(time.Hour).Unix()))
	Require(t, err)

	returnedData, err = agg.GetByHash(ctx, dataHash2)
	Require(t, err)
	if !bytes.Equal(data2, returnedData) {
		Fail(t, fmt.Sprintf("Returned data '%s' does not match expected '%s'", returnedData, data2))
	}

	err = server1.Shutdown()
	Require(t, err)
	err = server2.Shutdown()
	Require(t, err)
	err = server3.Shutdown()
	Require(t, err)

}
