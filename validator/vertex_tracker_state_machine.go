package validator

import (
	"github.com/OffchainLabs/challenge-protocol-v2/util"
)

// Defines the transition table for a finite state machine describing
// a challenge vertex tracker. Each time the vertex tracker wakes, it will
// take action depending on the state it is in within its fsm, and will only
// be allowed to transition states depending on the rules this transition table.
func newVertexTrackerFsm(startState vertexTrackerState) (
	*util.Fsm[vertexTrackerAction, vertexTrackerState],
	error,
) {
	transitions := []*util.FsmEvent[vertexTrackerAction, vertexTrackerState]{
		// Start states.
		{
			Typ: backToStart{},
			From: []vertexTrackerState{
				trackerAtOneStepFork,
				trackerPresumptive,
				trackerBisecting,
				trackerMerging,
			},
			To: trackerStarted,
		},
		{
			Typ: markPresumptive{},
			From: []vertexTrackerState{
				trackerStarted,
				trackerPresumptive,
				trackerBisecting,
				trackerMerging,
			},
			To: trackerPresumptive,
		},
		// One-step-proof states.
		{
			Typ: checkOneStepFork{},
			From: []vertexTrackerState{
				trackerStarted,
				trackerAtOneStepFork,
			},
			To: trackerAtOneStepFork,
		},
		{
			Typ:  checkOneStepProof{},
			From: []vertexTrackerState{trackerAtOneStepFork},
			To:   trackerAtOneStepProof,
		},
		{
			Typ:  openSubchallenge{},
			From: []vertexTrackerState{trackerAtOneStepFork},
			To:   trackerOpeningSubchallenge,
		},
		{
			Typ:  openSubchallengeLeaf{},
			From: []vertexTrackerState{trackerOpeningSubchallenge},
			To:   trackerAddingSubchallengeLeaf,
		},
		{
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
				trackerBisecting,
			},
			To: trackerBisecting,
		},
		{
			Typ: merge{},
			From: []vertexTrackerState{
				trackerStarted,
				trackerBisecting,
			},
			To: trackerMerging,
		},
		// Finishing.
		{
			Typ:  confirmWinner{},
			From: []vertexTrackerState{trackerAtOneStepProof},
			To:   trackerConfirming,
		},
	}
	return util.NewFsm(startState, transitions)
}
