// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbnode

import (
	"context"

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
	bc         *core.BlockChain
	strictness uint
}

func NewTxPreChecker(publisher TransactionPublisher, bc *core.BlockChain, strictness uint) *TxPreChecker {
	return &TxPreChecker{
		TransactionPublisher: publisher,
		bc:                   bc,
		strictness:           strictness,
	}
}

const TxPreCheckerStrictnessNone uint = 0
const TxPreCheckerStrictnessAlwaysCompatible uint = 10
const TxPreCheckerStrictnessLikelyCompatible uint = 20
const TxPreCheckerStrictnessFullValidation uint = 30

func PreCheckTx(chainConfig *params.ChainConfig, header *types.Header, statedb *state.StateDB, arbos *arbosState.ArbosState, tx *types.Transaction, strictness uint) error {
	if strictness < TxPreCheckerStrictnessAlwaysCompatible {
		return nil
	}
	if tx.Gas() < params.TxGas {
		return core.ErrIntrinsicGas
	}
	sender, err := types.Sender(types.MakeSigner(chainConfig, header.Number), tx)
	if err != nil {
		return core.ErrInvalidSender
	}
	baseFee := header.BaseFee
	if strictness < TxPreCheckerStrictnessLikelyCompatible {
		baseFee, err = arbos.L2PricingState().MinBaseFeeWei()
		if err != nil {
			return err
		}
	}
	if arbmath.BigLessThan(tx.GasFeeCap(), baseFee) {
		return core.ErrUnderpriced
	}
	stateNonce := statedb.GetNonce(sender)
	if tx.Nonce() < stateNonce {
		return core.ErrNonceTooLow
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
		return core.ErrInsufficientFunds
	}
	if strictness >= TxPreCheckerStrictnessFullValidation && tx.Nonce() > stateNonce {
		return core.ErrNonceTooHigh
	}
	dataCost, _ := arbos.L1PricingState().GetPosterInfo(tx, l1pricing.BatchPosterAddress)
	dataGas := arbmath.BigDiv(dataCost, header.BaseFee)
	if tx.Gas() < intrinsic+dataGas.Uint64() {
		return core.ErrIntrinsicGas
	}
	return nil
}

func (c *TxPreChecker) PublishTransaction(ctx context.Context, tx *types.Transaction) error {
	block := c.bc.CurrentBlock()
	statedb, err := c.bc.StateAt(block.Root())
	if err != nil {
		return err
	}
	arbos, err := arbosState.OpenSystemArbosState(statedb, nil, true)
	if err != nil {
		return err
	}
	err = PreCheckTx(c.bc.Config(), block.Header(), statedb, arbos, tx, c.strictness)
	if err != nil {
		return err
	}
	return c.TransactionPublisher.PublishTransaction(ctx, tx)
}
