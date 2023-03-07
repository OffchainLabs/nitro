package validator

import (
	"fmt"
)

type vertexTrackerState uint8

const (
	trackerInvalid vertexTrackerState = iota
	trackerStarted
	trackerPresumptive
	trackerFinished
	trackerAtOneStepFork
	trackerAtOneStepProof
	trackerOpeningSubchallenge
	trackerAddingSubchallengeLeaf
	trackerAwaitingSubchallengeResolution
	trackerBisecting
	trackerMerging
)

func (v vertexTrackerState) String() string {
	switch v {
	case trackerStarted:
		return "started"
	case trackerPresumptive:
		return "presumptive"
	case trackerFinished:
		return "finished"
	case trackerAtOneStepFork:
		return "one_step_fork"
	case trackerAtOneStepProof:
		return "one_step_proof"
	case trackerOpeningSubchallenge:
		return "opening_subchallenge"
	case trackerAddingSubchallengeLeaf:
		return "adding_subchallenge_leaf"
	case trackerAwaitingSubchallengeResolution:
		return "awaiting_resolution"
	case trackerBisecting:
		return "bisecting"
	case trackerMerging:
		return "merging"
	default:
		return "invalid"
	}
}

type vertexTrackerAction interface {
	fmt.Stringer
	isVertexTrackerAction() bool
}

type checkPresumptive struct{}
type markPresumptive struct{}
type checkOneStepFork struct{}
type bisect struct{}
type merge struct{}
type openSubchallenge struct{}
type openSubchallengeLeaf struct{}
type checkChallengeConfirmed struct{}
type checkVertexConfirmed struct{}
type checkSiblingConfirmed struct{}
type checkSibling struct{}
type awaitSubchallengeResolution struct{}
type checkOneStepProof struct{}
type confirmWinner struct{}
type transitionToChallengeComplete struct{}

func (_ markPresumptive) String() string {
	return "mark_presumptive"
}
func (_ checkOneStepFork) String() string {
	return "check_one_step_fork"
}
func (_ bisect) String() string {
	return "bisect"
}
func (_ merge) String() string {
	return "merge"
}
func (_ openSubchallenge) String() string {
	return "openSubchallenge"
}

func (_ markPresumptive) isVertexTrackerAction() bool {
	return true
}
func (_ checkOneStepFork) isVertexTrackerAction() bool {
	return true
}
func (_ bisect) isVertexTrackerAction() bool {
	return true
}
func (_ merge) isVertexTrackerAction() bool {
	return true
}
func (_ openSubchallenge) isVertexTrackerAction() bool {
	return true
}
