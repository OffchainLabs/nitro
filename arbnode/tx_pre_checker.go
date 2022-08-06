// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbnode

import (
	"context"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/arbos/l1pricing"
	"github.com/offchainlabs/nitro/util/arbmath"
)

type txPreCheckerState struct {
	header  *types.Header
	stateDb *state.StateDB
	arbos   *arbosState.ArbosState
}

type TxPreChecker struct {
	publisher    TransactionPublisher
	bc           *core.BlockChain
	strictness   uint
	latestState  atomic.Value // contains a txPreCheckerState
	subscription event.Subscription
	headChan     chan core.ChainHeadEvent
}

func NewTxPreChecker(publisher TransactionPublisher, bc *core.BlockChain, strictness uint) *TxPreChecker {
	headChan := make(chan core.ChainHeadEvent, 64)
	subscription := bc.SubscribeChainHeadEvent(headChan)
	c := &TxPreChecker{
		publisher:    publisher,
		bc:           bc,
		strictness:   strictness,
		latestState:  atomic.Value{}, // filled in in Initialize
		subscription: subscription,
		headChan:     headChan,
	}
	go func() {
		for {
			event, ok := <-headChan
			if !ok {
				return
			}
		BacklogLoop:
			for {
				// clear out any backed up events
				select {
				case event, ok = <-headChan:
					if !ok {
						return
					}
				default:
					break BacklogLoop
				}
			}
			err := c.updateLatestState(event.Block)
			if err != nil {
				log.Warn("failed to update tx pre-checker state to latest", "err", err)
			}
		}
	}()
	return c
}

func (c *TxPreChecker) updateLatestState(block *types.Block) error {
	stateDb, err := c.bc.StateAt(block.Root())
	if err != nil {
		return err
	}
	arbos, err := arbosState.OpenSystemArbosState(stateDb, nil, true)
	if err != nil {
		return err
	}
	fullState := txPreCheckerState{
		header:  block.Header(),
		stateDb: stateDb,
		arbos:   arbos,
	}
	c.latestState.Store(fullState)
	return nil
}

func (c *TxPreChecker) getLatestState() txPreCheckerState {
	state, ok := c.latestState.Load().(txPreCheckerState)
	if !ok {
		panic("invalid type stored in latestState")
	}
	return state
}

func (c *TxPreChecker) Initialize(ctx context.Context) error {
	err := c.updateLatestState(c.bc.CurrentBlock())
	if err != nil {
		return err
	}
	return c.publisher.Initialize(ctx)
}

func (c *TxPreChecker) Start(ctx context.Context) error {
	return c.publisher.Start(ctx)
}

func (c *TxPreChecker) StopAndWait() {
	c.subscription.Unsubscribe()
	close(c.headChan)
	c.publisher.StopAndWait()
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
	state := c.getLatestState()
	err := PreCheckTx(c.bc.Config(), state.header, state.stateDb, state.arbos, tx, c.strictness)
	if err != nil {
		return err
	}
	return c.publisher.PublishTransaction(ctx, tx)
}

func (c *TxPreChecker) CheckHealth(ctx context.Context) error {
	return c.publisher.CheckHealth(ctx)
}
