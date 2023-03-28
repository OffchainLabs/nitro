package validator

import (
	"fmt"

	"github.com/OffchainLabs/challenge-protocol-v2/protocol"
)

// Defines a state in a finite state machine that aids
// in deciding a challenge edge tracker's actions.
type edgeTrackerState uint8

const (
	// Start state of 0 can never happen to avoid silly mistakes with default Go values.
	// nolint:unused
	edgeInvalid edgeTrackerState = iota
	// The start state of the tracker.
	edgeStarted
	// The edge being tracked is presumptive.
	edgePresumptive
	// The edge was presumptive, but lost its presumptive status.
	// An edge can never be presumptive again after this.
	edgePresumptiveLost
	// The edge being tracked is at a one step fork.
	edgeAtOneStepFork
	// The edge being tracked is at a one step proof.
	edgeAtOneStepProof
	// The tracker is opening a subchallenge on the edge.
	edgeOpeningSubchallenge
	// The tracker is adding a subchallenge leaf on the edge's subchallenge.
	edgeAddingSubchallengeLeaf
	// The tracker is awaiting resolution of its subchallenges.
	edgeAwaitingSubchallenge
	// The tracker is attempting a bisection move.
	edgeBisecting
	// The tracker is confirming an edge.
	// TODO: There are other ways the edge can be confirmed, and perhaps should
	// be tracked in a separate goroutine then the tracker.
	edgeConfirming
)

// String turns an edge tracker state into a readable string.
func (s edgeTrackerState) String() string {
	switch s {
	case edgeStarted:
		return "started"
	case edgePresumptive:
		return "presumptive"
	case edgePresumptiveLost:
		return "lost_presumptive_status"
	case edgeAtOneStepFork:
		return "one_step_fork"
	case edgeAtOneStepProof:
		return "one_step_proof"
	case edgeOpeningSubchallenge:
		return "opening_subchallenge"
	case edgeAddingSubchallengeLeaf:
		return "adding_subchallenge_leaf"
	case edgeAwaitingSubchallenge:
		return "awaiting_subchallenge_resolution"
	case edgeBisecting:
		return "bisecting"
	case edgeConfirming:
		return "confirming"
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

// Transitions an edge from presumptive to non-presumptive.
type edgeLosePresumptive struct{}

// Tracker will act if the edge is at a one step fork.
type edgeHandleOneStepFork struct{}

// Tracker will act if the edge is at a one step proof.
type edgeHandleOneStepProof struct{}

// Tracker will open a subchallenge on its edge.
type edgeOpenSubchallenge struct{}

// Tracker will add a subchallenge on its edge's subchallenge.
type edgeOpenSubchallengeLeaf struct {
	subChallenge protocol.SpecChallenge
}

// Tracker will await subchallenge resolution.
type edgeAwaitSubchallengeResolution struct{}

// Tracker will attempt to bisect its edge.
type edgeBisect struct{}

// Tracker will attempt to confirm a challenge winner.
type edgeConfirm struct{}

func (_ edgeBackToStart) String() string {
	return "back_to_start"
}
func (_ edgeMarkPresumptive) String() string {
	return "mark_presumptive"
}
func (_ edgeLosePresumptive) String() string {
	return "lose_presumptive_status"
}
func (_ edgeHandleOneStepFork) String() string {
	return "check_one_step_fork"
}
func (_ edgeHandleOneStepProof) String() string {
	return "check_one_step_proof"
}
func (_ edgeOpenSubchallenge) String() string {
	return "open_subchallenge"
}
func (_ edgeOpenSubchallengeLeaf) String() string {
	return "open_subchallenge_leaf"
}
func (_ edgeAwaitSubchallengeResolution) String() string {
	return "await_subchallenge_resolution"
}
func (_ edgeBisect) String() string {
	return "bisect"
}
func (_ edgeConfirm) String() string {
	return "confirm"
}

func (_ edgeBackToStart) isEdgeTrackerAction() bool {
	return true
}
func (_ edgeMarkPresumptive) isEdgeTrackerAction() bool {
	return true
}
func (_ edgeLosePresumptive) isEdgeTrackerAction() bool {
	return true
}
func (_ edgeHandleOneStepFork) isEdgeTrackerAction() bool {
	return true
}
func (_ edgeHandleOneStepProof) isEdgeTrackerAction() bool {
	return true
}
func (_ edgeOpenSubchallenge) isEdgeTrackerAction() bool {
	return true
}
func (_ edgeOpenSubchallengeLeaf) isEdgeTrackerAction() bool {
	return true
}
func (_ edgeAwaitSubchallengeResolution) isEdgeTrackerAction() bool {
	return true
}
func (_ edgeBisect) isEdgeTrackerAction() bool {
	return true
}
func (_ edgeConfirm) isEdgeTrackerAction() bool {
	return true
}
