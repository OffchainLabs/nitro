// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package dapserver

import (
	"context"
	"net/http"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/daprovider/daclient"
	"github.com/offchainlabs/nitro/daprovider/data_streaming"
	"github.com/offchainlabs/nitro/daprovider/referenceda"
	"github.com/offchainlabs/nitro/util/signature"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

func TestInteractionBetweenClientAndProviderServer_SimpleStoreSucceeds(t *testing.T) {
	ctx := context.Background()
	server := setupProviderServer(ctx, t)
	client := setupClient(ctx, t, server.Addr, false)

	message := testhelpers.RandomizeSlice(make([]byte, 10)) // fits into the body limit

	_, err := client.Store(message, 0).Await(ctx)
	testhelpers.RequireImpl(t, err)
}

func TestInteractionBetweenClientAndProviderServer_StreamingStoreSucceeds(t *testing.T) {
	ctx := context.Background()
	server := setupProviderServer(ctx, t)
	client := setupClient(ctx, t, server.Addr, true)

	message := testhelpers.RandomizeSlice(make([]byte, 10)) // fits into the body limit

	_, err := client.Store(message, 0).Await(ctx)
	testhelpers.RequireImpl(t, err)
}

func TestInteractionBetweenClientAndProviderServer_StoreLongMessageSucceeds(t *testing.T) {
	ctx := context.Background()
	server := setupProviderServer(ctx, t)
	client := setupClient(ctx, t, server.Addr, true)

	message := testhelpers.RandomizeSlice(make([]byte, data_streaming.TestHttpBodyLimit+1))

	_, err := client.Store(message, 0).Await(ctx)
	testhelpers.RequireImpl(t, err)
}

func setupProviderServer(ctx context.Context, t *testing.T) *http.Server {
	providerServerConfig := ServerConfig{
		Addr:               "localhost",
		Port:               0,
		EnableDAWriter:     true,
		ServerTimeouts:     genericconf.HTTPServerTimeoutConfig{},
		RPCServerBodyLimit: data_streaming.TestHttpBodyLimit,
		JWTSecret:          "",
	}

	privateKey, err := crypto.GenerateKey()
	testhelpers.RequireImpl(t, err)
	dataSigner := signature.DataSignerFromPrivateKey(privateKey)

	// The services below will work fine as long as we don't need to do any action on-chain.
	dummyAddress := common.HexToAddress("0x0")
	storage := referenceda.GetInMemoryStorage()
	reader := referenceda.NewReader(storage, nil, dummyAddress)
	writer := referenceda.NewWriter(dataSigner, referenceda.DefaultConfig.MaxBatchSize)
	validator := referenceda.NewValidator(nil, dummyAddress)
	headerBytes := []byte{daprovider.DACertificateMessageHeaderFlag}

	providerServer, err := NewServerWithDAPProvider(ctx, &providerServerConfig, reader, writer, validator, headerBytes, data_streaming.PayloadCommitmentVerifier())
	testhelpers.RequireImpl(t, err)

	return providerServer
}

func setupClient(ctx context.Context, t *testing.T, providerServerAddress string, useDataStream bool) *daclient.Client {
	clientConfig := daclient.TestClientConfig(providerServerAddress)
	clientConfig.UseDataStreaming = useDataStream

	client, err := daclient.NewClient(ctx, clientConfig, data_streaming.PayloadCommiter())
	testhelpers.RequireImpl(t, err)

	return client
}
