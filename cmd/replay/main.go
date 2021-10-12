package main

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
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
	newBlock := buildBlock(statedb, lastBlockHeader, chainContext)
	if err == nil {
		fmt.Printf("New state root: %v\n", newBlock.Root())
		newBlockHash := newBlock.Hash()
		fmt.Printf("New block hash: %v\n", newBlockHash)

		wavmio.SetLastBlockHash(newBlockHash)
	} else {
		fmt.Printf("Error processing message: %v\n", err)
	}
}

var chainId = big.NewInt(0xA4B12) // TODO

func buildBlock(statedb *state.StateDB, lastBlockHeader *types.Header, chainContext core.ChainContext) *types.Block {
	var delayedMessagesRead uint64
	if lastBlockHeader != nil {
		delayedMessagesRead = lastBlockHeader.Nonce.Uint64()
	}
	inboxReader := NewInboxReader(delayedMessagesRead)
	blockBuilder := arbos.NewBlockBuilder(statedb, lastBlockHeader, chainContext)
	for {
		message, shouldEndBlock, err := inboxReader.Peek()
		if err != nil {
			log.Warn("error parsing inbox message: %v", err)
			break
		}
		segments, err := arbos.ExtractL1MessageSegments(message, chainId)
		if !blockBuilder.ShouldAddMessage(segments) {
			break
		}
		inboxReader.Advance()
		blockBuilder.AddMessage(segments)
		if shouldEndBlock {
			break
		}
	}
	return blockBuilder.ConstructBlock(inboxReader.DelayedMessagesRead())
}
