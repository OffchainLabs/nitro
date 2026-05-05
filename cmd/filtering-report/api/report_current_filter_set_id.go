// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package api

import (
	"context"
	"errors"

	"github.com/offchainlabs/nitro/execution/gethexec/addressfilter"
	"github.com/offchainlabs/nitro/util/httpclient"
)

func (a *FilteringReportAPI) ReportCurrentFilterSetID(ctx context.Context, report *addressfilter.FilterSetIDReport) error {
	if a.filterSetReport == nil {
		return nil
	}
	if report == nil {
		return errors.New("nil filter-set id report")
	}
	return httpclient.PostJSON(ctx, a.filterSetReport.client, a.filterSetReport.url, report)
}
