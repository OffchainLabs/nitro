// Copyright 2021-2024, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package gethexec

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
var RebuildingStartBlockHashKey []byte = []byte("_rebuildingStartBlockHash") // contains the block hash of starting block when rebuilding of wasm store first began
var RebuildingDone common.Hash = common.BytesToHash([]byte("_done"))         // indicates that the rebuilding is done, if RebuildingPositionKey holds this value it implies rebuilding was completed

func ReadFromKeyValueStore[T any](store ethdb.KeyValueStore, key []byte) (T, error) {
	var empty T
	posBytes, err := store.Get(key)
	if err != nil {
		return empty, err
	}
	var val T
	err = rlp.DecodeBytes(posBytes, &val)
	if err != nil {
		return empty, fmt.Errorf("error decoding value stored for key in the KeyValueStore: %w", err)
	}
	return val, nil
}

func WriteToKeyValueStore[T any](store ethdb.KeyValueStore, key []byte, val T) error {
	valBytes, err := rlp.EncodeToBytes(val)
	if err != nil {
		return err
	}
	err = store.Put(key, valBytes)
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
// It also stores a special value that is only set once when rebuilding commenced in RebuildingStartBlockHashKey as the block
// time of the latest block when rebuilding was first called, this is used to avoid recomputing of assembly and module of
// contracts that were created after rebuilding commenced since they would anyway already be added during sync.
func RebuildWasmStore(ctx context.Context, wasmStore ethdb.KeyValueStore, l2Blockchain *core.BlockChain, position, rebuildingStartBlockHash common.Hash) {
	var err error
	var stateDb *state.StateDB
	latestHeader := l2Blockchain.CurrentBlock()
	// Attempt to get state at the start block when rebuilding commenced, if not available (in case of non-archival nodes) use latest state
	rebuildingStartHeader := l2Blockchain.GetHeaderByHash(rebuildingStartBlockHash)
	stateDb, err = l2Blockchain.StateAt(rebuildingStartHeader.Root)
	if err != nil {
		log.Info("error getting state at start block of rebuilding wasm store, attempting rebuilding with latest state", "err", err)
		stateDb, err = l2Blockchain.StateAt(latestHeader.Root)
		if err != nil {
			log.Error("error getting state at latest block, aborting rebuilding", "err", err)
			return
		}
	}
	diskDb := stateDb.Database().DiskDB()
	arbState, err := arbosState.OpenSystemArbosState(stateDb, nil, true)
	if err != nil {
		log.Error("error getting arbos state, aborting rebuilding", "err", err)
		return
	}
	programs := arbState.Programs()
	iter := diskDb.NewIterator(rawdb.CodePrefix, position[:])
	for count := 1; iter.Next(); count++ {
		codeHashBytes := bytes.TrimPrefix(iter.Key(), rawdb.CodePrefix)
		codeHash := common.BytesToHash(codeHashBytes)
		code := iter.Value()
		if state.IsStylusProgram(code) {
			if err := programs.SaveActiveProgramToWasmStore(stateDb, codeHash, code, latestHeader.Time, l2Blockchain.Config().DebugMode(), rebuildingStartHeader.Time); err != nil {
				log.Error("error while rebuilding wasm store, aborting rebuilding", "err", err)
				return
			}
		}
		// After every fifty codeHash checks, update the rebuilding position
		// This also notifies user that we are working on rebuilding
		if count%50 == 0 || ctx.Err() != nil {
			log.Info("Storing rebuilding status to disk", "codeHash", codeHash)
			if err := WriteToKeyValueStore(wasmStore, RebuildingPositionKey, codeHash); err != nil {
				log.Error("error updating position to wasm store mid way though rebuilding, aborting", "err", err)
				return
			}
			// If outer context is cancelled we should terminate rebuilding
			// We attempted to write the latest checked codeHash to wasm store
			if ctx.Err() != nil {
				return
			}
		}
	}
	iter.Release()
	// Set rebuilding position to done indicating completion
	if err := WriteToKeyValueStore(wasmStore, RebuildingPositionKey, RebuildingDone); err != nil {
		log.Error("error updating rebuilding position to done", "err", err)
		return
	}
	log.Info("Rebuilding of wasm store was successful")
}
