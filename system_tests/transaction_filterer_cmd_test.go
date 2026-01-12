// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/cmd/transaction-filterer/api"
	"github.com/offchainlabs/nitro/util/rpcclient"
)

func TestTransactionFiltererCmd(t *testing.T) {
	ctx := t.Context()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	cleanup := builder.Build(t)
	defer cleanup()

	transactionFiltererStackConf := api.DefaultStackConfig
	// use a random available port
	transactionFiltererStackConf.HTTPPort = 0
	transactionFiltererStackConf.WSPort = 0
	transactionFiltererStackConf.AuthPort = 0

	transactionFiltererStack, err := api.NewStack(ctx, &transactionFiltererStackConf, builder.L2.Client)
	Require(t, err)
	err = transactionFiltererStack.Start()
	Require(t, err)
	defer transactionFiltererStack.Close()

	transactionFiltererRPCClientConfigFetcher := func() *rpcclient.ClientConfig {
		config := rpcclient.DefaultClientConfig
		config.URL = transactionFiltererStack.HTTPEndpoint()
		return &config
	}
	transactionFiltererRPCClient := rpcclient.NewRpcClient(transactionFiltererRPCClientConfigFetcher, nil)
	err = transactionFiltererRPCClient.Start(ctx)
	Require(t, err)

	txHash := common.Hash{}
	err = transactionFiltererRPCClient.CallContext(ctx, nil, "transactionfilterer_filter", txHash)
	Require(t, err)
}
