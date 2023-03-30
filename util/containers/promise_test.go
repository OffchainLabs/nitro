package containers

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestPromise(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tempPromise := NewPromise[int]()

	tempPromise.Produce(1)
	res, err := tempPromise.Await(ctx)
	if res != 1 || err != nil {
		t.Fatal("unexpected Promise.Await")
	}
	res, err = tempPromise.Current()
	if res != 1 || err != nil {
		t.Fatal("unexpected Promise.Current when ready")
	}

	tempPromise = NewPromise[int]()
	res, err = tempPromise.Current()
	if res != 0 || !errors.Is(err, ErrNotReady) {
		t.Fatal("unexpected Promise.Current when not ready")
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		res, err = tempPromise.Await(ctx)
		wg.Done()
	}()
	tempPromise.Produce(2)
	wg.Wait()
	if res != 2 || err != nil {
		t.Fatal("unexpected Promise.Await in parallel")
	}
	res, err = tempPromise.Current()
	if res != 2 || err != nil {
		t.Fatal("unexpected Promise.Current 2nd time")
	}

	cancelCalled := int64(0)
	tempPromise = NewPromise[int]()
	tempPromise.SetCancel(func() { atomic.AddInt64(&cancelCalled, 1) })
	shortCtx, shortCancel := context.WithTimeout(ctx, time.Millisecond*100)
	defer shortCancel()
	res, err = tempPromise.Await(shortCtx)
	if res != 0 || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal("unexpected Promise.Await with timeout")
	}
	if atomic.LoadInt64(&cancelCalled) != 1 {
		t.Fatal("cancel not called by await on timeout")
	}
	tempPromise.Cancel()
	if atomic.LoadInt64(&cancelCalled) != 2 {
		t.Fatal("cancel not called by promise.Cancel")
	}
}
