// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

//go:build redistest
// +build redistest

package arbtest

import "testing"

func TestRedisBatchPosterParallel(t *testing.T) {
	TestBatchPosterParallel(t)
}
