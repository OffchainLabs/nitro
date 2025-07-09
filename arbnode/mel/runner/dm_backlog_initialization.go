package melrunner

import (
	"context"
	"fmt"

	"github.com/offchainlabs/nitro/arbnode/mel"
	"github.com/offchainlabs/nitro/arbos/merkleAccumulator"
)

// InitializeDelayedMessageBacklog is to be only called by the Start fsm step of MEL. This function fills the backlog based on the seen and read count from the given mel state
func InitializeDelayedMessageBacklog(ctx context.Context, d *mel.DelayedMessageBacklog, db *Database, state *mel.State, finalizedAndReadIndexFetcher func(context.Context) (uint64, error)) error {
	// d.ctx = ctx
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
	// Get the merkleAccumulator that has accumulated delayed messages up until the position=targetDelayedMessagesRead
	acc, delayedMsgIndexToParentChainBlockNum, err := getMerkleAccumulatorAt(ctx, targetDelayedMessagesRead, db, state)
	if err != nil {
		return err
	}
	// Accumulator is now at the step we need, hence we start creating DelayedMessageBacklogEntry for all the delayed messages that are seen but not read
	for index := targetDelayedMessagesRead; index < state.DelayedMessagedSeen; index++ {
		msg, err := db.fetchDelayedMessage(index)
		if err != nil {
			return err
		}
		_, err = acc.Append(msg.Hash())
		if err != nil {
			return err
		}
		merkleRoot, err := acc.Root()
		if err != nil {
			return err
		}
		if err := d.Add(
			&mel.DelayedMessageBacklogEntry{
				Index:                       index,
				MerkleRoot:                  merkleRoot,
				MelStateParentChainBlockNum: delayedMsgIndexToParentChainBlockNum[index],
			}); err != nil {
			return err
		}
	}
	return nil
}

// getMerkleAccumulatorAt returns a merkle accumulator that has accumulated messages up until a given targetDelayedMessagesRead index
func getMerkleAccumulatorAt(ctx context.Context, targetDelayedMessagesRead uint64, db *Database, state *mel.State) (*merkleAccumulator.MerkleAccumulator, map[uint64]uint64, error) {
	// We first find the melState whose DelayedMessagedSeen is just before the targetDelayedMessagesRead
	// so that we can construct a merkleAccumulator that is relevant to us
	var prev *mel.State
	var err error
	delayedMsgIndexToParentChainBlockNum := make(map[uint64]uint64)
	curr := state
	for i := state.ParentChainBlockNumber - 1; i > 0; i-- {
		prev, err = db.State(ctx, i)
		if err != nil {
			return nil, nil, err
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
	if prev == nil {
		return nil, nil, fmt.Errorf("could not find relevant mel state while creating merkle accumulator while initializing backlog. targetDelayedMessagesRead: %d, state.delayedSeen: %d, state.delayedRead: %d",
			targetDelayedMessagesRead, state.DelayedMessagedSeen, state.DelayedMessagesRead)
	}
	acc, err := merkleAccumulator.NewNonpersistentMerkleAccumulatorFromPartials(
		mel.ToPtrSlice(prev.DelayedMessageMerklePartials),
	)
	if err != nil {
		return nil, nil, err
	}
	// We then walk forward the merkleAccumulator till targetDelayedMessagesRead
	for index := prev.DelayedMessagedSeen; index < targetDelayedMessagesRead; index++ {
		msg, err := db.fetchDelayedMessage(index)
		if err != nil {
			return nil, nil, err
		}
		_, err = acc.Append(msg.Hash())
		if err != nil {
			return nil, nil, err
		}
	}
	return acc, delayedMsgIndexToParentChainBlockNum, nil
}
