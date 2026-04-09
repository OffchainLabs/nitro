// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package gethexec

import (
	"context"

	"github.com/offchainlabs/nitro/execution/gethexec/addressfilter"
	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/util/rpcclient"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

const FilteringReportNamespace = "filteringreport"

var DefaultFilteringReportRPCClientConfig = func() rpcclient.ClientConfig {
	cfg := rpcclient.DefaultClientConfig
	cfg.URL = ""
	return cfg
}()

type FilteringReportRPCClient struct {
	stopwaiter.StopWaiter
	client *rpcclient.RpcClient
}

func NewFilteringReportRPCClient(config rpcclient.ClientConfigFetcher) *FilteringReportRPCClient {
	return &FilteringReportRPCClient{
		client: rpcclient.NewRpcClient(config, nil),
	}
}

func (c *FilteringReportRPCClient) Start(ctxIn context.Context) error {
	c.StopWaiter.Start(ctxIn, c)
	ctx := c.GetContext()
	return c.client.Start(ctx)
}

func (c *FilteringReportRPCClient) StopAndWait() {
	c.StopWaiter.StopAndWait()
	c.client.Close()
}

func (c *FilteringReportRPCClient) ReportFilteredTransactions(reports []addressfilter.FilteredTxReport) containers.PromiseInterface[struct{}] {
	return stopwaiter.LaunchPromiseThread(c, func(ctx context.Context) (struct{}, error) {
		err := c.client.CallContext(ctx, nil, FilteringReportNamespace+"_reportFilteredTransactions", reports)
		return struct{}{}, err
	})
}
