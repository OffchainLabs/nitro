// Copyright 2023-2024, Offchain Labs, Inc.
// For license information, see:
// https://github.com/offchainlabs/nitro/blob/master/LICENSE.md

package solimpl

import (
	"context"
	"math/big"
	"time"

	"github.com/ccoveille/go-safecast"
	"github.com/pkg/errors"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"

	protocol "github.com/offchainlabs/nitro/bold/chain-abstraction"
	"github.com/offchainlabs/nitro/bold/containers"
)

// ChainCommitter defines a type of chain backend that supports
// committing changes via a direct method, such as a simulated backend
// for testing purposes.
type ChainCommitter interface {
	Commit() common.Hash
}

type transactConfig struct {
	waitForDesiredBlockNum bool
}

type transactOpt func(tc *transactConfig)

func withoutSafeWait() transactOpt {
	return func(tc *transactConfig) {
		tc.waitForDesiredBlockNum = false
	}
}

// Runs a callback function meant to write to a chain backend, and if the
// chain backend supports committing directly, we call the commit function before
// returning. This function additionally waits for the transaction to complete and returns
// an optional transaction receipt. It returns an error if the
// transaction had a non-successful status on-chain, or if the execution of the callback
// errored directly.
func (a *AssertionChain) transact(
	ctx context.Context,
	backend protocol.ChainBackend,
	fn func(opts *bind.TransactOpts) (*types.Transaction, error),
	configOpts ...transactOpt,
) (*types.Receipt, error) {
	config := &transactConfig{
		waitForDesiredBlockNum: true,
	}
	for _, o := range configOpts {
		o(config)
	}
	// We do not send the tx, but instead estimate gas first.
	opts := copyTxOpts(a.txOpts)

	// No BOLD transactions require a value.
	opts.Value = big.NewInt(0)
	opts.NoSend = true
	tx, err := fn(opts)
	if err != nil {
		return nil, errors.Wrap(err, "test execution of tx errored before sending payable tx")
	}
	// Convert the transaction into a CallMsg.
	msg := ethereum.CallMsg{
		From:     opts.From,
		To:       tx.To(),
		Gas:      0, // Set to 0 to let the node decide
		GasPrice: opts.GasPrice,
		Value:    opts.Value,
		Data:     tx.Data(),
	}

	// Estimate the gas required for the transaction. This will catch errors early
	// without needing to pay for the transaction and waste funds.
	gas, err := backend.EstimateGas(ctx, msg)
	if err != nil {
		return nil, errors.Wrapf(err, "gas estimation errored for tx with hash %s", containers.Trunc(tx.Hash().Bytes()))
	}

	// Now, we send the tx with the estimated gas.
	defaultGasUint64, err := safecast.ToUint64(defaultBaseGas)
	if err != nil {
		return nil, errors.Wrap(err, "could not convert default base gas to uint64")
	}
	opts.GasLimit = gas + defaultGasUint64
	tx, err = a.transactor.SendTransaction(ctx, fn, opts, gas)
	if err != nil {
		return nil, err
	}

	if commiter, ok := backend.(ChainCommitter); ok {
		commiter.Commit()
	}
	ctxWaitMined, cancelWaitMined := context.WithTimeout(ctx, time.Minute)
	defer cancelWaitMined()
	receipt, err := bind.WaitMined(ctxWaitMined, backend, tx)
	if err != nil {
		return nil, err
	}

	if config.waitForDesiredBlockNum && a.rpcHeadBlockNumber != rpc.LatestBlockNumber {
		ctxWaitSafe, cancelWaitSafe := context.WithTimeout(ctx, time.Minute*20)
		defer cancelWaitSafe()
		receipt, err = a.waitForTxToBeSafe(ctxWaitSafe, backend, tx, receipt)
		if err != nil {
			return nil, err
		}
	}

	if receipt.Status != types.ReceiptStatusSuccessful {
		callMsg := ethereum.CallMsg{
			From:       opts.From,
			To:         tx.To(),
			Gas:        0,
			GasPrice:   nil,
			Value:      tx.Value(),
			Data:       tx.Data(),
			AccessList: tx.AccessList(),
		}
		if _, err := backend.CallContract(ctx, callMsg, nil); err != nil {
			return nil, errors.Wrap(err, "transaction errored")
		}
	}
	return receipt, nil
}

// waitForTxToBeSafe waits for the transaction to be mined in a block that is safe.
func (a *AssertionChain) waitForTxToBeSafe(
	ctx context.Context,
	backend protocol.ChainBackend,
	tx *types.Transaction,
	receipt *types.Receipt,
) (*types.Receipt, error) {
	for {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		latestSafeHeader, err := backend.HeaderByNumber(ctx, big.NewInt(int64(a.rpcHeadBlockNumber)))
		if err != nil {
			return nil, err
		}
		if !latestSafeHeader.Number.IsUint64() {
			return nil, errors.New("block number is not uint64")
		}
		latestSafeHeaderNumber := latestSafeHeader.Number.Uint64()
		txSafe := latestSafeHeaderNumber >= receipt.BlockNumber.Uint64()

		// If the tx is not yet safe, we can simply wait.
		if !txSafe {
			var blocksLeftForTxToBeSafe int64
			if receipt.BlockNumber.Uint64() > latestSafeHeaderNumber {
				blocksLeftForTxToBeSafe = 0
			} else {
				blocksLeftForTxToBeSafe, err = safecast.ToInt64(latestSafeHeaderNumber - receipt.BlockNumber.Uint64())
				if err != nil {
					return nil, errors.Wrap(err, "could not convert blocks left for tx to be safe to int64")
				}
			}
			timeToWait := a.averageTimeForBlockCreation * time.Duration(blocksLeftForTxToBeSafe)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(timeToWait):
			}
		} else {
			break
		}
	}

	// This is to handle the case where the transaction is mined in a block, but then the block is reorged.
	// In this case, we want to wait for the transaction to be mined again.
	receiptLatest, err := bind.WaitMined(ctx, backend, tx)
	if err != nil {
		return nil, err
	}
	// If the receipt block number is different from the latest receipt block number, we wait for the transaction
	// to be in the safe block again.
	if receiptLatest.BlockNumber.Cmp(receipt.BlockNumber) != 0 {
		return a.waitForTxToBeSafe(ctx, backend, tx, receiptLatest)
	}
	return receipt, nil
}

// copyTxOpts creates a deep copy of the given transaction options.
func copyTxOpts(opts *bind.TransactOpts) *bind.TransactOpts {
	copied := &bind.TransactOpts{
		From:     opts.From,
		Context:  opts.Context,
		NoSend:   opts.NoSend,
		Signer:   opts.Signer,
		GasLimit: opts.GasLimit,
	}

	if opts.Nonce != nil {
		copied.Nonce = new(big.Int).Set(opts.Nonce)
	}
	if opts.Value != nil {
		copied.Value = new(big.Int).Set(opts.Value)
	}
	if opts.GasPrice != nil {
		copied.GasPrice = new(big.Int).Set(opts.GasPrice)
	}
	if opts.GasFeeCap != nil {
		copied.GasFeeCap = new(big.Int).Set(opts.GasFeeCap)
	}
	if opts.GasTipCap != nil {
		copied.GasTipCap = new(big.Int).Set(opts.GasTipCap)
	}
	return copied
}
