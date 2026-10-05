// Copyright 2024-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package main

import (
	"context"
	"encoding/json"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
)

type ExpressLaneProxyAPI struct {
	proxy *ExpressLaneProxy
}

func (a *ExpressLaneProxyAPI) SendRawTransaction(ctx context.Context, input hexutil.Bytes) (common.Hash, error) {
	return a.proxy.SendRawTransaction(ctx, input)
}

func (a *ExpressLaneProxyAPI) ChainId(ctx context.Context) hexutil.Uint64 {
	return a.proxy.ChainId(ctx)
}

func (a *ExpressLaneProxyAPI) GetTransactionCount(ctx context.Context, address common.Address, blockNumOrHash rpc.BlockNumberOrHash) (hexutil.Uint64, error) {
	return a.proxy.GetTransactionCount(ctx, address, blockNumOrHash)
}

func (a *ExpressLaneProxyAPI) FeeHistory(ctx context.Context, blockCount hexutil.Uint64, lastBlock rpc.BlockNumber, rewardPercentiles []float64) (json.RawMessage, error) {
	return a.proxy.FeeHistory(ctx, blockCount, lastBlock, rewardPercentiles)
}

func (a *ExpressLaneProxyAPI) BlockNumber(ctx context.Context) (uint64, error) {
	return a.proxy.BlockNumber(ctx)
}

func (a *ExpressLaneProxyAPI) GetBlockByNumber(ctx context.Context, blockNum *rpc.BlockNumber, includeTxData bool) (json.RawMessage, error) {
	return a.proxy.GetBlockByNumber(ctx, blockNum, includeTxData)
}

func (a *ExpressLaneProxyAPI) GetTransactionReceipt(ctx context.Context, txHash hexutil.Bytes, opts *json.RawMessage) (json.RawMessage, error) {
	return a.proxy.GetTransactionReceipt(ctx, txHash, opts)
}
