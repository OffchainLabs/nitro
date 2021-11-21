//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/offchainlabs/arbstate/arbos"
	"github.com/offchainlabs/arbstate/arbstate"
)

func main() {
	dbPath := flag.String("db-path", "", "The path to the Geth LevelDB database")
	previousBlockHash := flag.String("previous-block-hash", "", "The previous block hash")
	dbRecordOutputPath := flag.String("db-record-output", "", "The path to output any read database entries to")
	flag.Parse()

	raw, err := rawdb.NewLevelDBDatabaseWithFreezer(*dbPath, 128, 128, filepath.Join(*dbPath, "/ancient"), "", true)
	if err != nil {
		panic(fmt.Sprintf("Error opening DB: %v\n", err))
	}

	lastBlockHash := common.HexToHash(*previousBlockHash)
	var lastBlockNumber uint64
	var lastHeader *types.Header
	if lastBlockHash != (common.Hash{}) {
		lastBlockNumber = *rawdb.ReadHeaderNumber(raw, lastBlockHash)
		lastHeader = rawdb.ReadHeader(raw, lastBlockHash, lastBlockNumber)
	}

	recordingDb := arbstate.NewRecordingDb(raw)
	db := state.NewDatabase(rawdb.NewDatabase(recordingDb))

	var lastBlockRoot common.Hash
	if lastHeader != nil {
		lastBlockRoot = lastHeader.Root
	}

	fmt.Printf("Previous state root: %v\n", lastBlockRoot)
	fmt.Printf("Previous block hash: %v\n", lastBlockHash)
	statedb, err := state.New(lastBlockRoot, db, nil)
	if err != nil {
		panic(fmt.Sprintf("Error opening state db: %v", err))
	}

	chainContext := arbstate.NewRecordingChainContext(raw, lastBlockNumber)
	builder := arbos.NewBlockBuilder(statedb, lastHeader, chainContext)
	// TODO add message(s) to builder

	newBlock, _, _ := builder.ConstructBlock(0)
	newBlockHash := newBlock.Hash()
	fmt.Printf("New block hash: %v\n", newBlockHash)

	readDbEntries := recordingDb.GetRecordedEntries()
	if *dbRecordOutputPath != "" {
		// Fill in block headers to readDbEntries
		for i := chainContext.GetMinBlockNumberAccessed(); i <= lastBlockNumber; i++ {
			hash := rawdb.ReadCanonicalHash(raw, i)
			header := rawdb.ReadHeader(raw, hash, i)
			bytes, err := rlp.EncodeToBytes(header)
			if err != nil {
				panic(fmt.Sprintf("Error RLP encoding header: %v\n", err))
			}
			readDbEntries[hash] = bytes
		}

		dbRecordOutput, err := os.Create(*dbRecordOutputPath)
		if err != nil {
			panic(fmt.Sprintf("Error creating db record output file: %v\n", err))
		}
		for _, value := range readDbEntries {
			err = binary.Write(dbRecordOutput, binary.LittleEndian, uint64(len(value)))
			if err != nil {
				panic(fmt.Sprintf("Error writing to db record output: %v\n", err))
			}
			_, err = dbRecordOutput.Write(value)
			if err != nil {
				panic(fmt.Sprintf("Error writing to db record output: %v\n", err))
			}
		}
	}
}
