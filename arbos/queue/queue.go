//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package queue

import (
	"github.com/offchainlabs/arbstate/arbos/storage"
	"github.com/offchainlabs/arbstate/arbos/util"

	"github.com/ethereum/go-ethereum/common"
)

type QueueInStorage struct {
	storage       *storage.Storage
	nextPutOffset int64
	nextGetOffset int64
}

func AllocateQueueInStorage(backingStorage *storage.Storage) (*QueueInStorage, common.Hash) {
	key := backingStorage.UniqueKey()
	storage := backingStorage.Open(key.Bytes())
	storage.SetByInt64(0, util.IntToHash(2))
	storage.SetByInt64(1, util.IntToHash(2))
	return &QueueInStorage{storage, 2, 2}, key
}

func OpenQueueInStorage(backingStorage *storage.Storage, key common.Hash) *QueueInStorage {
	storage := backingStorage.Open(key.Bytes())
	npo := storage.GetByInt64(0).Big().Int64()
	ngo := storage.GetByInt64(1).Big().Int64()
	return &QueueInStorage{storage, npo, ngo}
}

func (q *QueueInStorage) IsEmpty() bool {
	return q.nextPutOffset == q.nextGetOffset
}

func (q *QueueInStorage) Peek() *common.Hash { // returns nil iff queue is empty
	if q.IsEmpty() {
		return nil
	}
	res := q.storage.GetByInt64(q.nextGetOffset)
	return &res
}

func (q *QueueInStorage) Get() *common.Hash { // returns nil iff queue is empty
	if q.IsEmpty() {
		return nil
	}
	res := q.storage.Swap(util.IntToHash(q.nextGetOffset), common.Hash{})
	q.nextGetOffset++
	q.storage.SetByInt64(1, util.IntToHash(q.nextGetOffset))
	return &res
}

func (q *QueueInStorage) Put(val common.Hash) {
	q.storage.SetByInt64(q.nextPutOffset, val)
	q.nextPutOffset++
	q.storage.SetByInt64(0, util.IntToHash(q.nextPutOffset))
}
