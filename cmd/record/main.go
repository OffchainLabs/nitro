package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"math/big"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/offchainlabs/arbstate"
)

type GethBlockRetriever struct {
	db                     ethdb.Database
	minBlockNumberAccessed *uint64
}

func (r *GethBlockRetriever) GetBlockHash(num uint64) common.Hash {
	if num == 0 {
		return common.Hash{}
	}
	if *r.minBlockNumberAccessed > num {
		*r.minBlockNumberAccessed = num
	}
	return rawdb.ReadCanonicalHash(r.db, num)
}

func main() {
	dbPath := flag.String("db-path", "", "The path to the Geth LevelDB database")
	previousBlockHash := flag.String("previous-block-hash", "", "The previous block hash")
	txHash := flag.String("tx-hash", "", "The hash of the transaction being executed")
	inboxOutputPath := flag.String("inbox-output", "", "The path to output any read inbox messages to")
	dbRecordOutputPath := flag.String("db-record-output", "", "The path to output any read database entries to")
	flag.Parse()

	raw, err := rawdb.NewLevelDBDatabaseWithFreezer(*dbPath, 128, 128, filepath.Join(*dbPath, "/ancient"), "", true)
	if err != nil {
		panic(fmt.Sprintf("Error opening DB: %v\n", err))
	}

	tx, _, realTxBlockNumber, _ := rawdb.ReadTransaction(raw, common.HexToHash(*txHash))
	if tx == nil {
		panic("Transaction not present in database")
	}
	signer := types.MakeSigner(params.RinkebyChainConfig, new(big.Int).SetUint64(realTxBlockNumber))
	sender, err := signer.Sender(tx)
	if err != nil {
		panic(fmt.Sprintf("Error getting transaction signer: %v\n", err))
	}
	fmt.Printf("Sender address: %v\n", sender.String())
	msg := arbstate.ArbMessage{
		From:      sender,
		Deposit:   nil,
		Timestamp: 0,
		Tx:        tx,
	}

	lastBlockHash := common.HexToHash(*previousBlockHash)
	lastBlockNumber := rawdb.ReadHeaderNumber(raw, lastBlockHash)
	if lastBlockNumber == nil {
		panic("Previous block not present in database")
	}
	lastHeader := rawdb.ReadHeader(raw, lastBlockHash, *lastBlockNumber)

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

	senderBalance := statedb.GetBalance(sender)
	fmt.Printf("Sender balance: %v\n", senderBalance.String())

	minBlockNumberAccessed := *lastBlockNumber
	blockRetriever := &GethBlockRetriever{db: raw, minBlockNumberAccessed: &minBlockNumberAccessed}
	newBlockHeader, err := arbstate.Process(statedb, lastHeader, blockRetriever, msg)
	var newBlockHash common.Hash
	if err == nil {
		newBlockHash = newBlockHeader.Hash()
	} else {
		fmt.Printf("Error processing message: %v\n", err)
	}
	fmt.Printf("New block hash: %v\n", newBlockHash)

	senderBalance = statedb.GetBalance(sender)
	fmt.Printf("New sender balance: %v\n", senderBalance.String())

	if *inboxOutputPath != "" {
		inboxOutput, err := os.Create(*inboxOutputPath)
		if err != nil {
			panic(fmt.Sprintf("Error creating inbox output file: %v\n", err))
		}
		enc, err := rlp.EncodeToBytes(msg)
		if err != nil {
			panic(fmt.Sprintf("Error RLP encoding ArbMessage: %v\n", err))
		}
		err = binary.Write(inboxOutput, binary.LittleEndian, uint64(len(enc)))
		if err != nil {
			panic(fmt.Sprintf("Error writing to inbox output: %v\n", err))
		}
		_, err = inboxOutput.Write(enc)
		if err != nil {
			panic(fmt.Sprintf("Error writing to inbox output: %v\n", err))
		}
	}

	if *dbRecordOutputPath != "" {
		// Fill in block headers to readDbEntries
		for i := minBlockNumberAccessed; i <= *lastBlockNumber; i++ {
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
		for key, value := range readDbEntries {
			_, err = dbRecordOutput.Write(key.Bytes())
			if err != nil {
				panic(fmt.Sprintf("Error writing to db record output: %v\n", err))
			}
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
