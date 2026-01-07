// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/cmd/tx-filterer-manager/server"
	"github.com/offchainlabs/nitro/util/rpcclient"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

func TestTxFiltererManagerCmd(t *testing.T) {
	ctx := t.Context()

	listener, err := testhelpers.FreeTCPPortListener()
	Require(t, err)

	rpcServer, err := server.StartRPCServer(ctx, listener, genericconf.HTTPServerTimeoutConfigDefault)
	Require(t, err)
	defer func() {
		if rpcServer != nil {
			err = rpcServer.Shutdown(ctx)
			Require(t, err)
		}
	}()

	rpcClientConfigFetcher := func() *rpcclient.ClientConfig {
		config := rpcclient.DefaultClientConfig
		config.URL = "http://" + listener.Addr().String()
		return &config
	}
	rpcClient := rpcclient.NewRpcClient(rpcClientConfigFetcher, nil)
	err = rpcClient.Start(ctx)
	Require(t, err)
	txHash := common.Hash{}
	err = rpcClient.CallContext(ctx, nil, "txfilterermanager_filter", txHash)
	Require(t, err)
}
