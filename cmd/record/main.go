package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/offchainlabs/arbstate/arbos"
)

type RecordingChainContext struct {
	db                     ethdb.Database
	minBlockNumberAccessed uint64
}

func (r *RecordingChainContext) Engine() consensus.Engine {
	return arbos.Engine{}
}

func (r *RecordingChainContext) GetHeader(hash common.Hash, num uint64) *types.Header {
	if num == 0 {
		return nil
	}
	if num < r.minBlockNumberAccessed {
		r.minBlockNumberAccessed = num
	}
	return rawdb.ReadHeader(r.db, hash, num)
}

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

	readDbEntries := make(map[common.Hash][]byte)
	db := state.NewDatabase(rawdb.NewDatabase(RecordingDb{
		inner:         raw,
		readDbEntries: readDbEntries,
	}))

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

	chainContext := &RecordingChainContext{db: raw, minBlockNumberAccessed: lastBlockNumber}
	builder := arbos.NewBlockBuilder(statedb, lastHeader, chainContext)
	// TODO add message(s) to builder

	newBlock := builder.ConstructBlock(0)
	newBlockHash := newBlock.Hash()
	fmt.Printf("New block hash: %v\n", newBlockHash)

	if *dbRecordOutputPath != "" {
		// Fill in block headers to readDbEntries
		for i := chainContext.minBlockNumberAccessed; i <= lastBlockNumber; i++ {
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
