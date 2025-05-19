package extractionfunction

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/offchainlabs/nitro/arbcompress"
	"github.com/offchainlabs/nitro/arbnode"
	meltypes "github.com/offchainlabs/nitro/arbnode/message-extraction/types"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbos/l1pricing"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/daprovider"
)

var (
	// ErrInvalidParentChainBlock is returned when the parent chain block
	// hash does not match the expected hash in the state.
	ErrInvalidParentChainBlock = errors.New("invalid parent chain block")
)

// Defines a method that can read a delayed message from an external database.
type DelayedMessageDatabase interface {
	ReadDelayedMessage(
		ctx context.Context,
		state *meltypes.State,
		index uint64,
	) (*arbnode.DelayedInboxMessage, error)
}

// Defines a method that can fetch the receipt for a specific
// transaction index in a parent chain block.
type ReceiptFetcher interface {
	ReceiptForTransactionIndex(
		ctx context.Context,
		txIndex uint,
	) (*types.Receipt, error)
}

// ExtractMessages is a pure function that can read a parent chain block and
// and input MEL state to run a specific algorithm that extracts Arbitrum messages and
// delayed messages observed from transactions in the block. This function can be proven
// through a replay binary, and should also compile to WAVM in addition to running in native mode.
func ExtractMessages(
	ctx context.Context,
	inputState *meltypes.State,
	parentChainBlock *types.Block,
	dataProviders []daprovider.Reader,
	delayedMsgDatabase DelayedMessageDatabase,
	receiptFetcher ReceiptFetcher,
) (*meltypes.State, []*arbostypes.MessageWithMetadata, []*arbnode.DelayedInboxMessage, error) {
	state := inputState.Clone()
	// Clones the state to avoid mutating the input pointer in case of errors.
	// Check parent chain block hash linkage.
	if state.ParentChainBlockHash != parentChainBlock.ParentHash() {
		return nil, nil, nil, fmt.Errorf(
			"%w: expected %s, got %s",
			ErrInvalidParentChainBlock,
			state.ParentChainPreviousBlockHash.Hex(),
			parentChainBlock.ParentHash().Hex(),
		)
	}
	// Updates the fields in the state to corresponding to the
	// incoming parent chain block.
	state.ParentChainBlockHash = parentChainBlock.Hash()
	state.ParentChainBlockNumber = parentChainBlock.NumberU64()
	state.ParentChainPreviousBlockHash = parentChainBlock.ParentHash()
	// Now, check for any logs emitted by the sequencer inbox by txs
	// included in the parent chain block.
	batches, batchTxs, batchTxIndices, err := parseBatchesFromBlock(
		ctx,
		state,
		parentChainBlock,
		receiptFetcher,
	)
	if err != nil {
		return nil, nil, nil, err
	}
	delayedMessages, err := parseDelayedMessagesFromBlock(
		ctx,
		state,
		parentChainBlock,
		receiptFetcher,
	)
	if err != nil {
		return nil, nil, nil, err
	}
	// Update the delayed message accumulator in the MEL state.
	batchPostingReports := make([]*arbnode.DelayedInboxMessage, 0)
	for _, delayed := range delayedMessages {
		// If this message is a batch posting report, we save it for later
		// use in this function once we extract messages from batches and
		// need to fill in their batch posting report.
		if delayed.Message.Header.Kind == arbostypes.L1MessageType_BatchPostingReport {
			batchPostingReports = append(batchPostingReports, delayed)
		}
		state.DelayedMessagedSeen += 1
		state = state.AccumulateDelayedMessage(delayed)
	}

	// Batch posting reports are included in the same transaction as a batch, so there should
	// always be the same number of reports as there are batches.
	if len(batchPostingReports) != len(batches) {
		return nil, nil, nil, fmt.Errorf(
			"batch posting reports %d do not match the number of batches %d",
			len(batchPostingReports),
			len(batches),
		)
	}

	var messages []*arbostypes.MessageWithMetadata
	for i, batch := range batches {
		batchTx := batchTxs[i]
		txIndex := batchTxIndices[i]
		serialized, err := serializeBatch(
			ctx,
			batch,
			parentChainBlock,
			batchTx,
			txIndex,
			receiptFetcher,
		)
		if err != nil {
			return nil, nil, nil, err
		}
		rawSequencerMsg, err := arbstate.ParseSequencerMessage(
			ctx,
			batch.SequenceNumber,
			batch.BlockHash,
			serialized,
			dataProviders,
			daprovider.KeysetValidate,
		)
		if err != nil {
			return nil, nil, nil, err
		}

		messagesInBatch, err := extractMessagesInBatch(
			ctx,
			state,
			rawSequencerMsg,
			delayedMsgDatabase,
		)
		if err != nil {
			return nil, nil, nil, err
		}
		for _, msg := range messagesInBatch {
			report := batchPostingReports[i]
			_, _, batchHash, _, _, _, err := arbostypes.ParseBatchPostingReportMessageFields(bytes.NewReader(report.Message.L2msg))
			if err != nil {
				return nil, nil, nil, fmt.Errorf("failed to parse batch posting report: %w", err)
			}
			gotHash := crypto.Keccak256Hash(serialized)
			if gotHash != batchHash {
				return nil, nil, nil, fmt.Errorf(
					"batch data hash incorrect %v (wanted %v for batch %v)",
					gotHash,
					batchHash,
					batch.SequenceNumber,
				)
			}
			gas := arbostypes.ComputeBatchGasCost(serialized)

			// Fill in the message's batch gas cost from the batch posting report.
			msg.Message.BatchGasCost = &gas

			messages = append(messages, msg)
			state.MsgCount += 1
			state = state.AccumulateMessage(msg)
		}
	}
	return state, messages, delayedMessages, nil
}

