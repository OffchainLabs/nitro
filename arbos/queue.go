package arbos

import (
	"errors"
	"github.com/ethereum/go-ethereum/common"
)

type QueueInStorage struct {
	headSegment    *ArbosStorageSegment
	nextGetSegment *ArbosStorageSegment
	nextGetOffset  uint64
	nextPutSegment *ArbosStorageSegment
	nextPutOffset  uint64
}

func (q *QueueInStorage) NextGetSegment() *ArbosStorageSegment {
	return q.nextGetSegment
}

func (q* QueueInStorage) SetNextGetSegment(seg *ArbosStorageSegment) error {
	q.nextGetSegment = seg
	return q.headSegment.Set(0, seg.offset)
}

func (q *QueueInStorage) NextGetOffset() uint64 {
	return q.nextGetOffset
}

func (q* QueueInStorage) SetNextGetOffset(offset uint64) error {
	q.nextGetOffset = offset
	return q.headSegment.Set(1, IntToHash(int64(offset)))
}

func (q *QueueInStorage) NextPutSegment() *ArbosStorageSegment {
	return q.nextPutSegment
}

func (q* QueueInStorage) SetNextPutSegment(seg *ArbosStorageSegment) error {
	q.nextPutSegment = seg
	return q.headSegment.Set(2, seg.offset)
}

func (q *QueueInStorage) NextPutOffset() uint64 {
	return q.nextPutOffset
}

func (q* QueueInStorage) SetNextPutOffset(offset uint64) error {
	q.nextPutOffset = offset
	return q.headSegment.Set(3, IntToHash(int64(offset)))
}

// a queue contains a linked list of segments
// slot zero in each segment points to the next segment; remaining slots contain queue items
const QueueSegmentSize = 64

func OpenQueueInStorage(seg *ArbosStorageSegment) (*QueueInStorage, error) {
	nextGetSegmentOffset, err := seg.Get(0)
	if err != nil {
		return nil, err
	}
	nextGetSegment, err := seg.storage.OpenSegment(nextGetSegmentOffset)
	if err != nil {
		return nil, err
	}
	nextGetOffset, err := seg.GetAsUint64(1)
	if err != nil {
		return nil, err
	}

	nextPutSegmentOffset, err := seg.Get(2)
	if err != nil {
		return nil, err
	}
	nextPutSegment, err := seg.storage.OpenSegment(nextPutSegmentOffset)
	if err != nil {
		return nil, err
	}
	nextPutOffset, err := seg.GetAsUint64(3)
	if err != nil {
		return nil, err
	}

	return &QueueInStorage{
		seg,
		nextGetSegment,
		nextGetOffset,
		nextPutSegment,
		nextPutOffset,
	}, nil
}

func NewQueue(state *ArbosState) (*QueueInStorage, error) {
	qisSegment, err := state.AllocateSegment(4)
	if err != nil {
		return nil, err
	}
	firstSegment, err := state.AllocateSegment(QueueSegmentSize)
	if err != nil {
		return nil, err
	}
	if err := qisSegment.Set(0, firstSegment.offset); err != nil {
		return nil, err
	}
	if err := qisSegment.Set(1, IntToHash(1)); err != nil {
		return nil, err
	}
	if err := qisSegment.Set(2, firstSegment.offset); err != nil {
		return nil, err
	}
	if err := qisSegment.Set(3, IntToHash(1)); err != nil {
		return nil, err
	}
	return &QueueInStorage{
		qisSegment,
		firstSegment,
		1,
		firstSegment,
		1,
	}, nil
}

func (q *QueueInStorage) IsEmpty() bool {
	return q.nextGetSegment.Equals(q.nextPutSegment) && (q.nextGetOffset == q.nextPutOffset)
}

func (q *QueueInStorage) Peek() (common.Hash, error) {   // get the first item, but leave that item in the queue
	if q.IsEmpty() {
		return common.Hash{}, errors.New("tried to Get from empty queue")
	}
	return q.nextGetSegment.Get(q.nextGetOffset)
}

func (q *QueueInStorage) Get() (common.Hash, error) {
	if q.IsEmpty() {
		return common.Hash{}, errors.New("tried to Get from empty queue")
	}

	result, err := q.nextGetSegment.Get(q.nextGetOffset)
	if err != nil {
		return common.Hash{}, err
	}
	if err := q.nextGetSegment.Set(q.nextGetOffset, common.Hash{}); err != nil {  // clear the slot we did get from
		return common.Hash{}, err
	}
	q.nextGetOffset += 1
	if q.nextGetOffset == q.nextGetSegment.size {
		getSegOffset, err := q.nextGetSegment.Get(0)
		if err != nil {
			return common.Hash{}, err
		}
		getSeg, err := q.headSegment.storage.OpenSegment(getSegOffset)
		if err != nil {
			return common.Hash{}, err
		}
		if err := q.nextGetSegment.Set(0, common.Hash{}); err != nil {   // finish clearing the old segment
			return common.Hash{}, err
		}

		if err := q.SetNextGetSegment(getSeg); err != nil {
			return common.Hash{}, nil
		}
		if err := q.SetNextGetOffset(1); err != nil {
			return common.Hash{}, err
		}
	} else {
		if err :=q.headSegment.Set(1, IntToHash(int64(q.nextGetOffset))); err != nil {
			return common.Hash{}, err
		}
	}
	return result, nil
}

func (q *QueueInStorage) Put(val common.Hash) error {
	if err := q.nextPutSegment.Set(q.nextPutOffset, val); err != nil {
		return err
	}

	q.nextPutOffset += 1
	if q.nextPutOffset == q.nextPutSegment.size {
		// allocate a new segment move to it
		newSegment, err := q.headSegment.storage.AllocateSegment(QueueSegmentSize)
		if err != nil {
			return err
		}
		if err := q.nextPutSegment.Set(0, newSegment.offset); err != nil {
			return err
		}
		q.nextPutSegment = newSegment
		if err := q.headSegment.Set(2, newSegment.offset); err != nil {
			return err
		}
		q.nextPutOffset = 1
		if err := q.headSegment.Set(3, IntToHash(1)); err != nil {
			return err
		}
	} else {
		if err :=q.headSegment.Set(3, IntToHash(int64(q.nextPutOffset))); err != nil {
			return err
		}
	}
	return nil
}

