//go:build race
// +build race

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
