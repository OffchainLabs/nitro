// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package systest

import "testing"

func TestStateSchemeValid(t *testing.T) {
	cases := map[StateScheme]bool{
		"":                  false,
		StateSchemeHash:     true,
		StateSchemePath:     true,
		"banana":            false,
		StateScheme("HASH"): false, // case-sensitive
	}
	for s, want := range cases {
		if got := s.Valid(); got != want {
			t.Errorf("(%q).Valid() = %v, want %v", s, got, want)
		}
	}
}

func TestDBEngineValid(t *testing.T) {
	cases := map[DBEngine]bool{
		"":                  false,
		DBEnginePebble:      true,
		DBEngineLevelDB:     true,
		DBEngineInMemory:    true,
		DBEngine("rocksdb"): false,
	}
	for e, want := range cases {
		if got := e.Valid(); got != want {
			t.Errorf("(%q).Valid() = %v, want %v", e, got, want)
		}
	}
}
