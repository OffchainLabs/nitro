// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package main

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	meltypes "github.com/offchainlabs/nitro/arbnode/message-extraction/types"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/gethhook"
	melwavmio "github.com/offchainlabs/nitro/wavmio/mel"
)

func main() {
	melwavmio.StubInit()
	gethhook.RequireHookedGeth()

	glogger := log.NewGlogHandler(
		log.NewTerminalHandler(io.Writer(os.Stderr), false))
	glogger.Verbosity(log.LevelError)
	log.SetDefault(log.NewLogger(glogger))
	endParentChainBlockHash := melwavmio.GetEndParentChainBlockHash()
	startMelRoot := melwavmio.GetStartMELRoot()
	startStateBytes, err := melwavmio.ResolveTypedPreimage(
		arbutil.Keccak256PreimageType,
		startMelRoot,
	)
	if err != nil {
		panic(fmt.Errorf("Error resolving preimage: %w", err))
	}
	startState := new(meltypes.State)
	if err := rlp.Decode(bytes.NewBuffer(startStateBytes), &startState); err != nil {
		panic(fmt.Errorf("Error decoding start MEL state: %w", err))
	}

	// Extract the relevant blocks in the range from the
	// block hash of the start MEL state to the end parent chain block hash.
	// This is done by walking backwards from the end parent chain block hash
	// until we reach the block hash of the start MEL state as blocks are
	// only connected by parent linkages.
	blocks := walkBackwards(
		startState.ParentChainBlockHash,
		endParentChainBlockHash,
	)
	currentState := startState
	// Loops backwards over blocks, feeding them one by one into
	// the extract messages function.

	for i := len(blocks) - 1; i >= 0; i-- {
		block := blocks[i]
		log.Info("Extracting messages from block", "number", block.NumberU64(), "hash", block.Hash().Hex())
	}

	// In the end, we set the global state's MEL root to the hash
	// of the post MEL state that is created by running extract
	// messages over the blocks we processed.

	//	ExtractMessages(
	//	ctx context.Context,
	//	inputState *meltypes.State,
	//	parentChainBlock *types.Block,
	//	dataProviders []daprovider.Reader,
	//	delayedMsgDatabase DelayedMessageDatabase,
	//	receiptFetcher ReceiptFetcher,
	//
	// ) (*meltypes.State, []*arbostypes.MessageWithMetadata, []*arbnode.DelayedInboxMessage, error)
	melwavmio.SetMELStateHash(currentState.Hash())
	melwavmio.StubFinal()
}

// TODO: Define a max lookback?
func walkBackwards(
	startHash,
	endHash common.Hash,
) []*types.Block {
	blocks := make([]*types.Block, 0)
	curr := endHash
	for {
		block := getBlockByHash(curr)
		blocks = append(blocks, block)
		curr = block.ParentHash()
		if curr == startHash {
			break
		}
	}
	return blocks
}

func getBlockByHash(hash common.Hash) *types.Block {
	enc, err := melwavmio.ResolveTypedPreimage(arbutil.Keccak256PreimageType, hash)
	if err != nil {
		panic(fmt.Errorf("Error resolving preimage: %w", err))
	}
	block := &types.Block{}
	err = rlp.DecodeBytes(enc, &block)
	if err != nil {
		panic(fmt.Errorf("Error parsing resolved block: %w", err))
	}
	return block
}
