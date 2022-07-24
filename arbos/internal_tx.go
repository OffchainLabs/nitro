// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbos

import (
	"fmt"
	"math/big"

	"github.com/offchainlabs/nitro/util/arbmath"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/arbos/util"
)

func InternalTxStartBlock(
	chainId,
	l1BaseFee *big.Int,
	l1BlockNum uint64,
	header,
	lastHeader *types.Header,
) *types.ArbitrumInternalTx {

	l2BlockNum := header.Number.Uint64()
	timePassed := header.Time - lastHeader.Time

	if l1BaseFee == nil {
		l1BaseFee = big.NewInt(0)
	}
	data, err := util.PackInternalTxDataStartBlock(l1BaseFee, l1BlockNum, l2BlockNum, timePassed)
	if err != nil {
		panic(fmt.Sprintf("Failed to pack internal tx %v", err))
	}
	return &types.ArbitrumInternalTx{
		ChainId: chainId,
		Data:    data,
	}
}

func ApplyInternalTxUpdate(tx *types.ArbitrumInternalTx, state *arbosState.ArbosState, evm *vm.EVM) {
	switch *(*[4]byte)(tx.Data[:4]) {
	case InternalTxStartBlockMethodID:
		inputs, err := util.UnpackInternalTxDataStartBlock(tx.Data)
		if err != nil {
			panic(err)
		}
		l1BlockNumber, _ := inputs[1].(uint64) // current block's
		timePassed, _ := inputs[2].(uint64)    // actually the L2 block number, fixed in upgrade 3
		if state.FormatVersion() >= 3 {
			timePassed, _ = inputs[3].(uint64) // time passed since last block
		}

		nextL1BlockNumber, err := state.Blockhashes().NextBlockNumber()
		state.Restrict(err)

		l2BaseFee, err := state.L2PricingState().BaseFeeWei()
		state.Restrict(err)

		if l1BlockNumber >= nextL1BlockNumber {
			var prevHash common.Hash
			if evm.Context.BlockNumber.Sign() > 0 {
				prevHash = evm.Context.GetHash(evm.Context.BlockNumber.Uint64() - 1)
			}
			state.Restrict(state.Blockhashes().RecordNewL1Block(l1BlockNumber, prevHash))
		}

		currentTime := evm.Context.Time.Uint64()

		// Try to reap 2 retryables
		_ = state.RetryableState().TryToReapOneRetryable(currentTime, evm, util.TracingDuringEVM)
		_ = state.RetryableState().TryToReapOneRetryable(currentTime, evm, util.TracingDuringEVM)

		state.L2PricingState().UpdatePricingModel(l2BaseFee, timePassed, false)

		state.UpgradeArbosVersionIfNecessary(currentTime)
	case InternalTxBatchPostingReportMethodID:
		inputs, err := util.UnpackInternalTxDataBatchPostingReport(tx.Data)
		if err != nil {
			panic(err)
		}
		batchTimestamp, _ := inputs[0].(*big.Int)
		batchPosterAddress, _ := inputs[1].(common.Address)
		// ignore input[2], batchNumber, which exist because we might need them in the future
		batchDataGas, _ := inputs[3].(uint64)
		l1BaseFeeWei, _ := inputs[4].(*big.Int)

		l1p := state.L1PricingState()
		perBatchGas, err := l1p.PerBatchGasCost()
		if err != nil {
			log.Warn("L1Pricing PerBatchGas failed", "err", err)
		}
		gasSpent := arbmath.SaturatingAdd(perBatchGas, arbmath.SaturatingCast(batchDataGas))
		weiSpent := arbmath.BigMulByUint(l1BaseFeeWei, arbmath.SaturatingUCast(gasSpent))
		err = l1p.UpdateForBatchPosterSpending(
			evm.StateDB,
			evm,
			state.FormatVersion(),
			batchTimestamp.Uint64(),
			evm.Context.Time.Uint64(),
			batchPosterAddress,
			weiSpent,
			l1BaseFeeWei,
			util.TracingDuringEVM,
		)
		if err != nil {
			log.Warn("L1Pricing UpdateForSequencerSpending failed", "err", err)
		}
	}
}
