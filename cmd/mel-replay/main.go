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
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/offchainlabs/nitro/arbnode/mel"
	"github.com/offchainlabs/nitro/arbnode/melextraction"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/melwavmio"
	"github.com/offchainlabs/nitro/wavmio"
)

type preimageResolver interface {
	ResolveTypedPreimage(preimageType arbutil.PreimageType, hash common.Hash) ([]byte, error)
}

type wavmPreimageResolver struct{}

func (w *wavmPreimageResolver) ResolveTypedPreimage(
	preimageType arbutil.PreimageType, hash common.Hash) ([]byte, error) {
	return melwavmio.ResolveTypedPreimage(preimageType, hash)
}

type BlobPreimageReader struct {
}

func (r *BlobPreimageReader) GetBlobs(
	ctx context.Context,
	batchBlockHash common.Hash,
	versionedHashes []common.Hash,
) ([]kzg4844.Blob, error) {
	var blobs []kzg4844.Blob
	for _, h := range versionedHashes {
		var blob kzg4844.Blob
		preimage, err := wavmio.ResolveTypedPreimage(arbutil.EthVersionedHashPreimageType, h)
		if err != nil {
			return nil, err
		}
		if len(preimage) != len(blob) {
			return nil, fmt.Errorf("for blob %v got back preimage of length %v but expected blob length %v", h, len(preimage), len(blob))
		}
		copy(blob[:], preimage)
		blobs = append(blobs, blob)
	}
	return blobs, nil
}

func (r *BlobPreimageReader) Initialize(ctx context.Context) error {
	return nil
}

// Runs a replay binary of message extraction for Arbitrum chains. Given a start and end parent chain
// block hash, this program will extract all block header hashes in that range, and then run the
// message extraction algorithm over those block headers, starting from a starting MEL state and processing
// block headers one-by-one. At the end, a final MEL state is produced, and its hash is set into the
// machine using a wavmio method.
func main() {
	melwavmio.StubInit()

	glogger := log.NewGlogHandler(
		log.NewTerminalHandler(io.Writer(os.Stderr), false))
	glogger.Verbosity(log.LevelError)
	log.SetDefault(log.NewLogger(glogger))

	wavmio.PopulateEcdsaCaches()

	dapReaders := []daprovider.Reader{daprovider.NewReaderForBlobReader(&BlobPreimageReader{})}

	startMelRoot := melwavmio.GetStartMELRoot()
	endParentChainBlockHash := melwavmio.GetEndParentChainBlockHash()

	// Fetches start MEL state from the start MEL root.
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
			dapReaders,
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
