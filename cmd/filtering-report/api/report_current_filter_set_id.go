// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package api

import (
	"context"
	"errors"
	"net/http"
	"time"

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
	reporter := a.filterSetReporter
	return httpclient.PostJSON(ctx, reporter.client, reporter.url, report,
		func(req *http.Request, signedBody []byte) {
			reporter.signer.SignHTTPRequest(req, signedBody, time.Now())
		})
}
