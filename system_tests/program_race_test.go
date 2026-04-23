// Copyright 2024-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
//go:build race

// when running with race detection - skip block validation

package arbtest

import (
	"testing"
)

// used in program test
func validateBlocks(
	t *testing.T, start uint64, jit bool, builder *NodeBuilder,
) {
}

// used in program test
func validateBlockRange(
	t *testing.T, blocks []uint64, jit bool,
	builder *NodeBuilder,
) {
}
