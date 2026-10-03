// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package prometheusmetrics

import "metrics"

var (
	_ = metrics.NewRegisteredCounter("arb/filter_report/client/failure_total", nil)
	_ = metrics.NewRegisteredGauge("arb/batchposter/wallet/eth", nil)

	_ = metrics.NewRegisteredCounter("arb/filtering-report/api/sqs_send_failures_total", nil) // want `metric "arb/filtering-report/api/sqs_send_failures_total" translates to invalid Prometheus name "arb_filtering-report_api_sqs_send_failures_total"`
	_ = metrics.NewRegisteredGauge("arb/transaction-filterer/queue_depth", nil)               // want `metric "arb/transaction-filterer/queue_depth" translates to invalid Prometheus name "arb_transaction-filterer_queue_depth"`
)
