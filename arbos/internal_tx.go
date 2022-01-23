//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbos

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/offchainlabs/arbstate/arbos/arbosState"
)

// Types of ArbitrumInternalTx, distinguished by the first data byte
const (
	// Contains 8 bytes indicating the big endian L1 block number to set
	arbInternalTxBootArbOS           uint8 = 0
	arbInternalTxUpdateL1BlockNumber uint8 = 1
)

func InternalTxBootArbOS(l2BlockNumber, chainId *big.Int) *types.ArbitrumInternalTx {
	return &types.ArbitrumInternalTx{
		ChainId:     chainId,
		Type:        arbInternalTxBootArbOS,
		Data:        []byte{},
		BlockNumber: l2BlockNumber.Uint64(),
		TxIndex:     0,
	}
}

func InternalTxUpdateL1BlockNumber(l1BlockNumber, l2BlockNumber, chainId *big.Int) *types.ArbitrumInternalTx {
	return &types.ArbitrumInternalTx{
		ChainId:     chainId,
		Type:        arbInternalTxUpdateL1BlockNumber,
		Data:        rlp.AppendUint64([]byte{}, l1BlockNumber.Uint64()),
		BlockNumber: l2BlockNumber.Uint64(),
		TxIndex:     0,
	}
}

func ApplyInternalTxUpdate(tx *types.ArbitrumInternalTx, state *arbosState.ArbosState, blockContext vm.BlockContext) error {
	switch tx.Type {
	case arbInternalTxBootArbOS:
		// no need to do anything, the TxProcessor already initialized ArbOS
		// this is just a dummy tx to allow that to happen
		return nil
	case arbInternalTxUpdateL1BlockNumber:
		var l1BlockNumber uint64
		err := rlp.DecodeBytes(tx.Data, &l1BlockNumber)
		if err != nil {
			return err
		}
		var prevHash common.Hash
		if blockContext.BlockNumber.Sign() > 0 {
			prevHash = blockContext.GetHash(blockContext.BlockNumber.Uint64() - 1)
		}
		return state.Blockhashes().RecordNewL1Block(l1BlockNumber, prevHash)
	default:
		return fmt.Errorf("unknown ArbitrumInternalTx type %v", tx.Type)
	}
}
