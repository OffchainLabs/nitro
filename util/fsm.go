package util

import (
	"errors"
)

var (
	ErrFSMInvalidTransition = errors.New("invalid state transition")
)

type FsmOpt[E, T stringer] func(f *Fsm[E, T])

type stringer interface {
	String() string
}

// TODO: Should the event carry data?
type FsmEvent[E, T stringer] struct {
	Typ  E
	From []T
	To   T
}

type Fsm[E, T stringer] struct {
	trackingTransitions bool
	currState           T
}

func WithTrackedTransitions[E, T stringer]() FsmOpt[E, T] {
	return func(f *Fsm[E, T]) {
		f.trackingTransitions = true
	}
}

func NewFsm[E, T stringer](
	events []*FsmEvent[E, T],
	opts ...FsmOpt[E, T],
) (*Fsm[E, T], error) {
	f := &Fsm[E, T]{}
	for _, opt := range opts {
		opt(f)
	}
	return f, nil
}

func (f *Fsm[E, T]) Do(event E) error {
	return nil
}

func (f *Fsm[E, T]) Current() T {
	return f.currState
}

func (f *Fsm[E, T]) A() T {
	return f.currState
}

func (f *Fsm[E, T]) B() T {
	return f.currState
}

func (f *Fsm[E, T]) C() T {
	return f.currState
}

func (f *Fsm[E, T]) D() T {
	return f.currState
}
