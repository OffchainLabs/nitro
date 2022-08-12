// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package validator

import (
	"context"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rlp"
)

type LocalStateTracker struct {
	db ethdb.Database

	mutex                  sync.Mutex
	lastBlockValidated     uint64
	lastBlockValidatedHash common.Hash
	nextBlockToValidate    uint64
	nextGlobalState        GlobalStatePosition
	status                 map[uint64]*validationStatus
}

func NewLocalStateTracker(db ethdb.Database) (*LocalStateTracker, error) {
	t := &LocalStateTracker{
		db:     db,
		status: make(map[uint64]*validationStatus),
	}
	return t, nil
}

func (t *LocalStateTracker) Initialize(ctx context.Context, genesisBlock *types.Block) error {
	return t.readFromDisk(genesisBlock)
}

func (t *LocalStateTracker) readFromDisk(genesisBlock *types.Block) error {
	exists, err := t.db.Has(lastBlockValidatedInfoKey)
	if err != nil {
		return err
	}

	if !exists {
		// The db contains no validation info; start from the beginning.
		// This skips validating the genesis block.
		t.lastBlockValidated = genesisBlock.NumberU64()
		t.lastBlockValidatedHash = genesisBlock.Hash()
		t.nextBlockToValidate = genesisBlock.NumberU64() + 1
		t.nextGlobalState = GlobalStatePosition{
			BatchNumber: 1,
			PosInBatch:  0,
		}
		return nil
	}

	infoBytes, err := t.db.Get(lastBlockValidatedInfoKey)
	if err != nil {
		return err
	}

	var info lastBlockValidatedDbInfo
	err = rlp.DecodeBytes(infoBytes, &info)
	if err != nil {
		return err
	}

	t.lastBlockValidated = info.BlockNumber
	t.lastBlockValidatedHash = info.BlockHash
	t.nextBlockToValidate = t.lastBlockValidated + 1
	t.nextGlobalState = info.AfterPosition

	return nil
}

func (t *LocalStateTracker) LastBlockValidated(ctx context.Context) (uint64, error) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	return t.lastBlockValidated, nil
}

func (t *LocalStateTracker) LastBlockValidatedAndHash(ctx context.Context) (uint64, common.Hash, error) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	return t.lastBlockValidated, t.lastBlockValidatedHash, nil
}

func (t *LocalStateTracker) setLastValidated(blockNumber uint64, blockHash common.Hash, endPos GlobalStatePosition) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.lastBlockValidated = blockNumber
	t.lastBlockValidatedHash = blockHash

	info := lastBlockValidatedDbInfo{
		BlockNumber:   blockNumber,
		BlockHash:     blockHash,
		AfterPosition: endPos,
	}
	encodedInfo, err := rlp.EncodeToBytes(info)
	if err != nil {
		return err
	}
	return t.db.Put(lastBlockValidatedInfoKey, encodedInfo)
}

func (t *LocalStateTracker) GetNextValidation(ctx context.Context) (uint64, GlobalStatePosition, error) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	return t.nextBlockToValidate, t.nextGlobalState, nil
}

func (t *LocalStateTracker) BeginValidation(ctx context.Context, header *types.Header, startPos GlobalStatePosition, endPos GlobalStatePosition) (bool, error) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	num := header.Number.Uint64()
	if t.nextBlockToValidate != num || t.nextGlobalState != startPos {
		return false, nil
	}
	var prevHash common.Hash
	if num > t.lastBlockValidated+1 {
		prevHash = t.status[num-1].blockHash
	} else if num == t.lastBlockValidated+1 {
		prevHash = t.lastBlockValidatedHash
	} else {
		return false, fmt.Errorf("lastBlockValidated is %v but nextBlockToValidate is %v?", t.lastBlockValidated, num)
	}
	if header.ParentHash != prevHash {
		return false, fmt.Errorf("previous block %v hash is %v but attempting to validate next block with a previous hash of %v", num-1, prevHash, header.ParentHash)
	}
	t.status[num] = &validationStatus{
		prevHash:    header.ParentHash,
		blockHash:   header.Hash(),
		validated:   false,
		endPosition: endPos,
	}
	t.nextBlockToValidate = num + 1
	t.nextGlobalState = endPos
	return true, nil
}

func (t *LocalStateTracker) ValidationCompleted(ctx context.Context, initialEntry *validationEntry) (uint64, GlobalStatePosition, error) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	if initialEntry.BlockNumber >= t.nextBlockToValidate {
		return 0, GlobalStatePosition{}, fmt.Errorf("completed validation for block %v >= nextBlockToValidate %v", initialEntry.BlockNumber, t.nextBlockToValidate)
	}
	status, ok := t.status[initialEntry.BlockNumber]
	if !ok {
		return 0, GlobalStatePosition{}, fmt.Errorf("completed validation for unknown block %v", initialEntry.BlockNumber)
	}
	if status.blockHash != initialEntry.BlockHash {
		return 0, GlobalStatePosition{}, fmt.Errorf("completed validation for block %v with hash %v but we have hash %v saved", initialEntry.BlockNumber, initialEntry.BlockHash, status.blockHash)
	}
	status.validated = true
	var lastEndPosition GlobalStatePosition
	for {
		blockNum := t.lastBlockValidated + 1
		status, ok := t.status[blockNum]
		if !ok || !status.validated {
			break
		}
		if t.lastBlockValidatedHash != status.prevHash {
			return 0, GlobalStatePosition{}, fmt.Errorf("at block number %v last validated hash %v doesn't match new validation parent %v", t.lastBlockValidated, t.lastBlockValidatedHash, status.prevHash)
		}
		delete(t.status, blockNum)
		t.lastBlockValidated = blockNum
		t.lastBlockValidatedHash = status.blockHash
		lastEndPosition = status.endPosition
	}
	return t.lastBlockValidated, lastEndPosition, nil
}

func (t *LocalStateTracker) Reorg(ctx context.Context, blockNum uint64, blockHash common.Hash, nextPosition GlobalStatePosition, isValid func(uint64, common.Hash) bool) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if t.nextBlockToValidate <= blockNum+1 {
		return nil
	}

	for i := t.lastBlockValidated + 1; i < t.nextBlockToValidate; i++ {
		delete(t.status, i)
	}
	t.nextBlockToValidate = blockNum + 1
	t.nextGlobalState = nextPosition

	if t.lastBlockValidated > blockNum {
		err := t.setLastValidated(blockNum, blockHash, nextPosition)
		if err != nil {
			return err
		}
	}

	return nil
}
