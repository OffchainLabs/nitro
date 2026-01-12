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

	stackConf := api.DefaultStackConfig
	// use a random available port
	stackConf.HTTPPort = 0
	stackConf.WSPort = 0
	stackConf.AuthPort = 0

	stack, err := api.NewStack(&stackConf)
	Require(t, err)
	err = stack.Start()
	Require(t, err)
	defer stack.Close()

	rpcClientConfigFetcher := func() *rpcclient.ClientConfig {
		config := rpcclient.DefaultClientConfig
		config.URL = stack.HTTPEndpoint()
		return &config
	}
	rpcClient := rpcclient.NewRpcClient(rpcClientConfigFetcher, nil)
	err = rpcClient.Start(ctx)
	Require(t, err)
	txHash := common.Hash{}
	err = rpcClient.CallContext(ctx, nil, "transactionfilterer_filter", txHash)
	Require(t, err)
}
