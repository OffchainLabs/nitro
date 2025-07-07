// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/offchainlabs/nitro/arbnode/mel"
	melextraction "github.com/offchainlabs/nitro/arbnode/mel/extraction"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/gethhook"
	melwavmio "github.com/offchainlabs/nitro/wavmio/mel"
)

type preimageResolver interface {
	ResolveTypedPreimage(preimageType arbutil.PreimageType, hash common.Hash) ([]byte, error)
}

type wavmPreimageResolver struct{}

func (w *wavmPreimageResolver) ResolveTypedPreimage(
	preimageType arbutil.PreimageType, hash common.Hash) ([]byte, error) {
	return melwavmio.ResolveTypedPreimage(preimageType, hash)
}

func main() {
	melwavmio.StubInit()
	gethhook.RequireHookedGeth()

	glogger := log.NewGlogHandler(
		log.NewTerminalHandler(io.Writer(os.Stderr), false))
	glogger.Verbosity(log.LevelError)
	log.SetDefault(log.NewLogger(glogger))

	startMelRoot := melwavmio.GetStartMELRoot()
	endParentChainBlockHash := melwavmio.GetEndParentChainBlockHash()

	startStateBytes, err := melwavmio.ResolveTypedPreimage(
		arbutil.Keccak256PreimageType,
		startMelRoot,
	)
	if err != nil {
		panic(fmt.Errorf("error resolving preimage: %w", err))
	}
	startState := new(mel.State)
	if err := rlp.Decode(bytes.NewBuffer(startStateBytes), &startState); err != nil {
		panic(fmt.Errorf("error decoding start MEL state: %w", err))
	}

	// Extract the relevant header hashes in the range from the
	// block hash of the start MEL state to the end parent chain block hash.
	// This is done by walking backwards from the end parent chain block hash
	// until we reach the block hash of the start MEL state as blocks are
	// only connected by parent linkages.
	blockHeaderHashes := walkBackwards(
		startState.ParentChainBlockHash,
		endParentChainBlockHash,
	)
	currentState := startState

	// Loops backwards over blocks, feeding them one by one into the extract messages function.
	resolver := &wavmPreimageResolver{}
	delayedMsgDatabase := &delayedMessageDatabase{
		preimageResolver: resolver,
	}
	ctx := context.Background()
	for i := len(blockHeaderHashes) - 1; i >= 0; i-- {
		headerHash := blockHeaderHashes[i]
		header := getHeaderByHash(headerHash)
		log.Info("Extracting messages from block", "number", header.Number.Uint64(), "hash", header.Hash().Hex())
		receiptFetcher := &receiptFetcherForBlock{
			header:           header,
			preimageResolver: resolver,
		}
		txsFetcher := &txsFetcherForBlock{
			header:           header,
			preimageResolver: resolver,
		}
		postState, _, _, err := melextraction.ExtractMessages(
			ctx,
			currentState,
			header,
			nil, // TODO: Provide da readers here.
			delayedMsgDatabase,
			receiptFetcher,
			txsFetcher,
		)
		if err != nil {
			panic(fmt.Errorf("error extracting messages from block %s: %w", header.Hash().Hex(), err))
		}
		currentState = postState
	}

	// In the end, we set the global state's MEL root to the hash of the post MEL state
	// that is created by running extract messages over the blocks we processed.
	encodedFinalState, err := rlp.EncodeToBytes(currentState)
	if err != nil {
		panic(fmt.Errorf("error encoding final MEL state: %w", err))
	}
	computedEndMelRoot := crypto.Keccak256Hash(encodedFinalState)
	melwavmio.SetEndMELRoot(computedEndMelRoot)
	melwavmio.StubFinal()
}

func walkBackwards(
	startHash,
	endHash common.Hash,
) []common.Hash {
	headerHashes := make([]common.Hash, 0)
	curr := endHash
	for {
		header := getHeaderByHash(curr)
		headerHashes = append(headerHashes, curr)
		curr = header.ParentHash
		if curr == startHash {
			break
		}
	}
	return headerHashes
}

func getHeaderByHash(hash common.Hash) *types.Header {
	enc, err := melwavmio.ResolveTypedPreimage(arbutil.Keccak256PreimageType, hash)
	if err != nil {
		panic(fmt.Errorf("error resolving preimage: %w", err))
	}
	header := &types.Header{}
	err = rlp.DecodeBytes(enc, &header)
	if err != nil {
		panic(fmt.Errorf("error parsing resolved block header: %w", err))
	}
	return header
}
