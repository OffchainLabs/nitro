// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/nitro/blob/master/LICENSE

package das

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/offchainlabs/nitro/das/dastree"
)

func TestSimpleDASReaderAggregator(t *testing.T) { //nolint
	initTest(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	storage1, storage2, storage3 := NewMemoryBackedStorageService(ctx), NewMemoryBackedStorageService(ctx), NewMemoryBackedStorageService(ctx)

	data1 := []byte("Testing a restful server now.")
	dataHash1 := dastree.Hash(data1)

	server1, port1, err := NewRestfulDasServerOnRandomPort(LocalServerAddressForTest, storage1)
	Require(t, err)
	server2, port2, err := NewRestfulDasServerOnRandomPort(LocalServerAddressForTest, storage2)
	Require(t, err)
	server3, port3, err := NewRestfulDasServerOnRandomPort(LocalServerAddressForTest, storage3)
	Require(t, err)

	err = storage1.Put(ctx, data1, uint64(time.Now().Add(time.Hour).Unix()))
	Require(t, err)
	err = storage2.Put(ctx, data1, uint64(time.Now().Add(time.Hour).Unix()))
	Require(t, err)
	err = storage3.Put(ctx, data1, uint64(time.Now().Add(time.Hour).Unix()))
	Require(t, err)

	time.Sleep(100 * time.Millisecond)

	config := RestfulClientAggregatorConfig{
		Urls:                   []string{"http://localhost:" + strconv.Itoa(port1), "http://localhost:" + strconv.Itoa(port2), "http://localhost:" + strconv.Itoa(port3)},
		Strategy:               "testing-sequential",
		StrategyUpdateInterval: time.Second,
		WaitBeforeTryNext:      500 * time.Millisecond,
		MaxPerEndpointStats:    10,
	}

	agg, err := NewRestfulClientAggregator(ctx, &config)
	Require(t, err)

	returnedData, err := agg.GetByHash(ctx, dataHash1)
	Require(t, err)
	if !bytes.Equal(data1, returnedData) {
		Fail(t, fmt.Sprintf("Returned data '%s' does not match expected '%s'", returnedData, data1))
	}

	_, err = agg.GetByHash(ctx, dastree.Hash([]byte("absent data")))
	if err == nil || !strings.Contains(err.Error(), "404") {
		Fail(t, "Expected a 404 error")
	}

	data2 := []byte("Testing data that is only on the last REST endpoint.")
	dataHash2 := dastree.Hash(data2)

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
