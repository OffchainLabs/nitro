package mel

import "github.com/ethereum/go-ethereum/common"

// DelayedMeta contains metadata relating to delayed messages
type DelayedMeta struct {
	Index                       uint64
	MerkleRoot                  common.Hash
	MelStateParentChainBlockNum uint64
}

type DelayedMetaDeque struct {
	deque   []*DelayedMeta
	initMsg *DelayedInboxMessage
}

func NewDelayedMetaDeque() *DelayedMetaDeque {
	return &DelayedMetaDeque{
		deque:   make([]*DelayedMeta, 0),
		initMsg: nil,
	}
}

func (d *DelayedMetaDeque) Len() int                           { return len(d.deque) }
func (d *DelayedMetaDeque) GetByPos(index uint64) *DelayedMeta { return d.deque[index] } // Used for testing purposes

func (d *DelayedMetaDeque) Add(item *DelayedMeta) {
	d.deque = append(d.deque, item)
}

// Used exclusively while reading the init message
func (d *DelayedMetaDeque) SetInitMsg(msg *DelayedInboxMessage) { d.initMsg = msg }
func (d *DelayedMetaDeque) GetInitMsg() *DelayedInboxMessage    { return d.initMsg }

func (d *DelayedMetaDeque) GetByIndex(index uint64) *DelayedMeta {
	pos := index - d.deque[0].Index
	return d.deque[pos]
}

func (d *DelayedMetaDeque) Clone() *DelayedMetaDeque {
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
	return &DelayedMetaDeque{deque, nil} // Init msg should only be read once, no need to persist it
}

func (d *DelayedMetaDeque) ClearReorged(newDelayedMessagedSeen uint64) {
	if len(d.deque) == 0 {
		return
	}
	if newDelayedMessagedSeen >= d.deque[0].Index {
		// DelayedMessagedSeen rewinded
		rightTrimPos := newDelayedMessagedSeen - d.deque[0].Index
		d.deque = d.deque[:rightTrimPos]
	}
}

// ClearReadAndFinalized trims the DelayedMetaDeque from left, such that the item is only removed if the corresponding delayed message is
// read and the MelStateParentChainBlockNum is finalized- this is to make DelayedMetaDeque as reorg resistant as possible
func (d *DelayedMetaDeque) ClearReadAndFinalized(finalizedDelayedMessagesRead uint64) {
	if len(d.deque) == 0 {
		return
	}
	if finalizedDelayedMessagesRead > d.deque[0].Index {
		leftTrimPos := finalizedDelayedMessagesRead - d.deque[0].Index
		d.deque = d.deque[leftTrimPos:]
	}
}
