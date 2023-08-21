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

type boostableTx interface {
	id() string
	bid() uint64
	timestamp() time.Time
	innerTx() *types.Transaction
}

type timeBoostService struct {
	sync.Mutex
	txInputFeed  <-chan boostableTx
	txOutputFeed chan<- boostableTx
	prioQueue    timeBoostableTxs
	gFactor      time.Duration
}

type opt func(*timeBoostService)

func withMaxBoostFactor(gFactor uint64) opt {
	return func(s *timeBoostService) {
		s.gFactor = time.Millisecond * s.gFactor
	}
}

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
			afterChan = time.After(s.gFactor)
		case <-ctx.Done():
			return
		}
	}
}

type timeBoostableTxs []boostableTx

func (tb timeBoostableTxs) Len() int      { return len(tb) }
func (tb timeBoostableTxs) Swap(i, j int) { tb[i], tb[j] = tb[j], tb[i] }

// We want to implement a priority queue using a max heap.
func (tb timeBoostableTxs) Less(i, j int) bool {
	if tb[i].bid() == tb[j].bid() {
		return tb[i].timestamp().Before(tb[j].timestamp())
	}
	return tb[i].bid() > tb[j].bid()
}

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
