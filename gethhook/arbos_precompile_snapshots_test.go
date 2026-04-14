// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package gethhook

import (
	"reflect"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/precompiles"
)

// withIsolatedArbOSRegistry lets a test register its own precompiles
// without bleeding into the real init-registered state used by
// TestAllRealPrecompilesReachableAtDeclaredVersion and the existing
// geth_test.go chain tests. Tests share package state and must not
// use t.Parallel().
func withIsolatedArbOSRegistry(t *testing.T) {
	t.Helper()
	savedRegs := make([]arbOSPrecompileRegistration, len(arbOSPrecompileRegistrations))
	copy(savedRegs, arbOSPrecompileRegistrations)
	arbOSPrecompileRegistrations = nil
	arbOSPrecompileSnapshots = nil
	arbOSPrecompileOnce = sync.Once{}
	t.Cleanup(func() {
		arbOSPrecompileRegistrations = savedRegs
		arbOSPrecompileSnapshots = nil
		arbOSPrecompileOnce = sync.Once{}
	})
}

type stubPrecompile struct{ tag string }

func (s *stubPrecompile) RequiredGas(input []byte) uint64  { return 0 }
func (s *stubPrecompile) Run(input []byte) ([]byte, error) { return []byte(s.tag), nil }
func (s *stubPrecompile) Name() string                     { return "stub:" + s.tag }

func addrOf(hex string) common.Address { return common.HexToAddress(hex) }

// Empty registry must return (nil, nil) and hit the non-nil empty
// sentinel without panicking.
func TestArbOSPrecompilesForEmptyRegistry(t *testing.T) {
	withIsolatedArbOSRegistry(t)

	for _, v := range []uint64{0, 1, 30, 50, 60, 1 << 40} {
		m, a := arbOSPrecompilesFor(v)
		if m != nil || a != nil {
			t.Errorf("ArbOS %d: expected (nil, nil), got map=%v addrs=%v", v, m, a)
		}
	}
}

// Pins the core consensus fix: a pre-activation CALL must charge
// 2600 gas (cold account) not 100 (warm precompile). An earlier
// gethhook change unconditionally registered precompiles into every
// bucket and shifted state roots during historical replay.
func TestArbOSPrecompileActivationVersionGating(t *testing.T) {
	withIsolatedArbOSRegistry(t)

	a := addrOf("0xfeed")
	registerArbOSPrecompile(60, a, &stubPrecompile{tag: "v60"})

	for _, v := range []uint64{0, 30, 50, 59} {
		m, addrs := arbOSPrecompilesFor(v)
		if _, ok := m[a]; ok {
			t.Errorf("ArbOS %d: precompile %s should be hidden (activation 60)", v, a)
		}
		if containsAddr(addrs, a) {
			t.Errorf("ArbOS %d: address %s leaked into active list", v, a)
		}
	}
	for _, v := range []uint64{60, 61, 100, 1 << 40} {
		m, addrs := arbOSPrecompilesFor(v)
		if _, ok := m[a]; !ok {
			t.Errorf("ArbOS %d: precompile %s should be active", v, a)
		}
		if !containsAddr(addrs, a) {
			t.Errorf("ArbOS %d: address %s missing from active list", v, a)
		}
	}
}

// Non-fork-boundary activation: the previous fork-name bucket
// scheme (IsDia / IsStylus / IsArbitrum) silently widened any
// between-fork registration to the lower bucket's whole window.
func TestArbOSPrecompileIntermediateVersion(t *testing.T) {
	withIsolatedArbOSRegistry(t)

	a := addrOf("0xa1")
	registerArbOSPrecompile(41, a, &stubPrecompile{tag: "v41"})

	cases := []struct {
		v    uint64
		want bool
	}{
		{0, false}, {30, false}, {40, false},
		{41, true}, {45, true}, {50, true}, {60, true},
	}
	for _, c := range cases {
		m, _ := arbOSPrecompilesFor(c.v)
		_, got := m[a]
		if got != c.want {
			t.Errorf("ArbOS %d: present=%v want %v", c.v, got, c.want)
		}
	}
}

