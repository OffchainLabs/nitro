// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package programs

import (
	"testing"

	"github.com/stretchr/testify/require"

	gethParams "github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbos/burn"
	"github.com/offchainlabs/nitro/arbos/storage"
)

// TestStylusParamsActivationGasRoundTrip verifies that ActivationGas survives a
// Save/Params round-trip and that the slot-alignment padding does not corrupt
// adjacent fields (MaxFragmentCount lives in the same slot 0).
func TestStylusParamsActivationGasRoundTrip(t *testing.T) {
	sto := storage.NewMemoryBacked(burn.NewSystemBurner(nil, false))
	Initialize(gethParams.ArbosVersion_60, sto)
	progs := Open(gethParams.ArbosVersion_60, sto)

	// freshly initialised value must be zero
	p, err := progs.Params()
	require.NoError(t, err)
	require.Equal(t, uint64(0), p.ActivationGas)

	// write a distinctive non-zero value that fills all 8 bytes
	const testGas = uint64(0x0102030405060708)
	p.ActivationGas = testGas
	require.NoError(t, p.Save())

	// read back and verify exact value
	p2, err := progs.Params()
	require.NoError(t, err)
	require.Equal(t, testGas, p2.ActivationGas)

	// adjacent field in slot 0 must not be corrupted
	require.Equal(t, uint8(initialMaxFragmentCount), p2.MaxFragmentCount)
}

// TestStylusParamsActivationGasZeroOnOlderVersion checks that ActivationGas is
// zero (not read from storage) when the ArbOS version predates the feature.
func TestStylusParamsActivationGasZeroOnOlderVersion(t *testing.T) {
	sto := storage.NewMemoryBacked(burn.NewSystemBurner(nil, false))
	Initialize(gethParams.ArbosVersion_50, sto)
	progs := Open(gethParams.ArbosVersion_50, sto)

	p, err := progs.Params()
	require.NoError(t, err)
	require.Equal(t, uint64(0), p.ActivationGas)
}
