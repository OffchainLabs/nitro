// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

// Package v2 provides the test harness for Nitro system tests.
//
// Design overview:
//
//   - Tests declare their "compatibility surface" via BuilderSpec fields:
//     which state schemes they support, which ArbOS versions, etc.
//   - The runner decides how much of that surface to explore based on
//     CLI flags (single override, matrix expansion, or random sampling).
//   - testConfigX functions return []*BuilderSpec — data-only descriptions.
//   - testRunX functions receive a *TestEnv and contain only test logic.
//   - TestRunner owns the lifecycle: it expands the matrix, builds nodes,
//     injects TestEnv, runs testRunX, then tears down.
//   - Tests never create or cancel contexts, never call cleanup.
package v2

import (
	"context"
	"flag"
	"fmt"
	"math/rand/v2"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"

	arbtest "github.com/offchainlabs/nitro/system_tests"
)

// =========================================================================
// Resource weights
// =========================================================================

// ResourceWeight declares how many parallel slots a test consumes.
type ResourceWeight int

const (
	WeightLight  ResourceWeight = 1 // L2-only, single node
	WeightMedium ResourceWeight = 2 // L1+L2, standard setup
	WeightHeavy  ResourceWeight = 3 // Multi-node, L1+L2+secondary, or L3
	WeightMax    ResourceWeight = 4 // Challenge tests, staker tests, multi-L2
)

// =========================================================================
// CLI flags
// =========================================================================

var (
	flagArbOSVersion  = flag.Uint64("v2.arbos-version", 0, "Override ArbOS version (0 = use test default)")
	flagStateScheme   = flag.String("v2.state-scheme", "", "Override state scheme: hash, path")
	flagDBEngine      = flag.String("v2.db-engine", "", "Override DB engine: pebble, leveldb, in-memory")
	flagMaxWeight     = flag.Int("v2.max-weight", 0, "Max total weight of parallel tests (0 = GOMAXPROCS)")
	flagMatrixSchemes = flag.String("v2.matrix.state-scheme", "", "Comma-separated state schemes for matrix expansion")
	flagMatrixArbOS   = flag.String("v2.matrix.arbos-version", "", "Comma-separated ArbOS versions for matrix expansion")
	flagMatrixDB      = flag.String("v2.matrix.db-engine", "", "Comma-separated DB engines for matrix expansion")
	flagMatrixRandom  = flag.Int("v2.matrix.random", 0, "Pick N random combinations per test from the full matrix (0 = all)")
	flagCategories    = flag.String("v2.categories", "default", "Comma-separated test categories to enable (e.g. default,challenge,stylus)")
	flagWithoutRace   = flag.Bool("v2.skip-race-incompatible", false, "Skip tests marked as race-incompatible")
)

// =========================================================================
// BuilderSpec — the "compatibility surface" of a test
// =========================================================================

// BuilderSpec is a data-only description of one test run returned by testConfigX.
// It declares what the test supports, not what it wants to run with.
// The runner intersects these with CLI flags to produce actual test runs.
type BuilderSpec struct {
	// --- What the test needs ---
	NeedsL1 bool
	Weight  ResourceWeight

	// --- Scheduling ---
	Parallelizable bool

	// --- Variant name (when testConfigX returns multiple specs) ---
	VariantName string

	// --- Compatibility surface: what this test supports ---
	// Empty slices mean "supports everything". The runner intersects these
	// with CLI flags. If the intersection is empty, the test is skipped.
	Schemes   []string // e.g. ["hash", "path"] or nil (= both)
	DBEngines []string // e.g. ["pebble"] or nil (= all)

	// ArbOS version constraints.
	MinArbOSVersion uint64 // 0 = no minimum
	MaxArbOSVersion uint64 // 0 = no maximum
	PinArbOSVersion uint64 // non-zero = must use exactly this version

	// --- Post-test hooks (executed by the runner after testRunX completes) ---
	ValidateBlocks bool // runner validates all produced blocks via StatelessBlockValidator
	CheckMultiNode bool // runner spins up a 2nd node, syncs, and verifies block hashes match

	// --- Category (replaces build tags) ---
	Category TestCategory // empty = default category

	// --- Resolved values (filled by the runner after matrix expansion) ---
	// Tests should NOT set these. They are written by the runner.
	resolvedScheme    string
	resolvedDBEngine  string
	resolvedArbOS     uint64
	resolvedFromLabel string // human-readable label for subtest name
}

// TestCategory replaces build tags with runtime categories.
type TestCategory string

const (
	CategoryDefault         TestCategory = ""
	CategoryChallenge       TestCategory = "challenge"
	CategoryStylus          TestCategory = "stylus"
	CategoryCIOnly          TestCategory = "cionly"
	CategoryBenchmark       TestCategory = "benchmark"
	CategoryLegacyChallenge TestCategory = "legacychallenge"
	CategoryValidatorReorg  TestCategory = "validatorreorg"
)

