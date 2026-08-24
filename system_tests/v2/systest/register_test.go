// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package systest

import (
	"testing"

	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/util/containers"
)

func stubRegistry(t *testing.T, bs ...*builder) {
	t.Helper()
	savedReg, savedNames, savedFrozen := registry.items, registry.names, registry.frozen.Load()
	registry.items = bs
	registry.names = map[string]bool{}
	for _, b := range bs {
		registry.names[b.name] = true
	}
	registry.frozen.Store(false)
	t.Cleanup(func() {
		registry.items, registry.names = savedReg, savedNames
		registry.frozen.Store(savedFrozen)
	})
}

func TestScheduleDoesNotMutateRegistry(t *testing.T) {
	b := newBuilder()
	b.name = "X"
	stubRegistry(t, b)

	_ = schedule(scheduleParams{})
	if b.validation || len(b.postHooks) != 0 || b.topology != TopologyL2Only {
		t.Fatalf("schedule mutated the shared registry builder: %+v", b)
	}
}

func TestScheduleGlobFilter(t *testing.T) {
	eth := newBuilder()
	eth.name = "TransferEth"
	erc := newBuilder()
	erc.name = "TransferErc"
	deploy := newBuilder()
	deploy.name = "Deploy"
	stubRegistry(t, eth, erc, deploy)

	out := schedule(scheduleParams{Tests: map[string]bool{"Transfer*": true, "Deploy": true}})
	if len(out) != 3 {
		t.Fatalf("got %d scheduled tests, want 3 (glob + exact)", len(out))
	}
}

func TestScheduleRejectsDeadPattern(t *testing.T) {
	b := newBuilder()
	b.name = "Keep"
	stubRegistry(t, b)

	defer func() {
		if recover() == nil {
			t.Fatal("want panic for tests pattern matching no registered test")
		}
	}()
	schedule(scheduleParams{Tests: map[string]bool{"Keep": true, "Gone": true}})
}

func TestScheduleRejectsDeadCategory(t *testing.T) {
	b := newBuilder()
	b.name = "Keep"
	stubRegistry(t, b)

	defer func() {
		if recover() == nil {
			t.Fatal("want panic for category with no registered tests")
		}
	}()
	schedule(scheduleParams{Categories: map[string]bool{"ghost": true}})
}

func TestScheduleSortsHeaviestFirst(t *testing.T) {
	light1 := newBuilder()
	light1.name = "Light1"
	heavy := newBuilder()
	heavy.name = "Heavy"
	WithMultiNode()(heavy)
	light2 := newBuilder()
	light2.name = "Light2"
	stakingValidation := newBuilder()
	stakingValidation.name = "Max"
	WithStakingValidation()(stakingValidation)
	stubRegistry(t, light1, heavy, light2, stakingValidation)

	out := schedule(scheduleParams{})
	if len(out) != 4 {
		t.Fatalf("got %d scheduled tests, want 4", len(out))
	}
	for i := 1; i < len(out); i++ {
		if out[i-1].Spec.Weight < out[i].Spec.Weight {
			t.Fatalf("not heaviest-first at index %d", i)
		}
	}
	if out[0].Spec.Name != "Max" {
		t.Fatalf("want Max first, got %q", out[0].Spec.Name)
	}
	if out[2].Spec.Name != "Light1" || out[3].Spec.Name != "Light2" {
		t.Fatalf("equal-weight tests not stable: got %q, %q", out[2].Spec.Name, out[3].Spec.Name)
	}
}

func TestNamedResolvesBeforeDerivation(t *testing.T) {
	stubRegistry(t)

	closure := func() Scenario { return func(*Env) {} }()
	Test(closure, Named("X"))
	Test(testRunFoo)
	if !registry.names["X"] || !registry.names["Foo"] {
		t.Fatalf("want registrations X and Foo, got %v", registry.names)
	}
}

