package validator

import (
	"github.com/OffchainLabs/challenge-protocol-v2/util"
)

func newEdgeTrackerFsm(
	startState edgeTrackerState,
	fsmOpts ...util.FsmOpt[edgeTrackerAction, edgeTrackerState],
) (*util.Fsm[edgeTrackerAction, edgeTrackerState], error) {
	transitions := []*util.FsmEvent[edgeTrackerAction, edgeTrackerState]{
		{
			// Returns the tracker to the very beginning. Several states can cause
			// this, including challenge moves.
			Typ: edgeBackToStart{},
			From: []edgeTrackerState{
				edgePresumptive,
				edgeBisecting,
				edgeStarted,
				edgeAtOneStepProof,
				edgeAddingSubchallengeLeaf,
				edgePresumptive,
			},
			To: edgeStarted,
		},
		{
			// Marks a tracker as presumptive status. This can occur
			// soon after the tracker begins, or if a challenge move has been made.
			Typ: edgeMarkPresumptive{},
			From: []edgeTrackerState{
				edgeStarted,
				edgePresumptive,
			},
			To: edgePresumptive,
		},
		// One-step-proof states.
		{
			// The tracker will take some action if it has reached a one-step-fork.
			Typ:  edgeHandleOneStepFork{},
			From: []edgeTrackerState{edgeStarted},
			To:   edgeAtOneStepFork,
		},
		{
			// The tracker will take some action if it has reached a one-step-proof
			// in a small step challenge.
			Typ:  edgeHandleOneStepProof{},
			From: []edgeTrackerState{edgeStarted},
			To:   edgeAtOneStepProof,
		},
		{
			// The tracker will add a subchallenge leaf to its edge's subchallenge.
			Typ:  edgeOpenSubchallengeLeaf{},
			From: []edgeTrackerState{edgeAtOneStepFork, edgeAddingSubchallengeLeaf},
			To:   edgeAddingSubchallengeLeaf,
		},
		// Challenge moves.
		{
			Typ:  edgeBisect{},
			From: []edgeTrackerState{edgeStarted},
			To:   edgeBisecting,
		},
		{
			Typ:  edgeTryToConfirm{},
			From: []edgeTrackerState{edgeConfirming, edgeBisecting, edgeAtOneStepFork, edgeAddingSubchallengeLeaf},
			To:   edgeConfirming,
		},
		// Terminal state.
		{
			Typ:  edgeConfirm{},
			From: []edgeTrackerState{edgeConfirmed, edgeConfirming, edgeAtOneStepProof},
			To:   edgeConfirmed,
		},
	}
	return util.NewFsm(startState, transitions, fsmOpts...)
}
