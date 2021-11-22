package arbnode

import (
	"context"
	"errors"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
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

	chanHead := make(chan *types.Header, 20)
	headSubscribe, err := client.SubscribeNewHead(ctx, chanHead)
	if err != nil {
		return nil, err
	}
	defer headSubscribe.Unsubscribe()

	for {
		receipt, err := client.TransactionReceipt(ctx, txhash)
		if receipt != nil {
			return receipt, err
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
