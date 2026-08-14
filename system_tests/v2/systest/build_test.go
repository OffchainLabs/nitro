// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package systest

import (
	"testing"

	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/util/containers"
)

func TestChainConfigForSpec(t *testing.T) {
	if cc := chainConfigForSpec(Spec{}); cc.ArbitrumChainParams.InitialArbOSVersion != defaultArbOSVersion() {
		t.Fatalf("unset ArbOS: got %d, want default %d", cc.ArbitrumChainParams.InitialArbOSVersion, defaultArbOSVersion())
	}
	if cc := chainConfigForSpec(Spec{ArbOSVersion: containers.Some(params.ArbosVersion_40)}); cc.ArbitrumChainParams.InitialArbOSVersion != params.ArbosVersion_40 {
		t.Fatalf("pinned ArbOS: got %d, want 40", cc.ArbitrumChainParams.InitialArbOSVersion)
	}
}

func TestStackConfigForSpec(t *testing.T) {
	def := stackConfigForSpec(t, Spec{})
	if def.HTTPHost != "" || def.WSHost != "" {
		t.Fatalf("hosts must stay unset without ExposeRPC: %q %q", def.HTTPHost, def.WSHost)
	}
	engine := DBEnginePebble
	if def.DBEngine == string(engine) {
		engine = DBEngineLevelDB
	}
	if got := stackConfigForSpec(t, Spec{DBEngine: containers.Some(engine)}).DBEngine; got != string(engine) {
		t.Fatalf("pinned db engine: got %q, want %q", got, engine)
	}
	rpc := stackConfigForSpec(t, Spec{ExposeRPC: true})
	if rpc.HTTPHost != "127.0.0.1" || rpc.WSHost != "127.0.0.1" {
		t.Fatalf("ExposeRPC hosts: got %q %q", rpc.HTTPHost, rpc.WSHost)
	}
}

func TestRollbackGuard(t *testing.T) {
	var ran int
	var rb rollbackGuard
	rb.stage(func() { ran++ })
	rb.run() // uncommitted → fires
	if ran != 1 {
		t.Fatalf("uncommitted run should fire once, got %d", ran)
	}
	rb.commit()
	rb.run() // committed → quiet
	if ran != 1 {
		t.Fatalf("committed run must not fire, got %d", ran)
	}
	var empty rollbackGuard
	empty.run() // no stage → no panic
}
