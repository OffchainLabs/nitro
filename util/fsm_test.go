package util

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

type doorEvent interface {
	isDoorEvent() bool
	fmt.Stringer
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

// Creates a simple test where we have a door open/closed state machine.
func TestFSM_OpenClose(t *testing.T) {
	//nolint:all
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

// Checks if our FSM can correctly track state transitions when configured to do so.
func TestFSM_TrackTransitions(t *testing.T) {
	//nolint:all
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

type hvacState uint8

const (
	hvacStateInvalid = iota
	hvacOn
	hvacOff
	hvacHeating
	hvacCooling
)

func (h hvacState) String() string {
	switch h {
	case hvacOn:
		return "on"
	case hvacOff:
		return "off"
	case hvacHeating:
		return "heating"
	case hvacCooling:
		return "heating"
	default:
		return "invalid"
	}
}

type hvacEvent interface {
	isHvacEvent() bool
	fmt.Stringer
}

type On struct{}
type Off struct{}
type Heat struct {
	Temp float64
}
type Cool struct {
	Temp float64
}

func (_ On) isHvacEvent() bool {
	return true
}
func (_ Off) isHvacEvent() bool {
	return true
}
func (_ Heat) isHvacEvent() bool {
	return true
}
func (_ Cool) isHvacEvent() bool {
	return true
}
func (_ On) String() string {
	return "turn_on"
}
func (_ Off) String() string {
	return "turn_off"
}
func (_ Heat) String() string {
	return "heat"
}
func (_ Cool) String() string {
	return "cool"
}

// Tests a more complex fsm that describes an HVAC system which includes cycles.
func TestFSM_ComplexWithCycles(t *testing.T) {
	//nolint:all
	var startState hvacState
	startState = hvacOff
	transitions := []*FsmEvent[hvacEvent, hvacState]{
		{Typ: On{}, From: []hvacState{hvacOff}, To: hvacOn},
		{Typ: Off{}, From: []hvacState{hvacOn, hvacHeating, hvacCooling}, To: hvacOff},
		{Typ: Heat{}, From: []hvacState{hvacOn, hvacHeating, hvacCooling}, To: hvacHeating},
		{Typ: Cool{}, From: []hvacState{hvacOn, hvacHeating, hvacCooling}, To: hvacCooling},
	}
	fsm, err := NewFsm(startState, transitions)
	require.NoError(t, err)

	err = fsm.Do(On{})
	require.NoError(t, err)

	curr := fsm.Current()
	require.Equal(t, uint8(hvacOn), uint8(curr.State))

	err = fsm.Do(Off{})
	require.NoError(t, err)

	curr = fsm.Current()
	require.Equal(t, uint8(hvacOff), uint8(curr.State))

	err = fsm.Do(On{})
	require.NoError(t, err)

	curr = fsm.Current()
	require.Equal(t, uint8(hvacOn), uint8(curr.State))

	err = fsm.Do(Cool{Temp: 18.5})
	require.NoError(t, err)

	curr = fsm.Current()
	ev, ok := curr.SourceEvent.(Cool)
	require.Equal(t, true, ok)
	require.Equal(t, 18.5, ev.Temp)

	err = fsm.Do(Cool{Temp: 17.0})
	require.NoError(t, err)

	curr = fsm.Current()
	ev, ok = curr.SourceEvent.(Cool)
	require.Equal(t, true, ok)
	require.Equal(t, 17.0, ev.Temp)

	err = fsm.Do(Heat{Temp: 23.5})
	require.NoError(t, err)

	curr = fsm.Current()
	heatEv, ok := curr.SourceEvent.(Heat)
	require.Equal(t, true, ok)
	require.Equal(t, 23.5, heatEv.Temp)

	err = fsm.Do(On{})
	require.ErrorIs(t, err, ErrFsmInvalidTransition)

	err = fsm.Do(Off{})
	require.NoError(t, err)

	curr = fsm.Current()
	require.Equal(t, uint8(hvacOff), uint8(curr.State))
}
