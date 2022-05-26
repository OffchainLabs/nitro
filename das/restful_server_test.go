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

const LocalServerAddressForTest = "localhost"
const LocalServerPortForTest = 9877

func TestRestfulClientServer(t *testing.T) { //nolint
	initTest(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	storage := NewMemoryBackedStorageService(ctx)
	data := []byte("Testing a restful server now.")
	dataHash := crypto.Keccak256(data)

	server := NewRestfulDasServerHTTP(LocalServerAddressForTest, LocalServerPortForTest, storage)

	err := storage.Put(ctx, data, uint64(time.Now().Add(time.Hour).Unix()))
	Require(t, err)

	time.Sleep(100 * time.Millisecond)

	client := NewRestfulDasClient("http", LocalServerAddressForTest, LocalServerPortForTest)
	returnedData, err := client.GetByHash(ctx, dataHash)
	Require(t, err)
	if !bytes.Equal(data, returnedData) {
		Fail(t, fmt.Sprintf("Returned data '%s' does not match expected '%s'", returnedData, data))
	}

	_, err = client.GetByHash(ctx, crypto.Keccak256([]byte("absent data")))
	if err == nil || !strings.Contains(err.Error(), "404") {
		Fail(t, "Expected a 404 error")
	}

	err = server.Shutdown()
	Require(t, err)
}
