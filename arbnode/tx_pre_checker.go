// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbnode

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/arbitrum_types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/arbos/l1pricing"
	"github.com/offchainlabs/nitro/util/arbmath"
)

type TxPreChecker struct {
	TransactionPublisher
	bc                  *core.BlockChain
	getStrictness       func() uint
	getRequiredStateAge func() int64
}

func NewTxPreChecker(publisher TransactionPublisher, bc *core.BlockChain, getStrictness func() uint, getRequiredStateAge func() int64) *TxPreChecker {
	return &TxPreChecker{
		TransactionPublisher: publisher,
		bc:                   bc,
		getStrictness:        getStrictness,
		getRequiredStateAge:  getRequiredStateAge,
	}
}

const TxPreCheckerStrictnessNone uint = 0
const TxPreCheckerStrictnessAlwaysCompatible uint = 10
const TxPreCheckerStrictnessLikelyCompatible uint = 20
const TxPreCheckerStrictnessFullValidation uint = 30

type NonceError struct {
	sender     common.Address
	txNonce    uint64
	stateNonce uint64
}

func (e NonceError) Error() string {
	if e.txNonce < e.stateNonce {
		return fmt.Sprintf("%v: address %v, tx: %d state: %d", core.ErrNonceTooLow, e.sender, e.txNonce, e.stateNonce)
	} else if e.txNonce > e.stateNonce {
		return fmt.Sprintf("%v: address %v, tx: %d state: %d", core.ErrNonceTooHigh, e.sender, e.txNonce, e.stateNonce)
	} else {
		// This should be unreachable
		return fmt.Sprintf("invalid nonce error for address %v nonce %v", e.sender, e.txNonce)
	}
}

func (e NonceError) Unwrap() error {
	if e.txNonce < e.stateNonce {
		return core.ErrNonceTooLow
	} else if e.txNonce > e.stateNonce {
		return core.ErrNonceTooHigh
	} else {
		// This should be unreachable
		return nil
	}
}

func MakeNonceError(sender common.Address, txNonce uint64, stateNonce uint64) error {
	if txNonce != stateNonce {
		return NonceError{
			sender:     sender,
			txNonce:    txNonce,
			stateNonce: stateNonce,
		}
	} else {
		return nil
	}
}

func PreCheckTx(bc *core.BlockChain, chainConfig *params.ChainConfig, header *types.Header, statedb *state.StateDB, arbos *arbosState.ArbosState, tx *types.Transaction, options *arbitrum_types.ConditionalOptions, strictness uint, requiredStateAge int64) error {
	if strictness < TxPreCheckerStrictnessAlwaysCompatible {
		return nil
	}
	if tx.Gas() < params.TxGas {
		return core.ErrIntrinsicGas
	}
	sender, err := types.Sender(types.MakeSigner(chainConfig, header.Number), tx)
	if err != nil {
		return err
	}
	baseFee := header.BaseFee
	if strictness < TxPreCheckerStrictnessLikelyCompatible {
		baseFee, err = arbos.L2PricingState().MinBaseFeeWei()
		if err != nil {
			return err
		}
	}
	if arbmath.BigLessThan(tx.GasFeeCap(), baseFee) {
		return fmt.Errorf("%w: address %v, maxFeePerGas: %s baseFee: %s", core.ErrFeeCapTooLow, sender, tx.GasFeeCap(), header.BaseFee)
	}
	stateNonce := statedb.GetNonce(sender)
	if tx.Nonce() < stateNonce {
		return MakeNonceError(sender, tx.Nonce(), stateNonce)
	}
	intrinsic, err := core.IntrinsicGas(tx.Data(), tx.AccessList(), tx.To() == nil, chainConfig.IsHomestead(header.Number), true)
	if err != nil {
		return err
	}
	if tx.Gas() < intrinsic {
		return core.ErrIntrinsicGas
	}
	if strictness < TxPreCheckerStrictnessLikelyCompatible {
		return nil
	}
	balance := statedb.GetBalance(sender)
	cost := tx.Cost()
	if arbmath.BigLessThan(balance, cost) {
		return fmt.Errorf("%w: address %v have %v want %v", core.ErrInsufficientFunds, sender, balance, cost)
	}
	if options != nil {
		l1BlockNumber, err := arbos.Blockhashes().L1BlockNumber()
		if err != nil {
			// TODO
			return err
		}
		if err := options.PreCheck(l1BlockNumber, statedb); err != nil {
			return err
		}
		now := time.Now().Unix()
		oldHeader := header
		// find a block that's old enough
		for now-int64(oldHeader.Time) < requiredStateAge && oldHeader.Number.Uint64() > 0 {
			previousHeader := bc.GetHeader(oldHeader.ParentHash, oldHeader.Number.Uint64()-1)
			if previousHeader == nil {
				break
			}
			oldHeader = previousHeader
		}
		if oldHeader != header {
			secondOldStatedb, err := bc.StateAt(oldHeader.Root)
			if err != nil {
				return err
			}
			if err := options.CheckOnlyStorage(secondOldStatedb); err != nil {
				return err
			}
		}
	}
	if strictness >= TxPreCheckerStrictnessFullValidation && tx.Nonce() > stateNonce {
		return MakeNonceError(sender, tx.Nonce(), stateNonce)
	}
	dataCost, _ := arbos.L1PricingState().GetPosterInfo(tx, l1pricing.BatchPosterAddress)
	dataGas := arbmath.BigDiv(dataCost, header.BaseFee)
	if tx.Gas() < intrinsic+dataGas.Uint64() {
		return core.ErrIntrinsicGas
	}
	return nil
}

func (c *TxPreChecker) PublishTransaction(ctx context.Context, tx *types.Transaction, options *arbitrum_types.ConditionalOptions) error {
	block := c.bc.CurrentBlock()
	statedb, err := c.bc.StateAt(block.Root())
	if err != nil {
		return err
	}
	arbos, err := arbosState.OpenSystemArbosState(statedb, nil, true)
	if err != nil {
		return err
	}
	err = PreCheckTx(c.bc, c.bc.Config(), block.Header(), statedb, arbos, tx, options, c.getStrictness(), c.getRequiredStateAge())
	if err != nil {
		return err
	}
	return c.TransactionPublisher.PublishTransaction(ctx, tx, options)
}
