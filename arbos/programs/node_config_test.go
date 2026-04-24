// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package programs

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
)

func TestGetArbNodeConfig_NilReturnsNil(t *testing.T) {
	db := state.NewDatabaseForTesting()
	statedb, _ := state.New(types.EmptyRootHash, db)
	require.Nil(t, GetArbNodeConfig(statedb), "never-set config should return nil")
}

func TestGetArbNodeConfig_WrongTypeReturnsNil(t *testing.T) {
	db := state.NewDatabaseForTesting()
	db.SetArbNodeConfig("not a *ArbNodeConfig")
	statedb, _ := state.New(types.EmptyRootHash, db)
	require.Nil(t, GetArbNodeConfig(statedb),
		"wrong-type storage is a wiring bug; fail-open is safe because all limits "+
			"gated by this config are off-chain only")
}

func TestGetArbNodeConfig_RoundTrips(t *testing.T) {
	db := state.NewDatabaseForTesting()
	want := &ArbNodeConfig{MaxOpenPages: 42, MaxStylusCallDepth: 5}
	db.SetArbNodeConfig(want)
	statedb, _ := state.New(types.EmptyRootHash, db)
	got := GetArbNodeConfig(statedb)
	require.NotNil(t, got)
	require.Equal(t, want.MaxOpenPages, got.MaxOpenPages)
	require.Equal(t, want.MaxStylusCallDepth, got.MaxStylusCallDepth)
}

// Validate() only logs — we just assert it doesn't panic across the cases that
// trigger each log branch, including the edge between warn threshold and safe.
func TestArbNodeConfig_Validate_DoesNotPanic(t *testing.T) {
	cases := []ArbNodeConfig{
		{},                      // all zero — all checks skipped
		{MaxOpenPages: 1},       // below open-pages warn threshold
		{MaxOpenPages: 128},     // above open-pages warn threshold
		{MaxStylusCallDepth: 1}, // below call-depth warn threshold
		{MaxStylusCallDepth: 2}, // exactly at call-depth warn threshold
		{MaxStylusCallDepth: 8}, // above call-depth warn threshold
		{MaxOpenPages: 128, MaxStylusCallDepth: 8}, // both set
	}
	for _, c := range cases {
		c.Validate()
	}
}