// Berlin's ecrecover at minVersion 0 overwritten by Cancun's at
// ArbosVersion_Stylus. The address must appear exactly once in
// each snapshot's slice so iteration doesn't double-count.
func TestArbOSPrecompileLayeringOverwrites(t *testing.T) {
	withIsolatedArbOSRegistry(t)

	a := addrOf("0xec")
	early := &stubPrecompile{tag: "berlin"}
	late := &stubPrecompile{tag: "cancun"}
	registerArbOSPrecompile(0, a, early)
	registerArbOSPrecompile(params.ArbosVersion_Stylus, a, late)

	m0, addrs0 := arbOSPrecompilesFor(0)
	if m0[a] != early {
		t.Errorf("ArbOS 0: expected early contract, got %+v", m0[a])
	}
	if n := countAddr(addrs0, a); n != 1 {
		t.Errorf("ArbOS 0: address %s appears %d times, want 1", a, n)
	}

	mStylus, addrsStylus := arbOSPrecompilesFor(params.ArbosVersion_Stylus)
	if mStylus[a] != late {
		t.Errorf("ArbOS Stylus: expected late contract, got %+v", mStylus[a])
	}
	if n := countAddr(addrsStylus, a); n != 1 {
		t.Errorf("ArbOS Stylus: address %s appears %d times, want 1", a, n)
	}
}

// End-to-end through the real vm.ActivePrecompiledContracts /
// ActivePrecompiles path, plus a pointer-equality pin on
// arbOSPrecompilesFor so a refactor that allocates a fresh snapshot
// per call fails here.
func TestArbOSPrecompileDispatchEndToEnd(t *testing.T) {
	withIsolatedArbOSRegistry(t)

	active := addrOf("0xabcd01")
	future := addrOf("0xabcd02")
	registerArbOSPrecompile(0, active, &stubPrecompile{tag: "active"})
	registerArbOSPrecompile(60, future, &stubPrecompile{tag: "future"})

	rules50 := params.Rules{IsArbitrum: true, ArbOSVersion: 50}
	cloned := vm.ActivePrecompiledContracts(rules50)
	if _, ok := cloned[active]; !ok {
		t.Error("ArbOS 50: expected active precompile present via ActivePrecompiledContracts")
	}
	if _, ok := cloned[future]; ok {
		t.Error("ArbOS 50: future precompile leaked through ActivePrecompiledContracts")
	}

	rawMap1, _ := arbOSPrecompilesFor(50)
	rawMap2, _ := arbOSPrecompilesFor(50)
	if reflect.ValueOf(rawMap1).Pointer() != reflect.ValueOf(rawMap2).Pointer() {
		t.Error("arbOSPrecompilesFor returned a fresh map on repeat call -- cache broken")
	}

	rules60 := params.Rules{IsArbitrum: true, ArbOSVersion: 60}
	if _, ok := vm.ActivePrecompiledContracts(rules60)[future]; !ok {
		t.Error("ArbOS 60: future precompile should be active")
	}
	if !containsAddr(vm.ActivePrecompiles(rules60), future) {
		t.Error("ArbOS 60: ActivePrecompiles missing future address")
	}
}

// Pins the !rules.IsArbitrum gate in arbOSActivePrecompiles: a
// non-Arbitrum chain must never observe the Arbitrum snapshot, and
// a regression that dropped the gate would fail here.
func TestArbOSPrecompileDispatchFallsThroughForNonArbitrum(t *testing.T) {
	withIsolatedArbOSRegistry(t)

	a := addrOf("0xfeed")
	registerArbOSPrecompile(60, a, &stubPrecompile{tag: "v60"})

	rules := params.Rules{IsArbitrum: false, IsCancun: true, ArbOSVersion: 60}
	m := vm.ActivePrecompiledContracts(rules)
	if _, ok := m[a]; ok {
		t.Error("non-Arbitrum dispatch leaked the Arbitrum snapshot precompile")
	}
	// Also assert we landed in the Ethereum switch and didn't
	// just return empty.
	if _, ok := m[common.BytesToAddress([]byte{0x01})]; !ok {
		t.Error("non-Arbitrum Cancun dispatch missing ecrecover -- fall-through broken")
	}
	addrs := vm.ActivePrecompiles(rules)
	if containsAddr(addrs, a) {
		t.Error("non-Arbitrum ActivePrecompiles leaked the Arbitrum snapshot address")
	}
}