// ResolvedScheme returns the state scheme chosen by the runner for this run.
func (s *BuilderSpec) ResolvedScheme() string { return s.resolvedScheme }

// ResolvedDBEngine returns the DB engine chosen by the runner for this run.
func (s *BuilderSpec) ResolvedDBEngine() string { return s.resolvedDBEngine }

// ResolvedArbOSVersion returns the ArbOS version chosen by the runner for this run.
func (s *BuilderSpec) ResolvedArbOSVersion() uint64 { return s.resolvedArbOS }

// =========================================================================
// Matrix expansion
// =========================================================================

// matrixDimensions holds the requested test dimensions from CLI flags.
type matrixDimensions struct {
	schemes   []string
	arbos     []uint64
	dbEngines []string
	randomN   int // 0 = all combinations
}

func parseMatrixDimensions() matrixDimensions {
	if !flag.Parsed() {
		flag.Parse()
	}
	var d matrixDimensions

	// Matrix flags take precedence over single-value flags.
	if *flagMatrixSchemes != "" {
		d.schemes = splitCSV(*flagMatrixSchemes)
	} else if *flagStateScheme != "" {
		d.schemes = []string{*flagStateScheme}
	}

	if *flagMatrixArbOS != "" {
		for _, s := range splitCSV(*flagMatrixArbOS) {
			var v uint64
			if _, err := fmt.Sscanf(s, "%d", &v); err == nil {
				d.arbos = append(d.arbos, v)
			}
		}
	} else if *flagArbOSVersion != 0 {
		d.arbos = []uint64{*flagArbOSVersion}
	}

	if *flagMatrixDB != "" {
		d.dbEngines = splitCSV(*flagMatrixDB)
	} else if *flagDBEngine != "" {
		d.dbEngines = []string{*flagDBEngine}
	}

	d.randomN = *flagMatrixRandom
	return d
}

// expandSpec takes one BuilderSpec (the test's compatibility surface) and the
// requested matrix dimensions, and returns the concrete runs to execute.
// Each returned spec has resolvedScheme/resolvedDBEngine/resolvedArbOS filled in.
// Returns nil if the test is incompatible with the requested dimensions.
func expandSpec(spec *BuilderSpec, dims matrixDimensions) []*BuilderSpec {
	// Intersect schemes: what the test supports ∩ what the CLI requests.
	schemes := intersectOrDefault(spec.Schemes, dims.schemes, []string{""})
	dbEngines := intersectOrDefault(spec.DBEngines, dims.dbEngines, []string{""})
	arbosVersions := filterArbOS(spec, dims.arbos)

	if len(schemes) == 0 || len(dbEngines) == 0 || len(arbosVersions) == 0 {
		return nil // incompatible
	}

	// Cartesian product.
	var results []*BuilderSpec
	for _, s := range schemes {
		for _, db := range dbEngines {
			for _, v := range arbosVersions {
				clone := *spec
				clone.resolvedScheme = s
				clone.resolvedDBEngine = db
				clone.resolvedArbOS = v

				// Build a label for the subtest name.
				var parts []string
				if s != "" {
					parts = append(parts, s)
				}
				if db != "" {
					parts = append(parts, db)
				}
				if v != 0 {
					parts = append(parts, fmt.Sprintf("arbos%d", v))
				}
				clone.resolvedFromLabel = strings.Join(parts, "/")

				results = append(results, &clone)
			}
		}
	}

	// Random sampling if requested.
	if dims.randomN > 0 && dims.randomN < len(results) {
		rand.Shuffle(len(results), func(i, j int) {
			results[i], results[j] = results[j], results[i]
		})
		results = results[:dims.randomN]
	}

	return results
}

// intersectOrDefault returns the values to iterate for one dimension.
//
// When CLI requests nothing (cliRequests is empty), we return fallback — a single
// default value. This means without CLI matrix flags, each test runs exactly once.
// The test's declared surface is only explored when the CLI explicitly requests it.
//
// When CLI requests values:
//   - If testSupports is empty (nil), the test supports everything → use cliRequests as-is.
//   - Otherwise, return the intersection of testSupports and cliRequests.
func intersectOrDefault(testSupports, cliRequests, fallback []string) []string {
	if len(cliRequests) == 0 {
		return fallback // no CLI constraint → single default run
	}
	if len(testSupports) == 0 {
		return cliRequests // test supports everything → use CLI
	}
	set := make(map[string]bool, len(testSupports))
	for _, s := range testSupports {
		set[s] = true
	}
	var result []string
	for _, s := range cliRequests {
		if set[s] {
			result = append(result, s)
		}
	}
	return result
}

