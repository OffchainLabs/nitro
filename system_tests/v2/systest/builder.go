// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package systest

import (
	"fmt"
	"reflect"
	"runtime"
	"runtime/debug"
	"strings"
	"time"
	"unicode"

	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/statetransfer"
	"github.com/offchainlabs/nitro/util/containers"
)

// builder is the mutable state collected by TestOptions during Test.
// Pinned axes use containers.Option: None = unpinned. Matrix expansion pins them per cell.
type builder struct {
	name     string // Named() override, else derived from the scenario func
	scenario Scenario

	topology Topology
	timeout  time.Duration // 0 = use the runner default

	arbOS       containers.Option[uint64]
	stateScheme containers.Option[StateScheme]
	dbEngine    containers.Option[DBEngine]

	category string

	minArbOS  uint64
	maxArbOS  uint64
	arbOSInit *params.ArbOSInit

	skipStateSchemes []StateScheme
	skipOnRace       bool
	skipChainOwner   bool
	exposeRPC        bool

	validation bool

	postHooks []Hook
	dims      map[axis][]axisVariant

	nodeOverrides         []func(*arbnode.Config)
	execOverrides         []func(*gethexec.Config)
	stackOverrides        []func(*node.Config)
	initDataOverrides     []func(*statetransfer.ArbosInitializationInfo)
	chainConfigOverrides  []func(*params.ChainConfig)
	followerExecOverrides []func(*gethexec.Config)
}

func newBuilder() *builder {
	return &builder{category: defaultCategory, dims: map[axis][]axisVariant{}}
}

// clone drops dims: the result is one cell of the matrix, already collapsed.
func (b *builder) clone() *builder {
	out := *b
	if len(b.skipStateSchemes) > 0 {
		out.skipStateSchemes = append([]StateScheme{}, b.skipStateSchemes...)
	}
	if len(b.postHooks) > 0 {
		out.postHooks = append([]Hook{}, b.postHooks...)
	}
	if len(b.nodeOverrides) > 0 {
		out.nodeOverrides = append([]func(*arbnode.Config){}, b.nodeOverrides...)
	}
	if len(b.execOverrides) > 0 {
		out.execOverrides = append([]func(*gethexec.Config){}, b.execOverrides...)
	}
	if len(b.stackOverrides) > 0 {
		out.stackOverrides = append([]func(*node.Config){}, b.stackOverrides...)
	}
	if len(b.initDataOverrides) > 0 {
		out.initDataOverrides = append([]func(*statetransfer.ArbosInitializationInfo){}, b.initDataOverrides...)
	}
	if len(b.chainConfigOverrides) > 0 {
		out.chainConfigOverrides = append([]func(*params.ChainConfig){}, b.chainConfigOverrides...)
	}
	if len(b.followerExecOverrides) > 0 {
		out.followerExecOverrides = append([]func(*gethexec.Config){}, b.followerExecOverrides...)
	}
	out.dims = map[axis][]axisVariant{}
	return &out
}

// validate runs cross-field invariant checks that a single TestOption can't
// express — they must see the final state after all options, in any order.
func (b *builder) validate() {
	if b.maxArbOS != 0 && b.minArbOS > b.maxArbOS {
		panic(fmt.Sprintf("systest: MinArbOS(%d) > MaxArbOS(%d)", b.minArbOS, b.maxArbOS))
	}
	if b.validation && b.topology == TopologyL2Only {
		panic("systest: WithValidation requires a parent chain")
	}
	if b.validates() && b.stateScheme.IsSome() && b.stateScheme.Unwrap() != validationScheme {
		panic(fmt.Sprintf("systest: validation requires %s state scheme; conflicts with WithStateScheme(%s)", validationScheme, b.stateScheme.Unwrap()))
	}
	if len(b.followerExecOverrides) > 0 && b.topology != TopologyMultiNode && b.topology != TopologyStakingValidation {
		panic("systest: WithFollowerExecConfigOverride requires a topology with a follower node")
	}
}

// validates reports whether any node in this test runs block validation:
// requested via WithValidation, or TopologyStakingValidation's staker follower (buildFollowerNode).
func (b *builder) validates() bool {
	return b.validation || b.topology == TopologyStakingValidation
}

