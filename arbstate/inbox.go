// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbstate

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/offchainlabs/nitro/arbcompress"
	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbos/l1pricing"
	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/zeroheavy"
)

type InboxBackend interface {
	PeekSequencerInbox() ([]byte, common.Hash, error)

	GetSequencerInboxPosition() uint64
	AdvanceSequencerInbox()

	GetPositionWithinMessage() uint64
	SetPositionWithinMessage(pos uint64)

	ReadDelayedInbox(seqNum uint64) (*arbostypes.L1IncomingMessage, error)
}

// lint:require-exhaustive-initialization
type SequencerMessage struct {
	MinTimestamp         uint64
	MaxTimestamp         uint64
	MinL1Block           uint64
	MaxL1Block           uint64
	AfterDelayedMessages uint64
	Segments             [][]byte
}

// TODO: We probably don't even need this map if batch payloads are only used once.
type BatchPayloadMap map[common.Hash]daprovider.PayloadResult

const MaxDecompressedLen int = 1024 * 1024 * 16 // 16 MiB
const maxZeroheavyDecompressedLen = 101*MaxDecompressedLen/100 + 64
const MaxSegmentsPerSequencerMessage = 100 * 1024
const L1HeaderSize = 40

func HandleBlobs(ctx context.Context, batchNum uint64, batchBlockHash common.Hash, data []byte, dapReaders *daprovider.ReaderRegistry, keysetValidationMode daprovider.KeysetValidationMode) (daprovider.PayloadResult, error) {
	var result daprovider.PayloadResult
	payload := data[L1HeaderSize:]
	if len(payload) > 0 && dapReaders != nil {
		if dapReader, found := dapReaders.GetByHeaderByte(payload[0]); found {
			promise := dapReader.RecoverPayload(batchNum, batchBlockHash, data)
			res, err := promise.Await(ctx)
			if err != nil {
				return result, err
			}
			result = res
		}
	}
	return result, nil
}

