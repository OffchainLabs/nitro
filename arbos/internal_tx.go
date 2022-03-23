// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbos

import (
	"fmt"
	"log"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/arbos/util"
)

// Types of ArbitrumInternalTx, distinguished by the first data byte
const (
	// Contains 8 bytes indicating the big endian L1 block number to set
	arbInternalTxStartBlock uint8 = 0
)

type internalTxStartBlockContents struct {
	Header        *types.Header
	L1BlockNumber uint64
	L1BaseFee     *big.Int
}

func InternalTxStartBlock(
	chainId,
	l1BaseFee *big.Int,
	l1BlockNum,
	l2BlockNum uint64,
	header *types.Header,
) *types.ArbitrumInternalTx {
	data, err := rlp.EncodeToBytes(internalTxStartBlockContents{
		Header:        header,
		L1BlockNumber: l1BlockNum,
		L1BaseFee:     l1BaseFee,
	})
	if err != nil {
		panic(fmt.Sprintf("rlp encoding failure %v", err))
	}
	return &types.ArbitrumInternalTx{
		ChainId:       chainId,
		Type:          arbInternalTxStartBlock,
		Data:          data,
		L2BlockNumber: l2BlockNum,
	}
}

func ApplyInternalTxUpdate(tx *types.ArbitrumInternalTx, state *arbosState.ArbosState, evm *vm.EVM) {

	var contents internalTxStartBlockContents
	err := rlp.DecodeBytes(tx.Data, &contents)
	if err != nil {
		log.Fatal("internal tx failure", err)
	}

	nextL1BlockNumber, err := state.Blockhashes().NextBlockNumber()
	state.Restrict(err)

	if contents.L1BlockNumber >= nextL1BlockNumber {
		var prevHash common.Hash
		if evm.Context.BlockNumber.Sign() > 0 {
			prevHash = evm.Context.GetHash(evm.Context.BlockNumber.Uint64() - 1)
		}
		state.Restrict(state.Blockhashes().RecordNewL1Block(contents.L1BlockNumber, prevHash))
	}

	lastBlockHeader := contents.Header
	if lastBlockHeader == nil {
		return
	}

	// Try to reap 2 retryables
	_ = state.RetryableState().TryToReapOneRetryable(lastBlockHeader.Time, evm, util.TracingDuringEVM)
	_ = state.RetryableState().TryToReapOneRetryable(lastBlockHeader.Time, evm, util.TracingDuringEVM)

	timePassed := state.SetLastTimestampSeen(lastBlockHeader.Time)
	state.L2PricingState().UpdatePricingModel(lastBlockHeader, timePassed, false)
	state.L1PricingState().UpdatePricingModel(contents.L1BaseFee, timePassed)

	state.UpgradeArbosVersionIfNecessary(lastBlockHeader.Time)
}
