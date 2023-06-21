package edgetracker

import (
	"github.com/OffchainLabs/challenge-protocol-v2/containers/fsm"
)

func newEdgeTrackerFsm(
	startState edgeTrackerState,
	fsmOpts ...fsm.Opt[edgeTrackerAction, edgeTrackerState],
) (*fsm.Fsm[edgeTrackerAction, edgeTrackerState], error) {
	transitions := []*fsm.Event[edgeTrackerAction, edgeTrackerState]{
		{
			// Returns the tracker to the very beginning. Several states can cause
			// this, including challenge moves.
			Typ: edgeBackToStart{},
			From: []edgeTrackerState{
				edgeBisecting,
				edgeStarted,
				edgeAtOneStepProof,
				edgeAddingSubchallengeLeaf,
			},
			To: edgeStarted,
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
			From: []edgeTrackerState{edgeStarted, edgeAddingSubchallengeLeaf},
			To:   edgeAddingSubchallengeLeaf,
		},
		// Challenge moves.
		{
			Typ:  edgeBisect{},
			From: []edgeTrackerState{edgeStarted},
			To:   edgeBisecting,
		},
		// Awaiting confirmation.
		{
			Typ:  edgeAwaitConfirmation{},
			From: []edgeTrackerState{edgeStarted, edgeBisecting, edgeAddingSubchallengeLeaf, edgeConfirming},
			To:   edgeConfirming,
		},
		// Terminal state.
		{
			Typ:  edgeConfirm{},
			From: []edgeTrackerState{edgeStarted, edgeConfirming, edgeConfirmed, edgeAtOneStepProof},
			To:   edgeConfirmed,
		},
	}
	return fsm.New(startState, transitions, fsmOpts...)
}
