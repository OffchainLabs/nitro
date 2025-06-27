package melextraction

import (
	"bytes"
	"context"
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
// virtual segment in encountered.
func extractMessagesInBatch(
	ctx context.Context,
	melState *mel.State,
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
		if hasSegmentsRemaining(params) {
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

func hasSegmentsRemaining(p *arbosExtractionParams) bool {
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
	melState         *mel.State
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
		// After end of batch there might be "virtual" delayedMsgSegments
		log.Debug("Reading virtual delayed message segment")
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
				return nil, p, fmt.Errorf("no more delayed messages in db")
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