func ParseSequencerMessage(ctx context.Context, batchNum uint64, batchBlockHash common.Hash, data []byte, dapReaders *daprovider.ReaderRegistry, cachedPayloads *BatchPayloadMap, keysetValidationMode daprovider.KeysetValidationMode) (*SequencerMessage, error) {
	if len(data) < L1HeaderSize {
		return nil, errors.New("sequencer message missing L1 header")
	}
	parsedMsg := &SequencerMessage{
		MinTimestamp:         binary.BigEndian.Uint64(data[:8]),
		MaxTimestamp:         binary.BigEndian.Uint64(data[8:16]),
		MinL1Block:           binary.BigEndian.Uint64(data[16:24]),
		MaxL1Block:           binary.BigEndian.Uint64(data[24:32]),
		AfterDelayedMessages: binary.BigEndian.Uint64(data[32:40]),
		Segments:             [][]byte{},
	}
	payload := data[L1HeaderSize:]

	// Stage 0: Check if our node is out of date and we don't understand this batch type
	// If the parent chain sequencer inbox smart contract authenticated this batch,
	// an unknown header byte must mean that this node is out of date,
	// because the smart contract understands the header byte and this node doesn't.
	if len(payload) > 0 && daprovider.IsL1AuthenticatedMessageHeaderByte(payload[0]) && !daprovider.IsKnownHeaderByte(payload[0]) {
		return nil, fmt.Errorf("%w: batch number %d has unsupported authenticated header byte 0x%02x", arbosState.ErrFatalNodeOutOfDate, batchNum, payload[0])
	}

	// Stage 1: Extract the payload from any data availability header.
	// It's important that multiple DAS strategies can't both be invoked in the same batch,
	// as these headers are validated by the sequencer inbox and not other DASs.
	// Use the registry to find the appropriate reader for the header byte
	if len(payload) > 0 && dapReaders != nil {
		if dapReader, found := dapReaders.GetByHeaderByte(payload[0]); found {
			var result daprovider.PayloadResult
			var ok bool
			var err error

			// We first try to fetch payload from cache and if not available we call it from DA provider
			if cachedPayloads != nil {
				result, ok = (*cachedPayloads)[batchBlockHash]
			}

			// TODO: Do we want to fallback to DA provider if payload is not found in cache?
			if !ok {
				promise := dapReader.RecoverPayload(batchNum, batchBlockHash, data)
				result, err = promise.Await(ctx)
			} else {
				// Can we delete payload entry for batchBlockHash or would we need it later?
				// TODO: Maybe we don't even need for cachedPayloads to be a map if we're only keeping one record in it.
				defer delete(*cachedPayloads, batchBlockHash)
			}

			if err != nil {
				// Matches the way keyset validation was done inside DAS readers i.e logging the error
				//  But other daproviders might just want to return the error
				if strings.Contains(err.Error(), daprovider.ErrSeqMsgValidation.Error()) && daprovider.IsDASMessageHeaderByte(payload[0]) {
					if keysetValidationMode == daprovider.KeysetPanicIfInvalid {
						panic(err.Error())
					} else {
						log.Error(err.Error())
					}
				} else {
					return nil, err
				}
			} else {
				payload = result.Payload
			}
			if payload == nil {
				return parsedMsg, nil
			}
		} else {
			// No reader found for this header byte - check if it's a known type
			if daprovider.IsDASMessageHeaderByte(payload[0]) {
				return nil, fmt.Errorf("no DAS reader configured for DAS message (header byte 0x%02x)", payload[0])
			} else if daprovider.IsBlobHashesHeaderByte(payload[0]) {
				return nil, daprovider.ErrNoBlobReader
			} else if daprovider.IsDACertificateMessageHeaderByte(payload[0]) {
				return nil, fmt.Errorf("no DACertificate reader configured for certificate message (header byte 0x%02x)", payload[0])
			}
			// Otherwise it's not a DA message, continue processing
		}
	}

	// At this point, `payload` has not been validated by the sequencer inbox at all.
	// It's not safe to trust any part of the payload from this point onwards.

	// Stage 2: If enabled, decode the zero heavy payload (saves gas based on calldata charging).
	if len(payload) > 0 && daprovider.IsZeroheavyEncodedHeaderByte(payload[0]) {
		pl, err := io.ReadAll(io.LimitReader(zeroheavy.NewZeroheavyDecoder(bytes.NewReader(payload[1:])), int64(maxZeroheavyDecompressedLen)))
		if err != nil {
			log.Warn("error reading from zeroheavy decoder", err.Error())
			return parsedMsg, nil
		}
		payload = pl
	}

	// Stage 3: Decompress the brotli payload and fill the parsedMsg.segments list.
	if len(payload) > 0 && daprovider.IsBrotliMessageHeaderByte(payload[0]) {
		decompressed, err := arbcompress.Decompress(payload[1:], MaxDecompressedLen)
		if err == nil {
			reader := bytes.NewReader(decompressed)
			stream := rlp.NewStream(reader, uint64(MaxDecompressedLen))
			for {
				var segment []byte
				err := stream.Decode(&segment)
				if err != nil {
					if !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
						log.Warn("error parsing sequencer message segment", "err", err.Error())
					}
					break
				}
				if len(parsedMsg.Segments) >= MaxSegmentsPerSequencerMessage {
					log.Warn("too many segments in sequence batch")
					break
				}
				parsedMsg.Segments = append(parsedMsg.Segments, segment)
			}
		} else {
			log.Warn("sequencer msg decompression failed", "err", err)
		}
	} else {
		length := len(payload)
		if length == 0 {
			log.Warn("empty sequencer message")
		} else {
			log.Warn("unknown sequencer message format", "length", length, "firstByte", payload[0])
		}

	}

	return parsedMsg, nil
}

// lint:require-exhaustive-initialization
type inboxMultiplexer struct {
	backend                   InboxBackend
	delayedMessagesRead       uint64
	dapReaders                *daprovider.ReaderRegistry
	cachedSequencerMessage    *SequencerMessage
	cachedSequencerMessageNum uint64
	cachedSegmentNum          uint64
	cachedSegmentTimestamp    uint64
	cachedSegmentBlockNumber  uint64
	cachedSubMessageNumber    uint64
	cachedPayload             *BatchPayloadMap
	// keysetValidationMode is used for error handling in ParseSequencerMessage.
	// Note: DAS readers now handle validation internally based on their construction-time mode,
	// but ParseSequencerMessage still needs this to decide whether to panic or log on validation errors.
	// In replay mode, this allows proper error handling based on the position within the message.
	keysetValidationMode daprovider.KeysetValidationMode
}

