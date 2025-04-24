package mel

import (
	"fmt"

	"github.com/offchainlabs/bold/containers/fsm"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
)

type FSMState uint8

const (
	// Start state of 0 can never happen to avoid silly mistakes with default Go values.
	_ FSMState = iota
	Start
	ProcessingNextBlock
	SavingMessages
)

func (s FSMState) String() string {
	switch s {
	case Start:
		return "start"
	case ProcessingNextBlock:
		return "processing_next_block"
	case SavingMessages:
		return "saving_messages"
	default:
		return "invalid"
	}
}

type action interface {
	fmt.Stringer
	isEdgeTrackerAction() bool
}

type backToStart struct{}

type processNextBlock struct {
	melState *State
}

type saveMessages struct {
	melState *State
	messages []*arbostypes.MessageWithMetadata
}

func (backToStart) String() string {
	return "back_to_start"
}
func (processNextBlock) String() string {
	return "process_next_block"
}
func (saveMessages) String() string {
	return "save_messages"
}
func (backToStart) isEdgeTrackerAction() bool {
	return true
}
func (processNextBlock) isEdgeTrackerAction() bool {
	return true
}
func (saveMessages) isEdgeTrackerAction() bool {
	return true
}

func newFSM(
	startState FSMState,
	fsmOpts ...fsm.Opt[action, FSMState],
) (*fsm.Fsm[action, FSMState], error) {
	transitions := []*fsm.Event[action, FSMState]{
		{
			Typ: backToStart{},
			From: []FSMState{
				Start,
				ProcessingNextBlock,
			},
			To: startState,
		},
		{
			Typ:  processNextBlock{},
			From: []FSMState{Start, ProcessingNextBlock, SavingMessages},
			To:   ProcessingNextBlock,
		},
		{
			Typ:  saveMessages{},
			From: []FSMState{ProcessingNextBlock, SavingMessages},
			To:   SavingMessages,
		},
	}
	return fsm.New(startState, transitions, fsmOpts...)
}
