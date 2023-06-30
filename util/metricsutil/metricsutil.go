// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/nitro/blob/master/LICENSE

package metricsutil

import (
	"regexp"
)

// Prometheus metric names must contain only chars [a-zA-Z0-9:_]
func CanonicalizeMetricName(metric string) string {
	invalidPromCharRegex := regexp.MustCompile(`[^a-zA-Z0-9:_]+`)
	return invalidPromCharRegex.ReplaceAllString(metric, "_")

}
