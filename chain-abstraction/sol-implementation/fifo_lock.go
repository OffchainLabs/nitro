package solimpl

import (
	"sync"
)

type FIFO struct {
	lock      sync.Mutex
	queue     chan struct{}
	waitQueue chan chan struct{}
}

func NewFIFO(capacity int) *FIFO {
	return &FIFO{
		queue:     make(chan struct{}, 1),
		waitQueue: make(chan chan struct{}, capacity),
	}
}

func (f *FIFO) Lock() bool {
	waitCh := make(chan struct{})
	f.lock.Lock()
	select {
	// If the queue is empty, we can lock immediately
	case f.queue <- struct{}{}:
		f.lock.Unlock()
	// If the queue is not empty, we need to wait our turn
	default:
		// We add our wait channel to the wait queue
		if len(f.waitQueue) == cap(f.waitQueue) {
			f.lock.Unlock()
			// If the wait queue is full, lock acquisition fails
			return false
		}
		f.waitQueue <- waitCh
		f.lock.Unlock()
		// We wait for our turn
		<-waitCh
	}
	// Lock acquisition succeeded
	return true
}

func (f *FIFO) Unlock() {
	f.lock.Lock()
	defer f.lock.Unlock()
	select {
	// If the queue is not empty, we unlock and signal the next waiter
	case <-f.queue:
		// If there are waiters, we signal the next one
		if len(f.waitQueue) > 0 {
			// We pop the next waiter from the queue
			nextWaitCh := <-f.waitQueue
			// We acquire the lock for the next waiter
			f.queue <- struct{}{}
			// We signal the next waiter
			close(nextWaitCh)
		}
	default:
		panic("attempt to unlock unlocked mutex")
	}
}
