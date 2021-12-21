//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbstate

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math/big"

	"github.com/andybalholm/brotli"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/offchainlabs/arbstate/arbos"
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
	Pop() (*MessageWithMetadata, error)
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

const maxDecompressedLen int64 = 1024 * 1024 * 16 // 16 MiB

func parseSequencerMessage(data []byte) *sequencerMessage {
	if len(data) < 40 {
		panic("sequencer message missing L1 header")
	}
	minTimestamp := binary.BigEndian.Uint64(data[:8])
	maxTimestamp := binary.BigEndian.Uint64(data[8:16])
	minL1Block := binary.BigEndian.Uint64(data[16:24])
	maxL1Block := binary.BigEndian.Uint64(data[24:32])
	afterDelayedMessages := binary.BigEndian.Uint64(data[32:40])
	var segments [][]byte
	if len(data) >= 41 && data[40] == 0 {
		reader := io.LimitReader(brotli.NewReader(bytes.NewReader(data[41:])), maxDecompressedLen)
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
			segments = append(segments, segment)
		}
	} else {
		log.Warn("unknown sequencer message format")
	}
	return &sequencerMessage{
		minTimestamp:         minTimestamp,
		maxTimestamp:         maxTimestamp,
		minL1Block:           minL1Block,
		maxL1Block:           maxL1Block,
		afterDelayedMessages: afterDelayedMessages,
		segments:             segments,
	}
}

func (m sequencerMessage) Encode() []byte {
	var header [40]byte
	binary.BigEndian.PutUint64(header[:8], m.minTimestamp)
	binary.BigEndian.PutUint64(header[8:16], m.maxTimestamp)
	binary.BigEndian.PutUint64(header[16:24], m.minL1Block)
	binary.BigEndian.PutUint64(header[24:32], m.maxL1Block)
	binary.BigEndian.PutUint64(header[32:40], m.afterDelayedMessages)
	buf := new(bytes.Buffer)
	segmentsEnc, err := rlp.EncodeToBytes(&m.segments)
	if err != nil {
		panic("couldn't encode sequencerMessage")
	}

	writer := brotli.NewWriter(buf)
	defer writer.Close()
	_, err = writer.Write(segmentsEnc)
	if err != nil {
		panic("Could not write")
	}
	writer.Flush()
	return append(header[:], buf.Bytes()...)
}

type inboxMultiplexer struct {
	backend                   InboxBackend
	delayedMessagesRead       uint64
	cachedSequencerMessage    *sequencerMessage
	cachedSequencerMessageNum uint64
	cachedSegmentNum          uint64
	cachedSegmentTimestamp    uint64
	cachedSegmentBlockNumber  uint64
}

func NewInboxMultiplexer(backend InboxBackend, delayedMessagesRead uint64) InboxMultiplexer {
	return &inboxMultiplexer{
		backend:             backend,
		delayedMessagesRead: delayedMessagesRead,
	}
}

var SequencerAddress = common.HexToAddress("0xA4B000000000000000000073657175656e636572") // TODO

var invalidMessage *arbos.L1IncomingMessage = &arbos.L1IncomingMessage{
	Header: &arbos.L1IncomingMessageHeader{
		Kind: arbos.L1MessageType_Invalid,
	},
	L2msg: []byte{},
}

const BatchSegmentKindL2Message uint8 = 0
const BatchSegmentKindDelayedMessages uint8 = 1
const BatchSegmentKindAdvanceTimestamp uint8 = 2
const BatchSegmentKindAdvanceL1BlockNumber uint8 = 3

