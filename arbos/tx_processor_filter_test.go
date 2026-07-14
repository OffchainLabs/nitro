// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbos

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/arbitrum/filter"
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

type recordingCheckerState struct {
	touched []filter.FilteredAddressWithReason
}

func (r *recordingCheckerState) TouchAddress(t *filter.FilteredAddressWithReason) {
	r.touched = append(r.touched, *t)
}

func (r *recordingCheckerState) IsFiltered() (bool, []filter.FilteredAddressRecord) {
	return false, nil
}

func (r *recordingCheckerState) countTouches(addr common.Address, reason filter.FilterReasonType) int {
	count := 0
	for _, t := range r.touched {
		if t.Address == addr && t.Reason == reason {
			count++
		}
	}
	return count
}

// TestEndTxHookTouchesRefundTo verifies that a redeem's RefundTo address is
// reported to the address filter exactly once when it receives a refund, and
// not at all when the refund is fully withheld (MaxRefund = 0).
func TestEndTxHookTouchesRefundTo(t *testing.T) {
	maxUint256 := new(big.Int).Sub(new(big.Int).Exp(common.Big2, common.Big256, nil), common.Big1)

	testCases := []struct {
		name                string
		maxRefund           *big.Int
		submissionFeeRefund *big.Int
		success             bool
		expectTouches       int
	}{
		{
			name:                "positive refund touches RefundTo once",
			maxRefund:           maxUint256,
			submissionFeeRefund: big.NewInt(0),
			success:             true,
			expectTouches:       1,
		},
		{
			name:                "withheld refund does not touch RefundTo",
			maxRefund:           big.NewInt(0),
			submissionFeeRefund: big.NewInt(0),
			success:             true,
			expectTouches:       0,
		},
		{
			name:                "failed redeem still refunds gas and touches RefundTo",
			maxRefund:           maxUint256,
			submissionFeeRefund: big.NewInt(0),
			success:             false,
			expectTouches:       1,
		},
		{
			name:                "multiple refunds touch RefundTo only once",
			maxRefund:           maxUint256,
			submissionFeeRefund: big.NewInt(1e9),
			success:             true,
			expectTouches:       1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			const gasLimit uint64 = 1_000_000
			const gasLeft uint64 = 500_000
			from := common.HexToAddress("0x1111")
			refundTo := common.HexToAddress("0x2222")
			baseFee := big.NewInt(l2pricing.InitialBaseFeeWei)

			chainConfig := chaininfo.ArbitrumDevTestChainConfig()
			_, statedb := arbosState.NewArbosMemoryBackedArbOSStateWithConfig(chainConfig)
			evm := vm.NewEVM(vm.BlockContext{BaseFee: baseFee}, statedb, chainConfig, vm.Config{})

			checker := &recordingCheckerState{}
			statedb.SetAddressCheckerState(checker)

			msg := &core.Message{
				TxRunContext: core.NewMessageReplayContext(),
				From:         from,
				GasLimit:     gasLimit,
				GasPrice:     big.NewInt(0),
				GasFeeCap:    baseFee,
				GasTipCap:    big.NewInt(0),
			}
			txProcessor := NewTxProcessor(evm, msg)
			txProcessor.PosterFee = big.NewInt(0)

			inner := &types.ArbitrumRetryTx{
				From:                from,
				RefundTo:            refundTo,
				GasFeeCap:           baseFee,
				SubmissionFeeRefund: tc.submissionFeeRefund,
				MaxRefund:           tc.maxRefund,
				Value:               big.NewInt(0),
				TicketId:            common.HexToHash("0x01"),
			}
			msg.Tx = types.NewTx(inner)

			// Fund the sender so EndTxHook can undo Geth's unused-gas refund, and
			// the network fee account so it can pay the refund.
			gasRefund := arbmath.BigMulByUint(baseFee, gasLeft)
			util.MintBalance(&from, gasRefund, evm, util.TracingAfterEVM, tracing.BalanceIncreaseDeposit)
			networkFeeAccount, err := txProcessor.state.NetworkFeeAccount()
			require.NoError(t, err)
			util.MintBalance(
				&networkFeeAccount,
				arbmath.BigMulByUint(baseFee, gasLimit*2),
				evm,
				util.TracingAfterEVM,
				tracing.BalanceIncreaseNetworkFee,
			)

			txProcessor.EndTxHook(gasLeft, multigas.ZeroGas(), tc.success)

			require.Equal(t, tc.expectTouches, checker.countTouches(refundTo, filter.ReasonRetryableRefundTo),
				"unexpected number of RefundTo touches")
		})
	}
}
