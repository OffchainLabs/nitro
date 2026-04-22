// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package api

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/execution/gethexec/addressfilter"
)

// ReportFilteredTransactions enqueues each report to SQS. All reports are
// attempted even if some fail. SQS provides at-least-once delivery, so
// downstream consumers must deduplicate using FilteredTxReport.ID.
func (a *FilteringReportAPI) ReportFilteredTransactions(ctx context.Context, reports []addressfilter.FilteredTxReport) error {
	log.Debug("Sending filtered transaction reports to SQS", "count", len(reports))
	var failures []string
	for i, report := range reports {
		body, err := json.Marshal(report)
		if err != nil {
			failures = append(failures, fmt.Sprintf("report %d (id=%s, txHash=%s): marshal error: %v", i, report.ID, report.TxHash.Hex(), err))
			continue
		}
		err = a.queueClient.Send(ctx, string(body))
		if err != nil {
			log.Error("Failed to send filtered transaction report to SQS", "txHash", report.TxHash.Hex(), "err", err)
			failures = append(failures, fmt.Sprintf("report %d (id=%s, txHash=%s): %v", i, report.ID, report.TxHash.Hex(), err))
			continue
		}
		log.Debug("Successfully sent filtered transaction report to SQS", "txHash", report.TxHash.Hex())
	}
	if len(failures) > 0 {
		return fmt.Errorf("%d/%d reports failed to send: %s", len(failures), len(reports), strings.Join(failures, "; "))
	}
	return nil
}
