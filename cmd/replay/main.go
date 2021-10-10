package main

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/offchainlabs/arbstate"
	"github.com/offchainlabs/arbstate/arbos"
	"github.com/offchainlabs/arbstate/wavmio"
)

func getBlockHeaderByHash(hash common.Hash) *types.Header {
	enc := wavmio.ResolvePreImage(hash)
	header := &types.Header{}
	err := rlp.DecodeBytes(enc, &header)
	if err != nil {
		panic(fmt.Sprintf("Error parsing resolved block header: %v", err))
	}
	return header
}

type ChainContext struct{}

func (c ChainContext) Engine() consensus.Engine {
	return arbos.Engine{}
}

func (c ChainContext) GetHeader(hash common.Hash, num uint64) *types.Header {
	header := getBlockHeaderByHash(hash)
	if !header.Number.IsUint64() || header.Number.Uint64() != num {
		panic(fmt.Sprintf("Retrieved wrong block number for header hash %v -- requested %v but got %v", hash, num, header.Number.String()))
	}
	return header
}

func main() {
	raw := rawdb.NewDatabase(PreimageDb{})
	db := state.NewDatabase(raw)
	lastBlockHash := wavmio.GetLastBlockHash()

	fmt.Printf("Previous block hash: %v\n", lastBlockHash)
	var lastBlockHeader *types.Header
	var lastBlockStateRoot common.Hash
	if lastBlockHash != (common.Hash{}) {
		lastBlockHeader = getBlockHeaderByHash(lastBlockHash)
		lastBlockStateRoot = lastBlockHeader.Root
	}

	fmt.Printf("Previous block state root: %v\n", lastBlockStateRoot)
	statedb, err := state.New(lastBlockStateRoot, db, nil)
	if err != nil {
		panic(fmt.Sprintf("Error opening state db: %v", err))
	}

	chainContext := ChainContext{}
	blockData := buildBlockData(statedb, lastBlockHeader)
	newBlock, err := arbstate.BuildBlock(statedb, blockData, chainContext)
	if err == nil {
		fmt.Printf("New state root: %v\n", newBlock.Root())
		newBlockHash := newBlock.Hash()
		fmt.Printf("New block hash: %v\n", newBlockHash)

		wavmio.SetLastBlockHash(newBlockHash)
	} else {
		fmt.Printf("Error processing message: %v\n", err)
	}
}

func readSegment() *arbos.MessageSegment {
	inboxMessageBytes := wavmio.ReadInboxMessage()
	var chainId *big.Int // TODO: fill in from state
	inboxMessageSegments, err := arbos.SplitInboxMessage(inboxMessageBytes, chainId)
	if err != nil {
		fmt.Printf("Error splitting inbox message into segments: %v\n", err)
		wavmio.AdvanceInboxMessage()
		return nil
	}

	positionWithinMessage := wavmio.GetPositionWithinMessage()
	if positionWithinMessage+1 >= uint64(len(inboxMessageSegments)) {
		// This is the last segment in the message
		wavmio.AdvanceInboxMessage()
		wavmio.SetPositionWithinMessage(0)
		if positionWithinMessage >= uint64(len(inboxMessageSegments)) {
			// The message is empty
			return nil
		}
	} else {
		// There's remaining segment(s) in this message
		wavmio.SetPositionWithinMessage(positionWithinMessage + 1)
	}
	return inboxMessageSegments[positionWithinMessage]
}

func buildBlockData(statedb *state.StateDB, lastBlockHeader *types.Header) *arbos.BlockData {
	blockBuilder := arbos.NewBlockBuilder(statedb, lastBlockHeader)
	for {
		segment := readSegment()
		if segment == nil {
			continue
		}
		if blockData := blockBuilder.AddSegment(segment); blockData != nil {
			return blockData
		}
	}
}