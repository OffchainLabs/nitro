// Copyright 2026-2027, Offchain Labs, Inc.
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
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/offchainlabs/nitro/arbnode/mel"
	melextraction "github.com/offchainlabs/nitro/arbnode/mel/extraction"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/daprovider"
	melreplay "github.com/offchainlabs/nitro/mel-replay"
	"github.com/offchainlabs/nitro/melwavmio"
	"github.com/offchainlabs/nitro/wavmio"
)

func main() {
	melwavmio.StubInit()

	glogger := log.NewGlogHandler(
		log.NewTerminalHandler(io.Writer(os.Stderr), false))
	glogger.Verbosity(log.LevelError)
	log.SetDefault(log.NewLogger(glogger))

	wavmio.PopulateEcdsaCaches()

	melMsgHash := melwavmio.GetMELMsgHash()
	startMELStateHash := melwavmio.GetStartMELRoot()
	melState := readMELState(startMELStateHash)

	if melMsgHash != (common.Hash{}) {
		msgBytes := readPreimage(melMsgHash)
		var currentBlock *types.Block
		nextBlock := produceBlock(currentBlock, msgBytes)
		melwavmio.IncreasePositionInMEL()
		wavmio.SetLastBlockHash(nextBlock.Hash())
	} else {
		targetBlockHash := melwavmio.GetEndParentChainBlockHash()
		// TODO: Read the real chain config.
		melState = extractMessagesUpTo(nil /* nil chain config */, melState, targetBlockHash)
	}

	positionInMEL := melwavmio.GetPositionInMEL()
	if melState.MsgCount > positionInMEL {
		nextMsg, err := melState.ReadMessage(positionInMEL)
		if err != nil {
			panic(fmt.Errorf("error reading message idx %d: %w", positionInMEL, err))
		}
		msgHash, err := mel.MessageHash(nextMsg)
		if err != nil {
			panic(fmt.Errorf("error hashing message idx %d: %w", positionInMEL, err))
		}
		melwavmio.SetMELMsgHash(msgHash)
	} else {
		melwavmio.SetMELMsgHash(common.Hash{})
	}
}

func produceBlock(currentBlock *types.Block, msg []byte) *types.Block {
	// TODO: Implement.
	return nil
}

// Runs a replay binary of message extraction for Arbitrum chains. Given a start and end parent chain
// block hash, this program will extract all block header hashes in that range, and then run the
// message extraction algorithm over those block headers, starting from a starting MEL state and processing
// block headers one-by-one. At the end, a final MEL state is produced, and its hash is set into the
// machine using a wavmio method.
func extractMessagesUpTo(
	chainConfig *params.ChainConfig,
	startState *mel.State,
	targetBlockHash common.Hash,
) *mel.State {
	resolver := &wavmPreimageResolver{}
	dapReader := daprovider.NewDAProviderRegistry()
	blobReader := &BlobPreimageReader{}
	if err := dapReader.SetupBlobReader(daprovider.NewReaderForBlobReader(blobReader)); err != nil {
		panic(fmt.Errorf("error setting up blob reader: %w", err))
	}

	// Extract the relevant header hashes in the range from the
	// block hash of the start MEL state to the end parent chain block hash.
	// This is done by walking backwards from the end parent chain block hash
	// until we reach the block hash of the start MEL state as blocks are
	// only connected by parent linkages.
	blockHeaderHashes := walkBackwards(
		startState.ParentChainBlockHash,
		targetBlockHash,
	)
	currentState := startState

	// Loops backwards over blocks, feeding them one by one into the extract messages function.
	delayedMsgDatabase := melreplay.NewDelayedMessageDatabase(resolver)
	ctx := context.Background()
	for i := len(blockHeaderHashes) - 1; i >= 0; i-- {
		headerHash := blockHeaderHashes[i]
		header := getHeaderByHash(headerHash)
		log.Info("Extracting messages from block", "number", header.Number.Uint64(), "hash", header.Hash().Hex())
		txsFetcher := melreplay.NewTransactionFetcher(header, resolver)
		logsFetcher := melreplay.NewLogsFetcher(header, resolver)
		postState, _, _, _, err := melextraction.ExtractMessages(
			ctx,
			currentState,
			header,
			dapReader,
			delayedMsgDatabase,
			txsFetcher,
			logsFetcher,
			chainConfig,
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
	return currentState
}

// Extracts all block header hashes in the range from startHash to endHash.
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

// Gets a block header by its hash using the preimage resolver.
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

func readMELState(hash common.Hash) *mel.State {
	startStateBytes := readPreimage(hash)
	state := new(mel.State)
	if err := rlp.Decode(bytes.NewBuffer(startStateBytes), &state); err != nil {
		panic(fmt.Errorf("error decoding MEL state: %w", err))
	}
	return state
}

func readPreimage(hash common.Hash) []byte {
	preimage, err := melwavmio.ResolveTypedPreimage(arbutil.Keccak256PreimageType, hash)
	if err != nil {
		panic(fmt.Errorf("error resolving preimage: %w", err))
	}
	return preimage
}
