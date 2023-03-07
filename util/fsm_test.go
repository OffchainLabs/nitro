package util

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestFSM_OpenClose(t *testing.T) {
	var startState doorState
	startState = doorStateClosed
	transitions := []*FsmEvent[doorEvent, doorState]{
		{Typ: Open{}, From: []doorState{doorStateClosed}, To: doorStateOpened},
		{Typ: Close{}, From: []doorState{doorStateOpened}, To: doorStateClosed},
	}
	fsm, err := NewFsm(startState, transitions)
	require.NoError(t, err)

	t.Run("assert state state", func(t *testing.T) {
		curr := fsm.Current()
		require.Equal(t, uint8(doorStateClosed), uint8(curr.State))
	})
	t.Run("invalid transition", func(t *testing.T) {
		err = fsm.Do(Close{})
		require.ErrorIs(t, err, ErrFsmInvalidTransition)
	})
	t.Run("valid transitions", func(t *testing.T) {
		err = fsm.Do(Open{intruderName: "vitalik"})
		require.NoError(t, err)

		curr := fsm.Current()
		require.Equal(t, uint8(doorStateOpened), uint8(curr.State))
		openedEv, ok := curr.SourceEvent.(Open)
		require.Equal(t, true, ok)
		require.Equal(t, "vitalik", openedEv.intruderName)

		err = fsm.Do(Close{})
		require.NoError(t, err)

		curr = fsm.Current()
		require.Equal(t, uint8(doorStateClosed), uint8(curr.State))

		err = fsm.Do(Open{intruderName: "vitalik"})
		require.NoError(t, err)

		curr = fsm.Current()
		require.Equal(t, uint8(doorStateOpened), uint8(curr.State))
	})
	t.Run("unknown event", func(t *testing.T) {
		err = fsm.Do(SchrodingersDoorOpenedAndClosed{})
		require.ErrorIs(t, err, ErrFsmEventNotFound)
	})
}

func TestFSM_TrackTransitions(t *testing.T) {
	var startState doorState
	startState = doorStateClosed
	transitions := []*FsmEvent[doorEvent, doorState]{
		{Typ: Open{}, From: []doorState{doorStateClosed}, To: doorStateOpened},
		{Typ: Close{}, From: []doorState{doorStateOpened}, To: doorStateClosed},
	}
	fsm, err := NewFsm(
		startState,
		transitions,
		WithTrackedTransitions[doorEvent, doorState](),
	)
	require.NoError(t, err)

	err = fsm.Do(Open{intruderName: "vitalik"})
	require.NoError(t, err)

	err = fsm.Do(Close{})
	require.NoError(t, err)

	err = fsm.Do(Open{intruderName: "vitalik"})
	require.NoError(t, err)

	err = fsm.Do(Open{})
	require.ErrorIs(t, err, ErrFsmInvalidTransition)

	require.Equal(t, 3, len(fsm.transitionsExecuted))
	require.Equal(t, uint8(doorStateClosed), uint8(fsm.transitionsExecuted[0].From))
	require.Equal(t, uint8(doorStateOpened), uint8(fsm.transitionsExecuted[0].To))
	_, ok := fsm.transitionsExecuted[0].Event.(Open)
	require.Equal(t, true, ok)

	require.Equal(t, uint8(doorStateOpened), uint8(fsm.transitionsExecuted[1].From))
	require.Equal(t, uint8(doorStateClosed), uint8(fsm.transitionsExecuted[1].To))
	_, ok = fsm.transitionsExecuted[1].Event.(Close)
	require.Equal(t, true, ok)

	require.Equal(t, uint8(doorStateClosed), uint8(fsm.transitionsExecuted[2].From))
	require.Equal(t, uint8(doorStateOpened), uint8(fsm.transitionsExecuted[2].To))
	_, ok = fsm.transitionsExecuted[2].Event.(Open)
	require.Equal(t, true, ok)
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

type SchrodingersDoorOpenedAndClosed struct{}

func (c SchrodingersDoorOpenedAndClosed) String() string {
	return "open_and_closed"
}

func (c SchrodingersDoorOpenedAndClosed) isDoorEvent() bool {
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
