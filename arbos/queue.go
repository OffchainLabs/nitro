package arbos

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type QueueInStorage struct {
	segment       *StorageSegment
	nextPutOffset *common.Hash
	nextGetOffset *common.Hash
}

func AllocateQueueInStorage(state *ArbosState) *QueueInStorage {
	segment, err := state.AllocateSegment(2)
	if err != nil {
		panic(err)
	}
	contentsOffset := state.AllocateEmptyStorageOffset()
	segment.Set(0, *contentsOffset)
	segment.Set(1, *contentsOffset)
	return &QueueInStorage{ segment, contentsOffset, contentsOffset }
}

func OpenQueueInStorage(state *ArbosState, offset common.Hash) *QueueInStorage {
	segment := state.OpenSegment(offset)
	npo := segment.Get(0)
	ngo := segment.Get(1)
	return &QueueInStorage{ segment, &npo, &ngo }
}

func (q *QueueInStorage) IsEmpty() bool {
	return q.nextPutOffset.Big().Cmp(q.nextGetOffset.Big()) == 0
}

func (q *QueueInStorage) Peek() *common.Hash {   // returns nil iff queue is empty
	if q.IsEmpty() {
		return nil
	}
	res := q.segment.storage.Get(*q.nextGetOffset)
	return &res
}

func (q *QueueInStorage) Get() *common.Hash {   // returns nil iff queue is empty
	if q.IsEmpty() {
		return nil
	}
	res := q.segment.storage.Swap(*q.nextGetOffset, common.Hash{})
	nextGetOffset := common.BigToHash(new(big.Int).Add(q.nextGetOffset.Big(), big.NewInt(1)))
	q.nextGetOffset = &nextGetOffset
	q.segment.Set(1, nextGetOffset)
	return &res
}

func (q *QueueInStorage) Put(val common.Hash) {
	q.segment.storage.Set(*q.nextPutOffset, val)
	nextPutOffset := common.BigToHash(new(big.Int).Add(q.nextPutOffset.Big(), big.NewInt(1)))
	q.nextPutOffset = &nextPutOffset
	q.segment.Set(0, nextPutOffset)
}

func (q *QueueInStorage) Size() *big.Int {
	return new(big.Int).Sub(q.nextPutOffset.Big(), q.nextGetOffset.Big())
}
