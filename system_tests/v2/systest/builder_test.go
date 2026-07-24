// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package systest

import (
	"testing"

	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/util/containers"
)

func TestDerivedName(t *testing.T) {
	tests := []struct {
		name string
		fn   Scenario
		want string
	}{
		{name: "testRun stripped", fn: testRunFoo, want: "Foo"},
		{name: "no prefix, uppercase first", fn: fooBody, want: "FooBody"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := derivedName(tt.fn)
			if got != tt.want {
				t.Fatalf("derivedName: got %q, want %q", got, tt.want)
			}
		})
	}
}

func testRunFoo(*Env) {}

func TestDerivedNamePanicsOnClosure(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("derivedName must panic on a factory closure")
		}
	}()
	factory := func() Scenario { return func(*Env) {} }
	derivedName(factory())
}

func fooBody(*Env) {}

func TestShouldSkip(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(*builder)
		sp         scheduleParams
		wantReason string
	}{
		{
			name:       "no constraints",
			setup:      func(*builder) {},
			sp:         scheduleParams{},
			wantReason: "",
		},
		{
			name: "pin conflicts with params pin",
			setup: func(b *builder) {
				b.arbOS = containers.Some(params.ArbosVersion_31)
			},
			sp:         scheduleParams{ArbOS: containers.Some(params.ArbosVersion_40)},
			wantReason: "ArbOS 31 conflicts with ArbOS pin 40",
		},
		{
			name: "state scheme pin conflicts with params pin",
			setup: func(b *builder) {
				b.stateScheme = containers.Some(StateSchemePath)
			},
			sp:         scheduleParams{StateScheme: containers.Some(StateSchemeHash)},
			wantReason: `state scheme "path" conflicts with state scheme pin "hash"`,
		},
		{
			name: "db engine pin conflicts with params pin",
			setup: func(b *builder) {
				b.dbEngine = containers.Some(DBEngineLevelDB)
			},
			sp:         scheduleParams{DBEngine: containers.Some(DBEnginePebble)},
			wantReason: `db engine "leveldb" conflicts with db engine pin "pebble"`,
		},
		{
			name: "env default scheme not a conflict",
			setup: func(b *builder) {
				b.stateScheme = containers.Some(StateSchemePath)
			},
			sp:         scheduleParams{DefaultStateScheme: containers.Some(StateSchemeHash)},
			wantReason: "",
		},
		{
			name: "SkipOnStateSchemes matches env default",
			setup: func(b *builder) {
				b.skipStateSchemes = []StateScheme{StateSchemePath}
			},
			sp:         scheduleParams{DefaultStateScheme: containers.Some(StateSchemePath)},
			wantReason: `incompatible with state scheme "path"`,
		},
		{
			name: "below MinArbOS",
			setup: func(b *builder) {
				b.arbOS = containers.Some(params.ArbosVersion_20)
				b.minArbOS = params.ArbosVersion_30
			},
			sp:         scheduleParams{},
			wantReason: "requires ArbOS>=30, got 20",
		},
		{
			name: "incompatible state scheme",
			setup: func(b *builder) {
				b.stateScheme = containers.Some(StateSchemePath)
				b.skipStateSchemes = []StateScheme{StateSchemePath}
			},
			sp:         scheduleParams{},
			wantReason: `incompatible with state scheme "path"`,
		},
		{
			name: "above MaxArbOS",
			setup: func(b *builder) {
				b.arbOS = containers.Some(params.ArbosVersion_50)
				b.maxArbOS = params.ArbosVersion_40
			},
			sp:         scheduleParams{},
			wantReason: "requires ArbOS<=40, got 50",
		},
		{
			name: "category not enabled",
			setup: func(b *builder) {
				b.category = "challenge"
			},
			sp:         scheduleParams{Categories: map[string]bool{"default": true}},
			wantReason: `category "challenge" not enabled`,
		},
		{
			name: "multi-entry skipStateSchemes matches second",
			setup: func(b *builder) {
				b.stateScheme = containers.Some(StateSchemeHash)
				b.skipStateSchemes = []StateScheme{StateSchemePath, StateSchemeHash}
			},
			sp:         scheduleParams{},
			wantReason: `incompatible with state scheme "hash"`,
		},
	}
	for _, c := range tests {
		t.Run(c.name, func(t *testing.T) {
			b := newBuilder()
			c.setup(b)
			got := b.shouldSkip(c.sp)
			if got != c.wantReason {
				t.Fatalf("got %q, want %q", got, c.wantReason)
			}
		})
	}
}

func TestOverridesRegisterOnBuilder(t *testing.T) {
	b := newBuilder()
	WithExecConfigOverride(func(*gethexec.Config) {})(b)
	WithNodeConfigOverride(func(*arbnode.Config) {})(b)
	WithStackConfigOverride(func(*node.Config) {})(b)
	WithChainConfigOverride(func(*params.ChainConfig) {})(b)
	if got := len(b.execOverrides); got != 1 {
		t.Errorf("execOverrides: got %d, want 1", got)
	}
	if got := len(b.nodeOverrides); got != 1 {
		t.Errorf("nodeOverrides: got %d, want 1", got)
	}
	if got := len(b.stackOverrides); got != 1 {
		t.Errorf("stackOverrides: got %d, want 1", got)
	}
	if got := len(b.chainConfigOverrides); got != 1 {
		t.Errorf("chainConfigOverrides: got %d, want 1", got)
	}
}

func TestValidateRejectsMinAboveMax(t *testing.T) {
	mustPanic(t, "MinArbOS(40) > MaxArbOS(30)", func() {
		b := newBuilder()
		MinArbOS(params.ArbosVersion_40)(b)
		MaxArbOS(params.ArbosVersion_30)(b)
		b.validate()
	})
}

func TestValidateAllowsValidArbOSRange(t *testing.T) {
	b := newBuilder()
	MinArbOS(params.ArbosVersion_30)(b)
	MaxArbOS(params.ArbosVersion_40)(b)
	b.validate() // must not panic
}
