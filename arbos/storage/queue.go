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

func InitializeQueue(sto *Storage) error {
	_ = sto.SetUint64ByUint64(0, 2)
	return sto.SetUint64ByUint64(1, 2)
}

func OpenQueue(sto *Storage) *Queue {
	return &Queue{
		sto,
		sto.OpenStorageBackedUint64(0),
		sto.OpenStorageBackedUint64(1),
	}
}

func (q *Queue) IsEmpty() (bool, error) {
	put, _ := q.nextPutOffset.Get()
	get, err := q.nextGetOffset.Get()
	return put == get, err
}

func (q *Queue) Peek() (*common.Hash, error) { // returns nil iff queue is empty
	empty, _ := q.IsEmpty()
	if empty {
		return nil, nil
	}
	next, _ := q.nextGetOffset.Get()
	res, err := q.storage.GetByUint64(next)
	return &res, err
}

func (q *Queue) Get() (*common.Hash, error) { // returns nil iff queue is empty
	empty, _ := q.IsEmpty()
	if empty {
		return nil, nil
	}
	newOffset, _ := q.nextGetOffset.Increment()
	res, err := q.storage.Swap(util.UintToHash(newOffset-1), common.Hash{})
	return &res, err
}

func (q *Queue) Put(val common.Hash) error {
	newOffset, _ := q.nextPutOffset.Increment()
	return q.storage.SetByUint64(newOffset-1, val)
}
