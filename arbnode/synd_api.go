// Package arbnode provides the SyndAPI for Arbitrum Nitro node operations.
// The SyndAPI handles validation data retrieval, batch metadata operations,
// and preimage data collection for the Arbitrum network.

package arbnode

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"math/big"
	"slices"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/execution/gethexec"
)

// SyndAPI provides methods for retrieving validation data and batch information
// from the Arbitrum Nitro node. It handles the interaction between the block
// recorder and transaction streamer components.
type SyndAPI struct {
	recorder      *gethexec.BlockRecorder
	inboxStreamer *TransactionStreamer
}

// ValidationData contains all the information needed to validate a range of
// messages in the Arbitrum network. It includes batch metadata, delayed messages,
// and preimage data required for verification.
type ValidationData struct {
	// Note: Batch*BlockNum refer to parent-chain block numbers (L1), while startBlock parameter is a message index.
	// The produced message range is (startBlock, endBlock] i.e. exclusive of startBlock.
	BatchStartBlockNum uint64
	BatchEndBlockNum   uint64
	BatchStartIndex    uint64
	BatchEndIndex      uint64
	DelayedMessages    [][]byte
	StartDelayedAcc    common.Hash
	PreimageData       [][]byte
}

// Constants for validation and error checking
const (
	// Minimum size for delayed message entries (32 bytes for hash + message data)
	MinDelayedMessageSize = 32
)

// BatchMetadata returns the metadata for the given batch number.
func (a *SyndAPI) BatchMetadata(batch uint64) (BatchMetadata, error) {
	return a.inboxStreamer.inboxReader.Tracker().GetBatchMetadata(batch)
}

// BatchFromAcc returns the batch number for the given accumulator.
// It performs a linear search through all batches to find the one with
// the matching accumulator hash. Returns an error if no matching batch is found.
func (a *SyndAPI) BatchFromAcc(ctx context.Context, acc common.Hash) (uint64, error) {
	inboxTracker := a.inboxStreamer.inboxReader.Tracker()
	count, err := inboxTracker.GetBatchCount()
	if err != nil {
		return 0, fmt.Errorf("failed to get batch count: %w", err)
	}
	if count == 0 {
		return 0, errors.New("no batches found in inbox tracker")
	}

	// Linear search through batches
	for count > 0 {
		count--
		if err := ctx.Err(); err != nil {
			return 0, fmt.Errorf("context cancelled during batch search: %w", err)
		}
		batchAcc, err := inboxTracker.GetBatchAcc(count)
		if err != nil {
			return 0, fmt.Errorf("failed to get batch accumulator for batch %d: %w", count, err)
		}
		if batchAcc == acc {
			return count, nil
		}
	}
	return 0, fmt.Errorf("accumulator %s not found in any batch", acc)
}

// getDelayedMessage retrieves and validates a delayed message for the given sequence number.
// It performs RLP decoding and validates the message structure and request ID.
func (a *SyndAPI) getDelayedMessage(seqNum uint64) ([]byte, error) {
	data, err := a.inboxStreamer.inboxReader.Tracker().db.Get(dbKey(rlpDelayedMessagePrefix, seqNum))
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve delayed message data for sequence %d: %w", seqNum, err)
	}
	if len(data) < MinDelayedMessageSize {
		return nil, fmt.Errorf("delayed message entry too short: got %d bytes, expected at least %d", len(data), MinDelayedMessageSize)
	}

	var msg *arbostypes.L1IncomingMessage
	if err := rlp.DecodeBytes(data[MinDelayedMessageSize:], &msg); err != nil {
		return nil, fmt.Errorf("failed to RLP decode delayed message for sequence %d: %w", seqNum, err)
	}

	// Validate that the message request ID matches the expected sequence number
	expectedSeqNum := new(big.Int).SetUint64(seqNum)
	if msg.Header.RequestId.Big().Cmp(expectedSeqNum) != 0 {
		return nil, fmt.Errorf("request ID mismatch: got %s, expected %d", msg.Header.RequestId, seqNum)
	}

	return msg.Serialize()
}

