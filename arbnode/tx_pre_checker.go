// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbnode

import (
	"context"
	"fmt"
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
	header         *types.Header
	stateDb        *state.StateDB
	l1PricingState *l1pricing.L1PricingState
}

type TxPreChecker struct {
	publisher    TransactionPublisher
	bc           *core.BlockChain
	latestState  atomic.Value // contains a txPreCheckerState
	subscription event.Subscription
	headChan     chan core.ChainHeadEvent
}

func NewTxPreChecker(publisher TransactionPublisher, bc *core.BlockChain) *TxPreChecker {
	headChan := make(chan core.ChainHeadEvent, 64)
	subscription := bc.SubscribeChainHeadEvent(headChan)
	c := &TxPreChecker{
		publisher:    publisher,
		bc:           bc,
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
		header:         block.Header(),
		stateDb:        stateDb,
		l1PricingState: arbos.L1PricingState(),
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

func (c *TxPreChecker) PublishTransaction(ctx context.Context, tx *types.Transaction) error {
	if tx.Gas() < params.TxGas {
		return core.ErrIntrinsicGas
	}
	state := c.getLatestState()
	sender, err := types.Sender(types.LatestSigner(c.bc.Config()), tx)
	if err != nil {
		return err
	}
	if arbmath.BigLessThan(tx.GasFeeCap(), state.header.BaseFee) {
		return fmt.Errorf("%w: address %v, maxFeePerGas: %s baseFee: %s", core.ErrFeeCapTooLow, sender, tx.GasFeeCap(), state.header.BaseFee)
	}
	balance := state.stateDb.GetBalance(sender)
	cost := tx.Cost()
	if arbmath.BigLessThan(balance, cost) {
		return fmt.Errorf("%w: address %v have %v want %v", core.ErrInsufficientFunds, sender, balance, cost)
	}
	stateNonce := state.stateDb.GetNonce(sender)
	if tx.Nonce() < stateNonce {
		return fmt.Errorf("%w: address %v, tx: %d state: %d", core.ErrNonceTooLow, sender, tx.Nonce(), stateNonce)
	}
	intrinsic, err := core.IntrinsicGas(tx.Data(), tx.AccessList(), tx.To() == nil, c.bc.Config().IsHomestead(state.header.Number), true)
	if err != nil {
		return err
	}
	dataCost, _ := state.l1PricingState.GetPosterInfo(tx, l1pricing.BatchPosterAddress)
	dataGas := arbmath.BigDiv(dataCost, state.header.BaseFee)
	if tx.Gas() < intrinsic+dataGas.Uint64() {
		return core.ErrIntrinsicGas
	}
	return c.publisher.PublishTransaction(ctx, tx)
}
