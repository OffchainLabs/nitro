package mel

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

// DelayedMessageBacklogEntry contains metadata relating to delayed messages required for merkle-tree verification
type DelayedMessageBacklogEntry struct {
	Index                       uint64      // Global delayed index of a delayed inbox message wrt to the chain
	MsgHash                     common.Hash // Hash of the delayed inbox message
	MelStateParentChainBlockNum uint64      // ParentChainBlocknumber of the MEL state in which this delayed inbox message was SEEN
}

// DelayedMessageBacklog is a data structure that holds metadata related to delayed messages that have been SEEN by MEL but not yet READ.
// This enables verification of delayed messages read from a database against the current Merkle root of the head MEL state. The MEL state
// also contains compact witnesses of a Merkle tree representing all seen delayed messages. To prove that a delayed message is part of
// this Merkle tree, this data structure can be used to verify Merkle proofs against the MEL state.
type DelayedMessageBacklog struct {
	ctx                          context.Context
	capacity                     int
	entries                      []*DelayedMessageBacklogEntry
	initMessage                  *DelayedInboxMessage
	finalizedAndReadIndexFetcher func(context.Context) (uint64, error)
}

func NewDelayedMessageBacklog(ctx context.Context, capacity int, finalizedAndReadIndexFetcher func(context.Context) (uint64, error), opts ...func(*DelayedMessageBacklog)) (*DelayedMessageBacklog, error) {
	if capacity == 0 {
		return nil, fmt.Errorf("capacity of DelayedMessageBacklog cannot be zero")
	}
	if finalizedAndReadIndexFetcher == nil {
		return nil, fmt.Errorf("finalizedAndReadIndexFetcher of DelayedMessageBacklog cannot be nil")
	}
	backlog := &DelayedMessageBacklog{
		ctx:                          ctx,
		capacity:                     capacity,
		entries:                      make([]*DelayedMessageBacklogEntry, 0),
		initMessage:                  nil,
		finalizedAndReadIndexFetcher: finalizedAndReadIndexFetcher,
	}
	for _, opt := range opts {
		opt(backlog)
	}
	return backlog, nil
}

func WithUnboundedCapacity(d *DelayedMessageBacklog) {
	d.capacity = 0
	d.finalizedAndReadIndexFetcher = nil
}

// Add takes values of a DelayedMessageBacklogEntry and adds it to the backlog given the entry succeeds validation. It also attempts trimming of backlog if capacity is reached
func (d *DelayedMessageBacklog) Add(entry *DelayedMessageBacklogEntry) error {
	if len(d.entries) > 0 {
		expectedIndex := d.entries[0].Index + uint64(len(d.entries))
		if entry.Index != expectedIndex {
			return fmt.Errorf("message index %d is not sequential, expected %d", entry.Index, expectedIndex)
		}
	}
	d.entries = append(d.entries, entry)
	return d.clear()
}

func (d *DelayedMessageBacklog) Get(index uint64) (*DelayedMessageBacklogEntry, error) {
	if len(d.entries) == 0 {
		return nil, errors.New("delayed message backlog is empty")
	}
	if index < d.entries[0].Index || index > d.entries[len(d.entries)-1].Index {
		return nil, fmt.Errorf("queried index: %d out of bounds, delayed message backlog's starting index: %d, ending index: %d", index, d.entries[0].Index, d.entries[len(d.entries)-1].Index)
	}
	pos := index - d.entries[0].Index
	entry := d.entries[pos]
	if entry.Index != index {
		return nil, fmt.Errorf("index mismatch in the delayed message backlog entry. Queried index: %d, backlog entry's index: %d", index, entry.Index)
	}
	return entry, nil
}

func (d *DelayedMessageBacklog) Len() int                            { return len(d.entries) } // Used for testing InitializeDelayedMessageBacklog function in melrunner
func (d *DelayedMessageBacklog) GetInitMsg() *DelayedInboxMessage    { return d.initMessage }
func (d *DelayedMessageBacklog) setInitMsg(msg *DelayedInboxMessage) { d.initMessage = msg }

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

// Reorg removes from backlog the entries that corresponded to the reorged out parent chain blocks
func (d *DelayedMessageBacklog) reorg(newDelayedMessagedSeen uint64) error {
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

func (d *DelayedMessageBacklog) clone() *DelayedMessageBacklog {
	var deque []*DelayedMessageBacklogEntry
	for _, item := range d.entries {
		msgHash := common.Hash{}
		copy(msgHash[:], item.MsgHash[:])
		deque = append(deque, &DelayedMessageBacklogEntry{
			Index:                       item.Index,
			MsgHash:                     msgHash,
			MelStateParentChainBlockNum: item.MelStateParentChainBlockNum,
		})
	}
	return &DelayedMessageBacklog{d.ctx, d.capacity, deque, nil, d.finalizedAndReadIndexFetcher} // Init msg should only be read once, no need to persist it
}
