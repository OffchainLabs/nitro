package readymarker

import (
	"context"
	"errors"
	"sync/atomic"
)

type ReadyMarkerInt interface {
	Ready() bool
	ReadyChan() chan struct{}
	WaitReady(ctx context.Context) error
}

var ErrNotReady error = errors.New("not ready")

type ReadyMarker struct {
	chanReady chan struct{}
	boolReady int32
	err       error
}

func (d *ReadyMarker) Ready() bool {
	return atomic.LoadInt32(&d.boolReady) != 0
}

func (d *ReadyMarker) ReadyChan() chan struct{} {
	return d.chanReady
}

func (d *ReadyMarker) WaitReady(ctx context.Context) error {
	select {
	case <-d.chanReady:
		return d.err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (d *ReadyMarker) TestReady() error {
	if !d.Ready() {
		return ErrNotReady
	}
	return d.err
}

func (d *ReadyMarker) SignalReady(err error) {
	d.err = err
	atomic.StoreInt32(&d.boolReady, 1)
	close(d.chanReady)
}

func NewReadyMarker() ReadyMarker {
	return ReadyMarker{
		boolReady: 0,
		chanReady: make(chan struct{}),
	}
}
