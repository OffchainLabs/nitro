// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE

package solimpl

import (
	"context"
	"fmt"
	"math/big"
	"sort"
	"sync"

	protocol "github.com/OffchainLabs/bold/chain-abstraction"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/types"
)

// TrackedContractBackend implements a wrapper around a chain backend interface
// which can keep track of the number of calls and transactions made per
// method to a contract along with gas costs for transactions. These can then be
// printed out to a destination for analysis.
type TrackedContractBackend struct {
	protocol.ChainBackend
	metrics map[string]*MethodMetrics // method hash -> metrics
	mu      sync.RWMutex
}

type MethodMetrics struct {
	Calls    int       // Total number of calls to the method.
	Txs      int       // Total number of transactions to the method.
	GasCosts []big.Int // Gas costs for each tx.
}

func NewTrackedContractBackend(backend protocol.ChainBackend) *TrackedContractBackend {
	return &TrackedContractBackend{
		ChainBackend: backend,
		metrics:      make(map[string]*MethodMetrics),
	}
}

func (t *TrackedContractBackend) CallContract(ctx context.Context, call ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	data := call.Data
	if len(data) >= 4 { // if there's a method selector
		methodHash := fmt.Sprintf("%#x", data[:4]) // first 4 bytes are method selector
		t.mu.Lock()
		metric, ok := t.metrics[methodHash]
		if !ok {
			metric = &MethodMetrics{}
			t.metrics[methodHash] = metric
		}
		metric.Calls++
		// Assuming gas cost for call can be added here if needed
		t.mu.Unlock()
	}
	return t.ChainBackend.CallContract(ctx, call, blockNumber)
}

func (t *TrackedContractBackend) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	if tx != nil && len(tx.Data()) >= 4 {
		methodHash := fmt.Sprintf("%#x", tx.Data()[:4])
		t.mu.Lock()
		metric, ok := t.metrics[methodHash]
		if !ok {
			metric = &MethodMetrics{}
			t.metrics[methodHash] = metric
		}
		metric.Txs++
		gasCost := new(big.Int).Mul(tx.GasPrice(), new(big.Int).SetUint64(tx.Gas()))
		metric.GasCosts = append(metric.GasCosts, *gasCost)
		t.mu.Unlock()
	}
	return t.ChainBackend.SendTransaction(ctx, tx)
}

// Computes a median of big integers (gas costs)
func median(gasCosts []big.Int) *big.Int {
	if len(gasCosts) == 0 {
		return nil
	}
	sorted := make([]big.Int, len(gasCosts))
	copy(sorted, gasCosts)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Cmp(&sorted[j]) < 0
	})
	mid := len(sorted) / 2
	if len(sorted)%2 == 0 {
		return new(big.Int).Add(&sorted[mid-1], &sorted[mid]).Div(&sorted[mid], big.NewInt(2))
	}
	return &sorted[mid]
}

func (t *TrackedContractBackend) PrintMetrics() {
	t.mu.RLock()
	defer t.mu.RUnlock()
	for methodHash, metrics := range t.metrics {
		fmt.Printf("Method: %s\n", methodHash)
		fmt.Printf("Calls: %d\n", metrics.Calls)
		fmt.Printf("Transactions: %d\n", metrics.Txs)
		if med := median(metrics.GasCosts); med != nil {
			fmt.Printf("Median Gas Cost: %s\n", med.String())
		} else {
			fmt.Println("Median Gas Cost: N/A")
		}
		fmt.Println("-----------")
	}
}
