// Copyright 2024-2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package timeboost

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/big"
	"os"
	"runtime/debug"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/arbitrum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/filters"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/solgen/go/express_lane_auctiongen"
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
	var num = rpc.LatestBlockNumber
	if blockNumber != nil {
		num = rpc.BlockNumber(blockNumber.Int64())
	}

	state, header, err := a.apiBackend.StateAndHeaderByNumber(ctx, num)
	if err != nil {
		return nil, err
	}

	msg := &core.Message{
		From:                  call.From,
		To:                    call.To,
		Value:                 big.NewInt(0),
		GasLimit:              math.MaxUint64,
		GasPrice:              big.NewInt(0),
		GasFeeCap:             big.NewInt(0),
		GasTipCap:             big.NewInt(0),
		Data:                  call.Data,
		AccessList:            call.AccessList,
		SkipNonceChecks:       true,
		SkipTransactionChecks: true,
		TxRunContext:          core.NewMessageEthcallContext(), // Indicate this is an eth_call
		SkipL1Charging:        true,                            // Skip L1 data fees
	}

	evm := a.apiBackend.GetEVM(ctx, state, header, &vm.Config{NoBaseFee: true}, nil)
	gp := core.NewGasPool(math.MaxUint64)
	result, err := core.ApplyMessage(evm, msg, gp)
	if err != nil {
		return nil, err
	}

	return result.ReturnData, nil
}

func NewExpressLaneAuctionFromInternalAPI(
	apiBackend *arbitrum.APIBackend,
	filterSystem *filters.FilterSystem,
	auctionContractAddr common.Address,
) (*express_lane_auctiongen.ExpressLaneAuction, error) {
	var contractBackend bind.ContractBackend = &contractAdapter{filters.NewFilterAPI(filterSystem), nil, apiBackend}

	auctionContract, err := express_lane_auctiongen.NewExpressLaneAuction(auctionContractAddr, contractBackend)
	if err != nil {
		return nil, err
	}

	return auctionContract, nil
}

func GetRoundTimingInfo(
	auctionContract *express_lane_auctiongen.ExpressLaneAuction,
) (*RoundTimingInfo, error) {
	retries := 0

pending:
	rawRoundTimingInfo, err := auctionContract.RoundTimingInfo(&bind.CallOpts{})
	if err != nil {
		const maxRetries = 5
		if errors.Is(err, bind.ErrNoCode) && retries < maxRetries {
			wait := time.Millisecond * 250 * (1 << retries)
			log.Info("ExpressLaneAuction contract not ready, will retry after wait", "err", err, "wait", wait, "maxRetries", maxRetries)
			retries++
			time.Sleep(wait)
			goto pending
		}
		return nil, err
	}
	return NewRoundTimingInfo(rawRoundTimingInfo)
}
