package validator

import (
	"github.com/OffchainLabs/challenge-protocol-v2/util"
)

// Defines the transition table for a finite state machine describing
// a challenge vertex tracker. Each time the vertex tracker wakes, it will
// take action depending on the state it is in within its fsm, and will only
// be allowed to transition states depending on the rules this transition table.
func newVertexTrackerFsm(
	startState vertexTrackerState,
) (*util.Fsm[vertexTrackerAction, vertexTrackerState], error) {
	transitions := []*util.FsmEvent[vertexTrackerAction, vertexTrackerState]{
		// Start states.
		{
			// Returns the tracker to the very beginning. Several states can cause
			// this, including challenge moves.
			Typ: backToStart{},
			From: []vertexTrackerState{
				trackerAtOneStepFork,
				trackerBisecting,
				trackerMerging,
			},
			To: trackerStarted,
		},
		{
			// Marks a tracker as presumptive status. This can occur
			// soon after the tracker begins, or if a challenge move has been made.
			Typ: markPresumptive{},
			From: []vertexTrackerState{
				trackerStarted,
				trackerBisecting,
				trackerMerging,
			},
			To: trackerPresumptive,
		},
		// One-step-proof states.
		{
			// The tracker will take some action if it has reached a one-step-fork.
			Typ:  actOneStepFork{},
			From: []vertexTrackerState{trackerStarted},
			To:   trackerAtOneStepFork,
		},
		{
			// The tracker will take some action if it has reached a one-step-proof
			// in a small step challenge.
			Typ:  actOneStepProof{},
			From: []vertexTrackerState{trackerAtOneStepFork},
			To:   trackerAtOneStepProof,
		},
		{
			// The tracker will open a subchallenge on a vertex that is at a one-step-fork.
			Typ:  openSubchallenge{},
			From: []vertexTrackerState{trackerAtOneStepFork},
			To:   trackerOpeningSubchallenge,
		},
		{
			// The tracker will add a subchallenge leaf to its vertex's subchallenge.
			Typ:  openSubchallengeLeaf{},
			From: []vertexTrackerState{trackerOpeningSubchallenge},
			To:   trackerAddingSubchallengeLeaf,
		},
		{
			// The tracker will be awaiting subchallenge resolution until it will exit.
			Typ: awaitSubchallengeResolution{},
			From: []vertexTrackerState{
				trackerAtOneStepFork,
				trackerAddingSubchallengeLeaf,
			},
			To: trackerAwaitingSubchallengeResolution,
		},
		// Challenge moves.
		{
			Typ: bisect{},
			From: []vertexTrackerState{
				trackerStarted,
				trackerBisecting, // A vertex can bisect multiple times consecutively.
			},
			To: trackerBisecting,
		},
		{
			Typ: merge{},
			From: []vertexTrackerState{
				trackerStarted,
				trackerBisecting, // If a bisection attempt already exists, the tracker will try to merge.
			},
			To: trackerMerging,
		},
		// Finishing.
		{
			// Once a vertex tracker is at a one-step-proof, it will attempt to confirm a winner on-chain.
			Typ:  confirmWinner{},
			From: []vertexTrackerState{trackerAtOneStepProof},
			To:   trackerConfirming,
		},
	}
	return util.NewFsm(startState, transitions)
}
