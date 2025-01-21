// Copyright 2024-2025, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package gethexec

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/big"
	"os"
	"runtime/debug"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/arbitrum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/filters"
	"github.com/ethereum/go-ethereum/rpc"
)

// contractAdapter is an impl of bind.ContractBackend with necessary methods defined to work with the ExpressLaneAuction contract
type contractAdapter struct {
	*filters.FilterAPI
	bind.ContractTransactor // We leave this member unset as it is not used.

	apiBackend *arbitrum.APIBackend
}

func (a *contractAdapter) FilterLogs(ctx context.Context, q ethereum.FilterQuery) ([]types.Log, error) {
	logPointers, err := a.GetLogs(ctx, filters.FilterCriteria(q))
	if err != nil {
		return nil, err
	}
	logs := make([]types.Log, 0, len(logPointers))
	for _, log := range logPointers {
		logs = append(logs, *log)
	}
	return logs, nil
}

func (a *contractAdapter) SubscribeFilterLogs(ctx context.Context, q ethereum.FilterQuery, ch chan<- types.Log) (ethereum.Subscription, error) {
	fmt.Fprintf(os.Stderr, "contractAdapter doesn't implement SubscribeFilterLogs: Stack trace:\n%s\n", debug.Stack())
	return nil, errors.New("contractAdapter doesn't implement SubscribeFilterLogs - shouldn't be needed")
}

func (a *contractAdapter) CodeAt(ctx context.Context, contract common.Address, blockNumber *big.Int) ([]byte, error) {
	number := rpc.LatestBlockNumber
	if blockNumber != nil {
		number = rpc.BlockNumber(blockNumber.Int64())
	}

	statedb, _, err := a.apiBackend.StateAndHeaderByNumber(ctx, number)
	if err != nil {
		return nil, fmt.Errorf("contractAdapter error: %w", err)
	}
	code := statedb.GetCode(contract)
	return code, nil
}

func (a *contractAdapter) CallContract(ctx context.Context, call ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	var num rpc.BlockNumber = rpc.LatestBlockNumber
	if blockNumber != nil {
		num = rpc.BlockNumber(blockNumber.Int64())
	}

	state, header, err := a.apiBackend.StateAndHeaderByNumber(ctx, num)
	if err != nil {
		return nil, err
	}

	msg := &core.Message{
		From:              call.From,
		To:                call.To,
		Value:             big.NewInt(0),
		GasLimit:          math.MaxUint64,
		GasPrice:          big.NewInt(0),
		GasFeeCap:         big.NewInt(0),
		GasTipCap:         big.NewInt(0),
		Data:              call.Data,
		AccessList:        call.AccessList,
		SkipAccountChecks: true,
		TxRunMode:         core.MessageEthcallMode, // Indicate this is an eth_call
		SkipL1Charging:    true,                    // Skip L1 data fees
	}

	evm := a.apiBackend.GetEVM(ctx, msg, state, header, &vm.Config{NoBaseFee: true}, nil)
	gp := new(core.GasPool).AddGas(math.MaxUint64)
	result, err := core.ApplyMessage(evm, msg, gp)
	if err != nil {
		return nil, err
	}

	return result.ReturnData, nil
}
