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
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbos/l1pricing"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/arbstate/daprovider"
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
) (*State, []*arbostypes.MessageWithMetadata, error) {
	state := inputState.Clone()
	// Copies the state to avoid mutating the input in case of errors.
	// Check parent chain block hash linkage.
	if state.ParentChainPreviousBlockHash != parentChainBlock.ParentHash() {
		return nil, nil, fmt.Errorf(
			"%w: expected %s, got %s",
			ErrInvalidParentChainBlock,
			state.ParentChainPreviousBlockHash.Hex(),
			parentChainBlock.ParentHash().Hex(),
		)
	}
	// Now, check for any logs emitted by the sequencer inbox by txs
	// included in the parent chain block.
	prevBlockNum := parentChainBlock.NumberU64() - 1
	batches, err := m.sequencerInbox.LookupBatchesInRange(
		ctx,
		new(big.Int).SetUint64(prevBlockNum),
		parentChainBlock.Number(),
	)
	if err != nil {
		return nil, nil, err
	}
	var messages []*arbostypes.MessageWithMetadata
	for _, batch := range batches {
		serialized, err := batch.Serialize(ctx, m.l1Reader.Client())
		if err != nil {
			return nil, nil, err
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
			return nil, nil, err
		}
		_ = rawSequencerMsg
		msg, err := m.extractArbosMessage(rawSequencerMsg)
		if err != nil {
			return nil, nil, err
		}
		messages = append(messages, msg)
		state.AccumulateMessage(msg)
	}

	// Updates the fields in the state to corresponding to the
	// incoming parent chain block.
	state.ParentChainBlockHash = parentChainBlock.Hash()
	state.ParentChainBlockNumber = parentChainBlock.NumberU64()

	return state, messages, nil
}

func (m *MessageExtractor) extractArbosMessage(seqMsg *arbstate.SequencerMessage) (*arbostypes.MessageWithMetadata, error) {
	delayedMessagesRead := uint64(0) // TODO: Get from the MEL state.
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
			DelayedMessagesRead: delayedMessagesRead,
		}
	} else if kind == arbstate.BatchSegmentKindDelayedMessages {
		if delayedMessagesRead >= seqMsg.AfterDelayedMessages {
			if segmentNum < uint64(len(seqMsg.Segments)) {
				log.Warn(
					"Attempt to read past batch delayed message count",
					"delayedMessagesRead", delayedMessagesRead,
					"batchAfterDelayedMessages", seqMsg.AfterDelayedMessages,
				)
			}
			msg = &arbostypes.MessageWithMetadata{
				Message:             arbostypes.InvalidL1Message,
				DelayedMessagesRead: seqMsg.AfterDelayedMessages,
			}
		} else {
			// delayed, realErr := r.backend.ReadDelayedInbox(delayedMessagesRead)
			// if realErr != nil {
			// 	return nil, realErr
			// }
			// r.delayedMessagesRead += 1
			// msg = &arbostypes.MessageWithMetadata{
			// 	Message:             delayed,
			// 	DelayedMessagesRead: r.delayedMessagesRead,
			// }
			msg = &arbostypes.MessageWithMetadata{}
		}
	} else {
		log.Error("Bad sequencer message segment kind", "segmentNum", segmentNum, "kind", kind)
		return nil, nil
	}
	return msg, nil
}
