package discretetimeboost

import (
	"container/heap"
	"sync"
	"time"
)

const (
	defaultMaxBoostFactor = 500 * time.Millisecond
)

type timeBoostHeap[T boostableTx] struct {
	sync.RWMutex
	prioQueue timeBoostableTxs
	gFactor   time.Duration
}

func newTimeBoostHeap[T boostableTx](opts ...timeBoostOpt[T]) *timeBoostHeap[T] {
	prioQueue := make(timeBoostableTxs, 0)
	heap.Init(&prioQueue)
	srv := &timeBoostHeap[T]{
		gFactor:   defaultMaxBoostFactor,
		prioQueue: prioQueue,
	}
	for _, o := range opts {
		o(srv)
	}
	return srv
}

type timeBoostOpt[T boostableTx] func(*timeBoostHeap[T])

// Sets the "G" parameter for time boost, in milliseconds.
func withTimeboostParameter[T boostableTx](gFactor uint64) timeBoostOpt[T] {
	return func(s *timeBoostHeap[T]) {
		s.gFactor = time.Millisecond * s.gFactor
	}
}

func (tb *timeBoostHeap[T]) PushAll(txs []T) {
	tb.Lock()
	defer tb.Unlock()
	for _, tx := range txs {
		heap.Push(&tb.prioQueue, tx)
	}
}

func (tb *timeBoostHeap[T]) PopAll() []T {
	tb.Lock()
	defer tb.Unlock()
	txs := make([]T, 0, tb.prioQueue.Len())
	for tb.prioQueue.Len() > 0 {
		txs = append(txs, heap.Pop(&tb.prioQueue).(T))
	}
	return txs
}

// A boostable tx type that contains a bid and a timestamp.
type boostableTx interface {
	hash() string
	bid() uint64
	timestamp() time.Time
	gas() uint64
}

// Defines a type that implements the heap.Interface interface from
// the standard library.
type timeBoostableTxs []boostableTx

func (tb timeBoostableTxs) Len() int      { return len(tb) }
func (tb timeBoostableTxs) Swap(i, j int) { tb[i], tb[j] = tb[j], tb[i] }

// We want to implement a priority queue using a max heap.
func (tb timeBoostableTxs) Less(i, j int) bool {
	if tb[i].bid() == tb[j].bid() {
		// Ties are broken by earliest timestamp.
		return tb[i].timestamp().Before(tb[j].timestamp())
	}
	return tb[i].bid() > tb[j].bid()
}

// Push and Pop implement the required methods for the heap interface from the standard library.
func (tb *timeBoostableTxs) Push(item any) {
	tx := item.(boostableTx)
	*tb = append(*tb, tx)
}

func (tb *timeBoostableTxs) Pop() any {
	old := *tb
	n := len(old)
	item := old[n-1]
	old[n-1] = nil // avoid memory leak
	*tb = old[0 : n-1]
	return item
}
