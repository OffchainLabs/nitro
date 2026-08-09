// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package httperror

import (
	"fmt"
	"net/http"
)

type HTTPError struct {
	StatusCode int
	Body       string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Body)
}

func (e *HTTPError) IsRetryable() bool {
	if e.StatusCode >= 500 {
		return true
	}
	return e.StatusCode == http.StatusRequestTimeout ||
		e.StatusCode == http.StatusTooEarly ||
		e.StatusCode == http.StatusTooManyRequests
}
