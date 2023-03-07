package util

import (
	"errors"
	"sync"
)

var (
	ErrFsmInvalidTransition = errors.New("invalid state transition")
	ErrFsmEventNotFound     = errors.New("event not found")
)

type FsmOpt[E, T Stringer] func(f *Fsm[E, T])

type Stringer interface {
	String() string
}

type FsmEvent[E, T Stringer] struct {
	Typ  E
	From []T
	To   T
}

type CurrentState[E, T Stringer] struct {
	SourceEvent E
	State       T
}

func WithTrackedTransitions[E, T Stringer]() FsmOpt[E, T] {
	return func(f *Fsm[E, T]) {
		f.trackingTransitions = true
	}
}

type internalKey struct {
	eventType string
	src       string
}

type transitionMade[E, T Stringer] struct {
	From  T
	To    T
	Event E
}

type Fsm[E, T Stringer] struct {
	lock                sync.RWMutex
	trackingTransitions bool
	transitionsExecuted []*transitionMade[E, T]
	curr                *CurrentState[E, T]
	validEvents         map[string]bool
	validStates         map[string]bool
	validTransitions    map[internalKey]T
}

func NewFsm[E, T Stringer](
	startState T,
	eventConfig []*FsmEvent[E, T],
	opts ...FsmOpt[E, T],
) (*Fsm[E, T], error) {
	f := &Fsm[E, T]{
		curr: &CurrentState[E, T]{
			State: startState,
		},
		transitionsExecuted: make([]*transitionMade[E, T], 0),
	}
	for _, opt := range opts {
		opt(f)
	}
	f.validTransitions = make(map[internalKey]T)
	f.validEvents = make(map[string]bool)
	f.validStates = make(map[string]bool)
	for _, ev := range eventConfig {
		for _, from := range ev.From {
			f.validTransitions[internalKey{
				eventType: ev.Typ.String(),
				src:       from.String(),
			}] = ev.To
			f.validStates[from.String()] = true
		}
		f.validEvents[ev.Typ.String()] = true
	}
	return f, nil
}

func (f *Fsm[E, T]) CanTransition(event E) bool {
	f.lock.RLock()
	defer f.lock.RUnlock()
	_, ok := f.validTransitions[internalKey{
		eventType: event.String(),
		src:       f.curr.State.String(),
	}]
	return ok
}

func (f *Fsm[E, T]) Do(event E) error {
	f.lock.Lock()
	defer f.lock.Unlock()
	src := f.curr.State
	key := internalKey{
		eventType: event.String(),
		src:       src.String(),
	}
	to, ok := f.validTransitions[key]
	if !ok {
		for key := range f.validTransitions {
			if key.eventType == event.String() {
				return ErrFsmInvalidTransition
			}
		}
		return ErrFsmEventNotFound
	}
	f.curr = &CurrentState[E, T]{
		State:       to,
		SourceEvent: event,
	}
	if f.trackingTransitions {
		f.transitionsExecuted = append(
			f.transitionsExecuted,
			&transitionMade[E, T]{From: src, To: to, Event: event},
		)
	}
	return nil
}

func (f *Fsm[E, T]) Current() *CurrentState[E, T] {
	f.lock.RLock()
	defer f.lock.RUnlock()
	return f.curr
}
