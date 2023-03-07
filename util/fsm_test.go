package util

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSimpleFSM(t *testing.T) {
	var startState doorState
	startState = doorStateClosed
	transitions := []*FsmEvent[doorEvent, doorState]{
		{Typ: Open{}, From: []doorState{doorStateClosed}, To: doorStateOpened},
		{Typ: Close{}, From: []doorState{doorStateOpened}, To: doorStateClosed},
	}
	fsm, err := NewFsm(startState, transitions)
	require.NoError(t, err)

	require.Equal(t, uint8(doorStateClosed), uint8(fsm.Current()))

	err = fsm.Do(Close{})
	require.ErrorIs(t, err, ErrFsmInvalidTransition)

	require.Equal(t, uint8(doorStateClosed), uint8(fsm.Current()))

	err = fsm.Do(Open{intruderName: "vitalik"})
	require.NoError(t, err)

	require.Equal(t, uint8(doorStateOpened), uint8(fsm.Current()))

	err = fsm.Do(Close{})
	require.NoError(t, err)

	require.Equal(t, uint8(doorStateClosed), uint8(fsm.Current()))

	err = fsm.Do(Close{})
	require.ErrorIs(t, err, ErrFsmInvalidTransition)
}

type doorEvent interface {
	isDoorEvent() bool
	Stringer
}

type Open struct {
	intruderName string
}

func (o Open) String() string {
	return "open"
}

func (o Open) isDoorEvent() bool {
	return true
}

type Close struct{}

func (c Close) String() string {
	return "close"
}

func (c Close) isDoorEvent() bool {
	return true
}

type doorState uint8

const (
	doorStateInvalid = iota
	doorStateOpened
	doorStateClosed
)

func (d doorState) String() string {
	switch d {
	case doorStateOpened:
		return "opened"
	case doorStateClosed:
		return "closed"
	default:
		return "invalid"
	}
}
