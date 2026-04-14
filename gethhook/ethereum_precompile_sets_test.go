// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package gethhook

import (
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/core/vm"
)

// Hand-maintained string → var map; Go can't reflect on
// package-level names. Drift from knownUpstreamPrecompileSetSymbols
// is caught by TestEthereumPrecompileSetRelationships.
var upstreamPrecompileContractsBySymbol = map[string]vm.PrecompiledContracts{
	"PrecompiledContractsHomestead":  vm.PrecompiledContractsHomestead,
	"PrecompiledContractsByzantium":  vm.PrecompiledContractsByzantium,
	"PrecompiledContractsIstanbul":   vm.PrecompiledContractsIstanbul,
	"PrecompiledContractsBerlin":     vm.PrecompiledContractsBerlin,
	"PrecompiledContractsCancun":     vm.PrecompiledContractsCancun,
	"PrecompiledContractsPrague":     vm.PrecompiledContractsPrague,
	"PrecompiledContractsBLS":        vm.PrecompiledContractsBLS,
	"PrecompiledContractsVerkle":     vm.PrecompiledContractsVerkle,
	"PrecompiledContractsOsaka":      vm.PrecompiledContractsOsaka,
	"PrecompiledContractsP256Verify": vm.PrecompiledContractsP256Verify,
}

// Relative to the gethhook package dir, which is the cwd under
// `go test`.
const upstreamContractsPath = "../go-ethereum/core/vm/contracts.go"

// Primary safety net against silently dropping a newly-added
// upstream precompile set. Uses go/parser (not a regex) so
// block-form `var (...)` declarations can't slip past.
func TestAllUpstreamPrecompileSetsCataloged(t *testing.T) {
	seen := parseUpstreamPrecompileSymbols(t)
	if len(seen) == 0 {
		t.Fatalf("found no PrecompiledContracts* vars in %s — parser or submodule layout regressed", upstreamContractsPath)
	}

	for name := range seen {
		if _, ok := knownUpstreamPrecompileSetSymbols[name]; !ok {
			t.Errorf("upstream symbol %s is not cataloged in knownUpstreamPrecompileSetSymbols — decide how Arbitrum gates it, then add a rationale entry", name)
		}
	}
	// Inverse direction catches stale catalog entries that a
	// compile-time rename would miss.
	for name := range knownUpstreamPrecompileSetSymbols {
		if _, ok := seen[name]; !ok {
			t.Errorf("catalog entry %s no longer exists upstream — remove it from knownUpstreamPrecompileSetSymbols", name)
		}
	}
}

func parseUpstreamPrecompileSymbols(t *testing.T) map[string]struct{} {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, upstreamContractsPath, nil, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("parsing %s: %v", upstreamContractsPath, err)
	}
	seen := make(map[string]struct{})
	ast.Inspect(file, func(n ast.Node) bool {
		decl, ok := n.(*ast.GenDecl)
		if !ok || decl.Tok != token.VAR {
			return true
		}
		for _, spec := range decl.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for _, name := range vs.Names {
				if strings.HasPrefix(name.Name, "PrecompiledContracts") {
					seen[name.Name] = struct{}{}
				}
			}
		}
		return true
	})
	return seen
}

// Pins consistency between the catalog and the registration list.
func TestEveryRegisteredSetIsCataloged(t *testing.T) {
	for _, set := range ethereumPrecompileSets {
		rationale, ok := knownUpstreamPrecompileSetSymbols[set.upstreamSymbol]
		if !ok {
			t.Errorf("ethereumPrecompileSets entry %s missing from knownUpstreamPrecompileSetSymbols", set.upstreamSymbol)
			continue
		}
		if rationale != "registered" {
			t.Errorf("ethereumPrecompileSets entry %s is catalogued as %q, expected \"registered\"", set.upstreamSymbol, rationale)
		}
	}
}

// Backs the "covered by X" and "alias of X" rationales with real
// assertions — without this, a future upstream change that breaks
// the superset relationship (e.g. Prague adds a precompile Osaka
// doesn't carry) would silently drop the covered set. Iterates the
// catalog directly so unknown rationale forms and unknown targets
// fail loudly.
func TestEthereumPrecompileSetRelationships(t *testing.T) {
	for name := range knownUpstreamPrecompileSetSymbols {
		if _, ok := upstreamPrecompileContractsBySymbol[name]; !ok {
			t.Errorf("upstreamPrecompileContractsBySymbol is missing %s — add the var reference alongside the catalog entry", name)
		}
	}

	for name, rationale := range knownUpstreamPrecompileSetSymbols {
		covered, ok := upstreamPrecompileContractsBySymbol[name]
		if !ok {
			continue // already reported above
		}
		switch {
		case rationale == "registered":
			// Covered by TestEveryRegisteredSetIsCataloged +
			// TestEveryEthereumPrecompileAddressRegistered.
		case strings.HasPrefix(rationale, "covered by "):
			targetName := "PrecompiledContracts" + strings.TrimPrefix(rationale, "covered by ")
			target, ok := upstreamPrecompileContractsBySymbol[targetName]
			if !ok {
				t.Errorf("%s rationale %q points at unknown target %s", name, rationale, targetName)
				continue
			}
			for a := range covered {
				if _, ok := target[a]; !ok {
					t.Errorf("%s (%s): address %s missing from %s — catalog rationale is stale", name, rationale, a, targetName)
				}
			}
		case strings.HasPrefix(rationale, "alias of "):
			targetName := "PrecompiledContracts" + strings.TrimPrefix(rationale, "alias of ")
			target, ok := upstreamPrecompileContractsBySymbol[targetName]
			if !ok {
				t.Errorf("%s rationale %q points at unknown target %s", name, rationale, targetName)
				continue
			}
			if reflect.ValueOf(covered).Pointer() != reflect.ValueOf(target).Pointer() {
				t.Errorf("%s (%s): no longer shares a backing map with %s — catalog rationale is stale", name, rationale, targetName)
			}
		default:
			t.Errorf("%s has unknown rationale form %q — expected \"registered\", \"covered by <Set>\", or \"alias of <Set>\"", name, rationale)
		}
	}
}

// The nil-contracts guard exists specifically to prevent a silent
// `range nil_map → zero iterations` drop, so exercise the panic
// path directly — a regression that removed the guard would otherwise
// re-open the exact bug it was added to prevent.
func TestRegisterAllEthereumPrecompileSetsPanicsOnNilContracts(t *testing.T) {
	withIsolatedArbOSRegistry(t)
	saved := ethereumPrecompileSets
	t.Cleanup(func() { ethereumPrecompileSets = saved })

	ethereumPrecompileSets = []ethereumPrecompileSet{
		{upstreamSymbol: "PrecompiledContractsFake", contracts: nil, minVersion: 0},
	}

	defer func() {
		if r := recover(); r == nil {
			t.Error("registerAllEthereumPrecompileSets should panic on nil contracts")
		}
	}()
	registerAllEthereumPrecompileSets()
}

// Catches a registration-loop typo that silently drops addresses
// from dispatch.
func TestEveryEthereumPrecompileAddressRegistered(t *testing.T) {
	for _, set := range ethereumPrecompileSets {
		m, _ := arbOSPrecompilesFor(set.minVersion)
		for a := range set.contracts {
			if _, ok := m[a]; !ok {
				t.Errorf("Ethereum precompile %s from set %s not registered at activation version %d", a, set.upstreamSymbol, set.minVersion)
			}
		}
	}
}
