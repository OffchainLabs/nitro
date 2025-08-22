// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"context"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
)

// ClientWrapper wraps an RPC client to manipulate inbound requests.
type ClientWrapper struct {
	mutex                *sync.Mutex
	innerClient          rpc.ClientInterface
	chainInfo            *BlockchainTestInfo
	rawTransactionFilter common.Address
	rawTransactionChan   chan<- *types.Transaction
}

func NewClientWrapper(innerClient rpc.ClientInterface, chainInfo *BlockchainTestInfo) *ClientWrapper {
	return &ClientWrapper{
		mutex:       new(sync.Mutex),
		innerClient: innerClient,
		chainInfo:   chainInfo,
	}
}

func (w *ClientWrapper) CallContext(ctx context.Context, result interface{}, method string, args ...interface{}) error {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	if method == "eth_sendRawTransaction" && w.filterRawTransaction(result, args...) {
		return nil
	}
	return w.innerClient.CallContext(ctx, result, method, args...)
}

func (w *ClientWrapper) EthSubscribe(ctx context.Context, channel interface{}, args ...interface{}) (*rpc.ClientSubscription, error) {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	return w.innerClient.EthSubscribe(ctx, channel, args...)
}

func (w *ClientWrapper) BatchCallContext(ctx context.Context, b []rpc.BatchElem) error {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	return w.innerClient.BatchCallContext(ctx, b)
}

func (w *ClientWrapper) Close() {
	w.innerClient.Close()
}

// EnableRawTransactionFilter will filter sendRawTransaction requests with the given sender.
// The filtered requests will return OK and will be sent to the given channel instead of going through.
func (w *ClientWrapper) EnableRawTransactionFilter(sender common.Address, rawTransactionChan chan<- *types.Transaction) {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	w.rawTransactionFilter = sender
	w.rawTransactionChan = rawTransactionChan
}

func (w *ClientWrapper) DisableRawTransactionFilter() {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	w.rawTransactionFilter = common.Address{}
}

func (w *ClientWrapper) filterRawTransaction(result interface{}, args ...interface{}) bool {
	if w.rawTransactionFilter == (common.Address{}) {
		return false
	}
	if len(args) < 1 {
		return false
	}
	rawTx, ok := args[0].(string)
	if !ok {
		return false
	}
	rawTxBytes, err := hexutil.Decode(rawTx)
	if err != nil {
		return false
	}
	var tx types.Transaction
	err = tx.UnmarshalBinary(rawTxBytes)
	if err != nil {
		return false
	}
	from, err := types.Sender(w.chainInfo.Signer, &tx)
	if err != nil {
		return false
	}
	if from == w.rawTransactionFilter {
		w.rawTransactionChan <- &tx
		return true
	}
	return false
}
