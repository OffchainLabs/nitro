// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package blockhash

import (
	"encoding/binary"
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/arbos/storage"
)

type Blockhashes struct {
	backingStorage  *storage.Storage
	nextBlockNumber storage.StorageBackedUint64
}

func InitializeBlockhashes(backingStorage *storage.Storage) {
	// no need to do anything, nextBlockNumber is already zero and no hashes are needed when nextBlockNumber is zero
}

func OpenBlockhashes(backingStorage *storage.Storage) *Blockhashes {
	return &Blockhashes{backingStorage, backingStorage.OpenStorageBackedUint64(0)}
}

func (bh *Blockhashes) NextBlockNumber() (uint64, error) {
	return bh.nextBlockNumber.Get()
}

func (bh *Blockhashes) BlockHash(number uint64) (common.Hash, error) {
	currentNumber, err := bh.nextBlockNumber.Get()
	if err != nil {
		return common.Hash{}, err
	}
	if number >= currentNumber || number+256 < currentNumber {
		return common.Hash{}, errors.New("invalid block number for BlockHash")
	}
	return bh.backingStorage.GetByUint64(1 + (number % 256))
}

func (bh *Blockhashes) RecordNewL1Block(number uint64, blockHash common.Hash) error {
	nextNumber, err := bh.nextBlockNumber.Get()
	if err != nil {
		return err
	}
	if number < nextNumber {
		// we already have a stored hash for the block, so just return
		return nil
	}
	if nextNumber+256 < number {
		nextNumber = number - 256 // no need to record hashes that we're just going to discard
	}
	for nextNumber+1 < number {
		// fill in hashes for any "skipped over" blocks
		nextNumber++
		var nextNumBuf [8]byte
		binary.LittleEndian.Uint64(nextNumBuf[:])

		fill, err := bh.backingStorage.Keccak(blockHash.Bytes(), nextNumBuf[:])
		if err != nil {
			return err
		}
		err = bh.backingStorage.SetByUint64(
			1+(nextNumber%256),
			common.BytesToHash(fill),
		)
		if err != nil {
			return err
		}
	}

	err = bh.backingStorage.SetByUint64(1+(number%256), blockHash)
	if err != nil {
		return err
	}
	return bh.nextBlockNumber.Set(number + 1)
}