// Pins the maps.Clone wrapper: a refactor that aliased the cached
// snapshot would silently let one caller poison every subsequent
// lookup.
func TestActivePrecompiledContractsReturnsDistinctClone(t *testing.T) {
	withIsolatedArbOSRegistry(t)

	a := addrOf("0xcace01")
	registerArbOSPrecompile(0, a, &stubPrecompile{tag: "active"})

	rules := params.Rules{IsArbitrum: true, ArbOSVersion: 50}
	cloned := vm.ActivePrecompiledContracts(rules)
	if _, ok := cloned[a]; !ok {
		t.Fatalf("clone missing registered address %s", a)
	}
	cached, _ := arbOSPrecompilesFor(50)
	if reflect.ValueOf(cloned).Pointer() == reflect.ValueOf(cached).Pointer() {
		t.Fatal("ActivePrecompiledContracts returned the cached snapshot; must be a clone")
	}

	mut := addrOf("0xcace02")
	cloned[mut] = &stubPrecompile{tag: "mut"}
	again, _ := arbOSPrecompilesFor(50)
	if _, poisoned := again[mut]; poisoned {
		t.Error("mutation of the clone leaked into the cached snapshot")
	}
}

func TestRegisterArbOSPrecompileNilPanics(t *testing.T) {
	withIsolatedArbOSRegistry(t)

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on nil contract")
		}
	}()
	registerArbOSPrecompile(0, addrOf("0x1"), nil)
}

// A post-materialize registration would be silently ignored and
// consensus would diverge, so the panic must fire.
func TestRegisterArbOSPrecompileAfterMaterializePanics(t *testing.T) {
	withIsolatedArbOSRegistry(t)

	registerArbOSPrecompile(0, addrOf("0x1"), &stubPrecompile{tag: "early"})
	_, _ = arbOSPrecompilesFor(0) // materialize

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on registration after materialization")
		}
	}()
	registerArbOSPrecompile(0, addrOf("0x2"), &stubPrecompile{tag: "late"})
}

// Against the real init-registered state (not isolated): every
// Nitro precompile must be reachable at its declared ArbosVersion,
// wrapped in ArbosPrecompileWrapper, and absent at declared-1.
// Layering further back is covered by
// TestArbOSPrecompileActivationVersionGating.
func TestAllRealPrecompilesReachableAtDeclaredVersion(t *testing.T) {
	for addr, p := range precompiles.Precompiles() {
		name := p.Precompile().Name()
		declared := p.Precompile().ArbosVersion()

		m, _ := arbOSPrecompilesFor(declared)
		got, ok := m[addr]
		if !ok {
			t.Errorf("precompile %s (%s) not registered at its declared ArbosVersion %d", addr, name, declared)
			continue
		}
		if _, isWrapper := got.(ArbosPrecompileWrapper); !isWrapper {
			t.Errorf("precompile %s (%s) at version %d is registered but not wrapped in ArbosPrecompileWrapper", addr, name, declared)
		}

		if declared == 0 {
			continue
		}
		mPrev, _ := arbOSPrecompilesFor(declared - 1)
		if _, stillThere := mPrev[addr]; stillThere {
			t.Errorf("precompile %s (%s) declared at version %d is also present at version %d — off-by-one regression", addr, name, declared, declared-1)
		}
	}
}

func containsAddr(addrs []common.Address, want common.Address) bool {
	for _, a := range addrs {
		if a == want {
			return true
		}
	}
	return false
}

func countAddr(addrs []common.Address, want common.Address) int {
	n := 0
	for _, a := range addrs {
		if a == want {
			n++
		}
	}
	return n
}
