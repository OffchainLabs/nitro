// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE

package edgetracker

import (
	"github.com/OffchainLabs/bold/containers/fsm"
)

func newEdgeTrackerFsm(
	startState State,
	fsmOpts ...fsm.Opt[edgeTrackerAction, State],
) (*fsm.Fsm[edgeTrackerAction, State], error) {
	transitions := []*fsm.Event[edgeTrackerAction, State]{
		{
			// Returns the tracker to the very beginning. Several states can cause
			// this, including challenge moves.
			Typ: edgeBackToStart{},
			From: []State{
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
			From: []State{edgeStarted},
			To:   edgeAtOneStepProof,
		},
		{
			// The tracker will add a subchallenge leaf to its edge's subchallenge.
			Typ:  edgeOpenSubchallengeLeaf{},
			From: []State{edgeStarted, edgeAddingSubchallengeLeaf},
			To:   edgeAddingSubchallengeLeaf,
		},
		// Challenge moves.
		{
			Typ:  edgeBisect{},
			From: []State{edgeStarted},
			To:   edgeBisecting,
		},
		// Awaiting confirmation.
		{
			Typ:  edgeAwaitConfirmation{},
			From: []State{edgeStarted, edgeBisecting, edgeAddingSubchallengeLeaf, edgeConfirming},
			To:   edgeConfirming,
		},
		// Terminal state.
		{
			Typ:  edgeConfirm{},
			From: []State{edgeStarted, edgeConfirming, edgeConfirmed, edgeAtOneStepProof},
			To:   edgeConfirmed,
		},
	}
	return fsm.New(startState, transitions, fsmOpts...)
}
