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
	"flag"
	"fmt"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"

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

// -------------------------------------------------------------------------
// CLI flags — parsed once by ParseCLIParams
// -------------------------------------------------------------------------

var (
	flagArbOSVersion    = flag.Uint64("v2.arbos-version", 0, "Override ArbOS version (0 = use test default)")
	flagStateScheme     = flag.String("v2.state-scheme", "", "Override state scheme: hash, path (empty = use test default)")
	flagDBEngine        = flag.String("v2.db-engine", "", "Override DB engine: pebble, leveldb, in-memory (empty = use test default)")
	flagMatrixSchemes   = flag.String("v2.matrix.state-scheme", "", "Comma-separated state schemes for matrix expansion (e.g. hash,path)")
	flagMatrixArbOS     = flag.String("v2.matrix.arbos-version", "", "Comma-separated ArbOS versions for matrix expansion (e.g. 31,40,50)")
	flagMaxWeight       = flag.Int("v2.max-weight", 0, "Max total weight of parallel tests (0 = GOMAXPROCS)")
)

// TestParams holds external configuration overrides passed to the runner via CLI.
// Each field is a pointer so nil means "use the test's own default".
// testConfigX functions inspect these and return 0..N BuilderSpecs accordingly.
type TestParams struct {
	ArbOSVersion   *uint64
	StateScheme    *string // "hash" or "path"
	DatabaseEngine *string // "pebble", "leveldb", or "in-memory"

	// label is a human-readable suffix for subtest names when running in matrix mode.
	// e.g. "hash/arbos31"
	label string
}

// Label returns a non-empty label if this TestParams came from matrix expansion.
func (p TestParams) Label() string { return p.label }

// ParseCLIParams reads test-configuration overrides from command-line flags.
// If matrix flags are set, it returns the cartesian product of all specified dimensions.
// Otherwise it returns a single TestParams with whatever single-value overrides were given.
func ParseCLIParams() []TestParams {
	if !flag.Parsed() {
		flag.Parse()
	}

	// Build matrix dimensions.
	var schemes []string
	var arbosVersions []uint64
	var dbEngines []string

	if *flagMatrixSchemes != "" {
		schemes = splitCSV(*flagMatrixSchemes)
	} else if *flagStateScheme != "" {
		schemes = []string{*flagStateScheme}
	}

	if *flagMatrixArbOS != "" {
		for _, s := range splitCSV(*flagMatrixArbOS) {
			var v uint64
			if _, err := fmt.Sscanf(s, "%d", &v); err == nil {
				arbosVersions = append(arbosVersions, v)
			}
		}
	} else if *flagArbOSVersion != 0 {
		arbosVersions = []uint64{*flagArbOSVersion}
	}

	if *flagDBEngine != "" {
		dbEngines = []string{*flagDBEngine}
	}

	// If nothing specified at all, return a single empty params (all nil = test defaults).
	if len(schemes) == 0 && len(arbosVersions) == 0 && len(dbEngines) == 0 {
		return []TestParams{{}}
	}

	// Normalize: ensure at least one entry per dimension so cartesian product works.
	if len(schemes) == 0 {
		schemes = []string{""}
	}
	if len(arbosVersions) == 0 {
		arbosVersions = []uint64{0}
	}
	if len(dbEngines) == 0 {
		dbEngines = []string{""}
	}

	var out []TestParams
	for _, s := range schemes {
		for _, v := range arbosVersions {
			for _, d := range dbEngines {
				p := TestParams{}
				var parts []string

				if s != "" {
					cp := s
					p.StateScheme = &cp
					parts = append(parts, s)
				}
				if v != 0 {
					cp := v
					p.ArbOSVersion = &cp
					parts = append(parts, fmt.Sprintf("arbos%d", v))
				}
				if d != "" {
					cp := d
					p.DatabaseEngine = &cp
					parts = append(parts, d)
				}

				p.label = strings.Join(parts, "/")
				out = append(out, p)
			}
		}
	}
	return out
}

func splitCSV(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

// -------------------------------------------------------------------------
// BuilderSpec
// -------------------------------------------------------------------------

// BuilderSpec is the data-only description of one test run returned by testConfigX.
// It carries no *testing.T or context — those are injected by the runner.
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

	// MinArbOSVersion causes the test to be skipped if the params request a
	// version below this threshold.
	MinArbOSVersion uint64
}

// ApplyParams merges TestParams overrides into the spec. The spec's own values
// take precedence (test knows best), but if the spec left a field at zero/empty,
// the params fill it in. Returns false if the spec is incompatible with the params
// (e.g. params pin an ArbOS version below the spec's minimum).
func (s *BuilderSpec) ApplyParams(p TestParams) bool {
	// ArbOS version: params override wins if spec didn't pin one.
	if p.ArbOSVersion != nil {
		if s.MinArbOSVersion > 0 && *p.ArbOSVersion < s.MinArbOSVersion {
			return false // incompatible
		}
		if s.ArbOSVersion == 0 {
			s.ArbOSVersion = *p.ArbOSVersion
		}
	}

	// State scheme: params override wins if spec didn't pin one.
	if p.StateScheme != nil && s.StateScheme == "" {
		s.StateScheme = *p.StateScheme
	}

	// DB engine: params override wins if spec didn't pin one.
	if p.DatabaseEngine != nil && s.DatabaseEngine == "" {
		s.DatabaseEngine = *p.DatabaseEngine
	}

	return true
}

// -------------------------------------------------------------------------
// TestEnv
// -------------------------------------------------------------------------

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
func (e *TestEnv) Require(err error, args ...interface{}) {
	e.T.Helper()
	if err != nil {
		e.T.Fatal(append([]interface{}{err}, args...)...)
	}
}

// Fatal fails the test with a formatted message.
func (e *TestEnv) Fatal(args ...interface{}) {
	e.T.Helper()
	e.T.Fatal(args...)
}

// GetDefaultTransactOpts returns a TransactOpts for the named account.
func (e *TestEnv) GetDefaultTransactOpts(name string) bind.TransactOpts {
	e.T.Helper()
	return e.L2Info.GetDefaultTransactOpts(name, e.Ctx)
}

// EnsureTxSucceeded waits for a tx to be mined and asserts it succeeded.
func (e *TestEnv) EnsureTxSucceeded(tx *types.Transaction) *types.Receipt {
	e.T.Helper()
	return e.L2.WaitForTx(e.T, e.Ctx, tx)
}

// -------------------------------------------------------------------------
// Registry
// -------------------------------------------------------------------------

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
func RegisterTest(name string, config func(TestParams) []*BuilderSpec, run func(*TestEnv)) {
	globalRegistry = append(globalRegistry, TestEntry{
		Name:   name,
		Config: config,
		Run:    run,
	})
}

// GetRegistry returns all registered test entries.
func GetRegistry() []TestEntry {
	return globalRegistry
}

// TestName returns the subtest name for a (entry, spec, params) combination.
func TestName(entryName string, spec *BuilderSpec, paramLabel string) string {
	name := entryName
	if spec.VariantName != "" {
		name = fmt.Sprintf("%s/%s", name, spec.VariantName)
	}
	if paramLabel != "" {
		name = fmt.Sprintf("%s/%s", name, paramLabel)
	}
	return name
}

// MaxWeight returns the configured max weight, or 0 for the default (GOMAXPROCS).
func MaxWeight() int {
	return *flagMaxWeight
}
