// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package api

import (
	"context"
	"errors"

	"github.com/offchainlabs/nitro/execution/gethexec/addressfilter"
	"github.com/offchainlabs/nitro/util/httpclient"
)

// ReportCurrentFilterSetID forwards the sequencer's current address-filter
// set ID to the configured external HTTP endpoint. When no endpoint is
// configured the call is a no-op, which lets the RPC stay callable without
// blocking startup of callers that do not care about this feature.
func (a *FilteringReportAPI) ReportCurrentFilterSetID(ctx context.Context, report *addressfilter.FilterSetIDReport) error {
	if a.filterSetReport == nil {
		return nil
	}
	if report == nil {
		return errors.New("nil filter-set id report")
	}
	return httpclient.PostJSON(ctx, a.filterSetReport.client, a.filterSetReport.url, report)
}
