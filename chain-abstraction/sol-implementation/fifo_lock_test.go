package solimpl

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestFIFOSequentialLocking(t *testing.T) {
	f := NewFIFO(10)
	var output []int
	var wg sync.WaitGroup
	wg.Add(10)
	for i := 0; i < 10; i++ {
		a := i
		go func() {
			f.Lock()
			output = append(output, a)
			wg.Done()
		}()
		time.Sleep(time.Millisecond)
	}
	for i := 0; i < 10; i++ {
		f.Unlock()
		time.Sleep(time.Millisecond)
	}
	wg.Wait()
	require.Equal(t, []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}, output)
}

func TestFIFOUnlockPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for unlocking an already unlocked FIFO")
		}
	}()

	fifo := NewFIFO(1)

	// Try to unlock when the FIFO is already unlocked
	fifo.Unlock()
}

func TestFIFOOnlyOneLockAllowed(t *testing.T) {
	fifo := NewFIFO(1)

	// Acquire the lock
	fifo.Lock()

	// Try to acquire the lock from another goroutine
	doneCh := make(chan struct{})
	go func() {
		defer close(doneCh)
		fifo.Lock()
		defer fifo.Unlock()
	}()

	// Wait for some time to ensure that the second goroutine doesn't acquire the lock
	select {
	case <-doneCh:
		t.Error("Second lock acquisition didn't fail within the expected time")
	case <-time.After(time.Millisecond * 100):
		t.Log("Second lock acquisition failed as expected")
	}

	// Release the lock
	fifo.Unlock()
}