func NewInboxMultiplexer(backend InboxBackend, delayedMessagesRead uint64, dapReaders *daprovider.ReaderRegistry, payloadMap *BatchPayloadMap, keysetValidationMode daprovider.KeysetValidationMode) arbostypes.InboxMultiplexer {
	return &inboxMultiplexer{
		backend:                   backend,
		delayedMessagesRead:       delayedMessagesRead,
		dapReaders:                dapReaders,
		cachedSequencerMessage:    nil,
		cachedSequencerMessageNum: 0,
		cachedSegmentNum:          0,
		cachedSegmentTimestamp:    0,
		cachedSegmentBlockNumber:  0,
		cachedSubMessageNumber:    0,
		cachedPayload:             payloadMap,
		keysetValidationMode:      keysetValidationMode,
	}
}

const BatchSegmentKindL2Message uint8 = 0
const BatchSegmentKindL2MessageBrotli uint8 = 1
const BatchSegmentKindDelayedMessages uint8 = 2
const BatchSegmentKindAdvanceTimestamp uint8 = 3
const BatchSegmentKindAdvanceL1BlockNumber uint8 = 4

func (r *inboxMultiplexer) CacheBlobs(ctx context.Context) error {
	bytes, batchBlockHash, realErr := r.backend.PeekSequencerInbox()
	if realErr != nil {
		return realErr
	}
	r.cachedSequencerMessageNum = r.backend.GetSequencerInboxPosition()
	var err error
	payload, err := HandleBlobs(ctx, r.cachedSequencerMessageNum, batchBlockHash, bytes, r.dapReaders, r.keysetValidationMode)
	if err != nil {
		return err
	}

	(*r.cachedPayload)[batchBlockHash] = payload

	return nil
}

// Pop returns the message from the top of the sequencer inbox and removes it from the queue.
// Note: this does *not* return parse errors, those are transformed into invalid messages
func (r *inboxMultiplexer) Pop(ctx context.Context) (*arbostypes.MessageWithMetadata, error) {
	if r.cachedSequencerMessage == nil {
		// Note: batchBlockHash will be zero in the replay binary, but that's fine
		bytes, batchBlockHash, realErr := r.backend.PeekSequencerInbox()
		if realErr != nil {
			return nil, realErr
		}
		r.cachedSequencerMessageNum = r.backend.GetSequencerInboxPosition()
		var err error
		r.cachedSequencerMessage, err = ParseSequencerMessage(ctx, r.cachedSequencerMessageNum, batchBlockHash, bytes, r.dapReaders, r.cachedPayload, r.keysetValidationMode)
		if err != nil {
			return nil, err
		}
	}
	msg, err := r.getNextMsg()
	// advance even if there was an error
	if r.IsCachedSegementLast() {
		r.advanceSequencerMsg()
	} else {
		r.advanceSubMsg()
	}
	// parsing error in getNextMsg
	if msg == nil && err == nil {
		msg = &arbostypes.MessageWithMetadata{
			Message:             arbostypes.InvalidL1Message,
			DelayedMessagesRead: r.delayedMessagesRead,
		}
	}
	return msg, err
}

func (r *inboxMultiplexer) advanceSequencerMsg() {
	if r.cachedSequencerMessage != nil {
		r.delayedMessagesRead = r.cachedSequencerMessage.AfterDelayedMessages
	}
	r.backend.SetPositionWithinMessage(0)
	r.backend.AdvanceSequencerInbox()
	r.cachedSequencerMessage = nil
	r.cachedSegmentNum = 0
	r.cachedSegmentTimestamp = 0
	r.cachedSegmentBlockNumber = 0
	r.cachedSubMessageNumber = 0
}

func (r *inboxMultiplexer) advanceSubMsg() {
	prevPos := r.backend.GetPositionWithinMessage()
	r.backend.SetPositionWithinMessage(prevPos + 1)
}