func TestRegistryFreezePanicsOnLateRegister(t *testing.T) {
	saved := registry.frozen.Load()
	registry.frozen.Store(true)
	t.Cleanup(func() {
		registry.frozen.Store(saved)
	})

	mustPanic(t, "Test called after schedule()", func() {
		Test(testRunFoo, Named("LateRegisteredScenario"))
	})
}

func TestCLIPinsReachScheduledSpec(t *testing.T) {
	b := newBuilder()
	b.name = "Pinned"
	stubRegistry(t, b)

	out := schedule(scheduleParams{ArbOS: containers.Some(params.ArbosVersion_41), StateScheme: containers.Some(StateSchemeHash), DBEngine: containers.Some(DBEnginePebble)})
	if len(out) != 1 {
		t.Fatalf("scheduled %d tests, want 1", len(out))
	}
	if s := out[0].Spec; s.ArbOSVersion.UnwrapOr(0) != params.ArbosVersion_41 || s.StateScheme.UnwrapOr("") != StateSchemeHash || s.DBEngine.UnwrapOr("") != DBEnginePebble {
		t.Fatalf("params pins did not reach Spec: %+v", s)
	}
}

func TestDuplicateRegistrationPanics(t *testing.T) {
	stubRegistry(t)

	Test(testRunFoo, Named("Dup"))
	mustPanic(t, "duplicate", func() {
		Test(testRunFoo, Named("Dup"))
	})
}

func TestCategoryEnabled(t *testing.T) {
	cases := []struct {
		name string
		p    scheduleParams
		cat  string
		want bool
	}{
		{"default cat with empty params", scheduleParams{}, "default", true},
		{"default cat excluded when only challenge listed", scheduleParams{Categories: map[string]bool{"challenge": true}}, "default", false},
		{"challenge enabled when listed", scheduleParams{Categories: map[string]bool{"challenge": true}}, "challenge", true},
		{"no filter runs any category", scheduleParams{}, "challenge", true},
		{"unknown category disabled", scheduleParams{Categories: map[string]bool{"challenge": true}}, "stylus", false},
		{"AllCategories enables any", scheduleParams{AllCategories: true}, "stylus", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := c.p.categoryEnabled(c.cat)
			if got != c.want {
				t.Errorf("categoryEnabled(%q) = %v, want %v", c.cat, got, c.want)
			}
		})
	}
}

func TestResolvedSchemePrecedence(t *testing.T) {
	p := scheduleParams{StateScheme: containers.Some(StateSchemeHash), DefaultStateScheme: containers.Some(StateSchemePath)}
	if got := p.resolvedScheme(); got.UnwrapOr("") != StateSchemeHash {
		t.Fatalf("resolvedScheme = %q, want explicit pin %q over env default", got.UnwrapOr(""), StateSchemeHash)
	}
	p = scheduleParams{DefaultStateScheme: containers.Some(StateSchemePath)}
	if got := p.resolvedScheme(); got.UnwrapOr("") != StateSchemePath {
		t.Fatalf("resolvedScheme = %q, want env default %q", got.UnwrapOr(""), StateSchemePath)
	}
}

func TestMaxWeight(t *testing.T) {
	cases := []struct {
		name  string
		items []scheduledTest
		want  weight
	}{
		{"empty", nil, 0},
		{"all light", []scheduledTest{{Spec: Spec{Weight: weightLight}}, {Spec: Spec{Weight: weightLight}}}, weightLight},
		{"mixed", []scheduledTest{{Spec: Spec{Weight: weightLight}}, {Spec: Spec{Weight: weightHeavy}}, {Spec: Spec{Weight: weightMedium}}}, weightHeavy},
		{"max present", []scheduledTest{{Spec: Spec{Weight: weightLight}}, {Spec: Spec{Weight: weightMax}}}, weightMax},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := maxWeight(c.items); got != c.want {
				t.Fatalf("maxWeight = %d, want %d", got, c.want)
			}
		})
	}
}