func (b *builder) shouldSkip(sp scheduleParams) string {
	if raceEnabled && b.skipOnRace {
		return "skipped under -race"
	}
	if b.arbOS.IsSome() && sp.ArbOS.IsSome() && sp.ArbOS.Unwrap() != b.arbOS.Unwrap() {
		return fmt.Sprintf("ArbOS %d conflicts with ArbOS pin %d", b.arbOS.Unwrap(), sp.ArbOS.Unwrap())
	}
	effective := b.arbOS.UnwrapOr(sp.ArbOS.UnwrapOr(defaultArbOSVersion()))
	if b.minArbOS > 0 && effective < b.minArbOS {
		return fmt.Sprintf("requires ArbOS>=%d, got %d", b.minArbOS, effective)
	}
	if b.maxArbOS > 0 && effective > b.maxArbOS {
		return fmt.Sprintf("requires ArbOS<=%d, got %d", b.maxArbOS, effective)
	}
	if b.stateScheme.IsSome() && sp.StateScheme.IsSome() && b.stateScheme.Unwrap() != sp.StateScheme.Unwrap() {
		return fmt.Sprintf("state scheme %q conflicts with state scheme pin %q", b.stateScheme.Unwrap(), sp.StateScheme.Unwrap())
	}
	if b.dbEngine.IsSome() && sp.DBEngine.IsSome() && b.dbEngine.Unwrap() != sp.DBEngine.Unwrap() {
		return fmt.Sprintf("db engine %q conflicts with db engine pin %q", b.dbEngine.Unwrap(), sp.DBEngine.Unwrap())
	}
	scheme := b.resolvedStateScheme(sp)
	for _, s := range b.skipStateSchemes {
		if scheme.IsSome() && scheme.Unwrap() == s {
			return fmt.Sprintf("incompatible with state scheme %q", s)
		}
	}
	if b.validates() && scheme.IsSome() && scheme.Unwrap() != validationScheme {
		return fmt.Sprintf("validation requires %s state scheme", validationScheme)
	}
	if !sp.categoryEnabled(b.category) {
		return fmt.Sprintf("category %q not enabled", b.category)
	}
	return ""
}

// resolvedStateScheme is the scheme this test will actually run under:
// test pin (or matrix cell) → params pin → env default.
func (b *builder) resolvedStateScheme(sp scheduleParams) containers.Option[StateScheme] {
	if b.stateScheme.IsSome() {
		return b.stateScheme
	}
	return sp.resolvedScheme()
}

// defaultArbOSVersion is the ArbOS version a test gets when neither the test
// nor params pin one. Resolved from the dev test chain config.
func defaultArbOSVersion() uint64 {
	return chaininfo.ArbitrumDevTestChainConfig().ArbitrumChainParams.InitialArbOSVersion
}

// raceEnabled reports whether the binary was built with the race detector.
// Read once from build settings, so no build-tagged files are needed.
var raceEnabled = func() bool {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		panic("systest: debug.ReadBuildInfo() unavailable; cannot determine -race build")
	}
	for _, s := range info.Settings {
		if s.Key == "-race" {
			return s.Value == "true"
		}
	}
	return false
}()

// mergeParams fills unpinned axes from params/env defaults. Matrix axes are handled in expansion.
func (b *builder) mergeParams(sp scheduleParams) {
	if b.arbOS.IsNone() {
		b.arbOS = sp.ArbOS
	}
	if b.stateScheme.IsNone() {
		b.stateScheme = sp.resolvedScheme()
	}
	if b.dbEngine.IsNone() {
		b.dbEngine = sp.DBEngine
	}
}

// weight is the scheduler-slot cost: topology-derived, floored to weightMax
// when the run validates (JIT validation saturates a core).
func (b *builder) weight() weight {
	if b.validates() {
		return weightMax
	}
	return specWeight(b.topology)
}

func (b *builder) freeze(nameSuffix string) Spec {
	name := b.name
	if nameSuffix != "" {
		name = name + "/" + nameSuffix
	}
	return Spec{
		Name:           name,
		Weight:         b.weight(),
		ArbOSVersion:   b.arbOS,
		StateScheme:    b.stateScheme,
		DBEngine:       b.dbEngine,
		Category:       b.category,
		Topology:       b.topology,
		Timeout:        b.timeout,
		SkipChainOwner: b.skipChainOwner,
		ExposeRPC:      b.exposeRPC,
		arbOSInit:      b.arbOSInit,
		Validate:       b.validation,
	}
}

// derivedName extracts a test name from a scenario func: strips the package
// path and "testRun" prefix, then uppercases the first letter.
//
//	github.com/.../pkg.testRunTransfer -> "Transfer"
//	github.com/.../pkg.fooBar          -> "FooBar"
//
// Reflection is a convenience fallback. Prefer Named("X") for refactor-safety.
func derivedName(scenario Scenario) string {
	pc := reflect.ValueOf(scenario).Pointer()
	full := runtime.FuncForPC(pc).Name()
	if i := strings.LastIndex(full, "/"); i >= 0 {
		full = full[i+1:]
	}
	if i := strings.Index(full, "."); i >= 0 {
		full = full[i+1:]
	}
	full = strings.TrimPrefix(full, "testRun")
	// A remaining dot means a closure or method value (pkg.fn.func1, pkg.T.M):
	// no derivable name.
	if full == "" || strings.Contains(full, ".") {
		panic("systest: scenario func has no derivable name (bare 'testRun' or closure); use systest.Named(...) or rename the function")
	}
	runes := []rune(full)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}
