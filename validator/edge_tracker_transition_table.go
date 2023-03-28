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
		{
			// Loses an edge's presumptive status, which it can never regain.
			Typ: edgeLosePresumptive{},
			From: []edgeTrackerState{
				edgePresumptive,
				edgePresumptiveLost,
			},
			To: edgePresumptiveLost,
		},
		// One-step-proof states.
		{
			// The tracker will take some action if it has reached a one-step-fork.
			// TODO: Should go back to start? But then cannot go to ps again though? Maybe need
			// an intermediate state here?
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
			// The tracker will open a subchallenge on a vertex that is at a one-step-fork.
			Typ:  edgeOpenSubchallenge{},
			From: []edgeTrackerState{edgeAtOneStepFork, edgeOpeningSubchallenge},
			To:   edgeOpeningSubchallenge,
		},
		{
			// The tracker will add a subchallenge leaf to its vertex's subchallenge.
			Typ:  edgeOpenSubchallengeLeaf{},
			From: []edgeTrackerState{edgeOpeningSubchallenge, edgeAddingSubchallengeLeaf},
			To:   edgeAddingSubchallengeLeaf,
		},
		{
			// The tracker will be awaiting subchallenge resolution until it will exit.
			Typ: edgeAwaitSubchallengeResolution{},
			From: []edgeTrackerState{
				edgeAtOneStepFork,
				edgeAddingSubchallengeLeaf,
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
