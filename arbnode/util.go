package arbnode

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/arbstate/solgen/go/precompilesgen"
)

type L1Interface interface {
	bind.ContractBackend
	ethereum.ChainReader
	ethereum.TransactionReader
	TransactionSender(ctx context.Context, tx *types.Transaction, block common.Hash, index uint) (common.Address, error)
	BlockNumber(ctx context.Context) (uint64, error)
	PendingCallContract(ctx context.Context, msg ethereum.CallMsg) ([]byte, error)
}

func getPendingBlockNumberNonArbitrum(ctx context.Context, client L1Interface) (*big.Int, error) {
	msg := ethereum.CallMsg{
		// Pretend to be a contract deployment to execute EVM code without calling a contract.
		To: nil,
		// Contains the following EVM code, which returns the current block number:
		// NUMBER
		// PUSH1 0
		// MSTORE
		// PUSH1 32
		// PUSH1 0
		// RETURN
		Data: []byte{0x43, 0x60, 0x00, 0x52, 0x60, 0x20, 0x60, 0x00, 0xF3},
	}
	callRes, err := client.PendingCallContract(ctx, msg)
	if err != nil {
		return nil, err
	}
	return new(big.Int).SetBytes(callRes), nil
}

func GetPendingBlockNumber(ctx context.Context, client L1Interface) (*big.Int, error) {
	// Attempt to get the block number from ArbSys, if it exists
	arbSys, err := precompilesgen.NewArbSys(common.BigToAddress(big.NewInt(100)), client)
	if err != nil {
		return getPendingBlockNumberNonArbitrum(ctx, client)
	}
	blockNum, err := arbSys.ArbBlockNumber(&bind.CallOpts{Context: ctx})
	if err != nil {
		return getPendingBlockNumberNonArbitrum(ctx, client)
	}
	// Arbitrum chains don't have miners, so they're one block behind non-Arbitrum chains.
	return blockNum.Add(blockNum, common.Big1), nil
}

// Will wait until txhash is in the blockchain and return its receipt
func WaitForTx(ctxinput context.Context, client L1Interface, txhash common.Hash, timeout time.Duration) (*types.Receipt, error) {
	ctx, cancel := context.WithTimeout(ctxinput, timeout)
	defer cancel()

	chanHead, cancel := HeaderSubscribeWithRetry(ctx, client)
	defer cancel()

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
		select {
		case <-chanHead:
		case <-time.After(timeout / 5):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

func EnsureTxSucceeded(ctx context.Context, client L1Interface, tx *types.Transaction) (*types.Receipt, error) {
	return EnsureTxSucceededWithTimeout(ctx, client, tx, time.Second*2)
}

func SendTxAsCall(ctx context.Context, client L1Interface, tx *types.Transaction, from common.Address, blockNum *big.Int, unlimitedGas bool) ([]byte, error) {
	var gas uint64
	if unlimitedGas {
		gas = 0
	} else {
		gas = tx.Gas()
	}
	callMsg := ethereum.CallMsg{
		From:       from,
		To:         tx.To(),
		Gas:        gas,
		GasPrice:   tx.GasPrice(),
		GasFeeCap:  tx.GasFeeCap(),
		GasTipCap:  tx.GasTipCap(),
		Value:      tx.Value(),
		Data:       tx.Data(),
		AccessList: tx.AccessList(),
	}
	return client.CallContract(ctx, callMsg, blockNum)
}

func EnsureTxSucceededWithTimeout(ctx context.Context, client L1Interface, tx *types.Transaction, timeout time.Duration) (*types.Receipt, error) {
	txRes, err := WaitForTx(ctx, client, tx.Hash(), timeout)
	if err != nil {
		return nil, fmt.Errorf("waitFoxTx got: %w", err)
	}
	if txRes == nil {
		return nil, errors.New("expected receipt")
	}
	if txRes.Status != types.ReceiptStatusSuccessful {
		// Re-execute the transaction as a call to get a better error
		from, err := client.TransactionSender(ctx, tx, txRes.BlockHash, txRes.TransactionIndex)
		if err != nil {
			return txRes, fmt.Errorf("TransactionSender got: %w", err)
		}
		_, err = SendTxAsCall(ctx, client, tx, from, txRes.BlockNumber, false)
		if err == nil {
			return txRes, errors.New("tx failed but call succeeded")
		}
		_, err = SendTxAsCall(ctx, client, tx, from, txRes.BlockNumber, true)
		if err == nil {
			return txRes, core.ErrGasLimitReached
		}
		return txRes, fmt.Errorf("SendTxAsCall got: %w", err)
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
