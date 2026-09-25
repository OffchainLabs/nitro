// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

//go:build !wasm

package programs

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

// tryActivate drives the guard at the top of activateProgram. The wasm is
// deliberately invalid: a request the guard rejects returns before reading it,
// and one it admits fails later for an unrelated reason, which is all these
// tests distinguish.
func tryActivate(t *testing.T, allowOffchainActivation bool, runCtx *core.MessageRunContext) error {
	t.Helper()
	db := state.NewDatabaseForTesting()
	db.SetArbNodeConfig(&StylusTargetConfig{
		AllowOffchainActivation: allowOffchainActivation,
	})
	statedb, err := state.New(types.EmptyRootHash, db)
	require.NoError(t, err)
	_, err = activateProgram(
		statedb, common.Address{1}, common.Hash{2}, []byte{0},
		1, 1, params.ArbosVersion_Stylus, false,
		newTestBurner(30_000_000), runCtx,
	)
	return err
}

func TestAllowOffchainActivation(t *testing.T) {
	for _, tc := range []struct {
		name                    string
		allowOffchainActivation bool
		runCtx                  *core.MessageRunContext
		wantAllowed             bool
	}{
		{"disabled/gas estimation", false, core.NewMessageGasEstimationContext(), false},
		{"enabled/gas estimation", true, core.NewMessageGasEstimationContext(), true},
		{"disabled/ethcall", false, core.NewMessageEthcallContext(), false},
		{"enabled/ethcall", true, core.NewMessageEthcallContext(), true},
		{"disabled/commit", false, core.NewMessageCommitContext(nil), true},
		{"enabled/commit", true, core.NewMessageCommitContext(nil), true},
		{"disabled/sequencing", false, core.NewMessageSequencingContext(nil), true},
		{"enabled/sequencing", true, core.NewMessageSequencingContext(nil), true},
		{"disabled/delayed sequencing", false, core.NewMessageDelayedSequencingContext(nil), true},
		{"enabled/delayed sequencing", true, core.NewMessageDelayedSequencingContext(nil), true},
		{"disabled/replay", false, core.NewMessageReplayContext(), true},
		{"enabled/replay", true, core.NewMessageReplayContext(), true},
		{"disabled/recording", false, core.NewMessageRecordingContext(nil), true},
		{"enabled/recording", true, core.NewMessageRecordingContext(nil), true},
		{"disabled/prefetch", false, core.NewMessagePrefetchContext(), true},
		{"enabled/prefetch", true, core.NewMessagePrefetchContext(), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := tryActivate(t, tc.allowOffchainActivation, tc.runCtx)
			if tc.wantAllowed {
				require.NotErrorIs(t, err, ErrOffchainActivationNotAllowed)
			} else {
				require.ErrorIs(t, err, ErrOffchainActivationNotAllowed)
			}
		})
	}
}
