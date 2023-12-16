package discretetimeboost

import (
	"context"
	"fmt"
	"sync"
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

type retryQueue []*sequencerQueuedTx

func (r *retryQueue) Push(item any) {
	tx := item.(*sequencerQueuedTx)
	*r = append(*r, tx)
}

func (r *retryQueue) Pop() any {
	old := *r
	n := len(old)
	item := old[n-1]
	old[n-1] = nil // avoid memory leak
	*r = old[0 : n-1]
	return item
}

type timeBoostState struct {
	roundDeadline     time.Time
	lastRoundDeadline time.Time
	heap              *timeBoostHeap[boostableTx]
}

type sequencer struct {
	timeBoost       *timeBoostState
	recv            chan *sequencerQueuedTx
	inputQueue      []boostableTx
	inputQueueLock  sync.RWMutex
	retryQueue      retryQueue
	retryQueueLock  sync.RWMutex
	blockGasLimit   uint64
	blockSpeedLimit time.Duration
	outputFeed      chan<- *block
	currBlockNum    uint64
}

func newSequencer(outputFeed chan<- *block) *sequencer {
	heap := newTimeBoostHeap[boostableTx]()
	return &sequencer{
		recv:            make(chan *sequencerQueuedTx, 100_000),
		blockGasLimit:   100,
		blockSpeedLimit: time.Millisecond * 250,
		retryQueue:      make(retryQueue, 0),
		inputQueue:      make([]boostableTx, 0),
		outputFeed:      outputFeed,
		timeBoost: &timeBoostState{
			lastRoundDeadline: time.Now(),
			roundDeadline:     time.Now().Add(heap.gFactor),
			heap:              heap,
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
		// Await the round deadline for timeboost.
		case <-time.After(time.Until(s.timeBoost.roundDeadline)):
			var numBlocksProduced uint64
			roundTxs := s.timeBoost.heap.PopAll()
			remainingTxs := roundTxs
			fmt.Println("*** Round finished, producing blocks ***")
			fmt.Printf("time=%v, num_txs=%d\n", s.timeBoost.roundDeadline, len(roundTxs))
			fmt.Println("*** -------------------------------- ***")
			fmt.Println("")

			for {
				// Produce blocks as fast as possible until we consume all
				// txs received in the recently ended timeboost round.
				remainingTxs = s.createBlock(ctx, remainingTxs)
				numBlocksProduced += 1

				// Once we fully consume all txs in the round, we can start the next one,
				// with a delay proportional to number of blocks produced to keep the rate of
				// block production limited.
				if len(remainingTxs) == 0 {
					// Starts the next timeboost round by setting its deadline.
					// Adds a delay to the next timeboost round based on the block speed limit.
					// This delay is simply the number of blocks produced times the speed limit.
					roundDelay := time.Duration(numBlocksProduced) * s.blockSpeedLimit
					s.timeBoost.lastRoundDeadline = s.timeBoost.roundDeadline
					s.timeBoost.roundDeadline = time.Now().Add(roundDelay).Add(s.timeBoost.heap.gFactor)
					roundDuration := s.timeBoost.roundDeadline.Sub(s.timeBoost.lastRoundDeadline)
					fmt.Println("!!! Consumed full round !!!")
					fmt.Printf(
						"g_factor=%v, gas_limit=%d, block_speed_limit=%v\n",
						s.timeBoost.heap.gFactor,
						s.blockGasLimit,
						s.blockSpeedLimit,
					)
					fmt.Printf("num_blocks=%d\n", numBlocksProduced)
					fmt.Printf("round_duration=%v\n", roundDuration)
					fmt.Printf("delaying next round by %v for speed limit\n", roundDelay)
					fmt.Println("!!! ------------------- !!!")
					fmt.Println("")
					break
				}
			}
		}
	}
}

func (s *sequencer) listenForTxs(ctx context.Context) {
	for {
		select {
		case tx := <-s.recv:
			s.timeBoost.heap.Push(tx)
		case <-ctx.Done():
			return
		}
	}
}

// Question: but if we are inside of this `createBlock` function, is this the right place
// to "collect" all the txs for the round and then just time.Sleep? What about txes that are being
// received while this function is sleeping? We won't be able to process them in this case.
// Maybe we need to do it outside of here? Or perhaps in the "source" of where they originate?
func (s *sequencer) createBlock(
	ctx context.Context,
	txsToConsume []boostableTx,
) (remainingTxs []boostableTx) {
	// We gather the txs from the mempool, gathering those that are
	// coming from the retry queue first.
	queuedTxs := make([]boostableTx, 0)

	// The retry queue goes first.
	// NOTE: This toy retry queue does not consider the case where txs
	// keep failing and keep spamming the retry queue. It is just a prototype.
	// The real sequencer has a robust retry queue.
	s.retryQueueLock.Lock()
	for len(s.retryQueue) > 0 {
		queuedTxs = append(queuedTxs, (&s.retryQueue).Pop().(boostableTx))
	}
	s.retryQueueLock.Unlock()

	s.inputQueueLock.Lock()
	queuedTxs = append(queuedTxs, txsToConsume...)
	s.inputQueueLock.Unlock()

	if len(queuedTxs) == 0 {
		return
	}

	// Pre-filter txs for being malformed, nonces, etc.
	queuedTxs = s.prefilterTxs(queuedTxs)

	overflowedTxs := s.executeTxsAndProduceBlock(queuedTxs)
	if len(overflowedTxs) != 0 {
		remainingTxs = overflowedTxs
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
		number:    s.currBlockNum,
		txes:      make([]boostableTx, 0, len(readyTxs)),
		gasUsed:   0,
		timestamp: time.Now(),
	}
	overflowedTxs = make([]boostableTx, 0, len(readyTxs))
	for _, tx := range readyTxs {
		if blk.gasUsed+tx.gas() > s.blockGasLimit {
			overflowedTxs = append(overflowedTxs, tx)
		} else {
			blk.gasUsed += tx.gas()
			blk.txes = append(blk.txes, tx)
		}
	}
	// Emit the block to some output feed.
	s.currBlockNum += 1
	s.outputFeed <- blk
	return
}

func (s *sequencer) prefilterTxs(txs []boostableTx) []boostableTx {
	// TODO: Real sequencer will perform prefiltering.
	return txs
}
