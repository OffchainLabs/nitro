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
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"

	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/arbos/l2pricing"
	"github.com/offchainlabs/nitro/arbos/util"
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
	err := txProcessor.state.L2PricingState().CommitCurrentMultiGasFees()
	require.NoError(t, err)

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

func TestEndTxHookMultiGasRefundRetryableTx(t *testing.T) {
	const gasLimit uint64 = 1_000_000
	const gasLeft uint64 = 0

	from := common.HexToAddress("0x1111")
	refundTo := common.HexToAddress("0x2222")

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

	require.True(t, evm.StateDB.GetBalance(from).IsZero())
	require.True(t, evm.StateDB.GetBalance(refundTo).IsZero())

	gasUsed := gasLimit - gasLeft
	usedMultiGas := multigas.MultiGasFromPairs(
		multigas.Pair{Kind: multigas.ResourceKindComputation, Amount: gasUsed / 2},
		multigas.Pair{Kind: multigas.ResourceKindStorageAccess, Amount: gasUsed / 2},
	)

	// Set up multi-gas constraints and spin model to produce a different multi-dimensional cost.
	pricing := txProcessor.state.L2PricingState()
	pricing.ArbosVersion = l2pricing.ArbosMultiGasConstraintsVersion

	Require(t, pricing.AddMultiGasConstraint(
		100000,
		10,
		200000000000,
		map[uint8]uint64{
			uint8(multigas.ResourceKindComputation):   1,
			uint8(multigas.ResourceKindStorageGrowth): 10,
		},
	))
	pricing.UpdatePricingModel(100)
	err := txProcessor.state.L2PricingState().CommitCurrentMultiGasFees()
	require.NoError(t, err)

	baseFee, err := pricing.BaseFeeWei()
	require.NoError(t, err)

	// Align the EVM block basefee with the pricing state's base fee.
	evm.Context.BaseFee = new(big.Int).Set(baseFee)

	// For retryables, simple gas price is GasFeeCap. Use the same as BaseFeeWei.
	gasFeeCap := new(big.Int).Set(baseFee)
	simpleGasCost := new(big.Int).Mul(gasFeeCap, new(big.Int).SetUint64(gasUsed))

	multiDimensionalCost, err := pricing.MultiDimensionalPriceForRefund(usedMultiGas)
	require.NoError(t, err)

	expectedRefund := new(big.Int).Sub(simpleGasCost, multiDimensionalCost)
	require.True(t, expectedRefund.Sign() > 0, "expected refund to be positive", expectedRefund)

	// Big MaxRefund so the full multi-gas refund can go to RefundTo.
	maxRefund := new(big.Int).Mul(expectedRefund, big.NewInt(10))

	inner := &types.ArbitrumRetryTx{
		From:                from,
		RefundTo:            refundTo,
		GasFeeCap:           gasFeeCap,
		SubmissionFeeRefund: big.NewInt(0),
		MaxRefund:           maxRefund,
		Value:               big.NewInt(0),
		TicketId:            common.HexToHash("0x01"),
	}
	retryTx := types.NewTx(inner)
	msg.Tx = retryTx

	// Pre-fund network fee account so refund(...) can pay the user.
	networkFeeAccount, err := txProcessor.state.NetworkFeeAccount()
	require.NoError(t, err)

	util.MintBalance(
		&networkFeeAccount,
		new(big.Int).Mul(expectedRefund, big.NewInt(2)), // plenty
		evm,
		util.TracingAfterEVM,
		tracing.BalanceIncreaseNetworkFee,
	)

	refundToBefore := evm.StateDB.GetBalance(refundTo).ToBig()
	fromBefore := evm.StateDB.GetBalance(from).ToBig()

	// Retryable path check
	txProcessor.EndTxHook(gasLeft, usedMultiGas, true)

	refundToAfter := evm.StateDB.GetBalance(refundTo).ToBig()
	fromAfter := evm.StateDB.GetBalance(from).ToBig()

	refundToDelta := new(big.Int).Sub(refundToAfter, refundToBefore)
	fromDelta := new(big.Int).Sub(fromAfter, fromBefore)

	// Expect:
	// - SubmissionFeeRefund = 0
	// - gasLeft = 0 => gasRefund = 0
	// - MaxRefund is large enough, so the entire multi-gas refund goes to RefundTo, none to From.
	require.True(t, fromDelta.Sign() == 0, "expected no refund to From, got %v", fromDelta)
	require.True(
		t,
		arbmath.BigEquals(expectedRefund, refundToDelta),
		"unexpected multi-gas refund to RefundTo: got %v, want %v",
		refundToDelta,
		expectedRefund,
	)
}
