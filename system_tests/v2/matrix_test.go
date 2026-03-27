// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package v2

import (
	"testing"
)

// TestMatrixExpansion validates the matrix expansion logic without spinning up nodes.
func TestMatrixExpansion(t *testing.T) {
	t.Run("no_cli_flags_single_run", func(t *testing.T) {
		spec := &BuilderSpec{
			Weight:         WeightLight,
			Parallelizable: true,
			Schemes:        []string{"hash", "path"},
		}
		dims := matrixDimensions{} // no CLI flags
		results := expandSpec(spec, dims)
		if len(results) != 1 {
			t.Fatalf("expected 1 run (default), got %d", len(results))
		}
		// With no CLI flags, resolved scheme is empty → builder uses its own default.
		if results[0].resolvedScheme != "" {
			t.Fatalf("expected empty resolved scheme (use builder default), got %q", results[0].resolvedScheme)
		}
	})

	t.Run("cli_requests_scheme_test_supports_both", func(t *testing.T) {
		spec := &BuilderSpec{
			Weight:         WeightLight,
			Parallelizable: true,
			Schemes:        []string{"hash", "path"},
		}
		dims := matrixDimensions{schemes: []string{"hash", "path"}}
		results := expandSpec(spec, dims)
		if len(results) != 2 {
			t.Fatalf("expected 2 runs, got %d", len(results))
		}
		if results[0].resolvedScheme != "hash" || results[1].resolvedScheme != "path" {
			t.Fatalf("unexpected schemes: %q, %q", results[0].resolvedScheme, results[1].resolvedScheme)
		}
	})

	t.Run("cli_requests_scheme_test_supports_one", func(t *testing.T) {
		spec := &BuilderSpec{
			Weight:         WeightLight,
			Parallelizable: true,
			Schemes:        []string{"hash"},
		}
		dims := matrixDimensions{schemes: []string{"hash", "path"}}
		results := expandSpec(spec, dims)
		if len(results) != 1 {
			t.Fatalf("expected 1 run (intersection), got %d", len(results))
		}
		if results[0].resolvedScheme != "hash" {
			t.Fatalf("expected 'hash', got %q", results[0].resolvedScheme)
		}
	})

	t.Run("cli_requests_scheme_test_supports_neither", func(t *testing.T) {
		spec := &BuilderSpec{
			Weight:         WeightLight,
			Parallelizable: true,
			Schemes:        []string{"hash"},
		}
		dims := matrixDimensions{schemes: []string{"path"}}
		results := expandSpec(spec, dims)
		if len(results) != 0 {
			t.Fatalf("expected 0 runs (empty intersection), got %d", len(results))
		}
	})

	t.Run("arbos_min_version_filter", func(t *testing.T) {
		spec := &BuilderSpec{
			Weight:          WeightLight,
			Parallelizable:  true,
			MinArbOSVersion: 30,
		}
		dims := matrixDimensions{arbos: []uint64{20, 30, 50}}
		results := expandSpec(spec, dims)
		if len(results) != 2 {
			t.Fatalf("expected 2 runs (30, 50), got %d", len(results))
		}
		if results[0].resolvedArbOS != 30 || results[1].resolvedArbOS != 50 {
			t.Fatalf("unexpected versions: %d, %d", results[0].resolvedArbOS, results[1].resolvedArbOS)
		}
	})

	t.Run("arbos_pin_version", func(t *testing.T) {
		spec := &BuilderSpec{
			Weight:          WeightLight,
			Parallelizable:  true,
			PinArbOSVersion: 31,
		}
		dims := matrixDimensions{arbos: []uint64{20, 31, 50}}
		results := expandSpec(spec, dims)
		if len(results) != 1 {
			t.Fatalf("expected 1 run (pinned 31), got %d", len(results))
		}
		if results[0].resolvedArbOS != 31 {
			t.Fatalf("expected 31, got %d", results[0].resolvedArbOS)
		}
	})

	t.Run("arbos_pin_incompatible", func(t *testing.T) {
		spec := &BuilderSpec{
			Weight:          WeightLight,
			Parallelizable:  true,
			PinArbOSVersion: 31,
		}
		dims := matrixDimensions{arbos: []uint64{20, 50}}
		results := expandSpec(spec, dims)
		if len(results) != 0 {
			t.Fatalf("expected 0 runs (pin 31 not in [20,50]), got %d", len(results))
		}
	})

	t.Run("cartesian_product", func(t *testing.T) {
		spec := &BuilderSpec{
			Weight:         WeightLight,
			Parallelizable: true,
			Schemes:        []string{"hash", "path"},
		}
		dims := matrixDimensions{
			schemes: []string{"hash", "path"},
			arbos:   []uint64{30, 50},
		}
		results := expandSpec(spec, dims)
		// 2 schemes x 2 arbos = 4
		if len(results) != 4 {
			t.Fatalf("expected 4 runs, got %d", len(results))
		}
	})

	t.Run("random_sampling", func(t *testing.T) {
		spec := &BuilderSpec{
			Weight:         WeightLight,
			Parallelizable: true,
		}
		dims := matrixDimensions{
			schemes: []string{"hash", "path"},
			arbos:   []uint64{20, 30, 50},
			randomN: 2,
		}
		results := expandSpec(spec, dims)
		// 2 schemes x 3 arbos = 6, but randomN=2 picks 2
		if len(results) != 2 {
			t.Fatalf("expected 2 runs (random sampling), got %d", len(results))
		}
	})

	t.Run("unconstrained_test_cli_expands", func(t *testing.T) {
		// Test declares no constraints (nil Schemes, nil DBEngines, no version limits).
		// CLI requests hash+path. Test should run twice.
		spec := &BuilderSpec{
			Weight:         WeightLight,
			Parallelizable: true,
		}
		dims := matrixDimensions{schemes: []string{"hash", "path"}}
		results := expandSpec(spec, dims)
		if len(results) != 2 {
			t.Fatalf("expected 2 runs, got %d", len(results))
		}
	})

	t.Run("subtest_naming", func(t *testing.T) {
		spec := &BuilderSpec{
			Weight:         WeightLight,
			Parallelizable: true,
			VariantName:    "variant1",
		}
		dims := matrixDimensions{schemes: []string{"hash"}, arbos: []uint64{50}}
		results := expandSpec(spec, dims)
		if len(results) != 1 {
			t.Fatalf("expected 1 run, got %d", len(results))
		}
		name := TestName("TestFoo", results[0])
		expected := "TestFoo/variant1/hash/arbos50"
		if name != expected {
			t.Fatalf("expected name %q, got %q", expected, name)
		}
	})
}
