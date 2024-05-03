// Copyright 2023-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

package server_common

import (
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
)

var (
	wantLatestModuleRoot = "0xf4389b835497a910d7ba3ebfb77aa93da985634f3c052de1290360635be40c4a"
	wantModuleRoots      = []string{
		"0x8b104a2e80ac6165dc58b9048de12f301d70b02a0ab51396c22b4b4b802a16a4",
		"0x68e4fe5023f792d4ef584796c84d710303a5e12ea02d6e37e2b5e9c4332507c4",
		"0xf4389b835497a910d7ba3ebfb77aa93da985634f3c052de1290360635be40c4a",
	}
)

func TestNewMachineLocator(t *testing.T) {
	ml, err := NewMachineLocator("testdata")
	if err != nil {
		t.Fatalf("Error creating new machine locator: %v", err)
	}
	if ml.latest.Hex() != wantLatestModuleRoot {
		t.Errorf("NewMachineLocator() got latestModuleRoot: %v, want: %v", ml.latest, wantLatestModuleRoot)
	}
	var got []string
	for _, s := range ml.ModuleRoots() {
		got = append(got, s.Hex())
	}
	sort.Strings(got)
	sort.Strings(wantModuleRoots)
	if diff := cmp.Diff(got, wantModuleRoots); diff != "" {
		t.Errorf("NewMachineLocator() unexpected diff (-want +got):\n%s", diff)
	}
}
