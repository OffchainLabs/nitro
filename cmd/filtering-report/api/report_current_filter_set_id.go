// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/offchainlabs/nitro/execution/gethexec/addressfilter"
	"github.com/offchainlabs/nitro/util/httpclient"
)

func (a *FilteringReportAPI) ReportCurrentFilterSetID(ctx context.Context, report *addressfilter.FilterSetIDReport) error {
	if a.filterSetReporter == nil {
		return nil
	}
	if report == nil {
		return errors.New("nil filter-set id report")
	}
	body, err := json.Marshal(report)
	if err != nil {
		return fmt.Errorf("marshal filter-set id report: %w", err)
	}
	reporter := a.filterSetReporter
	return httpclient.PostJSON(ctx, reporter.httpClient, reporter.url, body, reporter.signer.SignHTTPRequest)
}
