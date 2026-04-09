// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package api

import (
	"context"

	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/execution/gethexec/addressfilter"
)

// ReportFilteredTransactions logs each filtered transaction report to stdout.
func (a *FilteringReportAPI) ReportFilteredTransactions(_ context.Context, reports []addressfilter.FilteredTxReport) error {
	for _, report := range reports {
		log.Info("Filtered transaction report",
			"id", report.ID,
			"txHash", report.TxHash,
			"blockNumber", report.BlockNumber,
			"positionInBlock", report.PositionInBlock,
			"isDelayed", report.IsDelayed,
			"filteredAddressCount", len(report.FilteredAddresses),
			"filteredAt", report.FilteredAt,
		)
		for _, addr := range report.FilteredAddresses {
			log.Info("Filtered address",
				"reportId", report.ID,
				"address", addr.Address,
				"reason", addr.Reason,
			)
		}
	}
	return nil
}
