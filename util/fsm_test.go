package util

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestFSM(t *testing.T) {
	transitions := []*FsmEvent[doorEvent, doorState]{
		{Typ: Open, From: []doorState{ClosedState}, To: OpenedState},
		{Typ: Close, From: []doorState{OpenedState}, To: ClosedState},
	}
	fsm, err := NewFsm(transitions)
	require.NoError(t, err)

	err = fsm.Do(Open)
	require.NoError(t, err)

	err = fsm.Do(Close)
	require.NoError(t, err)

	err = fsm.Do(Close)
	require.ErrorIs(t, err, ErrFSMInvalidTransition)

	switch fsm.Current() {
	case OpenedState:
	case ClosedState:
	case InvalidState:
	}
}

type doorState uint

const (
	InvalidState doorState = iota
	OpenedState
	ClosedState
)

type doorEvent uint

func (e doorEvent) String() string {
	return ""
}

const (
	InvalidEvent doorEvent = iota
	Open
	Close
)

func (v doorState) String() string {
	return ""
}
