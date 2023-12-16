package discretetimeboost

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type block struct {
	number    uint64
	txes      []boostableTx
	gasUsed   uint64
	timestamp time.Time
}

type sequencerQueuedTx struct {
	gasTipCap       uint64
	id              string
	gasToUse        uint64
	firstAppearance time.Time
}

func (t *sequencerQueuedTx) bid() uint64 {
	return t.gasTipCap
}

func (t *sequencerQueuedTx) timestamp() time.Time {
	return t.firstAppearance
}

func (t *sequencerQueuedTx) hash() string {
	return t.id
}

func (t *sequencerQueuedTx) gas() uint64 {
	return t.gasToUse
}

type sequencerQueue []*sequencerQueuedTx

func (r *sequencerQueue) Push(item any) {
	tx := item.(*sequencerQueuedTx)
	*r = append(*r, tx)
}

func (r *sequencerQueue) Pop() any {
	old := *r
	n := len(old)
	item := old[n-1]
	old[n-1] = nil // avoid memory leak
	*r = old[0 : n-1]
	return item
}

type timeBoostState struct {
	currentRoundOngoing *atomic.Bool
	roundDeadline       time.Time
	heap                *timeBoostHeap[boostableTx]
}

type sequencer struct {
	timeBoost       *timeBoostState
	recv            chan *sequencerQueuedTx
	inputQueue      []boostableTx
	inputQueueLock  sync.RWMutex
	retryQueue      sequencerQueue
	retryQueueLock  sync.RWMutex
	blockGasLimit   uint64
	blockSpeedLimit time.Duration
	outputFeed      chan<- *block
}

func newSequencer(outputFeed chan<- *block) *sequencer {
	heap := newTimeBoostHeap[boostableTx]()
	return &sequencer{
		recv:            make(chan *sequencerQueuedTx, 100_000),
		blockGasLimit:   100,
		blockSpeedLimit: time.Millisecond * 250,
		retryQueue:      make(sequencerQueue, 0),
		inputQueue:      make([]boostableTx, 0),
		outputFeed:      outputFeed,
		timeBoost: &timeBoostState{
			currentRoundOngoing: &atomic.Bool{},
			roundDeadline:       time.Now().Add(heap.gFactor),
			heap:                heap,
		},
	}
}

func (s *sequencer) start(ctx context.Context) {
	fmt.Println("Starting sequencer")
	go s.listenForTxs(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		default:
			fmt.Println("Running block production")
			var numBlocksProduced uint64
			for {
				// Produce blocks as fast as possible until we consume all
				// txs received in the recently ended timeboost round.
				consumedFullRound := s.createBlock(ctx)
				numBlocksProduced += 1

				// Once we fully consume all txs in the round, we can start the next one,
				// with a delay proportional to number of blocks produced to keep the rate of
				// block production limited.
				if consumedFullRound {
					// Starts the next timeboost round by setting its deadline.
					// Adds a delay to the next timeboost round based on the block speed limit.
					// This delay is simply the number of blocks produced times the speed limit.
					roundDelay := time.Duration(numBlocksProduced) * s.blockSpeedLimit
					s.timeBoost.roundDeadline = time.Now().Add(roundDelay).Add(s.timeBoost.heap.gFactor)
					s.timeBoost.currentRoundOngoing.Store(true)
				}
			}
		}
	}
}

func (s *sequencer) listenForTxs(ctx context.Context) {
	for {
		select {
		case tx := <-s.recv:
			s.inputQueueLock.Lock()
			s.inputQueue = append(s.inputQueue, tx)
			s.inputQueueLock.Unlock()
		case <-ctx.Done():
			return
		}
	}
}

// Question: but if we are inside of this `createBlock` function, is this the right place
// to "collect" all the txs for the round and then just time.Sleep? What about txes that are being
// received while this function is sleeping? We won't be able to process them in this case.
// Maybe we need to do it outside of here? Or perhaps in the "source" of where they originate?
func (s *sequencer) createBlock(ctx context.Context) (consumedFullRound bool) {
	// We gather the txs from the mempool, gathering those that are
	// coming from the retry queue first.
	queuedTxs := make([]boostableTx, 0)

	s.retryQueueLock.Lock()
	for len(s.retryQueue) > 0 {
		queuedTxs = append(queuedTxs, (&s.retryQueue).Pop().(boostableTx))
	}
	s.retryQueueLock.Unlock()

	s.inputQueueLock.Lock()
	queuedTxs = append(queuedTxs, s.inputQueue...)
	s.inputQueueLock.Unlock()

	if len(queuedTxs) == 0 {
		return
	}

	fmt.Println("Grabbed a few!", len(queuedTxs))

	// Pre-filter txs for being malformed, nonces, etc.
	queuedTxs = s.prefilterTxs(queuedTxs)

	// Figure out which timeboost heap this tx should go into.
	s.timeBoost.heap.PushAll(queuedTxs)

	var txes []boostableTx
	if s.timeBoost.currentRoundOngoing.Load() {
		// If we are still in a round, wait the remaining time to round completion.
		<-time.After(time.Until(s.timeBoost.roundDeadline))

		// The round is done, which means we can proceed with popping all the txs
		// from the timeboost heap and proceeding with producing a block.

		// We pop everything from the timeboost heap as we
		// can proceed with blocks from them as needed.
		// If the gas limit is reached, txs will get put into a retry queue,
		// in which they will get retried by this function which will run again immediately.
		txes = s.timeBoost.heap.PopAll()

		// We open up the next round's buffer for filling, but we don't know yet
		// when the next round's timer will end until we finish producing all the blocks here.
		s.timeBoost.currentRoundOngoing.Store(false)
	}
	overflowedTxs := s.executeTxsAndProduceBlock(txes)
	if len(overflowedTxs) == 0 {
		consumedFullRound = true
		return
	}
	s.retryQueueLock.Lock()
	for _, overflowedTx := range overflowedTxs {
		(&s.retryQueue).Push(overflowedTx)
	}
	s.retryQueueLock.Unlock()
	return
}

// Check if the txs fit into the block gas limit.
// Otherwise, return the list of indices that did not fit.
func (s *sequencer) executeTxsAndProduceBlock(readyTxs []boostableTx) (overflowedTxs []boostableTx) {
	blk := &block{
		number:    0,
		txes:      make([]boostableTx, 0, len(readyTxs)),
		gasUsed:   0,
		timestamp: time.Now(),
	}
	overflowedTxs = make([]boostableTx, 0, len(readyTxs))
	for _, tx := range readyTxs {
		if blk.gasUsed+tx.gas() > s.blockGasLimit {
			overflowedTxs = append(overflowedTxs, tx)
		} else {
			fmt.Println("non-overflow")
			blk.gasUsed += tx.gas()
			blk.txes = append(blk.txes, tx)
		}
	}
	// Emit the block to some output feed.
	s.outputFeed <- blk
	return
}

func (s *sequencer) prefilterTxs(txs []boostableTx) []boostableTx {
	// TODO: Real sequencer will perform prefiltering.
	return txs
}