// Extracts a list of arbos messages from the sequencer batch, which looks at the
// segments contained in a batch and extracts the correct ordering of messages
// from the segments, possibly reading delayed messages when a delayed message
// virtual segment in encountered.
func extractMessagesInBatch(
	ctx context.Context,
	melState *meltypes.State,
	seqMsg *arbstate.SequencerMessage,
	delayedMsgDB DelayedMessageDatabase,
) ([]*arbostypes.MessageWithMetadata, error) {
	var messages []*arbostypes.MessageWithMetadata
	params := &arbosExtractionParams{
		melState:         melState,
		seqMsg:           seqMsg,
		delayedMsgDB:     delayedMsgDB,
		targetSubMessage: 0,
		segmentNum:       0,
		submessageNumber: 0,
		blockNumber:      0,
		timestamp:        0,
	}
	var msg *arbostypes.MessageWithMetadata
	var err error
	for {
		if isLastSegment(params) {
			break
		}
		msg, params, err = extractArbosMessage(ctx, params)
		if err != nil {
			return nil, err
		}
		if msg == nil {
			msg = &arbostypes.MessageWithMetadata{
				Message:             arbostypes.InvalidL1Message,
				DelayedMessagesRead: params.melState.DelayedMessagesRead,
			}
		}
		messages = append(messages, msg)
		params.targetSubMessage += 1
	}
	return messages, nil
}

func isLastSegment(p *arbosExtractionParams) bool {
	// we issue delayed messages until reaching afterDelayedMessages
	if p.melState.DelayedMessagesRead < p.seqMsg.AfterDelayedMessages {
		return false
	}
	for segmentNum := p.segmentNum + 1; segmentNum < uint64(len(p.seqMsg.Segments)); segmentNum++ {
		segment := p.seqMsg.Segments[segmentNum]
		if len(segment) == 0 {
			continue
		}
		kind := segment[0]
		if kind == arbstate.BatchSegmentKindL2Message || kind == arbstate.BatchSegmentKindL2MessageBrotli {
			return false
		}
		if kind == arbstate.BatchSegmentKindDelayedMessages {
			return false
		}
	}
	return true
}

type arbosExtractionParams struct {
	melState         *meltypes.State
	seqMsg           *arbstate.SequencerMessage
	delayedMsgDB     DelayedMessageDatabase
	targetSubMessage uint64
	segmentNum       uint64
	submessageNumber uint64
	blockNumber      uint64
	timestamp        uint64
}

