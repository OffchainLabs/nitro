//
// Copyright 2021-2022, Offchain Labs, Inc. All rights reserved.
//

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
	l1BlockNum,
	l2BlockNum uint64,
	header *types.Header,
) *types.ArbitrumInternalTx {
	if l1BaseFee == nil {
		l1BaseFee = big.NewInt(0)
	}
	data, err := util.PackInternalTxDataStartBlock(l1BaseFee, header.BaseFee, l1BlockNum, header.Time)
	if err != nil {
		panic(fmt.Sprintf("Failed to pack internal tx %v", err))
	}
	return &types.ArbitrumInternalTx{
		ChainId:       chainId,
		Type:          arbInternalTxStartBlock,
		Data:          data,
		L2BlockNumber: l2BlockNum,
	}
}

func ApplyInternalTxUpdate(tx *types.ArbitrumInternalTx, state *arbosState.ArbosState, evm *vm.EVM) {
	inputs, err := util.UnpackInternalTxDataStartBlock(tx.Data)
	if err != nil {
		panic(err)
	}
	l1BaseFee, _ := inputs[0].(*big.Int)
	l2BaseFee, _ := inputs[1].(*big.Int)
	l1BlockNumber, _ := inputs[2].(uint64)
	timeLastBlock, _ := inputs[3].(uint64)

	nextL1BlockNumber, err := state.Blockhashes().NextBlockNumber()
	state.Restrict(err)

	if l1BlockNumber >= nextL1BlockNumber {
		var prevHash common.Hash
		if evm.Context.BlockNumber.Sign() > 0 {
			prevHash = evm.Context.GetHash(evm.Context.BlockNumber.Uint64() - 1)
		}
		state.Restrict(state.Blockhashes().RecordNewL1Block(l1BlockNumber, prevHash))
	}

	// Try to reap 2 retryables
	_ = state.RetryableState().TryToReapOneRetryable(timeLastBlock, evm, util.TracingDuringEVM)
	_ = state.RetryableState().TryToReapOneRetryable(timeLastBlock, evm, util.TracingDuringEVM)

	timePassed := state.SetLastTimestampSeen(timeLastBlock)
	state.L2PricingState().UpdatePricingModel(l2BaseFee, timePassed, false)
	state.L1PricingState().UpdatePricingModel(l1BaseFee, timePassed)

	state.UpgradeArbosVersionIfNecessary(timeLastBlock)
}
