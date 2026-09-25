// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package httpclient

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/util/httperror"
)

const errorBodyLimit = 1024

func PostJSON(ctx context.Context, client *http.Client, url string, body []byte, beforeSend func(req *http.Request, body []byte)) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request to %s: %w", url, err)
	}
	req.Header.Set("Content-Type", "application/json")
	if beforeSend != nil {
		beforeSend(req, body)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("post to %s: %w", url, err)
	}
	defer drainAndClose(resp, url)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return readHTTPError(resp)
	}
	return nil
}

func Get(ctx context.Context, client *http.Client, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request to %s: %w", url, err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get %s: %w", url, err)
	}
	defer drainAndClose(resp, url)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, readHTTPError(resp)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response from %s: %w", url, err)
	}
	return body, nil
}

// drainAndClose lets the transport reuse the connection.
func drainAndClose(resp *http.Response, url string) {
	if _, err := io.Copy(io.Discard, resp.Body); err != nil {
		log.Warn("failed draining response body", "err", err, "url", url)
	}
	resp.Body.Close()
}

func readHTTPError(resp *http.Response) *httperror.HTTPError {
	respBody, readErr := io.ReadAll(io.LimitReader(resp.Body, errorBodyLimit)) // cap error body to avoid unbounded reads
	respBodyStr := string(respBody)
	if readErr != nil {
		log.Warn("Failed reading error response body", "err", readErr, "statusCode", resp.StatusCode)
		respBodyStr = fmt.Sprintf("%s (body read error: %s)", respBodyStr, readErr)
	}
	return &httperror.HTTPError{StatusCode: resp.StatusCode, Body: respBodyStr}
}