func extractArbosMessage(
	ctx context.Context,
	p *arbosExtractionParams,
) (*arbostypes.MessageWithMetadata, *arbosExtractionParams, error) {
	seqMsg := p.seqMsg
	segmentNum := p.segmentNum
	timestamp := p.timestamp
	blockNumber := p.blockNumber
	submessageNumber := p.submessageNumber
	var segment []byte
	for {
		if segmentNum >= uint64(len(seqMsg.Segments)) {
			break
		}
		segment = seqMsg.Segments[segmentNum]
		if len(segment) == 0 {
			segmentNum++
			continue
		}
		segmentKind := segment[0]
		if segmentKind == arbstate.BatchSegmentKindAdvanceTimestamp || segmentKind == arbstate.BatchSegmentKindAdvanceL1BlockNumber {
			rd := bytes.NewReader(segment[1:])
			advancing, err := rlp.NewStream(rd, 16).Uint64()
			if err != nil {
				log.Warn("Error parsing sequencer advancing segment", "err", err)
				segmentNum++
				continue
			}
			if segmentKind == arbstate.BatchSegmentKindAdvanceTimestamp {
				timestamp += advancing
			} else if segmentKind == arbstate.BatchSegmentKindAdvanceL1BlockNumber {
				blockNumber += advancing
			}
			segmentNum++
		} else if submessageNumber < p.targetSubMessage {
			segmentNum++
			submessageNumber++
		} else {
			break
		}
	}
	p.segmentNum = segmentNum
	p.timestamp = timestamp
	p.blockNumber = blockNumber
	p.submessageNumber = submessageNumber
	if timestamp < seqMsg.MinTimestamp {
		timestamp = seqMsg.MinTimestamp
	} else if timestamp > seqMsg.MaxTimestamp {
		timestamp = seqMsg.MaxTimestamp
	}
	if blockNumber < seqMsg.MinL1Block {
		blockNumber = seqMsg.MinL1Block
	} else if blockNumber > seqMsg.MaxL1Block {
		blockNumber = seqMsg.MaxL1Block
	}
	if segmentNum >= uint64(len(seqMsg.Segments)) {
		// after end of batch there might be "virtual" delayedMsgSegments
		log.Warn("reading virtual delayed message segment")
		segment = []byte{arbstate.BatchSegmentKindDelayedMessages}
	} else {
		segment = seqMsg.Segments[segmentNum]
	}
	if len(segment) == 0 {
		log.Error("Empty sequencer message segment", "segmentNum", segmentNum)
		return nil, p, nil
	}
	kind := segment[0]
	segment = segment[1:]
	var msg *arbostypes.MessageWithMetadata
	if kind == arbstate.BatchSegmentKindL2Message || kind == arbstate.BatchSegmentKindL2MessageBrotli {
		if kind == arbstate.BatchSegmentKindL2MessageBrotli {
			decompressed, err := arbcompress.Decompress(segment, arbostypes.MaxL2MessageSize)
			if err != nil {
				log.Info("dropping compressed message", "err", err)
				return nil, p, nil
			}
			segment = decompressed
		}

		msg = &arbostypes.MessageWithMetadata{
			Message: &arbostypes.L1IncomingMessage{
				Header: &arbostypes.L1IncomingMessageHeader{
					Kind:        arbostypes.L1MessageType_L2Message,
					Poster:      l1pricing.BatchPosterAddress,
					BlockNumber: blockNumber,
					Timestamp:   timestamp,
					RequestId:   nil,
					L1BaseFee:   big.NewInt(0),
				},
				L2msg: segment,
			},
			DelayedMessagesRead: p.melState.DelayedMessagesRead,
		}
	} else if kind == arbstate.BatchSegmentKindDelayedMessages {
		if p.melState.DelayedMessagesRead >= seqMsg.AfterDelayedMessages {
			if segmentNum < uint64(len(seqMsg.Segments)) {
				log.Warn(
					"Attempt to read past batch delayed message count",
					"delayedMessagesRead", p.melState.DelayedMessagesRead,
					"batchAfterDelayedMessages", seqMsg.AfterDelayedMessages,
				)
			}
			msg = &arbostypes.MessageWithMetadata{
				Message:             arbostypes.InvalidL1Message,
				DelayedMessagesRead: seqMsg.AfterDelayedMessages,
			}
		} else {
			delayed, err := p.delayedMsgDB.ReadDelayedMessage(ctx, p.melState, p.melState.DelayedMessagesRead)
			if err != nil {
				return nil, p, err
			}
			if delayed == nil {
				log.Error("No more delayed messages in queue", "delayedMessagesRead", p.melState.DelayedMessagesRead)
				return nil, p, fmt.Errorf("no more delayed messages in queue")
			}
			p.melState.DelayedMessagesRead += 1
			msg = &arbostypes.MessageWithMetadata{
				Message:             delayed.Message,
				DelayedMessagesRead: p.melState.DelayedMessagesRead,
			}
		}
	} else {
		log.Error("Bad sequencer message segment kind", "segmentNum", segmentNum, "kind", kind)
		return nil, p, nil
	}
	return msg, p, nil
}
