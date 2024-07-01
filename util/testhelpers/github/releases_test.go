package github

import (
	"context"
	"testing"
)

func TestReleases(t *testing.T) {
	rels, err := NitroReleases(context.Background())
	if err != nil {
		t.Error(err)
	}
	if len(rels) == 0 {
		t.Error("No releases found")
	}
	if len(rels) != 50 {
		t.Errorf("Expected 50 releases, got %d", len(rels))
	}
}

func TestLatestConsensusRelease(t *testing.T) {
	rel, err := LatestConsensusRelease(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if rel == nil {
		t.Fatal("No consensus release found")
	}
	if rel.WavmModuleRoot == "" {
		t.Error("Unexpected empty WAVM module root.")
	}
	if rel.MachineWavmURL.String() == "" {
		t.Error("Unexpected empty machine WAVM URL.")
	}
	if rel.ReplayWasmURL.String() == "" {
		t.Error("Unexpected empty replay WASM URL.")
	}
}
