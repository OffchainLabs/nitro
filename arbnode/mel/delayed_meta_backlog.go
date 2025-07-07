package mel

import "github.com/ethereum/go-ethereum/common"

// DelayedMeta contains metadata relating to delayed messages
type DelayedMeta struct {
	Index                       uint64
	MerkleRoot                  common.Hash
	MelStateParentChainBlockNum uint64
}

// DelayedMetaBacklog is a data structure that holds a deque containing meta data related to delayed messages that are currently SEEN by MEL
// but not yet READ. This enables verification of correctness of delayed messages read from the database, i.e each MEL state has the
// DelayedMessageMerklePartials field that holds merkle partials array after all the delayed messages SEEN while constructing the state are
// accumulated into the merkle accumulator. Hence to prove while READing a delayed MSG that it was part of this merkle accumulator- we would
// need to have a way to go back to the state before current - fetch the partials - make an accumulator from these partials - accumulate messages
// upto this delayed message index - verify that the merkle root after accumulating the MSG matches the one stored in its DelayedMeta - that was
// created when that MSG was first SEEN
//
// This is to be initialized in the FetchInitialState function of InitialStateFetcher. If not, then Start fsm step of MEL runner initializes it
type DelayedMetaBacklog struct {
	// If deque grows past this capacity,
	// trim the read and finalized delayedMeta
	cap     int
	deque   []*DelayedMeta
	initMsg *DelayedInboxMessage
}

func NewDelayedMetaBacklog() *DelayedMetaBacklog {
	return &DelayedMetaBacklog{
		deque:   make([]*DelayedMeta, 0),
		initMsg: nil,
	}
}

func (d *DelayedMetaBacklog) Len() int                           { return len(d.deque) }   // Used for testing purposes
func (d *DelayedMetaBacklog) GetByPos(index uint64) *DelayedMeta { return d.deque[index] } // Used for testing purposes

func (d *DelayedMetaBacklog) SetTargetCapacity(cap int) { d.cap = cap }

func (d *DelayedMetaBacklog) Add(item *DelayedMeta) {
	d.deque = append(d.deque, item)
}

// Used exclusively while reading the init message
func (d *DelayedMetaBacklog) SetInitMsg(msg *DelayedInboxMessage) { d.initMsg = msg }
func (d *DelayedMetaBacklog) GetInitMsg() *DelayedInboxMessage    { return d.initMsg }

func (d *DelayedMetaBacklog) GetByIndex(index uint64) *DelayedMeta {
	pos := index - d.deque[0].Index
	return d.deque[pos]
}

func (d *DelayedMetaBacklog) Clone() *DelayedMetaBacklog {
	var deque []*DelayedMeta
	for _, item := range d.deque {
		merkleRoot := common.Hash{}
		copy(merkleRoot[:], item.MerkleRoot[:])
		deque = append(deque, &DelayedMeta{
			Index:                       item.Index,
			MerkleRoot:                  merkleRoot,
			MelStateParentChainBlockNum: item.MelStateParentChainBlockNum,
		})
	}
	return &DelayedMetaBacklog{d.cap, deque, nil} // Init msg should only be read once, no need to persist it
}

// Reorg trims the DelayedMetaBacklog from right upto the given DelayedMessagedSeen count from the current valid state
func (d *DelayedMetaBacklog) Reorg(newDelayedMessagedSeen uint64) {
	if len(d.deque) == 0 {
		return
	}
	if newDelayedMessagedSeen >= d.deque[0].Index {
		// DelayedMessagedSeen rewinded
		rightTrimPos := newDelayedMessagedSeen - d.deque[0].Index
		d.deque = d.deque[:rightTrimPos]
	}
}

// Clear trims the DelayedMetaBacklog from left, such that the item is only removed if the corresponding delayed message is
// read and the MelStateParentChainBlockNum is finalized- this is to make DelayedMetaBacklog as reorg resistant as possible
//
// This function takes a fetcher instead of finalizedDelayedMessagesRead directly, since getting the finalizedDelayedMessagesRead
// can be a costly operation and should only be used when deque grows past the backlog's target capacity
func (d *DelayedMetaBacklog) Clear(finalizedDelayedMessagesReadFetcher func() uint64) {
	if len(d.deque) <= d.cap {
		return
	}
	finalizedDelayedMessagesRead := finalizedDelayedMessagesReadFetcher()
	if finalizedDelayedMessagesRead > d.deque[0].Index {
		leftTrimPos := finalizedDelayedMessagesRead - d.deque[0].Index
		d.deque = d.deque[leftTrimPos:]
	}
}
