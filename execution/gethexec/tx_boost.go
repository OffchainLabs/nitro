package gethexec

import (
	"container/heap"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/offchainlabs/nitro/execution"
)

type txBoostHeap struct {
	sync.RWMutex
	prioQueue *boostableTxs
}

func newTxBoostHeap(policyScorer execution.BoostPolicyScorer) *txBoostHeap {
	state := &boostableTxs{
		txs:          make([]boostableTx, 0),
		policyScorer: policyScorer,
	}
	heap.Init(state)
	srv := &txBoostHeap{
		prioQueue: state,
	}
	return srv
}

func (tb *txBoostHeap) Push(tx boostableTx) {
	tb.Lock()
	defer tb.Unlock()
	heap.Push(tb.prioQueue, tx)
}

func (tb *txBoostHeap) Pop() boostableTx {
	tb.Lock()
	defer tb.Unlock()
	return heap.Pop(tb.prioQueue).(boostableTx)
}

// A boostable tx type that contains a bid and a timestamp.
type boostableTx interface {
	timestamp() time.Time
	innerTx() *types.Transaction
}

// Defines a type that implements the heap.Interface interface from
// the standard library.
type boostableTxs struct {
	txs          []boostableTx
	policyScorer execution.BoostPolicyScorer
}

func (tb boostableTxs) Len() int      { return len(tb.txs) }
func (tb boostableTxs) Swap(i, j int) { tb.txs[i], tb.txs[j] = tb.txs[j], tb.txs[i] }

// We want to implement a priority queue using a max heap. This will score txs
// according to a custom policy and sort them from highest score to lowest.
func (tb boostableTxs) Less(i, j int) bool {
	iScore := tb.policyScorer.ScoreTx(tb.txs[i].innerTx())
	jScore := tb.policyScorer.ScoreTx(tb.txs[j].innerTx())
	if iScore == jScore {
		// Ties are broken by earliest timestamp.
		return tb.txs[i].timestamp().Before(tb.txs[j].timestamp())
	}
	return iScore > jScore
}

// Push and Pop implement the required methods for the heap interface from the standard library.
func (tb *boostableTxs) Push(item any) {
	tx := item.(boostableTx)
	tb.txs = append(tb.txs, tx)
}

func (tb *boostableTxs) Pop() any {
	old := tb.txs
	n := len(old)
	item := old[n-1]
	old[n-1] = nil // avoid memory leak
	tb.txs = old[0 : n-1]
	return item
}
