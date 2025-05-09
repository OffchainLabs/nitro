package mel

import (
	"fmt"

	"github.com/offchainlabs/bold/containers/fsm"
	"github.com/offchainlabs/nitro/arbnode"
	meltypes "github.com/offchainlabs/nitro/arbnode/message-extraction/types"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
)

type FSMState uint8

const (
	// Start state of 0 can never happen to avoid silly mistakes with default Go values.
	_ FSMState = iota
	Start
	ProcessingNextBlock
	ReorgingToOldBlock
	SavingMessages
)

func (s FSMState) String() string {
	switch s {
	case Start:
		return "start"
	case ProcessingNextBlock:
		return "processing_next_block"
	case ReorgingToOldBlock:
		return "reorging_to_old_block"
	case SavingMessages:
		return "saving_messages"
	default:
		return "invalid"
	}
}

type action interface {
	fmt.Stringer
	isFsmAction() bool
}

type backToStart struct{}

type processNextBlock struct {
	melState *meltypes.State
}

type reorgingToOldBlock struct {
	melState *meltypes.State
}

type saveMessages struct {
	postState       *meltypes.State
	messages        []*arbostypes.MessageWithMetadata
	delayedMessages []*arbnode.DelayedInboxMessage
}

func (backToStart) String() string {
	return "back_to_start"
}
func (processNextBlock) String() string {
	return "process_next_block"
}
func (reorgingToOldBlock) String() string {
	return "reorging_to_old_block"
}
func (saveMessages) String() string {
	return "save_messages"
}
func (backToStart) isFsmAction() bool {
	return true
}
func (processNextBlock) isFsmAction() bool {
	return true
}
func (reorgingToOldBlock) isFsmAction() bool {
	return true
}
func (saveMessages) isFsmAction() bool {
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
			From: []FSMState{Start, ProcessingNextBlock, ReorgingToOldBlock, SavingMessages},
			To:   ProcessingNextBlock,
		},
		{
			Typ:  reorgingToOldBlock{},
			From: []FSMState{Start, ProcessingNextBlock},
			To:   ReorgingToOldBlock,
		},
		{
			Typ:  saveMessages{},
			From: []FSMState{ProcessingNextBlock, SavingMessages},
			To:   SavingMessages,
		},
	}
	return fsm.New(startState, transitions, fsmOpts...)
}
