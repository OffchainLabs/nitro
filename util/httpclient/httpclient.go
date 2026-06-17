// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/util/httperror"
)

const errorBodyLimit = 1024

func PostJSON(ctx context.Context, client *http.Client, url string, v any, beforeSend func(req *http.Request, body []byte)) error {
	body, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshal body: %w", err)
	}
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
	defer func() {
		if _, drainErr := io.Copy(io.Discard, resp.Body); drainErr != nil {
			log.Warn("failed draining response body", "err", drainErr, "url", url)
		}
		resp.Body.Close()
	}()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, readErr := io.ReadAll(io.LimitReader(resp.Body, errorBodyLimit)) // cap error body to avoid unbounded reads
		respBodyStr := string(respBody)
		if readErr != nil {
			log.Warn("Failed reading error response body", "err", readErr, "statusCode", resp.StatusCode)
			respBodyStr = fmt.Sprintf("%s (body read error: %s)", respBodyStr, readErr)
		}
		return &httperror.HTTPError{StatusCode: resp.StatusCode, Body: respBodyStr}
	}
	return nil
}
