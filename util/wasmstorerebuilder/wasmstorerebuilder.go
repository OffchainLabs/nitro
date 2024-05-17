// Copyright 2021-2024, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package wasmstorerebuilder

import (
	"bytes"
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/offchainlabs/nitro/arbos/arbosState"
)

var RebuildingPositionKey []byte = []byte("_rebuildingPosition")             // contains the codehash upto where rebuilding of wasm store was last completed
var RebuildingStartBlockTimeKey []byte = []byte("_rebuildingStartBlockTime") // contains the block time when rebuilding of wasm store first began
var RebuildingDone common.Hash = common.BytesToHash([]byte("_done"))         // indicates that the rebuilding is done, if RebuildingPositionKey holds this value it implies rebuilding was completed

func GetRebuildingParam[T any](wasmStore ethdb.KeyValueStore, key []byte) (T, error) {
	var empty T
	posBytes, err := wasmStore.Get(key)
	if err != nil {
		return empty, err
	}
	var val T
	err = rlp.DecodeBytes(posBytes, &val)
	if err != nil {
		return empty, fmt.Errorf("invalid value stored in rebuilding key or error decoding rebuildingBytes: %w", err)
	}
	return val, nil
}

func SetRebuildingParam[T any](wasmStore ethdb.KeyValueStore, key []byte, val T) error {
	valBytes, err := rlp.EncodeToBytes(val)
	if err != nil {
		return err
	}
	err = wasmStore.Put(key, valBytes)
	if err != nil {
		return err
	}
	return nil
}

// RebuildWasmStore function runs a loop looking at every codehash in diskDb, checking if its an activated stylus contract and
// saving it to wasm store if it doesnt already exists. When errored it logs them and silently returns
//
// It stores the status of rebuilding to wasm store by updating the codehash (of the latest sucessfully checked contract) in
// RebuildingPositionKey every 50 checks.
//
// It also stores a special value that is only set once when rebuilding commenced in RebuildingStartBlockTimeKey as the block
// time of the latest block when rebuilding was first called, this is used to avoid recomputing of assembly and module of
// contracts that were created after rebuilding commenced since they would anyway already be added during sync.
func RebuildWasmStore(ctx context.Context, wasmStore ethdb.KeyValueStore, l2Blockchain *core.BlockChain, key common.Hash, rebuildingStartBlockTime uint64) {
	latestHeader := l2Blockchain.CurrentBlock()
	latestState, err := l2Blockchain.StateAt(latestHeader.Root)
	if err != nil {
		log.Error("error getting state at latest block, aborting rebuilding", "err", err)
		return
	}
	diskDb := latestState.Database().DiskDB()
	arbState, err := arbosState.OpenSystemArbosState(latestState, nil, true)
	if err != nil {
		log.Error("error getting arbos state, aborting rebuilding", "err", err)
		return
	}
	programs := arbState.Programs()
	iter := diskDb.NewIterator(rawdb.CodePrefix, key[:])
	for count := 1; iter.Next(); count++ {
		// If outer context is cancelled we should terminate rebuilding. We probably wont be able to save codehash to wasm store (it might be closed)
		if ctx.Err() != nil {
			return
		}
		codeHashBytes := bytes.TrimPrefix(iter.Key(), rawdb.CodePrefix)
		codeHash := common.BytesToHash(codeHashBytes)
		code := iter.Value()
		if state.IsStylusProgram(code) {
			if err := programs.SaveActiveProgramToWasmStore(latestState, codeHash, code, latestHeader.Time, l2Blockchain.Config().DebugMode(), rebuildingStartBlockTime); err != nil {
				log.Error("error while rebuilding wasm store, aborting rebuilding", "err", err)
				return
			}
		}
		// After every fifty codeHash checks, update the rebuilding position
		// This also notifies user that we are working on rebuilding
		if count%50 == 0 {
			log.Info("Storing rebuilding status to disk", "codeHash", codeHash)
			if err := SetRebuildingParam(wasmStore, RebuildingPositionKey, codeHash); err != nil {
				log.Error("error updating position to wasm store mid way though rebuilding, aborting", "err", err)
				return
			}
		}
	}
	iter.Release()
	// Set rebuilding position to done indicating completion
	if err := SetRebuildingParam(wasmStore, RebuildingPositionKey, RebuildingDone); err != nil {
		log.Error("error updating rebuilding position to done", "err", err)
		return
	}
	log.Info("Rebuilding of wasm store was successful")
}
