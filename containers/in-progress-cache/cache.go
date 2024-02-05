package inprogresscache

import (
	"sync"

	"github.com/ethereum/go-ethereum/metrics"
)

var (
	inFlightRequestsCounter = metrics.NewRegisteredCounter("arb/validator/inprogresscache/inflight", nil)
	pendingRequestsCounter  = metrics.NewRegisteredCounter("arb/validator/inprogresscache/pending", nil)
)

// Cache for expensive computations that ensures only
// one request is in-flight at a time. If a future request comes in with the same request id
// as the ongoing computation, a goroutine is spawned that awaits the computation's completion
// instead of kicking off two expensive computations.
type Cache[K comparable, V any] struct {
	inProgress         map[K]bool
	awaitingCompletion map[K][]chan V
	lock               sync.RWMutex
}

func New[K comparable, V any]() *Cache[K, V] {
	return &Cache[K, V]{
		inProgress:         make(map[K]bool),
		awaitingCompletion: make(map[K][]chan V),
	}
}

// Compute an expensive closure. The request must be representable as a string.
func (c *Cache[K, V]) Compute(requestId K, f func() (V, error)) (V, error) {
	c.lock.RLock()
	if ok := c.inProgress[requestId]; ok {
		pendingRequestsCounter.Inc(1)

		c.lock.RUnlock()
		responseChan := make(chan V, 1)
		defer close(responseChan)

		c.lock.Lock()
		c.awaitingCompletion[requestId] = append(c.awaitingCompletion[requestId], responseChan)
		c.lock.Unlock()
		val := <-responseChan
		return val, nil
	}
	c.lock.RUnlock()

	c.lock.Lock()
	c.inProgress[requestId] = true
	inFlightRequestsCounter.Inc(1)
	c.lock.Unlock()

	// Do expensive operation
	var zeroVal V
	result, err := f()
	if err != nil {
		return zeroVal, err
	}

	c.lock.RLock()
	receiversWaiting, ok := c.awaitingCompletion[requestId]
	c.lock.RUnlock()

	if ok {
		for _, ch := range receiversWaiting {
			ch <- result
		}
	}

	c.lock.Lock()
	c.inProgress[requestId] = false
	c.awaitingCompletion[requestId] = make([]chan V, 0)
	c.lock.Unlock()
	return result, nil
}
