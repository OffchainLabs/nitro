package melrunner

import (
	"fmt"

	"github.com/offchainlabs/nitro/arbnode/mel"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/bold/containers/fsm"
)

// Defines a finite state machine (FSM) for the message extraction process.
type FSMState uint8

const (
	// Start state of 0 can never happen to avoid silly mistakes with default Go values.
	_ FSMState = iota
	Start
	ProcessingNextBlock
	Reorging
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
	case Reorging:
		return "reorging"
	default:
		return "invalid"
	}
}

type action interface {
	fmt.Stringer
	isFsmAction() bool
}

// An action that returns the FSM to the start state.
type backToStart struct{}

// An action that transitions the FSM to the processing next block state.
type processNextBlock struct {
	melState *mel.State
}

// An action that transitions the FSM to the saving messages state.
type saveMessages struct {
	preStateMsgCount uint64
	postState        *mel.State
	messages         []*arbostypes.MessageWithMetadata
	delayedMessages  []*mel.DelayedInboxMessage
}

// An action that transitions the FSM to the reorging state.
type reorgToOldBlock struct{}

func (backToStart) String() string {
	return "back_to_start"
}
func (processNextBlock) String() string {
	return "process_next_block"
}
func (saveMessages) String() string {
	return "save_messages"
}
func (reorgToOldBlock) String() string {
	return "reorg"
}
func (backToStart) isFsmAction() bool {
	return true
}
func (processNextBlock) isFsmAction() bool {
	return true
}
func (saveMessages) isFsmAction() bool {
	return true
}
func (reorgToOldBlock) isFsmAction() bool {
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
			From: []FSMState{Start, ProcessingNextBlock, SavingMessages, Reorging},
			To:   ProcessingNextBlock,
		},
		{
			Typ:  reorgToOldBlock{},
			From: []FSMState{Start, ProcessingNextBlock},
			To:   Reorging,
		},
		{
			Typ:  saveMessages{},
			From: []FSMState{ProcessingNextBlock, SavingMessages},
			To:   SavingMessages,
		},
	}
	return fsm.New(startState, transitions, fsmOpts...)
}
