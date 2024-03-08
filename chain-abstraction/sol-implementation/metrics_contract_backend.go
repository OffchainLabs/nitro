// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE

package solimpl

import (
	"context"
	"fmt"
	"math/big"

	protocol "github.com/OffchainLabs/bold/chain-abstraction"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/metrics"
)

type MetricsContractBackend struct {
	protocol.ChainBackend
}

func NewMetricsContractBackend(backend protocol.ChainBackend) *MetricsContractBackend {
	return &MetricsContractBackend{
		ChainBackend: backend,
	}
}

func (t *MetricsContractBackend) CallContract(ctx context.Context, call ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	data := call.Data
	if len(data) >= 4 { // if there's a method selector
		methodHash := fmt.Sprintf("%#x", data[:4]) // first 4 bytes are method selector
		metrics.GetOrRegisterCounter("arb/backend/call_contract/"+methodHash+"/count", nil).Inc(1)
	}
	return t.ChainBackend.CallContract(ctx, call, blockNumber)
}

func (t *MetricsContractBackend) CodeAt(ctx context.Context, contract common.Address, blockNumber *big.Int) ([]byte, error) {
	metrics.GetOrRegisterCounter("arb/backend/code_at/count", nil).Inc(1)
	return t.ChainBackend.CodeAt(ctx, contract, blockNumber)
}

func (t *MetricsContractBackend) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	metrics.GetOrRegisterCounter("arb/backend/header_by_number/count", nil).Inc(1)
	return t.ChainBackend.HeaderByNumber(ctx, number)
}

func (t *MetricsContractBackend) PendingCodeAt(ctx context.Context, account common.Address) ([]byte, error) {
	metrics.GetOrRegisterCounter("arb/backend/pending_code_at/count", nil).Inc(1)
	return t.ChainBackend.PendingCodeAt(ctx, account)
}

func (t *MetricsContractBackend) PendingNonceAt(ctx context.Context, account common.Address) (uint64, error) {
	metrics.GetOrRegisterCounter("arb/backend/pending_code_at/count", nil).Inc(1)
	return t.ChainBackend.PendingNonceAt(ctx, account)
}

func (t *MetricsContractBackend) SuggestGasPrice(ctx context.Context) (*big.Int, error) {
	metrics.GetOrRegisterCounter("arb/backend/suggest_gas_price/count", nil).Inc(1)
	return t.ChainBackend.SuggestGasPrice(ctx)
}

func (t *MetricsContractBackend) SuggestGasTipCap(ctx context.Context) (*big.Int, error) {
	metrics.GetOrRegisterCounter("arb/backend/suggest_gas_tip_cap/count", nil).Inc(1)
	return t.ChainBackend.SuggestGasTipCap(ctx)
}

func (t *MetricsContractBackend) EstimateGas(ctx context.Context, call ethereum.CallMsg) (gas uint64, err error) {
	data := call.Data
	if len(data) >= 4 { // if there's a method selector
		methodHash := fmt.Sprintf("%#x", data[:4]) // first 4 bytes are method selector
		metrics.GetOrRegisterCounter("arb/backend/estimate_gas/"+methodHash+"/count", nil).Inc(1)
	}
	return t.ChainBackend.EstimateGas(ctx, call)
}

func (t *MetricsContractBackend) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	if tx != nil && len(tx.Data()) >= 4 {
		methodHash := fmt.Sprintf("%#x", tx.Data()[:4])
		metrics.GetOrRegisterCounter("arb/backend/send_transaction/"+methodHash+"/count", nil).Inc(1)
	}
	return t.ChainBackend.SendTransaction(ctx, tx)
}

func (t *MetricsContractBackend) FilterLogs(ctx context.Context, query ethereum.FilterQuery) ([]types.Log, error) {
	metrics.GetOrRegisterCounter("arb/backend/filter_logs/count", nil).Inc(1)
	return t.ChainBackend.FilterLogs(ctx, query)
}

func (t *MetricsContractBackend) SubscribeFilterLogs(ctx context.Context, query ethereum.FilterQuery, ch chan<- types.Log) (ethereum.Subscription, error) {
	metrics.GetOrRegisterCounter("arb/backend/subscribe_filter_logs/count", nil).Inc(1)
	return t.ChainBackend.SubscribeFilterLogs(ctx, query, ch)
}

func (t *MetricsContractBackend) TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	metrics.GetOrRegisterCounter("arb/backend/transaction_receipt/count", nil).Inc(1)
	return t.ChainBackend.TransactionReceipt(ctx, txHash)
}