// ValidationData returns the validation data for the given range of messages.
// This method constructs a complete validation dataset including:
// - Batch metadata (start/end block numbers and indices)
// - Delayed messages within the range
// - Preimage data required for verification
//
// Parameters:
//   - startBlock: The starting message index (0-based)
//   - endBatch: The ending batch number (can be relative or absolute)
//   - isRelative: If true, endBatch is relative to the start batch
//
// Returns a ValidationData struct containing all necessary validation information.
func (a *SyndAPI) ValidationData(ctx context.Context, startBlock uint64, endBatch uint64, isRelative bool) (*ValidationData, error) {
	inboxTracker := a.inboxStreamer.inboxReader.Tracker()

	// Find the batch containing the start message
	startBatch, found, err := inboxTracker.FindInboxBatchContainingMessage(arbutil.MessageIndex(startBlock))
	if err != nil {
		return nil, fmt.Errorf("failed to find batch containing start message %d: %w", startBlock, err)
	}
	if !found {
		return nil, fmt.Errorf("start batch for message %d not found", startBlock)
	}

	// Calculate absolute end batch if relative
	if isRelative {
		endBatch += startBatch
	}

	// Validate batch range
	if startBatch > endBatch {
		return nil, fmt.Errorf("invalid batch range: start batch %d is after end batch %d", startBatch, endBatch)
	}

	// Get metadata for the start batch
	metadata, err := inboxTracker.GetBatchMetadata(startBatch)
	if err != nil {
		return nil, fmt.Errorf("failed to get metadata for start batch %d: %w", startBatch, err)
	}
	startCount := metadata.DelayedMessageCount

	// Get the delayed accumulator for the start batch
	var startDelayedAcc common.Hash
	if startCount > 0 {
		startDelayedAcc, err = inboxTracker.GetDelayedAcc(startCount - 1)
		if err != nil {
			return nil, fmt.Errorf("failed to get delayed accumulator for count %d: %w", startCount-1, err)
		}
	}

	batchStartBlockNum := metadata.ParentChainBlock

	// Handle single batch case (start and end are the same)
	if startBatch == endBatch {
		return &ValidationData{
			BatchStartBlockNum: batchStartBlockNum + 1,
			BatchEndBlockNum:   batchStartBlockNum,
			BatchStartIndex:    startBatch + 1,
			BatchEndIndex:      endBatch,
			DelayedMessages:    nil,
			StartDelayedAcc:    startDelayedAcc,
			PreimageData:       nil,
		}, nil
	}

	// Get metadata for the next batch to determine start block number
	metadata, err = inboxTracker.GetBatchMetadata(startBatch + 1)
	if err != nil {
		return nil, fmt.Errorf("failed to get metadata for batch %d: %w", startBatch+1, err)
	}
	batchStartBlockNum = metadata.ParentChainBlock

	// Calculate end block and validate message range
	endBlock, err := inboxTracker.GetBatchMessageCount(endBatch)
	if err != nil {
		return nil, fmt.Errorf("failed to get message count for end batch %d: %w", endBatch, err)
	}
	endBlock -= 1

	batchStartBlock, err := inboxTracker.GetBatchMessageCount(startBatch)
	if err != nil {
		return nil, fmt.Errorf("failed to get message count for start batch %d: %w", startBatch, err)
	}
	if startBlock+1 != uint64(batchStartBlock) {
		return nil, fmt.Errorf("batch %d starts at message %d, not %d", startBatch, uint64(batchStartBlock), startBlock+1)
	}

	// Get metadata for the end batch
	metadata, err = inboxTracker.GetBatchMetadata(endBatch)
	if err != nil {
		return nil, fmt.Errorf("failed to get metadata for end batch %d: %w", endBatch, err)
	}
	endCount := metadata.DelayedMessageCount

	// Validate delayed message count consistency
	if startCount > endCount {
		return nil, fmt.Errorf("inconsistent delayed message counts: start count %d > end count %d", startCount, endCount)
	}

	// Collect delayed messages in the range
	var messages [][]byte
	for i := startCount; i < endCount; i++ {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("context cancelled while collecting delayed messages: %w", err)
		}
		msg, err := a.getDelayedMessage(i)
		if err != nil {
			return nil, fmt.Errorf("failed to get delayed message %d: %w", i, err)
		}
		messages = append(messages, msg)
	}

	// Get preimage data for the message range
	preimageData, err := a.preimageData(ctx, arbutil.MessageIndex(startBlock), endBlock, startBatch, endBatch)
	if err != nil {
		return nil, fmt.Errorf("failed to get preimage data: %w", err)
	}

	return &ValidationData{
		BatchStartBlockNum: batchStartBlockNum,
		BatchEndBlockNum:   metadata.ParentChainBlock,
		BatchStartIndex:    startBatch + 1,
		BatchEndIndex:      endBatch,
		DelayedMessages:    messages,
		StartDelayedAcc:    startDelayedAcc,
		PreimageData:       preimageData,
	}, nil
}

