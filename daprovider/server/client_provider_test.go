// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package dapserver

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/daprovider/daclient"
	"github.com/offchainlabs/nitro/daprovider/das"
	"github.com/offchainlabs/nitro/daprovider/referenceda"
	"github.com/offchainlabs/nitro/util/rpcclient"
	"github.com/offchainlabs/nitro/util/signature"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

const RPCBodyLimit int = 1_000

func TestInteractionBetweenClientAndProviderServer_StoreSucceeds(t *testing.T) {
	ctx := context.Background()
	server := setupProviderServer(ctx, t)
	client := setupClient(ctx, t, server.Addr)

	message := testhelpers.RandomizeSlice(make([]byte, 10)) // fits into the body limit

	_, err := client.Store(ctx, message, 0, true)
	testhelpers.RequireImpl(t, err)
}

func TestInteractionBetweenClientAndProviderServer_StoreFailsDueToSize(t *testing.T) {
	ctx := context.Background()
	server := setupProviderServer(ctx, t)
	client := setupClient(ctx, t, server.Addr)

	message := testhelpers.RandomizeSlice(make([]byte, RPCBodyLimit+1))

	_, err := client.Store(ctx, message, 0, true)
	require.Regexp(t, ".*Request Entity Too Large.*", err.Error())
}

func setupProviderServer(ctx context.Context, t *testing.T) *http.Server {
	providerServerConfig := ServerConfig{
		Addr:               "localhost",
		Port:               0,
		EnableDAWriter:     true,
		ServerTimeouts:     genericconf.HTTPServerTimeoutConfig{},
		RPCServerBodyLimit: RPCBodyLimit,
	}

	privateKey, err := crypto.GenerateKey()
	testhelpers.RequireImpl(t, err)
	dataSigner := signature.DataSignerFromPrivateKey(privateKey)

	// The services below will work fine as long as we don't need to do any action on-chain.
	dummyAddress := common.HexToAddress("0x0")
	reader := referenceda.NewReader(nil, dummyAddress)
	writer := referenceda.NewWriter(dataSigner)
	validator := referenceda.NewValidator(nil, dummyAddress)

	signatureVerifier, err := das.NewSignatureVerifierWithSeqInboxCaller(nil, "")
	if err != nil {
		return nil
	}

	providerServer, err := NewServerWithDAPProvider(ctx, &providerServerConfig, reader, writer, validator, signatureVerifier)
	testhelpers.RequireImpl(t, err)

	return providerServer
}

func setupClient(ctx context.Context, t *testing.T, providerServerAddress string) *daclient.Client {
	clientConfig := func() *rpcclient.ClientConfig {
		return &rpcclient.ClientConfig{
			URL: providerServerAddress,
		}
	}

	privateKey, err := crypto.GenerateKey()
	testhelpers.RequireImpl(t, err)
	dataSigner := signature.DataSignerFromPrivateKey(privateKey)

	client, err := daclient.NewClient(ctx, clientConfig, RPCBodyLimit, dataSigner)
	testhelpers.RequireImpl(t, err)
	return client
}
