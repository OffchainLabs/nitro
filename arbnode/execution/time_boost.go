package execution

import (
	"container/heap"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
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
	txInputFeed    <-chan boostableTx
	txOutputFeed   chan<- boostableTx
	prioQueue      timeBoostableTxs
	gFactor        time.Duration
	nextRoundStart chan struct{}
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
		txInputFeed:    inputFeed,
		txOutputFeed:   outputFeed,
		prioQueue:      make(timeBoostableTxs, 0),
		gFactor:        defaultMaxBoostFactor * time.Millisecond,
		nextRoundStart: make(chan struct{}, 1),
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
	log.Info("Running timeboost loop, next round released in", fmt.Sprintf("%v", s.gFactor), "milliseconds")
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
			log.Info("Releasing", fmt.Sprintf("%d", s.prioQueue.Len()), "txs from timeboost queue")
			for s.prioQueue.Len() > 0 {
				tx := heap.Pop(&s.prioQueue).(boostableTx)
				s.txOutputFeed <- tx
			}
			s.Unlock()
		case <-s.nextRoundStart:
			log.Info("Notified to start next timeboost round")
			// We need to await an external notification to start the next round of time boost.
			// This should be triggered after all txs output in a round of time boost are made
			// public in the sequencer's output feed.
			afterChan = time.After(s.gFactor)
		case <-ctx.Done():
			return
		}
	}
}

func (s *timeBoostService) startNextRound() {
	s.nextRoundStart <- struct{}{}
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