func (r *inboxMultiplexer) IsCachedSegementLast() bool {
	seqMsg := r.cachedSequencerMessage
	// we issue delayed messages until reaching afterDelayedMessages
	if r.delayedMessagesRead < seqMsg.AfterDelayedMessages {
		return false
	}
	for segmentNum := r.cachedSegmentNum + 1; segmentNum < uint64(len(seqMsg.Segments)); segmentNum++ {
		segment := seqMsg.Segments[segmentNum]
		if len(segment) == 0 {
			continue
		}
		kind := segment[0]
		if kind == BatchSegmentKindL2Message || kind == BatchSegmentKindL2MessageBrotli {
			return false
		}
		if kind == BatchSegmentKindDelayedMessages {
			return false
		}
	}
	return true
}

// Returns a message, the segment number that had this message, and real/backend errors
// parsing errors will be reported to log, return nil msg and nil error
func (r *inboxMultiplexer) getNextMsg() (*arbostypes.MessageWithMetadata, error) {
	targetSubMessage := r.backend.GetPositionWithinMessage()
	seqMsg := r.cachedSequencerMessage
	segmentNum := r.cachedSegmentNum
	timestamp := r.cachedSegmentTimestamp
	blockNumber := r.cachedSegmentBlockNumber
	submessageNumber := r.cachedSubMessageNumber
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
		if segmentKind == BatchSegmentKindAdvanceTimestamp || segmentKind == BatchSegmentKindAdvanceL1BlockNumber {
			rd := bytes.NewReader(segment[1:])
			advancing, err := rlp.NewStream(rd, 16).Uint64()
			if err != nil {
				log.Warn("error parsing sequencer advancing segment", "err", err)
				segmentNum++
				continue
			}
			if segmentKind == BatchSegmentKindAdvanceTimestamp {
				timestamp += advancing
			} else if segmentKind == BatchSegmentKindAdvanceL1BlockNumber {
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
	r.cachedSegmentNum = segmentNum
	r.cachedSegmentTimestamp = timestamp
	r.cachedSegmentBlockNumber = blockNumber
	r.cachedSubMessageNumber = submessageNumber
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
		log.Warn("reading virtual delayed message segment", "delayedMessagesRead", r.delayedMessagesRead, "afterDelayedMessages", seqMsg.AfterDelayedMessages)
		segment = []byte{BatchSegmentKindDelayedMessages}
	} else {
		segment = seqMsg.Segments[segmentNum]
	}
	if len(segment) == 0 {
		log.Error("empty sequencer message segment", "sequence", r.cachedSegmentNum, "segmentNum", segmentNum)
		return nil, nil
	}
	kind := segment[0]
	segment = segment[1:]
	var msg *arbostypes.MessageWithMetadata
	if kind == BatchSegmentKindL2Message || kind == BatchSegmentKindL2MessageBrotli {

		if kind == BatchSegmentKindL2MessageBrotli {
			decompressed, err := arbcompress.Decompress(segment, arbostypes.MaxL2MessageSize)
			if err != nil {
				log.Info("dropping compressed message", "err", err, "delayedMsg", r.delayedMessagesRead)
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
			DelayedMessagesRead: r.delayedMessagesRead,
		}
	} else if kind == BatchSegmentKindDelayedMessages {
		if r.delayedMessagesRead >= seqMsg.AfterDelayedMessages {
			if segmentNum < uint64(len(seqMsg.Segments)) {
				log.Warn(
					"attempt to read past batch delayed message count",
					"delayedMessagesRead", r.delayedMessagesRead,
					"batchAfterDelayedMessages", seqMsg.AfterDelayedMessages,
				)
			}
			msg = &arbostypes.MessageWithMetadata{
				Message:             arbostypes.InvalidL1Message,
				DelayedMessagesRead: seqMsg.AfterDelayedMessages,
			}
		} else {
			delayed, realErr := r.backend.ReadDelayedInbox(r.delayedMessagesRead)
			if realErr != nil {
				return nil, realErr
			}
			r.delayedMessagesRead += 1
			msg = &arbostypes.MessageWithMetadata{
				Message:             delayed,
				DelayedMessagesRead: r.delayedMessagesRead,
			}
		}
	} else {
		log.Error("bad sequencer message segment kind", "sequence", r.cachedSegmentNum, "segmentNum", segmentNum, "kind", kind)
		return nil, nil
	}
	return msg, nil
}

func (r *inboxMultiplexer) DelayedMessagesRead() uint64 {
	return r.delayedMessagesRead
}
