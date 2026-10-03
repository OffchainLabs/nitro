// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package programs

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
)

func compareArbNodeConfig(t *testing.T, want *StylusTargetConfig, got *StylusTargetConfig) {
	require.NotNil(t, got)
	require.Equal(t, want.Arm64, got.Arm64)
	require.Equal(t, want.Amd64, got.Amd64)
	require.Equal(t, want.Host, got.Host)
	require.Equal(t, want.ExtraArchs, got.ExtraArchs)
	require.Equal(t, want.AllowFallback, got.AllowFallback)
	require.Equal(t, want.MaxOpenPages, got.MaxOpenPages)
	require.Equal(t, want.MaxStylusCallDepth, got.MaxStylusCallDepth)
	require.Equal(t, want.NativeStackSize, got.NativeStackSize)
	require.Equal(t, want.MaxSinglepassOutputSize, got.MaxSinglepassOutputSize)
	require.Equal(t, want.MaxWavmOps, got.MaxWavmOps)
}

func TestGetArbNodeConfig_NilReturnsDefault(t *testing.T) {
	db := state.NewDatabaseForTesting()
	statedb, _ := state.New(types.EmptyRootHash, db)
	compareArbNodeConfig(t, &DefaultStylusTargetConfig, GetStylusConfig(statedb))
}

func TestGetArbNodeConfig_WrongTypeReturnsNil(t *testing.T) {
	db := state.NewDatabaseForTesting()
	db.SetArbNodeConfig("not a *ArbNodeConfig")
	statedb, _ := state.New(types.EmptyRootHash, db)
	compareArbNodeConfig(t, &DefaultStylusTargetConfig, GetStylusConfig(statedb))
}

func TestGetArbNodeConfig_RoundTrips(t *testing.T) {
	db := state.NewDatabaseForTesting()
	want := &StylusTargetConfig{
		Arm64:                   "arm64-test-target",
		Amd64:                   "amd64-test-target",
		Host:                    "host-test-target",
		ExtraArchs:              []string{string(rawdb.TargetWavm), string(rawdb.TargetArm64)},
		AllowFallback:           false,
		MaxOpenPages:            42,
		MaxStylusCallDepth:      5,
		NativeStackSize:         2 * 1024 * 1024,
		MaxSinglepassOutputSize: 12 * 1024 * 1024,
		MaxWavmOps:              1 << 20,
	}
	db.SetArbNodeConfig(want)
	statedb, _ := state.New(types.EmptyRootHash, db)
	compareArbNodeConfig(t, want, GetStylusConfig(statedb))
}
