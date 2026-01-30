// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package client

import (
	"context"

	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/cmd/transaction-filterer/api"
	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/util/rpcclient"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

type TransactionFiltererRPCClient struct {
	stopwaiter.StopWaiter
	client *rpcclient.RpcClient
}

func NewTransactionFiltererRPCClient(config rpcclient.ClientConfigFetcher) *TransactionFiltererRPCClient {
	return &TransactionFiltererRPCClient{
		client: rpcclient.NewRpcClient(config, nil),
	}
}

func (c *TransactionFiltererRPCClient) Start(ctx_in context.Context) error {
	c.StopWaiter.Start(ctx_in, c)
	ctx := c.GetContext()
	return c.client.Start(ctx)
}

func (c *TransactionFiltererRPCClient) Filter(txHashToFilter common.Hash) containers.PromiseInterface[common.Hash] {
	return stopwaiter.LaunchPromiseThread(c, func(ctx context.Context) (common.Hash, error) {
		var res common.Hash
		err := c.client.CallContext(ctx, &res, api.Namespace+"_filter", txHashToFilter)
		return res, err
	})
}
