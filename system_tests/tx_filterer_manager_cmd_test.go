// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/cmd/tx-filterer-manager/api"
	"github.com/offchainlabs/nitro/util/rpcclient"
)

func TestTxFiltererManagerCmd(t *testing.T) {
	ctx := t.Context()

	stackConf := node.Config{
		DataDir:             node.DefaultDataDir(),
		HTTPPort:            node.DefaultHTTPPort,
		AuthAddr:            node.DefaultAuthHost,
		AuthPort:            node.DefaultAuthPort,
		AuthVirtualHosts:    node.DefaultAuthVhosts,
		HTTPModules:         []string{"txfilterermanager"},
		HTTPHost:            "localhost",
		HTTPVirtualHosts:    []string{"localhost"},
		HTTPTimeouts:        rpc.DefaultHTTPTimeouts,
		WSHost:              "localhost",
		WSPort:              node.DefaultWSPort,
		WSModules:           []string{"txfilterermanager"},
		GraphQLVirtualHosts: []string{"localhost"},
		P2P: p2p.Config{
			ListenAddr:  "",
			NoDiscovery: true,
			NoDial:      true,
		},
	}
	stack, err := node.New(&stackConf)
	Require(t, err)
	api.RegisterAPI(stack)
	err = stack.Start()
	Require(t, err)

	rpcClientConfigFetcher := func() *rpcclient.ClientConfig {
		config := rpcclient.DefaultClientConfig
		config.URL = stack.HTTPEndpoint()
		return &config
	}
	rpcClient := rpcclient.NewRpcClient(rpcClientConfigFetcher, nil)
	err = rpcClient.Start(ctx)
	Require(t, err)
	txHash := common.Hash{}
	err = rpcClient.CallContext(ctx, nil, "txfilterermanager_filter", txHash)
	Require(t, err)
}
