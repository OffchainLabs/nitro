// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbos

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/arbos/util"
)

// Types of ArbitrumInternalTx, distinguished by the first data byte
const (
	// Contains 8 bytes indicating the big endian L1 block number to set
	arbInternalTxStartBlock uint8 = 0
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
	data, err := util.PackInternalTxDataStartBlock(l1BaseFee, lastHeader.BaseFee, l1BlockNum, timePassed)
	if err != nil {
		panic(fmt.Sprintf("Failed to pack internal tx %v", err))
	}
	return &types.ArbitrumInternalTx{
		ChainId:       chainId,
		SubType:       arbInternalTxStartBlock,
		Data:          data,
		L2BlockNumber: l2BlockNum,
	}
}

func ApplyInternalTxUpdate(tx *types.ArbitrumInternalTx, state *arbosState.ArbosState, evm *vm.EVM) {
	inputs, err := util.UnpackInternalTxDataStartBlock(tx.Data)
	if err != nil {
		panic(err)
	}
	l1BaseFee, _ := inputs[0].(*big.Int)   // current block's
	l2BaseFee, _ := inputs[1].(*big.Int)   // the last L2 block's base fee (which is the result of the calculation 2 blocks ago)
	l1BlockNumber, _ := inputs[2].(uint64) // current block's
	timePassed, _ := inputs[3].(uint64)    // since last block

	nextL1BlockNumber, err := state.Blockhashes().NextBlockNumber()
	state.Restrict(err)

	if state.FormatVersion() >= 3 {
		// The `l2BaseFee` in the tx data is indeed the last block's base fee,
		// however, for the purposes of this function, we need the previous computed base fee.
		// Since the computed base fee takes one block to apply, the last block's base fee
		// is actually two calculations ago. Instead, as of ArbOS format version 3,
		// we use the current state's base fee, which is the result of the last calculation.
		l2BaseFee, err = state.L2PricingState().BaseFeeWei()
		state.Restrict(err)
	}

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
	state.L1PricingState().UpdatePricingModel(l1BaseFee, currentTime)

	state.UpgradeArbosVersionIfNecessary(currentTime, evm.ChainConfig())
}
