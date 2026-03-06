// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package gethexec

import (
	"context"

	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/util/rpcclient"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

const TransactionFiltererNamespace = "transactionfilterer"

var DefaultTransactionFiltererRPCClientConfig = rpcclient.ClientConfig{
	URL:                       "",
	JWTSecret:                 "",
	Retries:                   3,
	RetryErrors:               "websocket: close.*|dial tcp .*|.*i/o timeout|.*connection reset by peer|.*connection refused",
	ArgLogLimit:               2048,
	WebsocketMessageSizeLimit: 256 * 1024 * 1024,
}

type TransactionFiltererRPCClient struct {
	stopwaiter.StopWaiter
	client *rpcclient.RpcClient
}

func NewTransactionFiltererRPCClient(config rpcclient.ClientConfigFetcher) *TransactionFiltererRPCClient {
	return &TransactionFiltererRPCClient{
		client: rpcclient.NewRpcClient(config, nil),
	}
}

func (c *TransactionFiltererRPCClient) Start(ctxIn context.Context) error {
	c.StopWaiter.Start(ctxIn, c)
	ctx := c.GetContext()
	return c.client.Start(ctx)
}

func (c *TransactionFiltererRPCClient) StopAndWait() {
	c.StopWaiter.StopAndWait()
	c.client.Close()
}

func (c *TransactionFiltererRPCClient) Filter(txHashToFilter common.Hash) containers.PromiseInterface[common.Hash] {
	return stopwaiter.LaunchPromiseThread(c, func(ctx context.Context) (common.Hash, error) {
		var res common.Hash
		err := c.client.CallContext(ctx, &res, TransactionFiltererNamespace+"_filter", txHashToFilter)
		return res, err
	})
}
