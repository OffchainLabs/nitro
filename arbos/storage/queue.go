//
// Copyright 2021-2022, Offchain Labs, Inc. All rights reserved.
//

package storage

import (
	"github.com/offchainlabs/nitro/arbos/util"

	"github.com/ethereum/go-ethereum/common"
)

type Queue struct {
	storage       *Storage
	nextPutOffset StorageBackedUint64
	nextGetOffset StorageBackedUint64
}

func InitializeQueue(sto *Storage) error {
	err := sto.SetUint64ByUint64(0, 2)
	if err != nil {
		return err
	}
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
	put, err := q.nextPutOffset.Get()
	if err != nil {
		return false, err
	}
	get, err := q.nextGetOffset.Get()
	return put == get, err
}

func (q *Queue) Size() (uint64, error) {
	put, err := q.nextPutOffset.Get()
	if err != nil {
		return 0, err
	}
	get, err := q.nextGetOffset.Get()
	return put - get, err
}

func (q *Queue) Peek() (*common.Hash, error) { // returns nil iff queue is empty
	empty, err := q.IsEmpty()
	if empty || err != nil {
		return nil, err
	}
	next, err := q.nextGetOffset.Get()
	if err != nil {
		return nil, err
	}
	res, err := q.storage.GetByUint64(next)
	return &res, err
}

func (q *Queue) Get() (*common.Hash, error) { // returns nil iff queue is empty
	empty, err := q.IsEmpty()
	if empty || err != nil {
		return nil, err
	}
	newOffset, err := q.nextGetOffset.Increment()
	if err != nil {
		return nil, err
	}
	res, err := q.storage.Swap(util.UintToHash(newOffset-1), common.Hash{})
	return &res, err
}

func (q *Queue) Put(val common.Hash) error {
	newOffset, err := q.nextPutOffset.Increment()
	if err != nil {
		return err
	}
	return q.storage.SetByUint64(newOffset-1, val)
}
