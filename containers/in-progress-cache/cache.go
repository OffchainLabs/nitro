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
	awaitingCompletion map[K][]chan Response[V]
	lock               sync.RWMutex
}

type Response[V any] struct {
	value V
	err   error
}

func New[K comparable, V any]() *Cache[K, V] {
	return &Cache[K, V]{
		inProgress:         make(map[K]bool),
		awaitingCompletion: make(map[K][]chan Response[V]),
	}
}

// Compute an expensive closure. The request must be representable as a string.
func (c *Cache[K, V]) Compute(requestId K, f func() (V, error)) (V, error) {
	c.lock.RLock()
	if ok := c.inProgress[requestId]; ok {
		pendingRequestsCounter.Inc(1)

		c.lock.RUnlock()
		responseChan := make(chan Response[V])
		defer close(responseChan)

		c.lock.Lock()
		c.awaitingCompletion[requestId] = append(c.awaitingCompletion[requestId], responseChan)
		c.lock.Unlock()
		response := <-responseChan
		return response.value, response.err
	}
	c.lock.RUnlock()

	c.lock.Lock()
	c.inProgress[requestId] = true
	inFlightRequestsCounter.Inc(1)
	c.lock.Unlock()

	// Do expensive operation and notify all waiting goroutines of the result as well as the error
	result, err := f()

	c.lock.RLock()
	receiversWaiting, ok := c.awaitingCompletion[requestId]
	c.lock.RUnlock()

	if ok {
		for _, ch := range receiversWaiting {
			ch <- Response[V]{result, err}
		}
	}

	c.lock.Lock()
	c.inProgress[requestId] = false
	c.awaitingCompletion[requestId] = make([]chan Response[V], 0)
	c.lock.Unlock()
	return result, err
}
