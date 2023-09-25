// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE

package edgetracker

import (
	"fmt"
)

// State defines a finite state machine that aids
// in deciding a challenge edge tracker's actions.
type State uint8

const (
	// Start state of 0 can never happen to avoid silly mistakes with default Go values.
	_ State = iota
	// The start state of the tracker.
	EdgeStarted
	// The edge being tracked is at a one step proof.
	EdgeAtOneStepProof
	// The tracker is adding a subchallenge leaf on the edge's subchallenge.
	EdgeAddingSubchallengeLeaf
	// The tracker is attempting a bisection move.
	EdgeBisecting
	// Intermediary state in which an edge is doing nothing else but awaiting confirmation
	// whenever it is possible.
	EdgeConfirming
	// Terminal state
	EdgeConfirmed
)

// String turns an edge tracker state into a readable string.
func (s State) String() string {
	switch s {
	case EdgeStarted:
		return "started"
	case EdgeAtOneStepProof:
		return "one_step_proof"
	case EdgeAddingSubchallengeLeaf:
		return "adding_subchallenge_leaf"
	case EdgeBisecting:
		return "bisecting"
	case EdgeConfirming:
		return "confirming"
	case EdgeConfirmed:
		return "confirmed"
	default:
		return "invalid"
	}
}

// Defines structs that characterize actions an edge tracker
// can take to transition between states in its finite state machine.
type edgeTrackerAction interface {
	fmt.Stringer
	isEdgeTrackerAction() bool
}

// Transitions the edge tracker back to a start state.
type edgeBackToStart struct{}

// Tracker will act if the edge is at a one step proof.
type edgeHandleOneStepProof struct{}

// Tracker will add a subchallenge on its edge's subchallenge.
type edgeOpenSubchallengeLeaf struct{}

// Tracker will attempt to bisect its edge.
type edgeBisect struct{}

type edgeAwaitConfirmation struct{}

type edgeConfirm struct{}

func (edgeBackToStart) String() string {
	return "back_to_start"
}
func (edgeHandleOneStepProof) String() string {
	return "check_one_step_proof"
}
func (edgeOpenSubchallengeLeaf) String() string {
	return "open_subchallenge_leaf"
}
func (edgeBisect) String() string {
	return "bisect"
}
func (edgeAwaitConfirmation) String() string {
	return "await_confirmation"
}
func (edgeConfirm) String() string {
	return "confirm"
}

func (edgeBackToStart) isEdgeTrackerAction() bool {
	return true
}
func (edgeHandleOneStepProof) isEdgeTrackerAction() bool {
	return true
}
func (edgeOpenSubchallengeLeaf) isEdgeTrackerAction() bool {
	return true
}
func (edgeBisect) isEdgeTrackerAction() bool {
	return true
}
func (edgeAwaitConfirmation) isEdgeTrackerAction() bool {
	return true
}
func (edgeConfirm) isEdgeTrackerAction() bool {
	return true
}
