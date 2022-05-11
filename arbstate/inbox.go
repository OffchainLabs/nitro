// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbstate

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/offchainlabs/nitro/zeroheavy"
	"io"
	"math/big"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/offchainlabs/nitro/arbcompress"
	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbos/l1pricing"
)

type InboxBackend interface {
	PeekSequencerInbox() ([]byte, error)

	GetSequencerInboxPosition() uint64
	AdvanceSequencerInbox()

	GetPositionWithinMessage() uint64
	SetPositionWithinMessage(pos uint64)

	ReadDelayedInbox(seqNum uint64) ([]byte, error)
}

type MessageWithMetadata struct {
	Message             *arbos.L1IncomingMessage `json:"message"`
	DelayedMessagesRead uint64                   `json:"delayedMessagesRead"`
}

type InboxMultiplexer interface {
	Pop(context.Context) (*MessageWithMetadata, error)
	DelayedMessagesRead() uint64
}

type sequencerMessage struct {
	minTimestamp         uint64
	maxTimestamp         uint64
	minL1Block           uint64
	maxL1Block           uint64
	afterDelayedMessages uint64
	segments             [][]byte
}

const maxDecompressedLen int = 1024 * 1024 * 16 // 16 MiB
const maxZeroheavyDecompressedLen = 101*maxDecompressedLen/100 + 64
const MaxSegmentsPerSequencerMessage = 100 * 1024
const MinLifetimeSecondsForDataAvailabilityCert = 7 * 24 * 60 * 60 // one week

func parseSequencerMessage(ctx context.Context, data []byte, das DataAvailabilityServiceReader) (*sequencerMessage, error) {
	if len(data) < 40 {
		panic("sequencer message missing L1 header")
	}
	minTimestamp := binary.BigEndian.Uint64(data[:8])
	maxTimestamp := binary.BigEndian.Uint64(data[8:16])
	minL1Block := binary.BigEndian.Uint64(data[16:24])
	maxL1Block := binary.BigEndian.Uint64(data[24:32])
	afterDelayedMessages := binary.BigEndian.Uint64(data[32:40])
	var segments [][]byte

	var payload []byte
	if len(data) >= 41 {
		if IsDASMessageHeaderByte(data[40]) {
			if das == nil {
				log.Error("No DAS configured, but sequencer message found with DAS header")
			} else {
				cert, err := DeserializeDASCertFrom(bytes.NewReader(data[40:]))
				if err != nil {
					log.Error("Deserializing data availability cert failed", "err", err)
					return &sequencerMessage{
						minTimestamp:         minTimestamp,
						maxTimestamp:         maxTimestamp,
						minL1Block:           minL1Block,
						maxL1Block:           maxL1Block,
						afterDelayedMessages: afterDelayedMessages,
						segments:             segments,
					}, nil
				}
				keyset, err := cert.RecoverKeyset(ctx, das)
				if err != nil {
					return nil, errors.New("unable to recover keyset even though L1 thought it was valid")
				}
				if err := keyset.VerifySignature(cert.SignersMask, cert.SerializeSignableFields(), cert.Sig); err != nil { // safe because L1 verified keyset hash
					log.Error("Bad signature on DAS batch", "err", err)
					return &sequencerMessage{
						minTimestamp:         minTimestamp,
						maxTimestamp:         maxTimestamp,
						minL1Block:           minL1Block,
						maxL1Block:           maxL1Block,
						afterDelayedMessages: afterDelayedMessages,
						segments:             segments,
					}, nil
				}
				if cert.Timeout < maxTimestamp+MinLifetimeSecondsForDataAvailabilityCert {
					log.Error("Data availability cert expires too soon", "err", "")
					return &sequencerMessage{
						minTimestamp:         minTimestamp,
						maxTimestamp:         maxTimestamp,
						minL1Block:           minL1Block,
						maxL1Block:           maxL1Block,
						afterDelayedMessages: afterDelayedMessages,
						segments:             segments,
					}, nil
				}
				payload, err = das.Retrieve(ctx, cert) // safe because DA cert was verified
				if err != nil || !bytes.Equal(crypto.Keccak256(payload), cert.DataHash[:]) {
					panic("DAS retrieve failed") // should never happen--best to halt execution if it does
				}
			}
		} else if data[40] == 0 {
			payload = data[40:]
		}

		if len(payload) > 0 {
			if IsZeroheavyEncodedHeaderByte(data[40]) {
				pl, err := io.ReadAll(io.LimitReader(zeroheavy.NewZeroheavyDecoder(bytes.NewReader(payload)), int64(maxZeroheavyDecompressedLen)))
				if err != nil {
					log.Warn("error reading from zeroheavy decoder", err.Error())
					pl = []byte{}
				}
				payload = pl
			}
			decompressed, err := arbcompress.Decompress(payload[1:], maxDecompressedLen)
			if err == nil {
				reader := bytes.NewReader(decompressed)
				stream := rlp.NewStream(reader, uint64(maxDecompressedLen))
				for {
					var segment []byte
					err := stream.Decode(&segment)
					if err != nil {
						if !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
							log.Warn("error parsing sequencer message segment", "err", err.Error())
						}
						break
					}
					if len(segments) >= MaxSegmentsPerSequencerMessage {
						log.Warn("too many segments in sequence batch")
						break
					}
					segments = append(segments, segment)
				}
			} else {
				log.Warn("sequencer msg decompression failed", "err", err)
			}
		} else {
			log.Warn("unknown sequencer message format")
		}
	}
	return &sequencerMessage{
		minTimestamp:         minTimestamp,
		maxTimestamp:         maxTimestamp,
		minL1Block:           minL1Block,
		maxL1Block:           maxL1Block,
		afterDelayedMessages: afterDelayedMessages,
		segments:             segments,
	}, nil
}

