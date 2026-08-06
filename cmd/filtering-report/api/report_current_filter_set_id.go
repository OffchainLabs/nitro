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
	filterSetIDPostFailuresCounter = metrics.NewRegisteredCounter(
		"arb/filter_report/api/filter_set_id_post_failure_total", nil,
	)
	filterSetIDPostSuccessesCounter = metrics.NewRegisteredCounter(
		"arb/filter_report/api/filter_set_id_post_success_total", nil,
	)
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
	if err := httpclient.PostJSON(ctx, reporter.httpClient, reporter.url, body, reporter.signer.SignHTTPRequest); err != nil {
		filterSetIDPostFailuresCounter.Inc(1)
		return err
	}
	filterSetIDPostSuccessesCounter.Inc(1)
	return nil
}
