package util

import (
	"fmt"
	"sync"

	"github.com/pkg/errors"
)

var (
	ErrFsmInvalidTransition = errors.New("invalid state transition")
	ErrFsmEventNotFound     = errors.New("event not found")
)

// FsmEvent defines an event in the finite state machine, which includes
// a type, a set of source states, and a destination state
type FsmEvent[E, T fmt.Stringer] struct {
	Typ  E
	From []T
	To   T
}

// CurrentState of the finite state machine, including a source event (if any)
// and the actual state value.
type CurrentState[E, T fmt.Stringer] struct {
	SourceEvent E
	State       T
}

// Fsm defines a generic, finite state machine which can transition from a series
// of predefined states and events. It can optionally track the executed state
// transitions for debugging or exploratory purposes.
type Fsm[E, T fmt.Stringer] struct {
	lock                sync.RWMutex
	trackingTransitions bool
	transitionsExecuted []*transitionMade[E, T]
	curr                *CurrentState[E, T]
	validEvents         map[string]bool
	validStates         map[string]bool
	validTransitions    map[internalKey]T
}

// FsmOpt defines a configuration option for the fsm.
type FsmOpt[E, T fmt.Stringer] func(f *Fsm[E, T])

// WithTrackedTransitions configures the fsm to track all executed state
// transitions in an slice.
// NOTE: The growth of this slice is unbounded so this method is NOT
// recommended in production.
func WithTrackedTransitions[E, T fmt.Stringer]() FsmOpt[E, T] {
	return func(f *Fsm[E, T]) {
		f.trackingTransitions = true
	}
}

// NewFsm initializes an FSM from a list of valid events / states type
// in a transition table.
//
//	var startState doorState
//	startState = doorStateClosed
//	transitions := []*FsmEvent[doorEvent, doorState]{
//		{Typ: Open{}, From: []doorState{doorStateClosed}, To: doorStateOpened},
//		{Typ: Close{}, From: []doorState{doorStateOpened}, To: doorStateClosed},
//	}
//	fsm, err := NewFsm(startState, transitions)
//
// the example above showcases how to define an fsm for a simple door
// that can be opened and closed as long.
func NewFsm[E, T fmt.Stringer](
	startState T,
	transitionTable []*FsmEvent[E, T],
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
	for _, ev := range transitionTable {
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

// CanTransition checks if the fsm can transition from
// its current state using a specified event.
func (f *Fsm[E, T]) CanTransition(event E) bool {
	f.lock.RLock()
	defer f.lock.RUnlock()
	_, ok := f.validTransitions[internalKey{
		eventType: event.String(),
		src:       f.curr.State.String(),
	}]
	return ok
}

// Do executes an event based on the current state of the fsm
// and updates the current state to the destination.
// If the transition is not allowed based on the transition table
// from the fsm initialization, we return ErrFsmInvalidTransition.
// If the event is not found in the transition table, we return ErrFsmEventNotFound.
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
				return errors.Wrapf(ErrFsmInvalidTransition, "source: %s, event %s", src, event)
			}
		}
		return errors.Wrapf(ErrFsmEventNotFound, "event: %s", event)
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

// Current returns the current state of the FSM, containing the state value
// and source event it used to get there, if any.
func (f *Fsm[E, T]) Current() *CurrentState[E, T] {
	f.lock.RLock()
	defer f.lock.RUnlock()
	return f.curr
}

// An internal key the fsm uses to store data in maps.
type internalKey struct {
	eventType string
	src       string
}

// A struct keeping track of a state transition in the fsm.
type transitionMade[E, T fmt.Stringer] struct {
	From  T
	To    T
	Event E
}
