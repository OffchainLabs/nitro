package mel

import (
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
	targetBufferSize             int
	entries                      []*DelayedMessageBacklogEntry
	dirtiesStartPos              int // represents the starting point of dirties in the entries list, items added while processing a state
	initMessage                  *DelayedInboxMessage
	finalizedAndReadIndexFetcher func() (uint64, error)
}

func NewDelayedMessageBacklog(targetBufferSize int, finalizedAndReadIndexFetcher func() (uint64, error)) (*DelayedMessageBacklog, error) {
	if targetBufferSize == 0 {
		return nil, fmt.Errorf("targetBufferSize of DelayedMessageBacklog cannot be zero")
	}
	if finalizedAndReadIndexFetcher == nil {
		return nil, fmt.Errorf("finalizedAndReadIndexFetcher of DelayedMessageBacklog cannot be nil")
	}
	backlog := &DelayedMessageBacklog{
		targetBufferSize:             targetBufferSize,
		entries:                      make([]*DelayedMessageBacklogEntry, 0),
		initMessage:                  nil,
		finalizedAndReadIndexFetcher: finalizedAndReadIndexFetcher,
	}
	return backlog, nil
}

// Add takes values of a DelayedMessageBacklogEntry and adds it to the backlog given the entry succeeds validation. It also attempts trimming of backlog if targetBufferSize is reached
func (d *DelayedMessageBacklog) Add(entry *DelayedMessageBacklogEntry) error {
	if len(d.entries) > 0 {
		expectedIndex := d.entries[0].Index + uint64(len(d.entries))
		if entry.Index != expectedIndex {
			return fmt.Errorf("message index %d is not sequential, expected %d", entry.Index, expectedIndex)
		}
	}
	d.entries = append(d.entries, entry)
	if len(d.entries) <= d.targetBufferSize {
		return nil
	}
	d.trimFinalizedAndReadEntries()
	return nil
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

func (d *DelayedMessageBacklog) CommitDirties()                      { d.dirtiesStartPos = len(d.entries) } // Add dirties to the entries by moving dirtiesStartPos to the end
func (d *DelayedMessageBacklog) Len() int                            { return len(d.entries) }              // Used for testing InitializeDelayedMessageBacklog function in melrunner
func (d *DelayedMessageBacklog) GetInitMsg() *DelayedInboxMessage    { return d.initMessage }
func (d *DelayedMessageBacklog) setInitMsg(msg *DelayedInboxMessage) { d.initMessage = msg }

// trimFinalizedAndReadEntries removes from backlog (if exceeds targetBufferSize) the entries that correspond to the delayed messages that are both READ
// and belong to finalized parent chain blocks. We should not interrupt delayed messages accumulation if we cannot trim the backlog, since its not high priority
func (d *DelayedMessageBacklog) trimFinalizedAndReadEntries() {
	if d.finalizedAndReadIndexFetcher != nil && d.dirtiesStartPos > 0 { // if all entries are currently dirty we dont trim the finalized ones
		finalizedDelayedMessagesRead, err := d.finalizedAndReadIndexFetcher()
		if err != nil {
			log.Error("Unable to trim finalized and read delayed messages from DelayedMessageBacklog, will be retried later", "err", err)
			return
		}
		if finalizedDelayedMessagesRead > d.entries[0].Index {
			leftTrimPos := min(finalizedDelayedMessagesRead-d.entries[0].Index, uint64(len(d.entries)))
			// #nosec G115
			leftTrimPos = min(leftTrimPos, uint64(d.dirtiesStartPos)) // cannot trim dirties yet, they will be trimmed out in the next attempt
			d.entries = d.entries[leftTrimPos:]
			// #nosec G115
			d.dirtiesStartPos -= int(leftTrimPos) // adjust start position of dirties
		}
	}
}

// Reorg removes from backlog the entries that corresponded to the reorged out parent chain blocks
func (d *DelayedMessageBacklog) reorg(newDelayedMessagedSeen uint64) error {
	if d.dirtiesStartPos != len(d.entries) {
		return fmt.Errorf("delayedMessageBacklog dirties is non-empty when reorg was called, size of dirties:%d", len(d.entries)-d.dirtiesStartPos)
	}
	if len(d.entries) == 0 {
		return nil
	}
	if newDelayedMessagedSeen >= d.entries[0].Index {
		rightTrimPos := newDelayedMessagedSeen - d.entries[0].Index
		if rightTrimPos > uint64(len(d.entries)) {
			return fmt.Errorf("newDelayedMessagedSeen: %d during a reorg is greater (by more than 1) than the greatest delayed message index stored in backlog: %d", newDelayedMessagedSeen, d.entries[len(d.entries)-1].Index)
		}
		d.entries = d.entries[:rightTrimPos]
	} else {
		d.entries = make([]*DelayedMessageBacklogEntry, 0)
	}
	d.dirtiesStartPos = len(d.entries)
	return nil
}

// clone is a shallow clone of DelayedMessageBacklog
func (d *DelayedMessageBacklog) clone() *DelayedMessageBacklog {
	// Remove dirties from entries
	d.entries = d.entries[:d.dirtiesStartPos]
	return d
}
