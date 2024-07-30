// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/offchainlabs/nitro/cmd/daprovider/das/dastree"
	"github.com/offchainlabs/nitro/cmd/genericconf"
)

const LocalServerAddressForTest = "localhost"

func NewRestfulDasServerOnRandomPort(address string, storageService StorageService) (*RestfulDasServer, int, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:0", address))
	if err != nil {
		return nil, 0, err
	}
	tcpAddr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		return nil, 0, errors.New("attempt to listen on TCP returned non-TCP address")
	}
	rds, err := NewRestfulDasServerOnListener(listener, genericconf.HTTPServerTimeoutConfigDefault, storageService, storageService)
	if err != nil {
		return nil, 0, err
	}
	return rds, tcpAddr.Port, nil
}

func TestRestfulClientServer(t *testing.T) {
	initTest(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	storage := NewMemoryBackedStorageService(ctx)
	data := []byte("Testing a restful server now.")
	dataHash := dastree.Hash(data)

	server, port, err := NewRestfulDasServerOnRandomPort(LocalServerAddressForTest, storage)
	Require(t, err)

	err = storage.Put(ctx, data, uint64(time.Now().Add(time.Hour).Unix()))
	Require(t, err)

	time.Sleep(100 * time.Millisecond)

	client := NewRestfulDasClient("http", LocalServerAddressForTest, port)
	returnedData, err := client.GetByHash(ctx, dataHash)
	Require(t, err)
	if !bytes.Equal(data, returnedData) {
		Fail(t, fmt.Sprintf("Returned data '%s' does not match expected '%s'", returnedData, data))
	}

	_, err = client.GetByHash(ctx, dastree.Hash([]byte("absent data")))
	if err == nil || !strings.Contains(err.Error(), "404") {
		Fail(t, "Expected a 404 error")
	}

	err = server.Shutdown()
	Require(t, err)
}
