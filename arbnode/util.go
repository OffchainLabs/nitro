package arbnode

import (
	"context"
	"errors"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

type L1Interface interface {
	bind.ContractBackend
	ethereum.ChainReader
	ethereum.TransactionReader
	TransactionSender(ctx context.Context, tx *types.Transaction, block common.Hash, index uint) (common.Address, error)
	BlockNumber(ctx context.Context) (uint64, error)
}

// Will wait until txhash is in the blockchain and return its receipt
func WaitForTx(ctxinput context.Context, client L1Interface, txhash common.Hash, timeout time.Duration) (*types.Receipt, error) {
	ctx, cancel := context.WithTimeout(ctxinput, timeout)
	defer cancel()

	chanHead, cancel := HeaderSubscribeWithRetry(ctx, client)
	defer cancel()

	for {
		reciept, err := client.TransactionReceipt(ctx, txhash)
		if reciept != nil {
			return reciept, err
		}
		select {
		case <-chanHead:
		case <-time.After(timeout / 5):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

func EnsureTxSucceeded(ctx context.Context, client L1Interface, tx *types.Transaction) (*types.Receipt, error) {
	return EnsureTxSucceededWithTimeout(ctx, client, tx, time.Second)
}

func EnsureTxSucceededWithTimeout(ctx context.Context, client L1Interface, tx *types.Transaction, timeout time.Duration) (*types.Receipt, error) {
	txRes, err := WaitForTx(ctx, client, tx.Hash(), timeout)
	if err != nil {
		return nil, err
	}
	if txRes == nil {
		return nil, errors.New("expected receipt")
	}
	if txRes.Status != types.ReceiptStatusSuccessful {
		// Re-execute the transaction as a call to get a better error
		from, err := client.TransactionSender(ctx, tx, txRes.BlockHash, txRes.TransactionIndex)
		if err != nil {
			return nil, err
		}
		callMsg := ethereum.CallMsg{
			From:       from,
			To:         tx.To(),
			Gas:        tx.Gas(),
			GasPrice:   tx.GasPrice(),
			GasFeeCap:  tx.GasFeeCap(),
			GasTipCap:  tx.GasTipCap(),
			Value:      tx.Value(),
			Data:       tx.Data(),
			AccessList: tx.AccessList(),
		}
		_, err = client.CallContract(ctx, callMsg, txRes.BlockNumber)
		if err != nil {
			return nil, err
		}
		return nil, errors.New("tx failed but call succeeded")
	}
	return txRes, nil
}

func headerSubscribeMainLoop(chanOut chan<- *types.Header, ctx context.Context, client ethereum.ChainReader) {
	headerSubscription, err := client.SubscribeNewHead(ctx, chanOut)
	if err != nil {
		if ctx.Err() == nil {
			log.Error("failed sunscribing to header", "err", err)
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
