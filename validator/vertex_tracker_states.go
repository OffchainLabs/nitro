package validator

import (
	"fmt"

	"github.com/OffchainLabs/challenge-protocol-v2/protocol"
	"github.com/ethereum/go-ethereum/common"
)

// Defines a state in a finite state machine that aids
// in deciding a challenge vertex tracker's actions.
type vertexTrackerState uint8

const (
	// Start state of 0 can never happen to avoid silly mistakes with default Go values.
	// nolint:unused
	trackerInvalid vertexTrackerState = iota
	// The start state of the tracker.
	trackerStarted
	// The vertex being tracked is presumptive.
	trackerPresumptive
	// The vertex being tracked is at a one step fork.
	trackerAtOneStepFork
	// The vertex being tracked is at a one step proof.
	trackerAtOneStepProof
	// The tracker is opening a subchallenge on the vertex.
	trackerOpeningSubchallenge
	// The tracker is adding a subchallenge leaf on the vertex's subchallenge.
	trackerAddingSubchallengeLeaf
	// The tracker is awaiting resolution of its subchallenges.
	trackerAwaitingSubchallengeResolution
	// The tracker is attempting a bisection move.
	trackerBisecting
	// The tracker is attempting a merge move.
	trackerMerging
	// The tracker is confirming a vertex after it has won a one-step-proof.
	// TODO: There are other ways the vertex can be confirmed, and perhaps should
	// be tracked in a separate goroutine then the vertex tracker.
	trackerConfirming
)

// String turns a vertex tracker state into a readable string.
func (v vertexTrackerState) String() string {
	switch v {
	case trackerStarted:
		return "started"
	case trackerPresumptive:
		return "presumptive"
	case trackerAtOneStepFork:
		return "one_step_fork"
	case trackerAtOneStepProof:
		return "one_step_proof"
	case trackerOpeningSubchallenge:
		return "opening_subchallenge"
	case trackerAddingSubchallengeLeaf:
		return "adding_subchallenge_leaf"
	case trackerAwaitingSubchallengeResolution:
		return "awaiting_subchallenge_resolution"
	case trackerBisecting:
		return "bisecting"
	case trackerMerging:
		return "merging"
	case trackerConfirming:
		return "confirming"
	default:
		return "invalid"
	}
}

// Defines structs that characterize actions a vertex tracker
// can take to transition between states in its finite state machine.
type vertexTrackerAction interface {
	fmt.Stringer
	isVertexTrackerAction() bool
}

// Transitions the vertex tracker back to a start state.
type backToStart struct{}

// Transitions the vertex tracker to a presumptive state.
type markPresumptive struct{}

// Tracker will act if the vertex is at a one step fork.
type actOneStepFork struct {
	// The parent vertex of the rival vertices in the one-step-fork.
	forkPointVertex protocol.ChallengeVertex
}

// Tracker will act if the vertex is at a one step proof.
type actOneStepProof struct{}

// Tracker will open a subchallenge on its vertex.
type openSubchallenge struct {
	// The parent vertex of the rival vertices in the one-step-fork.
	challengeForkVertex protocol.ChallengeVertex
}

// Tracker will add a subchallenge on its vertex's subchallenge.
type openSubchallengeLeaf struct {
	// The parent vertex of the rival vertices in the one-step-fork.
	forkPointVertex protocol.ChallengeVertex
	subChallenge    protocol.Challenge
}

// Tracker will await subchallenge resolution.
type awaitSubchallengeResolution struct{}

// Tracker will attempt to bisect its vertex.
type bisect struct {
	bisectingTo       uint64
	bisectingToCommit common.Hash
}

// Tracker will attempt to merge its vertex.
type merge struct {
	bisectingTo       uint64
	bisectingToCommit common.Hash
}

// Tracker will attempt to confirm a challenge winner.
type confirmWinner struct{}

func (_ backToStart) String() string {
	return "back_to_start"
}
func (_ markPresumptive) String() string {
	return "mark_presumptive"
}
func (_ actOneStepFork) String() string {
	return "check_one_step_fork"
}
func (_ actOneStepProof) String() string {
	return "check_one_step_proof"
}
func (_ openSubchallenge) String() string {
	return "open_subchallenge"
}
func (_ openSubchallengeLeaf) String() string {
	return "open_subchallenge_leaf"
}
func (_ awaitSubchallengeResolution) String() string {
	return "await_subchallenge_resolution"
}
func (_ bisect) String() string {
	return "bisect"
}
func (_ merge) String() string {
	return "merge"
}
func (_ confirmWinner) String() string {
	return "confirm_winner"
}

func (_ backToStart) isVertexTrackerAction() bool {
	return true
}
func (_ markPresumptive) isVertexTrackerAction() bool {
	return true
}
func (_ actOneStepFork) isVertexTrackerAction() bool {
	return true
}
func (_ actOneStepProof) isVertexTrackerAction() bool {
	return true
}
func (_ openSubchallenge) isVertexTrackerAction() bool {
	return true
}
func (_ openSubchallengeLeaf) isVertexTrackerAction() bool {
	return true
}
func (_ awaitSubchallengeResolution) isVertexTrackerAction() bool {
	return true
}
func (_ bisect) isVertexTrackerAction() bool {
	return true
}
func (_ merge) isVertexTrackerAction() bool {
	return true
}
func (_ confirmWinner) isVertexTrackerAction() bool {
	return true
}
