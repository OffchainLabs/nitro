package meltypes

import "github.com/ethereum/go-ethereum/common"

// DelayedMeta contains metadata relating to delayed messages
type DelayedMeta struct {
	Index                       uint64
	Read                        bool
	MerkleRoot                  common.Hash
	MelStateParentChainBlockNum uint64
}

type DelayedMetaDeque struct {
	deque []*DelayedMeta
}

func (d *DelayedMetaDeque) Len() int                           { return len(d.deque) }   // Used for testing purposes
func (d *DelayedMetaDeque) GetByPos(index uint64) *DelayedMeta { return d.deque[index] } // Used for testing purposes

func (d *DelayedMetaDeque) Add(item *DelayedMeta) {
	d.deque = append(d.deque, item)
}

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
			Read:                        item.Read,
			MerkleRoot:                  merkleRoot,
			MelStateParentChainBlockNum: item.MelStateParentChainBlockNum,
		})
	}
	return &DelayedMetaDeque{deque}
}

func (d *DelayedMetaDeque) ClearReorged(delayedMessagesRead, newDelayedMessagesRead, delayedMessagedSeen, newDelayedMessagedSeen uint64) {
	if len(d.deque) > 0 {
		if newDelayedMessagedSeen < delayedMessagedSeen {
			// DelayedMessagedSeen rewinded
			rightTrimPos := newDelayedMessagedSeen - d.deque[0].Index
			d.deque = d.deque[:rightTrimPos]
		}
		if newDelayedMessagesRead < delayedMessagesRead {
			// DelayedMessagesRead rewinded
			for _, delayedMeta := range d.deque {
				if !delayedMeta.Read {
					break
				}
				if delayedMeta.Index >= newDelayedMessagesRead {
					delayedMeta.Read = false
				}
			}
		}
	}
}

// ClearReadAndFinalized trims the DelayedMetaDeque from left, such that the item is only removed if the corresponding delayed message is
// read and the MelStateParentChainBlockNum is finalized- this is to make DelayedMetaDeque as reorg resistant as possible
func (d *DelayedMetaDeque) ClearReadAndFinalized(finalizedBlock uint64) {
	i := 0
	for i < len(d.deque) {
		if !d.deque[i].Read || d.deque[i].MelStateParentChainBlockNum > finalizedBlock {
			break
		}
		i++
	}
	d.deque = d.deque[i:]
}
