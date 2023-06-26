package edgetracker

import (
	"github.com/OffchainLabs/challenge-protocol-v2/containers/fsm"
)

func newEdgeTrackerFsm(
	startState EdgeTrackerState,
	fsmOpts ...fsm.Opt[edgeTrackerAction, EdgeTrackerState],
) (*fsm.Fsm[edgeTrackerAction, EdgeTrackerState], error) {
	transitions := []*fsm.Event[edgeTrackerAction, EdgeTrackerState]{
		{
			// Returns the tracker to the very beginning. Several states can cause
			// this, including challenge moves.
			Typ: edgeBackToStart{},
			From: []EdgeTrackerState{
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
			From: []EdgeTrackerState{edgeStarted},
			To:   edgeAtOneStepProof,
		},
		{
			// The tracker will add a subchallenge leaf to its edge's subchallenge.
			Typ:  edgeOpenSubchallengeLeaf{},
			From: []EdgeTrackerState{edgeStarted, edgeAddingSubchallengeLeaf},
			To:   edgeAddingSubchallengeLeaf,
		},
		// Challenge moves.
		{
			Typ:  edgeBisect{},
			From: []EdgeTrackerState{edgeStarted},
			To:   edgeBisecting,
		},
		// Awaiting confirmation.
		{
			Typ:  edgeAwaitConfirmation{},
			From: []EdgeTrackerState{edgeStarted, edgeBisecting, edgeAddingSubchallengeLeaf, edgeConfirming},
			To:   edgeConfirming,
		},
		// Terminal state.
		{
			Typ:  edgeConfirm{},
			From: []EdgeTrackerState{edgeStarted, edgeConfirming, edgeConfirmed, edgeAtOneStepProof},
			To:   edgeConfirmed,
		},
	}
	return fsm.New(startState, transitions, fsmOpts...)
}
