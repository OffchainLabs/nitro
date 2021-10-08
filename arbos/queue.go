package arbos

import (
	"github.com/ethereum/go-ethereum/common"
	"math/big"
)

type QueueInStorage struct {
	segment       *SizedArbosStorageSegment
	nextPutOffset *common.Hash
	nextGetOffset *common.Hash
}

func AllocateQueueInStorage(state *ArbosState) (*QueueInStorage, error) {
	segment, err := state.AllocateSizedSegment(2)
	if err != nil {
		return nil, err
	}
	contentsOffset := state.AllocateEmptyStorageOffset()
	if err := segment.Set(0, *contentsOffset); err != nil {
		return nil, err
	}
	if err := segment.Set(1, *contentsOffset); err != nil {
		return nil, err
	}
	return &QueueInStorage{ segment, contentsOffset, contentsOffset }, nil
}

func OpenQueueInStorage(state *ArbosState, offset common.Hash) (*QueueInStorage, error) {
	segment, err := state.OpenSizedSegment(offset)
	if err != nil {
		return nil, err
	}
	npo, err := segment.Get(0)
	if err != nil {
		return nil, err
	}
	ngo, err := segment.Get(1)
	if err != nil {
		return nil, err
	}
	return &QueueInStorage{ segment, &npo, &ngo }, nil
}

func (q *QueueInStorage) IsEmpty() bool {
	return q.nextPutOffset.Big().Cmp(q.nextGetOffset.Big()) == 0
}

func (q *QueueInStorage) Peek() *common.Hash {   // returns nil iff queue is empty
	if q.IsEmpty() {
		return nil
	}
	res := q.segment.storage.backingStorage.Get(*q.nextGetOffset)
	return &res
}

func (q *QueueInStorage) Get() *common.Hash {   // returns nil iff queue is empty
	if q.IsEmpty() {
		return nil
	}
	res := q.segment.storage.backingStorage.Swap(*q.nextGetOffset, common.Hash{})
	nextGetOffset := common.BigToHash(new(big.Int).Add(q.nextGetOffset.Big(), big.NewInt(1)))
	q.nextGetOffset = &nextGetOffset
	if err := q.segment.Set(1, nextGetOffset); err != nil {
		panic(err)
	}
	return &res
}

func (q *QueueInStorage) Put(val common.Hash) {
	q.segment.storage.backingStorage.Set(*q.nextPutOffset, val)
	nextPutOffset := common.BigToHash(new(big.Int).Add(q.nextPutOffset.Big(), big.NewInt(1)))
	q.nextPutOffset = &nextPutOffset
	if err := q.segment.Set(0, nextPutOffset); err != nil {
		panic(err)
	}
}
