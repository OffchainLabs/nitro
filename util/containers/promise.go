package containers

import (
	"context"
	"errors"
	"sync/atomic"
)

type PromiseInterface[R any] interface {
	Ready() bool
	ReadyChan() chan struct{}
	Await(ctx context.Context) (R, error)
	Current() (R, error) // doesn't wait
	Cancel()
}

var ErrNotReady error = errors.New("not ready")

type Promise[R any] struct {
	chanReady chan struct{}
	result    R
	err       error
	produced  atomic.Bool
	cancel    func()
}

func (p *Promise[R]) Ready() bool {
	select {
	case <-p.chanReady:
		return true
	default:
		return false
	}
}

func (p *Promise[R]) ReadyChan() chan struct{} {
	return p.chanReady
}

func (p *Promise[R]) Await(ctx context.Context) (R, error) {
	select {
	case <-p.chanReady:
		return p.result, p.err
	case <-ctx.Done():
		var empty R
		p.Cancel()
		return empty, ctx.Err()
	}
}

func (p *Promise[R]) Current() (R, error) {
	if !p.Ready() {
		var empty R
		return empty, ErrNotReady
	}
	return p.result, p.err
}

func (p *Promise[R]) Cancel() {
	if p.cancel == nil {
		return
	}
	if p.Ready() {
		return
	}
	p.cancel()
}

func (p *Promise[R]) ProduceErrorSafe(err error) error {
	if !p.produced.CompareAndSwap(false, true) {
		return errors.New("cannot produce two values")
	}
	p.err = err
	close(p.chanReady)
	return nil
}

func (p *Promise[R]) ProduceError(err error) {
	errSafe := p.ProduceErrorSafe(err)
	if errSafe != nil {
		panic(errSafe)
	}
}

func (p *Promise[R]) ProduceSafe(value R) error {
	if !p.produced.CompareAndSwap(false, true) {
		return errors.New("cannot produce two values")
	}
	p.result = value
	close(p.chanReady)
	return nil
}

func (p *Promise[R]) Produce(value R) {
	errSafe := p.ProduceSafe(value)
	if errSafe != nil {
		panic(errSafe)
	}
}

// cancel might be called multiple times while no value or error is produced
// cancel will be called by Await if it's context is done
func NewPromise[R any](cancel func()) Promise[R] {
	return Promise[R]{
		chanReady: make(chan struct{}),
		cancel:    cancel,
	}
}

func NewReadyPromise[R any](val R, err error) PromiseInterface[R] {
	promise := NewPromise[R](nil)
	if err == nil {
		promise.Produce(val)
	} else {
		promise.ProduceError(err)
	}
	return &promise
}
