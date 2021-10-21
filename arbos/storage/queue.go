//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package storage

import (
	"github.com/offchainlabs/arbstate/arbos/util"

	"github.com/ethereum/go-ethereum/common"
)

type Queue struct {
	storage       *Storage
	nextPutOffset int64
	nextGetOffset int64
}

func InitializeQueue(sto *Storage) {
	sto.SetByInt64(0, util.IntToHash(2))
	sto.SetByInt64(1, util.IntToHash(2))
}

func OpenQueue(sto *Storage) *Queue {
	return &Queue{
		sto,
		sto.GetByInt64(0).Big().Int64(),
		sto.GetByInt64(1).Big().Int64(),
	}
}

func (q *Queue) IsEmpty() bool {
	return q.nextPutOffset == q.nextGetOffset
}

func (q *Queue) Peek() *common.Hash { // returns nil iff queue is empty
	if q.IsEmpty() {
		return nil
	}
	res := q.storage.GetByInt64(q.nextGetOffset)
	return &res
}

func (q *Queue) Get() *common.Hash { // returns nil iff queue is empty
	if q.IsEmpty() {
		return nil
	}
	res := q.storage.Swap(util.IntToHash(q.nextGetOffset), common.Hash{})
	q.nextGetOffset++
	q.storage.SetByInt64(1, util.IntToHash(q.nextGetOffset))
	return &res
}

func (q *Queue) Put(val common.Hash) {
	q.storage.SetByInt64(q.nextPutOffset, val)
	q.nextPutOffset++
	q.storage.SetByInt64(0, util.IntToHash(q.nextPutOffset))
}
