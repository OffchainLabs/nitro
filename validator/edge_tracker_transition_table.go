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
				edgeAtOneStepFork,
				edgePresumptive,
				edgeBisecting,
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
				edgeBisecting,
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
			From: []edgeTrackerState{edgeAtOneStepFork, edgeAtOneStepProof},
			To:   edgeAtOneStepProof,
		},
		{
			// The tracker will add a subchallenge leaf to its edge's subchallenge.
			Typ:  edgeOpenSubchallengeLeaf{},
			From: []edgeTrackerState{edgeAtOneStepFork, edgeAddingSubchallengeLeaf},
			To:   edgeAddingSubchallengeLeaf,
		},
		{
			// The tracker will be awaiting subchallenge resolution until it will exit.
			Typ: edgeAwaitSubchallengeResolution{},
			From: []edgeTrackerState{
				edgeAtOneStepFork,
				edgeAddingSubchallengeLeaf,
				edgeBisecting,
				edgeAwaitingSubchallenge,
			},
			To: edgeAwaitingSubchallenge,
		},
		// Challenge moves.
		{
			Typ: edgeBisect{},
			From: []edgeTrackerState{
				edgeStarted,
				edgeBisecting, // TODO: Can this still happen?
			},
			To: edgeBisecting,
		},
		// Finishing.
		{
			Typ:  edgeConfirm{},
			From: []edgeTrackerState{edgeAtOneStepProof, edgeConfirming},
			To:   edgeConfirming,
		},
	}
	return util.NewFsm(startState, transitions, fsmOpts...)
}
