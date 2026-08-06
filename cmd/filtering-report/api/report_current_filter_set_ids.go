// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/metrics"

	"github.com/offchainlabs/nitro/execution/gethexec/addressfilter"
	"github.com/offchainlabs/nitro/util/httpclient"
)

var (
	filterSetIDsPostFailuresCounter = metrics.NewRegisteredCounter(
		"arb/filter_report/api/filter_set_ids_post_failure_total", nil,
	)
	filterSetIDsPostSuccessesCounter = metrics.NewRegisteredCounter(
		"arb/filter_report/api/filter_set_ids_post_success_total", nil,
	)
)

func (a *FilteringReportAPI) ReportCurrentFilterSetIDs(ctx context.Context, report *addressfilter.FilterSetIDsReport) error {
	if a.filterSetReporter == nil {
		return nil
	}
	if report == nil {
		return errors.New("nil filter-set ids report")
	}
	body, err := json.Marshal(report)
	if err != nil {
		return fmt.Errorf("marshal filter-set ids report: %w", err)
	}
	reporter := a.filterSetReporter
	if err := httpclient.PostJSON(ctx, reporter.httpClient, reporter.url, body, reporter.signer.SignHTTPRequest); err != nil {
		filterSetIDsPostFailuresCounter.Inc(1)
		return err
	}
	filterSetIDsPostSuccessesCounter.Inc(1)
	return nil
}
