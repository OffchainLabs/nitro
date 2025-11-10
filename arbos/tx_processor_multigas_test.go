// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbos

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/arbitrum/multigas"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/vm"

	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
)

func newMockEVMForTestingWithBaseFee(baseFee *big.Int) *vm.EVM {
	chainConfig := chaininfo.ArbitrumDevTestChainConfig()
	_, statedb := arbosState.NewArbosMemoryBackedArbOSStateWithConfig(chainConfig)

	context := vm.BlockContext{
		BaseFee: baseFee,
	}
	evm := vm.NewEVM(context, statedb, chainConfig, vm.Config{})
	return evm
}

func TestStartTxHookReturnsMultigas(t *testing.T) {
	const plentyOfGas = 13_000_000_000_000 // comfortably > 12.9T posterGas

	testCases := []struct {
		name         string
		gasRemaining uint64
		expectErr    error
		expectZeroMG bool
	}{
		{
			name:         "gasRemaining < gasNeededToStartEVM",
			gasRemaining: 1,
			expectErr:    core.ErrIntrinsicGas,
			expectZeroMG: true,
		},
		{
			name:         "charge posterGas",
			gasRemaining: plentyOfGas,
			expectErr:    nil,
			expectZeroMG: false,
		},
	}

	for _, c := range testCases {
		t.Run(c.name, func(t *testing.T) {
			// Base fee > 0 to enable L1-charging branch when not skipping
			evm := newMockEVMForTestingWithBaseFee(big.NewInt(1))

			msg := &core.Message{
				TxRunContext: core.NewMessageReplayContext(),
				GasTipCap:    big.NewInt(1),
				GasFeeCap:    big.NewInt(1),
			}

			txProcessor := NewTxProcessor(evm, msg)

			gasRem := c.gasRemaining
			_, mg, err := txProcessor.GasChargingHook(&gasRem, 0)

			require.Equal(t, c.expectErr, err, "GasChargingHook error mismatch")

			if c.expectZeroMG {
				require.Equal(t, multigas.ZeroGas(), mg, "expected ZeroGas for this case")
			} else {
				require.Greater(t, mg.Get(multigas.ResourceKindL1Calldata), uint64(0), "expected L1Calldata > 0")
			}
		})
	}
}
