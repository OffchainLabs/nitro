// Copyright 2021-2024, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbutil

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/rpc"
)

type L1Interface interface {
	bind.ContractBackend
	ethereum.ChainReader
	ethereum.ChainStateReader
	ethereum.TransactionReader
	TransactionSender(ctx context.Context, tx *types.Transaction, block common.Hash, index uint) (common.Address, error)
	BlockNumber(ctx context.Context) (uint64, error)
	CallContractAtHash(ctx context.Context, msg ethereum.CallMsg, blockHash common.Hash) ([]byte, error)
	PendingCallContract(ctx context.Context, msg ethereum.CallMsg) ([]byte, error)
	ChainID(ctx context.Context) (*big.Int, error)
	Client() rpc.ClientInterface
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

func GetPendingCallBlockNumber(ctx context.Context, client L1Interface) (*big.Int, error) {
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

func DetailTxError(ctx context.Context, client L1Interface, tx *types.Transaction, txRes *types.Receipt) error {
	// Re-execute the transaction as a call to get a better error
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if txRes == nil {
		return errors.New("expected receipt")
	}
	if txRes.Status == types.ReceiptStatusSuccessful {
		return nil
	}
	from, err := client.TransactionSender(ctx, tx, txRes.BlockHash, txRes.TransactionIndex)
	if err != nil {
		return fmt.Errorf("TransactionSender got: %w for tx %v", err, tx.Hash())
	}
	_, err = SendTxAsCall(ctx, client, tx, from, txRes.BlockNumber, false)
	if err == nil {
		return fmt.Errorf("tx failed but call succeeded for tx hash %v", tx.Hash())
	}
	_, err = SendTxAsCall(ctx, client, tx, from, txRes.BlockNumber, true)
	if err == nil {
		return fmt.Errorf("%w for tx hash %v", vm.ErrOutOfGas, tx.Hash())
	}
	return fmt.Errorf("SendTxAsCall got: %w for tx hash %v", err, tx.Hash())
}

func DetailTxErrorUsingCallMsg(ctx context.Context, client L1Interface, txHash common.Hash, txRes *types.Receipt, callMsg ethereum.CallMsg) error {
	// Re-execute the transaction as a call to get a better error
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if txRes == nil {
		return errors.New("expected receipt")
	}
	if txRes.Status == types.ReceiptStatusSuccessful {
		return nil
	}
	var err error
	if _, err = client.CallContract(ctx, callMsg, txRes.BlockNumber); err == nil {
		return fmt.Errorf("tx failed but call succeeded for tx hash %v", txHash)
	}
	callMsg.Gas = 0
	if _, err = client.CallContract(ctx, callMsg, txRes.BlockNumber); err == nil {
		return fmt.Errorf("%w for tx hash %v", vm.ErrOutOfGas, txHash)
	}
	return fmt.Errorf("SendTxAsCall got: %w for tx hash %v", err, txHash)
}
