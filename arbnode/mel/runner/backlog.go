package melrunner

import (
	"context"
	"fmt"

	"github.com/offchainlabs/nitro/arbnode/mel"
)

// InitializeDelayedMessageBacklog is to be only called by the Start fsm step of MEL. This function fills the backlog based on the seen and read count from the given mel state
func InitializeDelayedMessageBacklog(ctx context.Context, d *mel.DelayedMessageBacklog, db *Database, state *mel.State, finalizedAndReadIndexFetcher func(context.Context) (uint64, error)) error {
	if state.DelayedMessagedSeen == 0 && state.DelayedMessagesRead == 0 { // this is the first mel state so no need to initialize backlog even if the state isnt finalized yet
		return nil
	}
	finalizedDelayedMessagesRead := state.DelayedMessagesRead // Assume to be finalized, then update if needed
	var err error
	if finalizedAndReadIndexFetcher != nil {
		finalizedDelayedMessagesRead, err = finalizedAndReadIndexFetcher(ctx)
		if err != nil {
			return err
		}
	}
	if state.DelayedMessagedSeen == state.DelayedMessagesRead && state.DelayedMessagesRead <= finalizedDelayedMessagesRead {
		return nil
	}
	// To make the delayedMessageBacklog reorg resistant we will need to add more delayedMessageBacklogEntry even though those messages are `Read`
	// this is only relevant if the current head Mel state's ParentChainBlockNumber is not yet finalized
	targetDelayedMessagesRead := min(state.DelayedMessagesRead, finalizedDelayedMessagesRead)
	delayedMsgIndexToParentChainBlockNum, err := indexToParentChainBlockMap(ctx, targetDelayedMessagesRead, db, state)
	if err != nil {
		return err
	}
	if uint64(len(delayedMsgIndexToParentChainBlockNum)) < state.DelayedMessagedSeen-targetDelayedMessagesRead {
		return fmt.Errorf("number of mappings from index to ParentChainBlockNum: %d are insufficient, needed atleast: %d", uint64(len(delayedMsgIndexToParentChainBlockNum)), state.DelayedMessagedSeen-targetDelayedMessagesRead)
	}

	// Create DelayedMessageBacklogEntry for all the delayed messages that are seen but not read
	for index := targetDelayedMessagesRead; index < state.DelayedMessagedSeen; index++ {
		msg, err := db.fetchDelayedMessage(index)
		if err != nil {
			return err
		}
		melStateParentChainBlockNum, ok := delayedMsgIndexToParentChainBlockNum[index]
		if !ok {
			return fmt.Errorf("delayed index: %d not found in the mapping of index to ParentChainBlockNum", index)
		}
		if err := d.Add(
			&mel.DelayedMessageBacklogEntry{
				Index:                       index,
				MsgHash:                     msg.Hash(),
				MelStateParentChainBlockNum: melStateParentChainBlockNum,
			}); err != nil {
			return err
		}
	}
	return nil
}

func indexToParentChainBlockMap(ctx context.Context, targetDelayedMessagesRead uint64, db *Database, state *mel.State) (map[uint64]uint64, error) {
	// We first find the melState whose DelayedMessagedSeen is just before the targetDelayedMessagesRead
	var prev *mel.State
	var err error
	delayedMsgIndexToParentChainBlockNum := make(map[uint64]uint64)
	curr := state
	for i := state.ParentChainBlockNumber - 1; i > 0; i-- {
		prev, err = db.State(ctx, i)
		if err != nil {
			return nil, err
		}
		if curr.DelayedMessagedSeen > prev.DelayedMessagedSeen { // Meaning the 'curr' melState has seen some delayed messages
			for j := prev.DelayedMessagedSeen; j < curr.DelayedMessagedSeen; j++ {
				delayedMsgIndexToParentChainBlockNum[j] = curr.ParentChainBlockNumber
			}
		}
		if prev.DelayedMessagedSeen <= targetDelayedMessagesRead {
			break
		}
		curr = prev
	}
	return delayedMsgIndexToParentChainBlockNum, nil
}
