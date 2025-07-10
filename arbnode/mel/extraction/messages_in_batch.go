package melextraction

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/offchainlabs/nitro/arbcompress"
	"github.com/offchainlabs/nitro/arbnode/mel"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbos/l1pricing"
	"github.com/offchainlabs/nitro/arbstate"
)

// Extracts a list of arbos messages from the sequencer batch, which looks at the
// segments contained in a batch and extracts the correct ordering of messages
// from the segments, possibly reading delayed messages when a delayed message
// virtual segment in encountered, or at the end if we have not read enough
// delayed messages in our MEL state when compared to the sequencer message's.
func messagesFromBatchSegments(
	ctx context.Context,
	melState *mel.State,
	seqMsg *arbstate.SequencerMessage,
	delayedMsgDB DelayedMessageDatabase,
) ([]*arbostypes.MessageWithMetadata, error) {
	messages := make([]*arbostypes.MessageWithMetadata, 0)
	timestamp := uint64(0)
	blockNumber := uint64(0)
	for idx, segment := range seqMsg.Segments {
		msg, newBlockNumber, newTimestamp, err := messageFromSegment(
			ctx,
			melState,
			delayedMsgDB,
			seqMsg,
			idx,
			segment,
			timestamp,
			blockNumber,
		)
		if err != nil {
			if errors.Is(err, ErrParsingAdvancingSegment) {
				continue // We ignore being able to parse an advance segment.
			}
			return nil, fmt.Errorf(
				"error parsing segment %d: %w", idx, err,
			)
		}
		timestamp = newTimestamp
		blockNumber = newBlockNumber
		if msg == nil {
			continue
		}
		messages = append(messages, msg)
	}

	// If the mel state delayed messages read even after we completed reading
	// all segments is less than the sequencer message's
	// after delayed messages, we need to read more delayed messages here.
	for melState.DelayedMessagesRead < seqMsg.AfterDelayedMessages {
		msg, err := extractDelayedMessageFromSegment(
			ctx,
			melState,
			seqMsg,
			delayedMsgDB,
		)
		if err != nil {
			return nil, fmt.Errorf("error extracting delayed message: %w", err)
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

var ErrParsingAdvancingSegment = fmt.Errorf("error parsing advancing segment")

func messageFromSegment(
	ctx context.Context,
	melState *mel.State,
	delayedMsgDB DelayedMessageDatabase,
	seqMsg *arbstate.SequencerMessage,
	segmentIdx int,
	segment []byte,
	timestamp uint64,
	blockNumber uint64,
) (*arbostypes.MessageWithMetadata, uint64, uint64, error) {
	if len(segment) == 0 {
		log.Warn("Empty segment in sequencer message", "seqMsg", seqMsg)
		return nil, blockNumber, timestamp, nil
	}
	kind := segment[0]
	if kind == arbstate.BatchSegmentKindAdvanceTimestamp || kind == arbstate.BatchSegmentKindAdvanceL1BlockNumber {
		rd := bytes.NewReader(segment[1:])
		advancing, err := rlp.NewStream(rd, 16).Uint64()
		if err != nil {
			log.Warn("Error parsing sequencer advancing segment", "err", err)
			return nil, blockNumber, timestamp, ErrParsingAdvancingSegment
		}
		if kind == arbstate.BatchSegmentKindAdvanceTimestamp {
			timestamp += advancing
		} else if kind == arbstate.BatchSegmentKindAdvanceL1BlockNumber {
			blockNumber += advancing
		}
		return nil, blockNumber, timestamp, nil
	} else if kind == arbstate.BatchSegmentKindL2Message || kind == arbstate.BatchSegmentKindL2MessageBrotli {
		segment = segment[1:]
		msg := produceL2Message(
			kind,
			seqMsg,
			segment,
			blockNumber,
			timestamp,
			melState.DelayedMessagesRead,
		)
		return msg, blockNumber, timestamp, nil
	} else if kind == arbstate.BatchSegmentKindDelayedMessages {
		msg, err := extractDelayedMessageFromSegment(
			ctx,
			melState,
			seqMsg,
			delayedMsgDB,
		)
		if err != nil {
			return nil, blockNumber, timestamp, err
		}
		return msg, blockNumber, timestamp, nil
	} else {
		log.Error("Bad sequencer message segment kind", "segmentNum", segmentIdx, "kind", kind)
		return &arbostypes.MessageWithMetadata{
			Message:             arbostypes.InvalidL1Message,
			DelayedMessagesRead: melState.DelayedMessagesRead,
		}, blockNumber, timestamp, nil
	}
}

func produceL2Message(
	kind byte,
	seqMsg *arbstate.SequencerMessage,
	segment []byte,
	blockNumber uint64,
	timestamp uint64,
	delayedMessagesRead uint64,
) *arbostypes.MessageWithMetadata {
	// We constrain the bounds of the timestamp and block number.
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
	seg := segment
	if kind == arbstate.BatchSegmentKindL2MessageBrotli {
		decompressed, err := arbcompress.Decompress(segment, arbostypes.MaxL2MessageSize)
		if err != nil {
			log.Info("dropping compressed message", "err", err)
			return &arbostypes.MessageWithMetadata{
				Message:             arbostypes.InvalidL1Message,
				DelayedMessagesRead: delayedMessagesRead,
			}
		}
		seg = decompressed
	}
	return &arbostypes.MessageWithMetadata{
		Message: &arbostypes.L1IncomingMessage{
			Header: &arbostypes.L1IncomingMessageHeader{
				Kind:        arbostypes.L1MessageType_L2Message,
				Poster:      l1pricing.BatchPosterAddress,
				BlockNumber: blockNumber,
				Timestamp:   timestamp,
				RequestId:   nil,
				L1BaseFee:   big.NewInt(0),
			},
			L2msg: seg,
		},
		DelayedMessagesRead: delayedMessagesRead,
	}
}

func extractDelayedMessageFromSegment(
	ctx context.Context,
	melState *mel.State,
	seqMsg *arbstate.SequencerMessage,
	delayedMsgDB DelayedMessageDatabase,
) (*arbostypes.MessageWithMetadata, error) {
	if melState.DelayedMessagesRead >= seqMsg.AfterDelayedMessages {
		return &arbostypes.MessageWithMetadata{
			Message:             arbostypes.InvalidL1Message,
			DelayedMessagesRead: seqMsg.AfterDelayedMessages,
		}, nil
	}
	delayed, err := delayedMsgDB.ReadDelayedMessage(ctx, melState, melState.DelayedMessagesRead)
	if err != nil {
		return nil, err
	}
	if delayed == nil {
		log.Error("No more delayed messages in queue", "delayedMessagesRead", melState.DelayedMessagesRead)
		return nil, fmt.Errorf("no more delayed messages in db")
	}

	// Increment the delayed messages read count in the mel state.
	melState.DelayedMessagesRead += 1

	return &arbostypes.MessageWithMetadata{
		Message:             delayed.Message,
		DelayedMessagesRead: melState.DelayedMessagesRead,
	}, nil
}
