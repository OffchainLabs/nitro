//
// Copyright 2021-2022, Offchain Labs, Inc. All rights reserved.
//

package arbtest

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
)

func GetPendingBlockNumber(ctx context.Context, client arbutil.L1Interface) (*big.Int, error) {
	// Attempt to get the block number from ArbSys, if it exists
	arbSys, err := precompilesgen.NewArbSys(common.BigToAddress(big.NewInt(100)), client)
	if err != nil {
		return arbutil.GetPendingCallBlockNumber(ctx, client)
	}
	blockNum, err := arbSys.ArbBlockNumber(&bind.CallOpts{Context: ctx})
	if err != nil {
		return arbutil.GetPendingCallBlockNumber(ctx, client)
	}
	// Arbitrum chains don't have miners, so they're one block behind non-Arbitrum chains.
	return blockNum.Add(blockNum, common.Big1), nil
}

// Will wait until txhash is in the blockchain and return its receipt
func WaitForTx(ctxinput context.Context, client arbutil.L1Interface, txhash common.Hash, timeout time.Duration) (*types.Receipt, error) {
	ctx, cancel := context.WithTimeout(ctxinput, timeout)
	defer cancel()

	chanHead, cancel := HeaderSubscribeWithRetry(ctx, client)
	defer cancel()

	checkInterval := timeout / 50
	if checkInterval > time.Second {
		checkInterval = time.Second
	}
	for {
		receipt, err := client.TransactionReceipt(ctx, txhash)
		if err == nil && receipt != nil {
			// For some reason, Geth has a weird property of giving out receipts and updating the latest block number
			// before calls will actually use the new block's state as pending. This leads to failures down the line,
			// as future calls/gas estimations will use a state before a transaction that is thought to have succeeded.
			// To prevent this, we do an eth_call to check what the pending state's block number is before returning the receipt.
			blockNumber, err := GetPendingBlockNumber(ctx, client)
			if err != nil {
				return nil, err
			}
			if blockNumber.Cmp(receipt.BlockNumber) > 0 {
				// The latest pending state contains the state of our transaction.
				return receipt, nil
			}
		}
		// Note: time.After won't free the timer until after it expires.
		// However, that's fine here, as checkInterval is at most a second.
		select {
		case <-chanHead:
		case <-time.After(checkInterval):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

func EnsureTxSucceeded(ctx context.Context, client arbutil.L1Interface, tx *types.Transaction) (*types.Receipt, error) {
	return EnsureTxSucceededWithTimeout(ctx, client, tx, time.Second*5)
}

func EnsureTxSucceededWithTimeout(ctx context.Context, client arbutil.L1Interface, tx *types.Transaction, timeout time.Duration) (*types.Receipt, error) {
	txRes, err := WaitForTx(ctx, client, tx.Hash(), timeout)
	if err != nil {
		return nil, fmt.Errorf("waitFoxTx (tx=%s) got: %w", tx.Hash().Hex(), err)
	}
	return txRes, arbutil.DetailTxError(ctx, client, tx, txRes)
}

func headerSubscribeMainLoop(chanOut chan<- *types.Header, ctx context.Context, client ethereum.ChainReader) {
	headerSubscription, err := client.SubscribeNewHead(ctx, chanOut)
	if err != nil {
		if ctx.Err() == nil {
			log.Error("failed subscribing to header", "err", err)
		}
		return
	}

	for {
		select {
		case err := <-headerSubscription.Err():
			if ctx.Err() == nil {
				return
			}
			log.Warn("error in subscription to L1 headers", "err", err)
			for {
				headerSubscription, err = client.SubscribeNewHead(ctx, chanOut)
				if err != nil {
					log.Warn("error re-subscribing to L1 headers", "err", err)
					select {
					case <-ctx.Done():
						return
					case <-time.After(time.Second):
					}
				} else {
					break
				}
			}
		case <-ctx.Done():
			headerSubscription.Unsubscribe()
			return
		}
	}
}

// returned channel will reconnect to client if disconnected, until context closed or cancel called
func HeaderSubscribeWithRetry(ctx context.Context, client ethereum.ChainReader) (<-chan *types.Header, context.CancelFunc) {
	chanOut := make(chan *types.Header)

	childCtx, cancelFunc := context.WithCancel(ctx)
	go headerSubscribeMainLoop(chanOut, childCtx, client)

	return chanOut, cancelFunc
}
