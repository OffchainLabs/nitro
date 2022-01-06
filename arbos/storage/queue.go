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
	nextPutOffset StorageBackedUint64
	nextGetOffset StorageBackedUint64
}

func InitializeQueue(sto *Storage) {
	sto.SetUint64ByUint64(0, 2)
	sto.SetUint64ByUint64(1, 2)
}

func OpenQueue(sto *Storage) *Queue {
	return &Queue{
		sto,
		sto.NewStorageBackedUint64(0),
		sto.NewStorageBackedUint64(1),
	}
}

func (q *Queue) IsEmpty() bool {
	return q.nextPutOffset.Get() == q.nextGetOffset.Get()
}

func (q *Queue) Peek() *common.Hash { // returns nil iff queue is empty
	if q.IsEmpty() {
		return nil
	}
	res := q.storage.GetByUint64(q.nextGetOffset.Get())
	return &res
}

func (q *Queue) Get() *common.Hash { // returns nil iff queue is empty
	if q.IsEmpty() {
		return nil
	}
	newOffset := q.nextGetOffset.Increment()
	res := q.storage.Swap(util.UintToHash(newOffset-1), common.Hash{})
	return &res
}

func (q *Queue) Put(val common.Hash) {
	newOffset := q.nextPutOffset.Increment()
	q.storage.SetByUint64(newOffset-1, val)
}