type inboxMultiplexer struct {
	backend                   InboxBackend
	delayedMessagesRead       uint64
	das                       DataAvailabilityServiceReader
	cachedSequencerMessage    *sequencerMessage
	cachedSequencerMessageNum uint64
	cachedSegmentNum          uint64
	cachedSegmentTimestamp    uint64
	cachedSegmentBlockNumber  uint64
	cachedSubMessageNumber    uint64
}

func NewInboxMultiplexer(backend InboxBackend, delayedMessagesRead uint64, das DataAvailabilityServiceReader) InboxMultiplexer {
	return &inboxMultiplexer{
		backend:             backend,
		delayedMessagesRead: delayedMessagesRead,
		das:                 das,
	}
}

var InvalidL1Message *arbos.L1IncomingMessage = &arbos.L1IncomingMessage{
	Header: &arbos.L1IncomingMessageHeader{
		Kind: arbos.L1MessageType_Invalid,
	},
	L2msg: []byte{},
}

const BatchSegmentKindL2Message uint8 = 0
const BatchSegmentKindL2MessageBrotli uint8 = 1
const BatchSegmentKindDelayedMessages uint8 = 2
const BatchSegmentKindAdvanceTimestamp uint8 = 3
const BatchSegmentKindAdvanceL1BlockNumber uint8 = 4

// This does *not* return parse errors, those are transformed into invalid messages
func (r *inboxMultiplexer) Pop(ctx context.Context) (*MessageWithMetadata, error) {
	if r.cachedSequencerMessage == nil {
		bytes, realErr := r.backend.PeekSequencerInbox()
		if realErr != nil {
			return nil, realErr
		}
		r.cachedSequencerMessageNum = r.backend.GetSequencerInboxPosition()
		var err error
		r.cachedSequencerMessage, err = parseSequencerMessage(ctx, bytes, r.das)
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
		msg = &MessageWithMetadata{
			Message:             InvalidL1Message,
			DelayedMessagesRead: r.delayedMessagesRead,
		}
	}
	return msg, err
}