// This does *not* return parse errors, those are transformed into invalid messages
func (r *inboxMultiplexer) Pop() (*MessageWithMetadata, error) {
	if r.cachedSequencerMessage == nil {
		bytes, realErr := r.backend.PeekSequencerInbox()
		if realErr != nil {
			return nil, realErr
		}
		r.cachedSequencerMessageNum = r.backend.GetSequencerInboxPosition()
		r.cachedSequencerMessage = parseSequencerMessage(bytes)
	}
	currentPosition := r.backend.GetPositionWithinMessage()
	msg, positionRead, err := r.getNextMsg(currentPosition)
	// advance even if there was an error
	// we issue delayed messages untill reaching afterDelayedMessages
	if (positionRead+1 >= uint64(len(r.cachedSequencerMessage.segments))) && r.delayedMessagesRead >= r.cachedSequencerMessage.afterDelayedMessages {
		r.advanceSequencerMsg()
	} else {
		r.backend.SetPositionWithinMessage(positionRead + 1)
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
}

// Returns a message, the delayed messages being read up to if applicable, and any *parsing* error
func (r *inboxMultiplexer) getNextMsg(targetSegment uint64) (*MessageWithMetadata, uint64, error) {
	seqMsg := r.cachedSequencerMessage
	segmentNum := r.cachedSegmentNum
	timestamp := r.cachedSegmentTimestamp
	blockNumber := r.cachedSegmentBlockNumber
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
			advancing, err := rlp.NewStream(rd, 16).Uint()
			if err != nil {
				log.Warn("error parsing sequencer advancing segment", "err", err)
				continue
			}
			if segmentKind == BatchSegmentKindAdvanceTimestamp {
				timestamp += advancing
			} else if segmentKind == BatchSegmentKindAdvanceL1BlockNumber {
				blockNumber += advancing
			}
			segmentNum++
		} else if segmentNum < targetSegment {
			segmentNum++
		} else {
			break
		}
	}
	r.cachedSegmentNum = segmentNum
	r.cachedSegmentTimestamp = timestamp
	r.cachedSegmentBlockNumber = blockNumber
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
		segment = []byte{BatchSegmentKindDelayedMessages}
	} else {
		segment = seqMsg.segments[int(segmentNum)]
	}
	if len(segment) == 0 {
		return nil, segmentNum, errors.New("empty sequencer message segment")
	}
	segmentKind := segment[0]
	var msg *MessageWithMetadata
	if segmentKind == BatchSegmentKindL2Message {
		// L2 message
		var blockNumberHash common.Hash
		copy(blockNumberHash[:], math.U256Bytes(new(big.Int).SetUint64(blockNumber)))
		var timestampHash common.Hash
		copy(timestampHash[:], math.U256Bytes(new(big.Int).SetUint64(timestamp)))
		var requestId common.Hash
		// TODO: a consistent request id. Right now we just don't set the request id when it isn't needed.
		if len(segment) < 2 || segment[1] != arbos.L2MessageKind_SignedTx {
			requestId[0] = 1 << 6
			binary.BigEndian.PutUint64(requestId[(32-16):(32-8)], r.cachedSequencerMessageNum)
			binary.BigEndian.PutUint64(requestId[(32-8):], segmentNum)
		}
		msg = &MessageWithMetadata{
			Message: &arbos.L1IncomingMessage{
				Header: &arbos.L1IncomingMessageHeader{
					Kind:        arbos.L1MessageType_L2Message,
					Poster:      SequencerAddress,
					BlockNumber: blockNumberHash,
					Timestamp:   timestampHash,
					RequestId:   requestId,
					GasPriceL1:  common.Hash{},
				},
				L2msg: segment[1:],
			},
			DelayedMessagesRead: r.delayedMessagesRead,
		}
	} else if segmentKind == BatchSegmentKindDelayedMessages {
		if r.delayedMessagesRead >= seqMsg.afterDelayedMessages {
			log.Warn("attempt to access delayed msg", "msg", r.delayedMessagesRead, "segment_upto", seqMsg.afterDelayedMessages)
			msg = &MessageWithMetadata{
				Message:             invalidMessage,
				DelayedMessagesRead: seqMsg.afterDelayedMessages,
			}
		} else {
			data, realErr := r.backend.ReadDelayedInbox(r.delayedMessagesRead)
			if realErr != nil {
				return nil, segmentNum, realErr
			}
			delayed, parseErr := arbos.ParseIncomingL1Message(bytes.NewReader(data))
			if parseErr != nil {
				log.Warn("error parsing delayed message", "err", parseErr, "delayedMsg", r.delayedMessagesRead)
				delayed = invalidMessage
			}
			r.delayedMessagesRead += 1
			msg = &MessageWithMetadata{
				Message:             delayed,
				DelayedMessagesRead: r.delayedMessagesRead,
			}
		}
	} else {
		return nil, segmentNum, fmt.Errorf("bad sequencer message segment kind %v", segmentKind)
	}
	return msg, segmentNum, nil
}

func (r *inboxMultiplexer) DelayedMessagesRead() uint64 {
	return r.delayedMessagesRead
}
