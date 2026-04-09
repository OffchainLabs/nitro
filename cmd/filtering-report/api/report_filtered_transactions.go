// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package api

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/aws/aws-sdk-go-v2/service/sqs"

	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/execution/gethexec/addressfilter"
)

func (a *FilteringReportAPI) ReportFilteredTransactions(ctx context.Context, reports []addressfilter.FilteredTxReport) error {
	if a.sqsClient == nil {
		return errors.New("SQS client not configured")
	}
	log.Debug("Sending filtered transaction reports to SQS", "count", len(reports))
	for _, report := range reports {
		body, err := json.Marshal(report)
		if err != nil {
			return err
		}
		bodyStr := string(body)
		_, err = a.sqsClient.SendMessage(ctx, &sqs.SendMessageInput{
			QueueUrl:    &a.sqsClient.QueueURL,
			MessageBody: &bodyStr,
		})
		if err != nil {
			log.Error("Failed to send filtered transaction report to SQS", "txHash", report.TxHash.Hex(), "err", err)
			return err
		}
		log.Debug("Successfully sent filtered transaction report to SQS", "txHash", report.TxHash.Hex())
	}
	return nil
}