// filterArbOS returns the ArbOS versions from cliVersions that are compatible
// with the spec's constraints (min/max/pin). If cliVersions is empty, returns
// {spec.PinArbOSVersion} or {0} (meaning "use default").
func filterArbOS(spec *BuilderSpec, cliVersions []uint64) []uint64 {
	if spec.PinArbOSVersion != 0 {
		// Test requires an exact version.
		if len(cliVersions) == 0 {
			return []uint64{spec.PinArbOSVersion}
		}
		for _, v := range cliVersions {
			if v == spec.PinArbOSVersion {
				return []uint64{v}
			}
		}
		return nil // CLI requested versions that don't match the pin
	}

	if len(cliVersions) == 0 {
		return []uint64{0} // no constraint, use default
	}

	var result []uint64
	for _, v := range cliVersions {
		if spec.MinArbOSVersion != 0 && v < spec.MinArbOSVersion {
			continue
		}
		if spec.MaxArbOSVersion != 0 && v > spec.MaxArbOSVersion {
			continue
		}
		result = append(result, v)
	}
	return result
}

// =========================================================================
// TestEnv — what testRunX receives
// =========================================================================

// TestEnv is what testRunX receives after the runner has built the node.
// It holds only the handles needed to interact with the running chain.
// Lifecycle (ctx cancel, node cleanup) is managed by the runner, not the test.
type TestEnv struct {
	T      *testing.T
	Ctx    context.Context
	L1     *L1Handle                    // nil for L2-only tests
	L1Info *arbtest.BlockchainTestInfo  // nil for L2-only tests
	L2     *L2Handle
	L2Info *arbtest.BlockchainTestInfo
	Spec   *BuilderSpec // the resolved spec for this run
}

// Require fails the test immediately if err is non-nil.
func (e *TestEnv) Require(err error, args ...any) {
	e.T.Helper()
	if err != nil {
		e.T.Fatal(append([]any{err}, args...)...)
	}
}

// Fatal fails the test with a formatted message.
func (e *TestEnv) Fatal(args ...any) {
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

// =========================================================================
// Test registry
// =========================================================================

// TestEntry ties together the name, config function, and run function for one test.
type TestEntry struct {
	Name   string
	Config func(TestParams) []*BuilderSpec
	Run    func(*TestEnv)
}

// TestParams holds external configuration overrides passed to the runner.
// testConfigX functions can inspect these to decide what specs to return.
// This is mostly useful for tests that need special logic beyond what
// BuilderSpec fields can express.
type TestParams struct {
	Dims matrixDimensions
}

var globalRegistry []TestEntry

// RegisterTest adds a test to the global registry.
// Call from init() in each test file.
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

// =========================================================================
// Naming
// =========================================================================

// TestName returns the subtest name for a (entry, spec) pair.
func TestName(entryName string, spec *BuilderSpec) string {
	name := entryName
	if spec.VariantName != "" {
		name = fmt.Sprintf("%s/%s", name, spec.VariantName)
	}
	if spec.resolvedFromLabel != "" {
		name = fmt.Sprintf("%s/%s", name, spec.resolvedFromLabel)
	}
	return name
}

// MaxWeight returns the configured max weight, or 0 for the default (GOMAXPROCS).
func MaxWeight() int {
	return *flagMaxWeight
}

// =========================================================================
// Helpers
// =========================================================================

// =========================================================================
// Simplified config constructors — reduce boilerplate in testConfigX
// =========================================================================

// L2Light returns a config function that always returns a single lightweight
// L2-only spec with no constraints. This is the most common config pattern.
func L2Light() func(TestParams) []*BuilderSpec {
	return func(_ TestParams) []*BuilderSpec {
		return []*BuilderSpec{{
			Weight:         WeightLight,
			Parallelizable: true,
		}}
	}
}

// L2WithMinArbOS returns a config function for a lightweight L2-only spec
// that requires a minimum ArbOS version.
func L2WithMinArbOS(minVersion uint64) func(TestParams) []*BuilderSpec {
	return func(_ TestParams) []*BuilderSpec {
		return []*BuilderSpec{{
			Weight:          WeightLight,
			Parallelizable:  true,
			MinArbOSVersion: minVersion,
		}}
	}
}

// L1L2 returns a config function for a standard L1+L2 test.
func L1L2() func(TestParams) []*BuilderSpec {
	return func(_ TestParams) []*BuilderSpec {
		return []*BuilderSpec{{
			NeedsL1:        true,
			Weight:         WeightMedium,
			Parallelizable: true,
		}}
	}
}

// EnabledCategories returns the set of categories enabled via CLI flags.
func EnabledCategories() map[TestCategory]bool {
	if !flag.Parsed() {
		flag.Parse()
	}
	cats := make(map[TestCategory]bool)
	for cat := range strings.SplitSeq(*flagCategories, ",") {
		cat = strings.TrimSpace(cat)
		if cat == "default" || cat == "" {
			cats[CategoryDefault] = true
		} else {
			cats[TestCategory(cat)] = true
		}
	}
	return cats
}

func splitCSV(s string) []string {
	var out []string
	for part := range strings.SplitSeq(s, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
