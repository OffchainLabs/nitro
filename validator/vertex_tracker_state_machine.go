package validator

import (
	"fmt"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
)

func newVertexTrackerFSM() error {
	var startState vertexTrackerState
	startState = trackerStarted
	transitions := []*util.FsmEvent[vertexTrackerAction, vertexTrackerState]{
		// All start states.
		{
			Typ:  checkPresumptive{},
			From: []vertexTrackerState{trackerStarted, trackerPresumptive},
			To:   trackerPresumptive,
		},
		{
			Typ:  checkChallengeConfirmed{},
			From: []vertexTrackerState{trackerStarted},
			To:   trackerFinished,
		},
		{
			Typ:  checkVertexConfirmed{},
			From: []vertexTrackerState{trackerStarted},
			To:   trackerFinished,
		},
		{
			Typ:  checkSiblingConfirmed{},
			From: []vertexTrackerState{trackerStarted},
			To:   trackerFinished,
		},
		// One-step-proof checks.
		{
			Typ:  checkOneStepFork{},
			From: []vertexTrackerState{trackerStarted, trackerAtOneStepFork},
			To:   trackerAtOneStepFork,
		},
		{
			Typ:  openSubchallenge{},
			From: []vertexTrackerState{trackerAtOneStepFork},
			To:   trackerOpeningSubchallenge,
		},
	}
	fsm, err := util.NewFsm(startState, transitions)
	if err != nil {
		return err
	}
	fmt.Println(fsm)
	return nil
}
