package mel

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/offchainlabs/nitro/arbcompress"
	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbos/l1pricing"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/arbstate/daprovider"
	"github.com/offchainlabs/nitro/arbutil"
)

var (
	// ErrInvalidParentChainBlock is returned when the parent chain block
	// hash does not match the expected hash in the state.
	ErrInvalidParentChainBlock = errors.New("invalid parent chain block")
)

func (m *MessageExtractor) extractMessages(
	ctx context.Context,
	inputState *State,
	parentChainBlock *types.Block,
) (*State, []*arbostypes.MessageWithMetadata, []*arbnode.DelayedInboxMessage, error) {
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
	blockNum := parentChainBlock.Number()
	batches, err := m.sequencerInbox.LookupBatchesInRange(
		ctx,
		blockNum,
		blockNum,
	)
	if err != nil {
		return nil, nil, nil, err
	}
	delayedMessages, err := m.delayedBridge.LookupMessagesInRange(
		ctx,
		blockNum,
		blockNum,
		func(batchNum uint64) ([]byte, error) {
			if len(batches) > 0 && batchNum >= batches[0].SequenceNumber {
				idx := batchNum - batches[0].SequenceNumber
				if idx < uint64(len(batches)) {
					return batches[idx].Serialize(ctx, m.l1Reader.Client())
				}
				return nil, fmt.Errorf("missing batch %d", batchNum)
			}
			return nil, fmt.Errorf("batch %d not found", batchNum)
		})
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
		// TODO: Create a unique, delayed message hash beyond just
		// the underlying message's hash.
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
		serialized, err := batch.Serialize(ctx, m.l1Reader.Client())
		if err != nil {
			return nil, nil, nil, err
		}
		rawSequencerMsg, err := arbstate.ParseSequencerMessage(
			ctx,
			batch.SequenceNumber,
			batch.BlockHash,
			serialized,
			m.dataProviders,
			daprovider.KeysetValidate,
		)
		if err != nil {
			return nil, nil, nil, err
		}

		// TODO: This needs to extract more than one message. It needs to loop
		// until we extracted all messages from the batch being processed.
		msg, err := m.extractArbosMessage(ctx, state, rawSequencerMsg, m.melDB)
		if err != nil {
			return nil, nil, nil, err
		}
		report := batchPostingReports[i]
		if report.Message.BatchGasCost == nil {
			return nil, nil, nil, fmt.Errorf(
				"batch posting report %+v does not have a batch gas cost",
				report.Message,
			)
		}
		// Fill in the message's batch gas cost from the batch posting report.
		msg.Message.BatchGasCost = report.Message.BatchGasCost
		messages = append(messages, msg)
		state.MsgCount += 1
		msgIdx := arbutil.MessageIndex(state.MsgCount) - 1
		msgHash, err := msg.Hash(msgIdx, state.ParentChainId)
		if err != nil {
			return nil, nil, nil, err
		}
		state = state.AccumulateMessage(msgHash)
	}
	return state, messages, delayedMessages, nil
}

func (m *MessageExtractor) extractArbosMessage(
	ctx context.Context,
	melState *State,
	seqMsg *arbstate.SequencerMessage,
	melDB StateDatabase,
) (*arbostypes.MessageWithMetadata, error) {
	segmentNum := uint64(0)
	submessageNumber := uint64(0)
	targetSubMessage := uint64(0)
	blockNumber := uint64(0)
	timestamp := uint64(0)
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
		} else if submessageNumber < targetSubMessage {
			segmentNum++
			submessageNumber++
		} else {
			break
		}
	}
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
		return nil, nil
	}
	kind := segment[0]
	segment = segment[1:]
	var msg *arbostypes.MessageWithMetadata
	if kind == arbstate.BatchSegmentKindL2Message || kind == arbstate.BatchSegmentKindL2MessageBrotli {
		if kind == arbstate.BatchSegmentKindL2MessageBrotli {
			decompressed, err := arbcompress.Decompress(segment, arbostypes.MaxL2MessageSize)
			if err != nil {
				log.Info("dropping compressed message", "err", err)
				return nil, nil
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
			DelayedMessagesRead: melState.DelayedMessagesRead,
		}
	} else if kind == arbstate.BatchSegmentKindDelayedMessages {
		if melState.DelayedMessagesRead >= seqMsg.AfterDelayedMessages {
			if segmentNum < uint64(len(seqMsg.Segments)) {
				log.Warn(
					"Attempt to read past batch delayed message count",
					"delayedMessagesRead", melState.DelayedMessagesRead,
					"batchAfterDelayedMessages", seqMsg.AfterDelayedMessages,
				)
			}
			msg = &arbostypes.MessageWithMetadata{
				Message:             arbostypes.InvalidL1Message,
				DelayedMessagesRead: seqMsg.AfterDelayedMessages,
			}
		} else {
			delayed, err := melDB.ReadDelayedMessage(ctx, melState.DelayedMessagesRead)
			if err != nil {
				return nil, err
			}
			if delayed == nil {
				log.Error("No more delayed messages in queue", "delayedMessagesRead", melState.DelayedMessagesRead)
				return nil, fmt.Errorf("no more delayed messages in queue")
			}
			// TODO: Verify that this delayed message retrieved from an external source, such as a DB,
			// is a child of the delayed message accumulator in the MEL state.
			melState.DelayedMessagesRead += 1
			msg = &arbostypes.MessageWithMetadata{
				Message:             delayed.Message,
				DelayedMessagesRead: melState.DelayedMessagesRead,
			}
		}
	} else {
		log.Error("Bad sequencer message segment kind", "segmentNum", segmentNum, "kind", kind)
		return nil, nil
	}
	return msg, nil
}
