// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

// Package v2 provides the a custom harness for system_tests.
//
// Design overview:
//   - testConfigX functions return []*BuilderSpec — zero or more descriptions of
//     test runs. Returning nil/empty skips the test for the current configuration.
//   - testRunX functions receive a *TestEnv (the built node) and contain only
//     the test logic itself: send txs, check state, assert invariants.
//     They never create or cancel contexts, never call cleanup.
//   - TestRunner (in runner_test.go) owns the lifecycle: it builds nodes,
//     injects TestEnv, runs testRunX, then tears down.
//     It also implements the worker-pool concurrency model so that tests are
//     only "running" from Go's perspective when they actually hold a worker.
//   - RegisterTest is called from init() in each test file. TestRunner
//     imports this package, triggering those init() calls automatically.
package v2

import (
	"context"
	"fmt"
	"testing"

	arbtest "github.com/offchainlabs/nitro/system_tests"
)

// ResourceWeight declares how many parallel slots a test consumes.
// TestRunner uses this to bound concurrency without Go's naive -parallel flag.
type ResourceWeight int

const (
	WeightLight  ResourceWeight = 1 // L2-only, single node
	WeightMedium ResourceWeight = 2 // L1+L2, standard setup
	WeightHeavy  ResourceWeight = 3 // Multi-node, L1+L2+secondary, or L3
	WeightMax    ResourceWeight = 4 // Challenge tests, staker tests, multi-L2
)

// TestParams holds external configuration overrides passed to the runner via CLI.
// Each field is a pointer so nil means "use the test's own default".
// testConfigX functions inspect these and return 0..N BuilderSpecs accordingly.
type TestParams struct {
	ArbOSVersion   *uint64
	StateScheme    *string // "hash" or "path" TODO: Use an enum instead.
	DatabaseEngine *string // "pebble", "leveldb", or "in-memory"
	Category       *string
	// Matrix expansion: the runner may call testConfigX multiple times with
	// different TestParams drawn from a cartesian product of CLI overrides.
}

// BuilderSpec is the data-only description of one test run returned by testConfigX.
// It carries no *testing.T or context — those are injected by the runner in runOneV2.
// The runner inspects these fields to decide how to build the node and schedule the test.
type BuilderSpec struct {
	NeedsL1        bool
	Weight         ResourceWeight
	Parallelizable bool

	// VariantName is non-empty when testConfigX returns multiple specs for the
	// same test (e.g. one per ArbOS version). The runner appends it to the
	// subtest name: "TestFoo/arbos30", "TestFoo/arbos40".
	VariantName string

	// Optional overrides applied by the runner when building the node.
	// Empty string/zero means "use the TestParams default or the suite default".
	StateScheme    string
	DatabaseEngine string
	ArbOSVersion   uint64 // 0 = ends up using the suite's default version.
}

// TestEnv is what testRunX receives after the runner has built the node.
// It holds only the handles needed to interact with the running chain.
// Lifecycle (ctx cancel, node cleanup) is managed by the runner, not the test.
type TestEnv struct {
	T      *testing.T
	Ctx    context.Context
	L2     *L2Handle
	L2Info *arbtest.BlockchainTestInfo
}

// Require fails the test immediately if err is non-nil.
func (e *TestEnv) Require(err error) {
	e.T.Helper()
	if err != nil {
		e.T.Fatal(err)
	}
}

// TestEntry ties together the name, config function, and run function for one test.
type TestEntry struct {
	Name   string
	Config func(TestParams) []*BuilderSpec
	Run    func(*TestEnv)
}

var globalRegistry []TestEntry

// RegisterTest adds a test pair to the global registry.
// Call this from an init() function in each test file so that TestRunner
// discovers it automatically.
//
//	func init() {
//	    RegisterTest("TestTransfer", testConfigTransfer, testRunTransfer)
//	}
func RegisterTest(name string, config func(TestParams) []*BuilderSpec, run func(*TestEnv)) {
	globalRegistry = append(globalRegistry, TestEntry{
		Name:   name,
		Config: config,
		Run:    run,
	})
}

// GetRegistry returns all registered test entries.
// Called by TestRunner to iterate over the full test suite.
func GetRegistry() []TestEntry {
	return globalRegistry
}

// ParseCLIParams reads test-configuration overrides from command-line flags.
// Currently returns an empty TestParams (all fields nil = use test defaults).
// Phase 2 will register --arbos-version, --state-scheme, --matrix, etc.
func ParseCLIParams() TestParams {
	return TestParams{}
}

// TestName returns the subtest name for a (entry, spec) pair.
func TestName(entryName string, spec *BuilderSpec) string {
	if spec.VariantName != "" {
		return fmt.Sprintf("%s/%s", entryName, spec.VariantName)
	}
	return entryName
}
