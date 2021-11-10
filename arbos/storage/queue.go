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
	nextPutOffset *StorageBackedInt64
	nextGetOffset *StorageBackedInt64
}

func InitializeQueue(sto *Storage) {
	sto.SetByInt64(0, util.IntToHash(2))
	sto.SetByInt64(1, util.IntToHash(2))
}

func OpenQueue(sto *Storage) *Queue {
	return &Queue{
		sto,
		sto.OpenStorageBackedInt64(util.IntToHash(0)),
		sto.OpenStorageBackedInt64(util.IntToHash(1)),
	}
}

func (q *Queue) IsEmpty() bool {
	return q.nextPutOffset.Get() == q.nextGetOffset.Get()
}

func (q *Queue) Peek() *common.Hash { // returns nil iff queue is empty
	if q.IsEmpty() {
		return nil
	}
	res := q.storage.GetByInt64(q.nextGetOffset.Get())
	return &res
}

func (q *Queue) Get() *common.Hash { // returns nil iff queue is empty
	if q.IsEmpty() {
		return nil
	}
	ngo := q.nextGetOffset.Get()
	res := q.storage.Swap(util.IntToHash(ngo), common.Hash{})
	ngo++
	q.nextGetOffset.Set(ngo)
	return &res
}

func (q *Queue) Put(val common.Hash) {
	npo := q.nextPutOffset.Get()
	q.storage.SetByInt64(npo, val)
	npo++
	q.nextPutOffset.Set(npo)
}
