// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbos

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/arbitrum/multigas"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/vm"

	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/arbos/l2pricing"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/util/arbmath"
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

func TestEndTxHookMultiGasRefundNormalTx(t *testing.T) {
	const gasLimit uint64 = 1000000
	const gasLeft uint64 = 0
	from := common.HexToAddress("0x1234")

	evm := newMockEVMForTestingWithBaseFee(big.NewInt(l2pricing.InitialBaseFeeWei))

	msg := &core.Message{
		TxRunContext: core.NewMessageReplayContext(),
		From:         from,
		GasLimit:     gasLimit,
		GasPrice:     big.NewInt(0),
		GasFeeCap:    big.NewInt(1),
		GasTipCap:    big.NewInt(0),
	}

	txProcessor := NewTxProcessor(evm, msg)
	txProcessor.PosterFee = big.NewInt(0)

	initialBalance := evm.StateDB.GetBalance(from)
	require.True(t, initialBalance.IsZero())

	gasUsed := gasLimit - gasLeft

	// Distribute used gas equally between computation and storage access.
	usedMultiGas := multigas.MultiGasFromPairs(
		multigas.Pair{Kind: multigas.ResourceKindComputation, Amount: gasUsed / 2},
		multigas.Pair{Kind: multigas.ResourceKindStorageAccess, Amount: gasUsed / 2},
	)

	// Set up multi-gas constraints and spin model to produce different multi-dimensional cost.
	txProcessor.state.L2PricingState().ArbosVersion = l2pricing.ArbosMultiGasConstraintsVersion

	Require(t, txProcessor.state.L2PricingState().AddMultiGasConstraint(
		100000,
		10,
		200000000000,
		map[uint8]uint64{
			uint8(multigas.ResourceKindComputation):   1,
			uint8(multigas.ResourceKindStorageGrowth): 10,
		},
	))
	txProcessor.state.L2PricingState().UpdatePricingModel(100)

	baseFee, err := txProcessor.state.L2PricingState().BaseFeeWei()
	require.NoError(t, err)

	// Align the EVM block basefee with the pricing state's min basefee.
	evm.Context.BaseFee = new(big.Int).Set(baseFee)

	singleGasCost := new(big.Int).Mul(baseFee, new(big.Int).SetUint64(gasUsed))

	multiDimensionalCost, err := txProcessor.state.L2PricingState().MultiDimensionalPriceForRefund(usedMultiGas)
	require.NoError(t, err)

	expectedRefund := new(big.Int).Sub(singleGasCost, multiDimensionalCost)
	require.True(t, expectedRefund.Sign() > 0, "expected refund to be positive", expectedRefund)

	txProcessor.EndTxHook(gasLeft, usedMultiGas, true)

	finalBalance := evm.StateDB.GetBalance(from)
	require.True(
		t,
		arbmath.BigEquals(expectedRefund, finalBalance.ToBig()),
		"unexpected multi-gas refund amount to sender: got %v, want %v",
		finalBalance.ToBig(),
		expectedRefund,
	)
}
