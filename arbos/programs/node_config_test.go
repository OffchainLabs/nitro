// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package programs

import (
	"strconv"
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/core"
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
	require.Equal(t, want.AllowOffchainActivation, got.AllowOffchainActivation)
	require.Equal(t, want.MaxOpenPages, got.MaxOpenPages)
	require.Equal(t, want.MaxStylusCallDepth, got.MaxStylusCallDepth)
	require.Equal(t, want.MaxStorageCacheSlots, got.MaxStorageCacheSlots)
	require.Equal(t, want.NativeStackSize, got.NativeStackSize)
	require.Equal(t, want.MaxSinglepassOutputSize, got.MaxSinglepassOutputSize)
	require.Equal(t, want.MaxWavmOps, got.MaxWavmOps)
}

func TestGetStylusConfig_NilReturnsDefault(t *testing.T) {
	db := state.NewDatabaseForTesting()
	statedb, _ := state.New(types.EmptyRootHash, db)
	compareArbNodeConfig(t, &DefaultStylusTargetConfig, GetStylusConfig(statedb))
}

func TestDefaultStorageCacheLimitIsDisabled(t *testing.T) {
	require.Zero(t, DefaultStylusTargetConfig.MaxStorageCacheSlots)
}

func TestMaxSinglepassOutputSizeFlag(t *testing.T) {
	flags := pflag.NewFlagSet(t.Name(), pflag.ContinueOnError)
	StylusTargetConfigAddOptions("stylus-target", flags)
	const limit uint64 = 12 * 1024 * 1024
	require.NoError(t, flags.Parse([]string{
		"--stylus-target.max-singlepass-output-size=" + strconv.FormatUint(limit, 10),
	}))
	got, err := flags.GetUint64("stylus-target.max-singlepass-output-size")
	require.NoError(t, err)
	require.Equal(t, limit, got)
}

func TestMaxSinglepassOutputSizeZeroDisablesLimit(t *testing.T) {
	cfg := DefaultStylusTargetConfig
	cfg.MaxSinglepassOutputSize = 0
	require.NoError(t, cfg.Validate())
	require.Zero(t, cfg.MaxSinglepassOutputSize)
}

func TestGetStylusConfig_WrongTypeReturnsNil(t *testing.T) {
	db := state.NewDatabaseForTesting()
	statedb, _ := state.New(types.EmptyRootHash, db)
	statedb.Database().CodeDB().SetArbNodeConfig("not a *StylusTargetConfig")
	compareArbNodeConfig(t, &DefaultStylusTargetConfig, GetStylusConfig(statedb))
}

func TestGetStylusConfig_RoundTrips(t *testing.T) {
	db := state.NewDatabaseForTesting()
	want := &StylusTargetConfig{
		Arm64:                   "arm64-test-target",
		Amd64:                   "amd64-test-target",
		Host:                    "host-test-target",
		ExtraArchs:              []string{string(rawdb.TargetWavm), string(rawdb.TargetArm64)},
		AllowFallback:           false,
		MaxOpenPages:            42,
		MaxStylusCallDepth:      5,
		MaxStorageCacheSlots:    1234,
		NativeStackSize:         2 * 1024 * 1024,
		MaxSinglepassOutputSize: 12 * 1024 * 1024,
		MaxWavmOps:              1 << 20,
	}
	statedb, _ := state.New(types.EmptyRootHash, db)
	statedb.Database().CodeDB().SetArbNodeConfig(want)
	compareArbNodeConfig(t, want, GetStylusConfig(statedb))
}

func TestStylusStorageCacheLimitRunModes(t *testing.T) {
	db := state.NewDatabaseForTesting()
	statedb, _ := state.New(types.EmptyRootHash, db)
	statedb.Database().CodeDB().SetArbNodeConfig(&StylusTargetConfig{MaxStorageCacheSlots: 1234})

	tests := []struct {
		name string
		ctx  *core.MessageRunContext
		want uint32
	}{
		{"nil", nil, 0},
		{"eth call", core.NewMessageEthcallContext(), 1234},
		{"gas estimation", core.NewMessageGasEstimationContext(), 1234},
		{"sequencing", core.NewMessageSequencingContext(nil), 1234},
		{"commit", core.NewMessageCommitContext(nil), 0},
		{"delayed sequencing", core.NewMessageDelayedSequencingContext(nil), 0},
		{"replay", core.NewMessageReplayContext(), 0},
		{"recording", core.NewMessageRecordingContext(nil), 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.want, stylusStorageCacheLimit(statedb, test.ctx))
		})
	}
}
