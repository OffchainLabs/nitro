// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package gethexec

import (
	"context"
	"encoding/json"

	"github.com/ethereum/go-ethereum/arbitrum/filter"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"

	"github.com/offchainlabs/nitro/execution/gethexec/addressfilter"
	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/util/rpcclient"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

const FilteringReportNamespace = "filteringreport"

const maxLoggedReportBytesLen = 256

type ReportProducer string

const (
	ReportProducerPrechecker ReportProducer = "prechecker"
	ReportProducerSequencer  ReportProducer = "sequencer"
)

var (
	reportFilteredTransactionsCallFailuresCounter = metrics.NewRegisteredCounter(
		"arb/filter_report/client/failure_total", nil,
	)
	reportFilteredTransactionsCallSuccessesCounter = metrics.NewRegisteredCounter(
		"arb/filter_report/client/success_total", nil,
	)
)

var DefaultFilteringReportRPCClientConfig = rpcclient.ClientConfig{
	URL:                       "",
	JWTSecret:                 "",
	Retries:                   3,
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

func (c *FilteringReportRPCClient) ReportFilteredTransactions(producer ReportProducer, reports []addressfilter.FilteredTxReport) containers.PromiseInterface[struct{}] {
	return stopwaiter.LaunchPromiseThread(c, func(ctx context.Context) (struct{}, error) {
		for i := range reports {
			logged := reportForLog(&reports[i])
			reportJSON, err := json.Marshal(logged)
			if err != nil {
				log.Info("filtered tx report", "producer", producer, "report", logged, "jsonMarshalErr", err)
				continue
			}
			log.Info("filtered tx report", "producer", producer, "report", string(reportJSON))
		}
		err := c.client.CallContext(ctx, nil, FilteringReportNamespace+"_reportFilteredTransactions", reports)
		if err != nil {
			reportFilteredTransactionsCallFailuresCounter.Inc(1)
		} else {
			reportFilteredTransactionsCallSuccessesCounter.Inc(1)
		}
		return struct{}{}, err
	})
}

// reportForLog returns a copy of the report with its unbounded byte fields (the
// raw transaction and event log data) truncated so log entries stay compact.
func reportForLog(report *addressfilter.FilteredTxReport) addressfilter.FilteredTxReport {
	logged := *report
	logged.TxRLP = truncateLoggedBytes(logged.TxRLP)
	if len(report.FilteredAddresses) > 0 {
		addresses := make([]filter.FilteredAddressRecord, len(report.FilteredAddresses))
		copy(addresses, report.FilteredAddresses)
		for i := range addresses {
			if addresses[i].EventRuleMatch == nil || addresses[i].EventRuleMatch.RawLog == nil {
				continue
			}
			rawLog := *addresses[i].EventRuleMatch.RawLog
			rawLog.Data = truncateLoggedBytes(rawLog.Data)
			eventRuleMatch := *addresses[i].EventRuleMatch
			eventRuleMatch.RawLog = &rawLog
			addresses[i].EventRuleMatch = &eventRuleMatch
		}
		logged.FilteredAddresses = addresses
	}
	return logged
}

func truncateLoggedBytes(b hexutil.Bytes) hexutil.Bytes {
	if len(b) > maxLoggedReportBytesLen {
		return b[:maxLoggedReportBytesLen]
	}
	return b
}
