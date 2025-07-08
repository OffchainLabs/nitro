package melrunner

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbnode/mel"
	"github.com/offchainlabs/nitro/arbos/merkleAccumulator"
)

// DelayedMessageBacklogEntry contains metadata relating to delayed messages required for merkle-tree verification
type DelayedMessageBacklogEntry struct {
	Index                       uint64
	MerkleRoot                  common.Hash
	MelStateParentChainBlockNum uint64
}

// DelayedMessageBacklog is a data structure that holds metadata related to delayed messages that have been SEEN by MEL but not yet READ.
// This enables verification of delayed messages read from a database against a Merkle root in the MEL state. The MEL state also contains
// compact witnesses of a Merkle tree representing all seen delayed messages. To prove that a delayed message is part of this Merkle tree,
// this data structure can be used to verify Merkle proofs against the MEL state.
type DelayedMessageBacklog struct {
	ctx                          context.Context
	capacity                     int // 0 = no trimming
	entries                      []*DelayedMessageBacklogEntry
	initMessage                  *mel.DelayedInboxMessage
	finalizedAndReadIndexFetcher func(context.Context) (uint64, error) // nil = no trimming
}

func NewDelayedMessageBacklog(capacity int, finalizedAndReadIndexFetcher func(context.Context) (uint64, error)) *DelayedMessageBacklog {
	return &DelayedMessageBacklog{
		capacity:                     capacity,
		entries:                      make([]*DelayedMessageBacklogEntry, 0),
		initMessage:                  nil,
		finalizedAndReadIndexFetcher: finalizedAndReadIndexFetcher,
	}
}

// Initialize is to be only called by the Start fsm step of MEL. This function fills the backlog based on the seen and read count from the given mel state
func (d *DelayedMessageBacklog) Initialize(ctx context.Context, db *Database, state *mel.State) error {
	d.ctx = ctx
	finalizedDelayedMessagesRead := state.DelayedMessagesRead // Assume to be finalized, then update if needed
	var err error
	if d.finalizedAndReadIndexFetcher != nil {
		finalizedDelayedMessagesRead, err = d.finalizedAndReadIndexFetcher(ctx)
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
	// We first find the melState whose DelayedMessagedSeen is just before the targetDelayedMessagesRead, so that we can construct a merkleAccumulator
	// thats relevant to us
	var prev *mel.State
	delayedMsgIndexToParentChainBlockNum := make(map[uint64]uint64)
	curr := state
	for i := state.ParentChainBlockNumber - 1; i > 0; i-- {
		prev, err = db.State(ctx, i)
		if err != nil {
			return err
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
		return nil
	}
	acc, err := merkleAccumulator.NewNonpersistentMerkleAccumulatorFromPartials(
		mel.ToPtrSlice(prev.DelayedMessageMerklePartials),
	)
	if err != nil {
		return err
	}
	// We then walk forward the merkleAccumulator till targetDelayedMessagesRead
	for index := prev.DelayedMessagedSeen; index < targetDelayedMessagesRead; index++ {
		msg, err := db.fetchDelayedMessage(index)
		if err != nil {
			return err
		}
		_, err = acc.Append(msg.Hash())
		if err != nil {
			return err
		}
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
		if err := d.Add(index, merkleRoot, delayedMsgIndexToParentChainBlockNum[index]); err != nil {
			return err
		}
	}
	return nil
}

// Add takes values of a DelayedMessageBacklogEntry and adds it to the backlog given the entry succeeds validation. It also attempts trimming of backlog if capacity is reached
func (d *DelayedMessageBacklog) Add(index uint64, merkleRoot common.Hash, parentChainBlockNum uint64) error {
	if len(d.entries) > 0 {
		expectedIndex := d.entries[0].Index + uint64(len(d.entries))
		if index != expectedIndex {
			return fmt.Errorf("message index %d is not sequential, expected %d", index, expectedIndex)
		}
	}
	d.entries = append(d.entries, &DelayedMessageBacklogEntry{index, merkleRoot, parentChainBlockNum})
	return d.clear()
}

func (d *DelayedMessageBacklog) Get(index uint64) (common.Hash, uint64, error) {
	if index < d.entries[0].Index || index > d.entries[len(d.entries)-1].Index {
		return common.Hash{}, 0, fmt.Errorf("queried index: %d out of bounds, delayed message backlog's starting index: %d, ending index: %d", index, d.entries[0].Index, d.entries[len(d.entries)-1].Index)
	}
	entry := d.entries[index-d.entries[0].Index]
	if entry.Index != index {
		return common.Hash{}, 0, fmt.Errorf("index mismatch in the backlog entry. Queried index: %d, backlog entry's index: %d", index, entry.Index)
	}
	return entry.MerkleRoot, entry.MelStateParentChainBlockNum, nil
}

func (d *DelayedMessageBacklog) Clone() mel.DelayedMessageBacklog {
	var deque []*DelayedMessageBacklogEntry
	for _, item := range d.entries {
		merkleRoot := common.Hash{}
		copy(merkleRoot[:], item.MerkleRoot[:])
		deque = append(deque, &DelayedMessageBacklogEntry{
			Index:                       item.Index,
			MerkleRoot:                  merkleRoot,
			MelStateParentChainBlockNum: item.MelStateParentChainBlockNum,
		})
	}
	return &DelayedMessageBacklog{d.ctx, d.capacity, deque, nil, d.finalizedAndReadIndexFetcher} // Init msg should only be read once, no need to persist it
}

// Reorg removes from backlog the entries that corresponded to the reorged out parent chain blocks
func (d *DelayedMessageBacklog) Reorg(newDelayedMessagedSeen uint64) error {
	if len(d.entries) == 0 {
		return nil
	}
	if newDelayedMessagedSeen >= d.entries[0].Index {
		rightTrimPos := newDelayedMessagedSeen - d.entries[0].Index
		if rightTrimPos > uint64(len(d.entries)) {
			return fmt.Errorf("newDelayedMessagedSeen: %d durign a reorg is greater (by more than 1) than the greatest delayed message index stored in backlog: %d", newDelayedMessagedSeen, d.entries[len(d.entries)-1].Index)
		}
		d.entries = d.entries[:rightTrimPos]
	} else {
		d.entries = make([]*DelayedMessageBacklogEntry, 0)
	}
	return nil
}

// clear removes from backlog (if exceeds capacity) the entries that correspond to the delayed messages that are both READ and belong to finalized parent chain blocks
func (d *DelayedMessageBacklog) clear() error {
	if len(d.entries) <= d.capacity {
		return nil
	}
	if d.finalizedAndReadIndexFetcher != nil {
		finalizedDelayedMessagesRead, err := d.finalizedAndReadIndexFetcher(d.ctx)
		if err != nil {
			log.Error("Unable to trim finalized and read delayed messages from DelayedMessageBacklog, will be retried later", "err", err)
			return nil // we should not interrupt delayed messages accumulation if we cannot trim the backlog, since its not high priority
		}
		if finalizedDelayedMessagesRead > d.entries[0].Index {
			leftTrimPos := min(finalizedDelayedMessagesRead-d.entries[0].Index, uint64(len(d.entries)))
			d.entries = d.entries[leftTrimPos:]
		}
	}
	return nil
}

func (d *DelayedMessageBacklog) SetInitMsg(msg *mel.DelayedInboxMessage) { d.initMessage = msg }
func (d *DelayedMessageBacklog) GetInitMsg() *mel.DelayedInboxMessage    { return d.initMessage }
