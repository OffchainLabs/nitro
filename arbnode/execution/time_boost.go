package execution

import (
	"container/heap"
	"context"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
)

const (
	defaultMaxBoostFactor = 500 // defined as the `g` parameter in the time boost specification.
)

// A boostable tx type that contains a bid and a timestamp.
type boostableTx interface {
	id() string
	bid() uint64
	timestamp() time.Time
	innerTx() *types.Transaction
}

// Time boost service defines a background service that can receive boostable
// transactions from an input channel and output potentially reordered txs to
// an output channel using the discrete time boost protocol for sequencing.
type timeBoostService struct {
	sync.Mutex
	txInputFeed  <-chan boostableTx
	txOutputFeed chan<- boostableTx
	prioQueue    timeBoostableTxs
	gFactor      time.Duration
}

type opt func(*timeBoostService)

// Sets the "G" parameter for time boost.
func withMaxBoostFactor(gFactor uint64) opt {
	return func(s *timeBoostService) {
		s.gFactor = time.Millisecond * s.gFactor
	}
}

// Initializes a time boost service from a channel of input transactions
// and releases potentially reordered txs to a specified output feed.
func newTimeBoostService(
	inputFeed <-chan boostableTx,
	outputFeed chan<- boostableTx,
	opts ...opt,
) *timeBoostService {
	prioQueue := make(timeBoostableTxs, 0)
	heap.Init(&prioQueue)
	s := &timeBoostService{
		txInputFeed:  inputFeed,
		txOutputFeed: outputFeed,
		prioQueue:    make(timeBoostableTxs, 0),
		gFactor:      defaultMaxBoostFactor * time.Millisecond,
	}
	for _, o := range opts {
		o(s)
	}
	return s
}

// Runs a discrete time boost service for the sequencer. It receives txs
// which it inserts into a max heap where txs with the highest
// bid win (ties are broken by timestamp).
func (s *timeBoostService) run(ctx context.Context) {
	afterChan := time.After(s.gFactor)
	for {
		select {
		case tx := <-s.txInputFeed:
			// Process a tx by inserting it into the priority queue.
			s.Lock()
			heap.Push(&s.prioQueue, tx)
			s.Unlock()
		case <-afterChan:
			// Releasing all items from the queue.
			s.Lock()
			for s.prioQueue.Len() > 0 {
				tx := heap.Pop(&s.prioQueue).(boostableTx)
				s.txOutputFeed <- tx
			}
			s.Unlock()
			// We start the next round of time boost.
			afterChan = time.After(s.gFactor)
		case <-ctx.Done():
			return
		}
	}
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
