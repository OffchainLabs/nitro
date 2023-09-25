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
				EdgeBisecting,
				EdgeStarted,
				EdgeAtOneStepProof,
				EdgeAddingSubchallengeLeaf,
			},
			To: EdgeStarted,
		},
		{
			// The tracker will take some action if it has reached a one-step-proof
			// in a small step challenge.
			Typ:  edgeHandleOneStepProof{},
			From: []State{EdgeStarted},
			To:   EdgeAtOneStepProof,
		},
		{
			// The tracker will add a subchallenge leaf to its edge's subchallenge.
			Typ:  edgeOpenSubchallengeLeaf{},
			From: []State{EdgeStarted, EdgeAddingSubchallengeLeaf},
			To:   EdgeAddingSubchallengeLeaf,
		},
		// Challenge moves.
		{
			Typ:  edgeBisect{},
			From: []State{EdgeStarted},
			To:   EdgeBisecting,
		},
		// Awaiting confirmation.
		{
			Typ:  edgeAwaitConfirmation{},
			From: []State{EdgeStarted, EdgeBisecting, EdgeAddingSubchallengeLeaf, EdgeConfirming},
			To:   EdgeConfirming,
		},
		// Terminal state.
		{
			Typ:  edgeConfirm{},
			From: []State{EdgeStarted, EdgeConfirming, EdgeConfirmed, EdgeAtOneStepProof},
			To:   EdgeConfirmed,
		},
	}
	return fsm.New(startState, transitions, fsmOpts...)
}
