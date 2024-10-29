// Copyright 2021-2024, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package gethexec

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/arbitrum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/offchainlabs/nitro/arbos/arbosState"
)

var RebuildingPositionKey []byte = []byte("_rebuildingPosition")             // contains the codehash upto which rebuilding of wasm store was last completed. Initialized to common.Hash{} at the start
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
// RebuildingPositionKey after every second of work.
//
// It also stores a special value that is only set once when rebuilding commenced in RebuildingStartBlockHashKey as the block
// time of the latest block when rebuilding was first called, this is used to avoid recomputing of assembly and module of
// contracts that were created after rebuilding commenced since they would anyway already be added during sync.
func RebuildWasmStore(ctx context.Context, wasmStore ethdb.KeyValueStore, chainDb ethdb.Database, maxRecreateStateDepth int64, targetConfig *StylusTargetConfig, l2Blockchain *core.BlockChain, position, rebuildingStartBlockHash common.Hash) error {
	var err error
	var stateDb *state.StateDB

	if err := PopulateStylusTargetCache(targetConfig); err != nil {
		return fmt.Errorf("error populating stylus target cache: %w", err)
	}

	latestHeader := l2Blockchain.CurrentBlock()
	// Attempt to get state at the start block when rebuilding commenced, if not available (in case of non-archival nodes) use latest state
	rebuildingStartHeader := l2Blockchain.GetHeaderByHash(rebuildingStartBlockHash)
	stateDb, _, err = arbitrum.StateAndHeaderFromHeader(ctx, chainDb, l2Blockchain, maxRecreateStateDepth, rebuildingStartHeader, nil)
	if err != nil {
		log.Info("Error getting state at start block of rebuilding wasm store, attempting rebuilding with latest state", "err", err)
		stateDb, _, err = arbitrum.StateAndHeaderFromHeader(ctx, chainDb, l2Blockchain, maxRecreateStateDepth, latestHeader, nil)
		if err != nil {
			return fmt.Errorf("error getting state at latest block, aborting rebuilding: %w", err)
		}
	}
	diskDb := stateDb.Database().DiskDB()
	arbState, err := arbosState.OpenSystemArbosState(stateDb, nil, true)
	if err != nil {
		return fmt.Errorf("error getting arbos state, aborting rebuilding: %w", err)
	}
	programs := arbState.Programs()
	iter := diskDb.NewIterator(rawdb.CodePrefix, position[:])
	defer iter.Release()
	lastStatusUpdate := time.Now()
	for iter.Next() {
		codeHashBytes := bytes.TrimPrefix(iter.Key(), rawdb.CodePrefix)
		codeHash := common.BytesToHash(codeHashBytes)
		code := iter.Value()
		if state.IsStylusProgram(code) {
			if err := programs.SaveActiveProgramToWasmStore(stateDb, codeHash, code, latestHeader.Time, l2Blockchain.Config().DebugMode(), rebuildingStartHeader.Time); err != nil {
				return fmt.Errorf("error while rebuilding of wasm store, aborting rebuilding: %w", err)
			}
		}
		// After every one second of work, update the rebuilding position
		// This also notifies user that we are working on rebuilding
		if time.Since(lastStatusUpdate) >= time.Second || ctx.Err() != nil {
			log.Info("Storing rebuilding status to disk", "codeHash", codeHash)
			if err := WriteToKeyValueStore(wasmStore, RebuildingPositionKey, codeHash); err != nil {
				return fmt.Errorf("error updating codehash position in rebuilding of wasm store: %w", err)
			}
			// If outer context is cancelled we should terminate rebuilding
			// We attempted to write the latest checked codeHash to wasm store
			if ctx.Err() != nil {
				return ctx.Err()
			}
			lastStatusUpdate = time.Now()
		}
	}
	// Set rebuilding position to done indicating completion
	if err := WriteToKeyValueStore(wasmStore, RebuildingPositionKey, RebuildingDone); err != nil {
		return fmt.Errorf("error updating codehash position in rebuilding of wasm store to done: %w", err)
	}
	log.Info("Rebuilding of wasm store was successful")
	return nil
}
