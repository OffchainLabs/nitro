// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package gethhook

import (
	"fmt"

	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
)

// upstreamSymbol pins the upstream var name as a string so
// TestAllUpstreamPrecompileSetsCataloged can cross-check this list
// against a go/parser walk of core/vm/contracts.go.
type ethereumPrecompileSet struct {
	upstreamSymbol string
	contracts      vm.PrecompiledContracts
	minVersion     uint64
}

// Each entry references an upstream vm.PrecompiledContractsXxx var
// directly, so a rename or removal upstream fails to compile.
// Upstream ADDITIONS are caught by knownUpstreamPrecompileSetSymbols
// + TestAllUpstreamPrecompileSetsCataloged.
var ethereumPrecompileSets = []ethereumPrecompileSet{
	{
		upstreamSymbol: "PrecompiledContractsBerlin",
		contracts:      vm.PrecompiledContractsBerlin,
		minVersion:     0,
	},
	{
		upstreamSymbol: "PrecompiledContractsCancun",
		contracts:      vm.PrecompiledContractsCancun,
		minVersion:     params.ArbosVersion_Stylus,
	},
	{
		upstreamSymbol: "PrecompiledContractsOsaka",
		contracts:      vm.PrecompiledContractsOsaka,
		minVersion:     params.ArbosVersion_Dia,
	},
	{
		upstreamSymbol: "PrecompiledContractsP256Verify",
		contracts:      vm.PrecompiledContractsP256Verify,
		minVersion:     params.ArbosVersion_Stylus,
	},
}

// Exhaustive catalog of upstream PrecompiledContractsXxx symbols.
// TestAllUpstreamPrecompileSetsCataloged fails when this map drifts
// from the real contracts.go, so a geth rebase that adds a new set
// surfaces the decision loudly.
//
// Each rationale must take one of three forms, enforced by
// TestEthereumPrecompileSetRelationships:
//
//   - "registered": has an entry in ethereumPrecompileSets above.
//   - "covered by <Set>": a registered entry that contains every
//     address this symbol exposes (superset check).
//   - "alias of <Set>": a Go-level `var X = Y` alias (pointer
//     equality check).
var knownUpstreamPrecompileSetSymbols = map[string]string{
	"PrecompiledContractsHomestead":  "covered by Berlin",
	"PrecompiledContractsByzantium":  "covered by Berlin",
	"PrecompiledContractsIstanbul":   "covered by Berlin",
	"PrecompiledContractsBerlin":     "registered",
	"PrecompiledContractsCancun":     "registered",
	"PrecompiledContractsPrague":     "covered by Osaka",
	"PrecompiledContractsBLS":        "alias of Prague",
	"PrecompiledContractsVerkle":     "alias of Berlin",
	"PrecompiledContractsOsaka":      "registered",
	"PrecompiledContractsP256Verify": "registered",
}

// registerAllEthereumPrecompileSets panics on an empty/nil
// contracts field so `range nil_map → zero iterations` can't
// silently drop a set from every snapshot.
func registerAllEthereumPrecompileSets() {
	for _, set := range ethereumPrecompileSets {
		if len(set.contracts) == 0 {
			panic(fmt.Sprintf("gethhook: Ethereum precompile set %s has empty/nil contracts — check the entry in ethereumPrecompileSets", set.upstreamSymbol))
		}
		for addr, precompile := range set.contracts {
			registerArbOSPrecompile(set.minVersion, addr, precompile)
		}
	}
}
