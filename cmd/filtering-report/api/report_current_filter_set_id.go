// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/execution/gethexec/addressfilter"
)

// errorBodyLimit caps how much of a non-2xx response body we surface in errors.
const errorBodyLimit = 1024

// ReportCurrentFilterSetId forwards the sequencer's current address-filter
// set id to the configured external HTTP endpoint. When no endpoint is
// configured the call is a no-op, which lets the RPC stay callable without
// blocking startup of callers that do not care about this feature.
func (a *FilteringReportAPI) ReportCurrentFilterSetId(ctx context.Context, report addressfilter.FilterSetIdReport) error {
	endpoint := a.filterSetReporting.URL
	if endpoint == "" {
		return nil
	}
	body, err := json.Marshal(report)
	if err != nil {
		return fmt.Errorf("marshal filter-set id report: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request to %s: %w", endpoint, err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("post to %s: %w", endpoint, err)
	}
	defer func() {
		if _, drainErr := io.Copy(io.Discard, resp.Body); drainErr != nil {
			log.Warn("failed draining filter-set id report response body", "err", drainErr)
		}
		resp.Body.Close()
	}()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, readErr := io.ReadAll(io.LimitReader(resp.Body, errorBodyLimit))
		if readErr != nil {
			return fmt.Errorf("post to %s returned status %d (body read error: %w)", endpoint, resp.StatusCode, readErr)
		}
		return fmt.Errorf("post to %s returned status %d: %q", endpoint, resp.StatusCode, respBody)
	}
	return nil
}
