// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package api

import (
	"context"
	"encoding/json"

	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/execution/gethexec/addressfilter"
)

func (a *FilteringReportAPI) ReportFilteredTransactions(ctx context.Context, reports []addressfilter.FilteredTxReport) error {
	log.Debug("Sending filtered transaction reports to SQS", "count", len(reports))
	for _, report := range reports {
		body, err := json.Marshal(report)
		if err != nil {
			return err
		}
		bodyStr := string(body)
		err = a.sqsClient.Send(ctx, bodyStr)
		if err != nil {
			log.Error("Failed to send filtered transaction report to SQS", "txHash", report.TxHash.Hex(), "err", err)
			return err
		}
		log.Debug("Successfully sent filtered transaction report to SQS", "txHash", report.TxHash.Hex())
	}
	return nil
}
