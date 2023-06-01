package validator

import (
	"fmt"
)

// Defines a state in a finite state machine that aids
// in deciding a challenge edge tracker's actions.
type edgeTrackerState uint8

const (
	// Start state of 0 can never happen to avoid silly mistakes with default Go values.
	_ edgeTrackerState = iota
	// The start state of the tracker.
	edgeStarted
	// The edge being tracked is presumptive.
	edgePresumptive
	// The edge being tracked is at a one step fork.
	edgeAtOneStepFork
	// The edge being tracked is at a one step proof.
	edgeAtOneStepProof
	// The tracker is adding a subchallenge leaf on the edge's subchallenge.
	edgeAddingSubchallengeLeaf
	// The tracker is attempting a bisection move.
	edgeBisecting
	// The tracker is confirming an edge.
	// TODO: There are other ways the edge can be confirmed, and perhaps should
	// be tracked in a separate goroutine then the tracker.
	edgeConfirming
	// Terminal state
	edgeConfirmed
)

// String turns an edge tracker state into a readable string.
func (s edgeTrackerState) String() string {
	switch s {
	case edgeStarted:
		return "started"
	case edgePresumptive:
		return "presumptive"
	case edgeAtOneStepFork:
		return "one_step_fork"
	case edgeAtOneStepProof:
		return "one_step_proof"
	case edgeAddingSubchallengeLeaf:
		return "adding_subchallenge_leaf"
	case edgeBisecting:
		return "bisecting"
	case edgeConfirming:
		return "confirming"
	case edgeConfirmed:
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

// Transitions the edge tracker to a presumptive state.
type edgeMarkPresumptive struct{}

// Tracker will act if the edge is at a one step fork.
type edgeHandleOneStepFork struct{}

// Tracker will act if the edge is at a one step proof.
type edgeHandleOneStepProof struct{}

// Tracker will add a subchallenge on its edge's subchallenge.
type edgeOpenSubchallengeLeaf struct{}

// Tracker will attempt to bisect its edge.
type edgeBisect struct{}

// Tracker will attempt to confirm a challenge winner.
type edgeTryToConfirm struct{}

type edgeConfirm struct{}

func (edgeBackToStart) String() string {
	return "back_to_start"
}
func (edgeMarkPresumptive) String() string {
	return "mark_presumptive"
}
func (edgeHandleOneStepFork) String() string {
	return "check_one_step_fork"
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
func (edgeTryToConfirm) String() string {
	return "trying_to_confirm"
}
func (edgeConfirm) String() string {
	return "confirm"
}

func (edgeBackToStart) isEdgeTrackerAction() bool {
	return true
}
func (edgeMarkPresumptive) isEdgeTrackerAction() bool {
	return true
}
func (edgeHandleOneStepFork) isEdgeTrackerAction() bool {
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
func (edgeTryToConfirm) isEdgeTrackerAction() bool {
	return true
}
func (edgeConfirm) isEdgeTrackerAction() bool {
	return true
}
