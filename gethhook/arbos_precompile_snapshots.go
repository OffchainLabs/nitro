// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package gethhook

import (
	"fmt"
	"sort"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

// Arbitrum precompiles are registered per ArbOS activation version
// through registerArbOSPrecompile. Dispatch picks the snapshot with
// the largest activationVersion <= rules.ArbOSVersion, so a
// precompile introduced at version V is hidden on blocks < V —
// matching consensus, which treats the address as a regular account
// on pre-activation blocks (cold CALL 2600 gas, not warm precompile
// 100 gas). Without this gate, historical replay diverges on state
// root.
//
// Registrations are init-only and materialized lazily under
// sync.Once. All ArbOS-versioning semantics live here rather than
// in core/vm to keep the fork's delta vs upstream ethereum/go-ethereum
// minimal.

type arbOSPrecompileRegistration struct {
	minVersion uint64
	addr       common.Address
	contract   vm.PrecompiledContract
}

// arbOSPrecompileSnapshot holds the contracts and addresses active
// at a specific ArbOS version. Every mutation goes through
// addPrecompile so the two views never drift.
type arbOSPrecompileSnapshot struct {
	activationVersion uint64
	contracts         vm.PrecompiledContracts
	addresses         []common.Address
}

// addPrecompile permits re-registration at the same address so
// layering (higher-minVersion registration overwrites lower) works:
// the later contract replaces the earlier in the map, and the
// existing address slot is left in place so the slice never grows
// duplicates. Called only from materializeArbOSPrecompileSnapshots;
// snapshots are immutable after that.
func (s *arbOSPrecompileSnapshot) addPrecompile(addr common.Address, c vm.PrecompiledContract) {
	if _, exists := s.contracts[addr]; !exists {
		s.addresses = append(s.addresses, addr)
	}
	s.contracts[addr] = c
}

var (
	arbOSPrecompileRegistrations []arbOSPrecompileRegistration
	// Sorted descending so dispatch returns on the first match.
	// Non-nil after materialization (even on the empty-registrations
	// sentinel path), which keeps the post-materialize guard in
	// registerArbOSPrecompile armed.
	arbOSPrecompileSnapshots []arbOSPrecompileSnapshot
	arbOSPrecompileOnce      sync.Once
)

// registerArbOSPrecompile must be called from init(). Multiple
// registrations at the same address with different minVersions are
// allowed: the highest-minVersion entry <= each snapshot's
// activationVersion wins, so e.g. Berlin's ecrecover can be
// registered at 0 and replaced by Cancun's at
// params.ArbosVersion_Stylus.
func registerArbOSPrecompile(minVersion uint64, addr common.Address, c vm.PrecompiledContract) {
	if c == nil {
		panic(fmt.Sprintf("registerArbOSPrecompile: nil contract for %s at version %d", addr, minVersion))
	}
	if arbOSPrecompileSnapshots != nil {
		panic(fmt.Sprintf("registerArbOSPrecompile: %s registered at version %d after snapshots were materialized; must be called from init()", addr, minVersion))
	}
	arbOSPrecompileRegistrations = append(arbOSPrecompileRegistrations, arbOSPrecompileRegistration{
		minVersion: minVersion,
		addr:       addr,
		contract:   c,
	})
}

// arbOSPrecompilesFor is installed into core/vm via
// vm.SetArbOSPrecompileResolver. Returns (nil, nil) when no
// registration sits at or below v so Arbitrum dispatch falls
// through to account handling. The views are shared read-only.
func arbOSPrecompilesFor(v uint64) (vm.PrecompiledContracts, []common.Address) {
	arbOSPrecompileOnce.Do(materializeArbOSPrecompileSnapshots)
	for _, s := range arbOSPrecompileSnapshots {
		if s.activationVersion <= v {
			return s.contracts, s.addresses
		}
	}
	return nil, nil
}

// materializeArbOSPrecompileSnapshots builds one snapshot per
// distinct activation version. Snapshot count and registration
// count are both tiny in practice, so we rebuild from scratch
// rather than maintain snapshots incrementally.
func materializeArbOSPrecompileSnapshots() {
	if len(arbOSPrecompileRegistrations) == 0 {
		// Non-nil zero-length sentinel: keeps the
		// post-materialize guard in registerArbOSPrecompile armed.
		arbOSPrecompileSnapshots = []arbOSPrecompileSnapshot{}
		return
	}

	versionSet := make(map[uint64]struct{}, 8)
	for _, r := range arbOSPrecompileRegistrations {
		versionSet[r.minVersion] = struct{}{}
	}
	versions := make([]uint64, 0, len(versionSet))
	for v := range versionSet {
		versions = append(versions, v)
	}
	sort.Slice(versions, func(i, j int) bool { return versions[i] < versions[j] })

	// Replay in ascending minVersion order so higher-minVersion
	// entries overwrite lower ones at the same address. Stable sort
	// keeps the address slice deterministic across runs.
	regs := make([]arbOSPrecompileRegistration, len(arbOSPrecompileRegistrations))
	copy(regs, arbOSPrecompileRegistrations)
	sort.SliceStable(regs, func(i, j int) bool { return regs[i].minVersion < regs[j].minVersion })

	snapshots := make([]arbOSPrecompileSnapshot, len(versions))
	for i, v := range versions {
		snap := arbOSPrecompileSnapshot{
			activationVersion: v,
			contracts:         make(vm.PrecompiledContracts),
		}
		for _, r := range regs {
			if r.minVersion > v {
				break // regs is ascending; remainder is all > v.
			}
			snap.addPrecompile(r.addr, r.contract)
		}
		snapshots[i] = snap
	}
	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].activationVersion > snapshots[j].activationVersion
	})
	arbOSPrecompileSnapshots = snapshots
}