// PreimageData returns the preimage data for the given range of messages.
// This method collects all preimage data required for validating messages
// in the specified range, including batch data and message preimages.
//
// Parameters:
//   - startBlock: The starting message index (0-based)
//   - endBatch: The ending batch number (can be relative or absolute)
//   - isRelative: If true, endBatch is relative to the start batch
//
// Returns a slice of preimage data bytes required for validation.
func (a *SyndAPI) PreimageData(ctx context.Context, startBlock uint64, endBatch uint64, isRelative bool) ([][]byte, error) {
	inboxTracker := a.inboxStreamer.inboxReader.Tracker()

	// Find the batch containing the start message
	startBatch, found, err := inboxTracker.FindInboxBatchContainingMessage(arbutil.MessageIndex(startBlock))
	if err != nil {
		return nil, fmt.Errorf("failed to find batch containing start message %d: %w", startBlock, err)
	}
	if !found {
		return nil, fmt.Errorf("start batch for message %d not found", startBlock)
	}

	// Calculate absolute end batch if relative
	if isRelative {
		endBatch += startBatch
	}

	// Validate batch range
	if startBatch > endBatch {
		return nil, fmt.Errorf("invalid batch range: start batch %d is after end batch %d", startBatch, endBatch)
	}

	// Return empty result for single batch case
	if startBatch == endBatch {
		return nil, nil
	}

	// Calculate end block
	endBlock, err := inboxTracker.GetBatchMessageCount(endBatch)
	if err != nil {
		return nil, fmt.Errorf("failed to get message count for end batch %d: %w", endBatch, err)
	}
	endBlock -= 1

	// Validate start block alignment
	batchStartBlock, err := inboxTracker.GetBatchMessageCount(startBatch)
	if err != nil {
		return nil, fmt.Errorf("failed to get message count for start batch %d: %w", startBatch, err)
	}
	if startBlock+1 != uint64(batchStartBlock) {
		return nil, fmt.Errorf("batch %d starts at message %d, not %d", startBatch, uint64(batchStartBlock), startBlock+1)
	}

	return a.preimageData(ctx, arbutil.MessageIndex(startBlock), endBlock, startBatch, endBatch)
}

// preimageData collects all preimage data required for validating messages in the specified range.
// This includes both batch data from past batches and message preimages generated during block recording.
//
// The method performs the following steps:
// 1. Collects messages in the range and identifies required past batches
// 2. Fetches batch data for required past batches
// 3. Records blocks to generate message preimages
// 4. Combines all preimage data into a single result
func (a *SyndAPI) preimageData(ctx context.Context, startBlock arbutil.MessageIndex, endBlock arbutil.MessageIndex, startBatch uint64, endBatch uint64) ([][]byte, error) {
	// Map to store preimage data with hash as key to avoid duplicates
	m := make(map[common.Hash][]byte)
	var msgs []*arbostypes.MessageWithMetadata

	// Process each message in the range
	for i := startBlock + 1; i <= endBlock; i++ {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("context cancelled while processing message %d: %w", i, err)
		}

		// Get the message
		msg, err := a.inboxStreamer.GetMessage(i)
		if err != nil {
			return nil, fmt.Errorf("failed to get message %d: %w", i, err)
		}

		// Get required past batches for this message
		prevBatchNums, err := msg.Message.PastBatchesRequired()
		if err != nil {
			return nil, fmt.Errorf("failed to get past batches required for message %d: %w", i, err)
		}

		// Fetch batch data for required past batches
		for _, batchNum := range prevBatchNums {
			// Validate batch number is within expected range
			if batchNum > endBatch {
				return nil, fmt.Errorf("required batch %d is after endBatch %d (range %d..%d)", batchNum, endBatch, startBatch, endBatch)
			}

			// Only fetch batches that are before or equal to startBatch
			if batchNum <= startBatch {
				// Fetch sequencer message bytes for the batch (makes eth_getLogs request)
				batch, _, err := a.inboxStreamer.inboxReader.GetSequencerMessageBytes(ctx, batchNum)
				if err != nil {
					return nil, fmt.Errorf("failed to get sequencer message bytes for batch %d: %w", batchNum, err)
				}
				// Store batch data with its hash as key
				m[crypto.Keccak256Hash(batch)] = batch
			}
		}
		msgs = append(msgs, msg)
	}

	// Record blocks to generate message preimages (no network requests)
	recording, err := a.recorder.RecordBlocks(ctx, startBlock, msgs)
	if err != nil {
		return nil, fmt.Errorf("failed to record blocks from %d to %d: %w", startBlock, endBlock, err)
	}

	// Validate recording position
	if recording.Pos != endBlock {
		return nil, fmt.Errorf("unexpected recording position: got %d, expected %d", recording.Pos, endBlock)
	}

	// Merge preimage data from recording with batch data
	maps.Copy(recording.Preimages, m)

	// Convert map values to slice and return
	return slices.Collect(maps.Values(recording.Preimages)), nil
}
