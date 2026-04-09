// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package api

import (
	"context"

	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/execution/gethexec"
)

// ReportFilteredTransactions logs each filtered transaction report to stdout.
func (a *FilteringReportAPI) ReportFilteredTransactions(_ context.Context, reports []gethexec.FilteredTxReport) error {
	for _, report := range reports {
		log.Info("Filtered transaction report",
			"id", report.Id,
			"txHash", report.TxHash,
			"blockNumber", report.BlockNumber,
			"positionInBlock", report.PositionInBlock,
			"isDelayed", report.IsDelayed,
			"chainId", report.ChainId,
			"filteredAddressCount", len(report.FilteredAddresses),
			"filteredAt", report.FilteredAt,
		)
		for _, addr := range report.FilteredAddresses {
			log.Info("Filtered address",
				"reportId", report.Id,
				"address", addr.Address,
				"reason", addr.Reason,
			)
		}
	}
	return nil
}
