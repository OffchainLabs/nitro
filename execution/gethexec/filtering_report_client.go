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

// DefaultFilteringReportRPCClientConfig keeps Retries=0 by default. The
// filter-set-id report is rescheduled on each periodic tick, so transport-
// level retries would only risk duplicate posts. Filtered-tx reports run
// through this same client and accept the same trade-off — a network blip
// drops a single report, which is preferable to the duplicate-delivery risk
// retries would create downstream.
var DefaultFilteringReportRPCClientConfig = rpcclient.ClientConfig{
	URL:                       "",
	JWTSecret:                 "",
	Retries:                   0,
	RetryErrors:               "websocket: close.*|dial tcp .*|.*i/o timeout|.*connection reset by peer|.*connection refused",
	ArgLogLimit:               2048,
	WebsocketMessageSizeLimit: 256 * 1024 * 1024,
}

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

func (c *FilteringReportRPCClient) ReportCurrentFilterSetID(report *addressfilter.FilterSetIDReport) containers.PromiseInterface[struct{}] {
	return stopwaiter.LaunchPromiseThread(c, func(ctx context.Context) (struct{}, error) {
		err := c.client.CallContext(ctx, nil, FilteringReportNamespace+"_reportCurrentFilterSetID", report)
		return struct{}{}, err
	})
}
