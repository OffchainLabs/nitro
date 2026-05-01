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
)

// errorBodyLimit caps how much of a non-2xx response body we surface in errors,
// so a misbehaving upstream that returns megabytes of HTML doesn't dominate
// our logs.
const errorBodyLimit = 1024

// PostJSON marshals v as JSON and POSTs it to url with the given client.
// Non-2xx responses are surfaced as an error, including up to errorBodyLimit
// bytes of the response body.
func PostJSON(ctx context.Context, client *http.Client, url string, v any) error {
	body, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshal body: %w", err)
	}
	return PostJSONBody(ctx, client, url, body)
}

// PostJSONBody POSTs an already-serialized JSON payload. Callers that have a
// Go value should use PostJSON instead.
func PostJSONBody(ctx context.Context, client *http.Client, url string, body []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request to %s: %w", url, err)
	}
	req.Header.Set("Content-Type", "application/json")
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
		respBody, readErr := io.ReadAll(io.LimitReader(resp.Body, errorBodyLimit))
		if readErr != nil {
			return fmt.Errorf("post to %s returned status %d (body read error: %w)", url, resp.StatusCode, readErr)
		}
		return fmt.Errorf("post to %s returned status %d: %q", url, resp.StatusCode, respBody)
	}
	return nil
}