func (r *inboxMultiplexer) advanceSequencerMsg() {
	if r.cachedSequencerMessage != nil {
		r.delayedMessagesRead = r.cachedSequencerMessage.afterDelayedMessages
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
	if r.delayedMessagesRead < seqMsg.afterDelayedMessages {
		return false
	}
	for segmentNum := int(r.cachedSegmentNum) + 1; segmentNum < len(seqMsg.segments); segmentNum++ {
		segment := seqMsg.segments[segmentNum]
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
func (r *inboxMultiplexer) getNextMsg() (*MessageWithMetadata, error) {
	targetSubMessage := r.backend.GetPositionWithinMessage()
	seqMsg := r.cachedSequencerMessage
	segmentNum := r.cachedSegmentNum
	timestamp := r.cachedSegmentTimestamp
	blockNumber := r.cachedSegmentBlockNumber
	submessageNumber := r.cachedSubMessageNumber
	var segment []byte
	for {
		if segmentNum >= uint64(len(seqMsg.segments)) {
			break
		}
		segment = seqMsg.segments[int(segmentNum)]
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
	if timestamp < seqMsg.minTimestamp {
		timestamp = seqMsg.minTimestamp
	} else if timestamp > seqMsg.maxTimestamp {
		timestamp = seqMsg.maxTimestamp
	}
	if blockNumber < seqMsg.minL1Block {
		blockNumber = seqMsg.minL1Block
	} else if blockNumber > seqMsg.maxL1Block {
		blockNumber = seqMsg.maxL1Block
	}
	if segmentNum >= uint64(len(seqMsg.segments)) {
		// after end of batch there might be "virtual" delayedMsgSegments
		log.Warn("reading virtual delayed message segment", "delayedMessagesRead", r.delayedMessagesRead, "afterDelayedMessages", seqMsg.afterDelayedMessages)
		segment = []byte{BatchSegmentKindDelayedMessages}
	} else {
		segment = seqMsg.segments[int(segmentNum)]
	}
	if len(segment) == 0 {
		log.Error("empty sequencer message segment", "sequence", r.cachedSegmentNum, "segmentNum", segmentNum)
		return nil, nil
	}
	kind := segment[0]
	segment = segment[1:]
	var msg *MessageWithMetadata
	if kind == BatchSegmentKindL2Message || kind == BatchSegmentKindL2MessageBrotli {

		if kind == BatchSegmentKindL2MessageBrotli {
			decompressed, err := arbcompress.Decompress(segment[1:], arbos.MaxL2MessageSize)
			if err != nil {
				log.Info("dropping compressed message", "err", err, "delayedMsg", r.delayedMessagesRead)
				return nil, nil
			}
			segment = decompressed
		}

		msg = &MessageWithMetadata{
			Message: &arbos.L1IncomingMessage{
				Header: &arbos.L1IncomingMessageHeader{
					Kind:        arbos.L1MessageType_L2Message,
					Poster:      l1pricing.SequencerAddress,
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
		if r.delayedMessagesRead >= seqMsg.afterDelayedMessages {
			if segmentNum < uint64(len(seqMsg.segments)) {
				log.Warn(
					"attempt to read past batch delayed message count",
					"delayedMessagesRead", r.delayedMessagesRead,
					"batchAfterDelayedMessages", seqMsg.afterDelayedMessages,
				)
			}
			msg = &MessageWithMetadata{
				Message:             InvalidL1Message,
				DelayedMessagesRead: seqMsg.afterDelayedMessages,
			}
		} else {
			data, realErr := r.backend.ReadDelayedInbox(r.delayedMessagesRead)
			if realErr != nil {
				return nil, realErr
			}
			r.delayedMessagesRead += 1
			delayed, parseErr := arbos.ParseIncomingL1Message(bytes.NewReader(data))
			if parseErr != nil {
				log.Warn("error parsing delayed message", "err", parseErr, "delayedMsg", r.delayedMessagesRead)
				return nil, nil
			}
			msg = &MessageWithMetadata{
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
