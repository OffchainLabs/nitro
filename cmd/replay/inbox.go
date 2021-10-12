package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/andybalholm/brotli"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/offchainlabs/arbstate/arbos"
	"github.com/offchainlabs/arbstate/wavmio"
)

type InboxReader interface {
	// Returns a message and if it must end the block
	Peek() (*arbos.L1IncomingMessage, bool, error)
	Advance()
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

func parseSequencerMessage(data []byte) sequencerMessage {
	if len(data) < 32 {
		panic("sequencer message missing L1 header")
	}
	minTimestamp := binary.LittleEndian.Uint64(data[:8])
	maxTimestamp := binary.LittleEndian.Uint64(data[8:16])
	minL1Block := binary.LittleEndian.Uint64(data[16:24])
	maxL1Block := binary.LittleEndian.Uint64(data[24:32])
	afterDelayedMessages := binary.LittleEndian.Uint64(data[32:40])
	reader := io.LimitReader(brotli.NewReader(bytes.NewReader(data[40:])), maxDecompressedLen)
	var segments [][]byte
	err := rlp.Decode(reader, &segments)
	if err != nil {
		fmt.Printf("Error parsing sequencer message segments: %s\n", err.Error())
		segments = nil
	}
	return sequencerMessage{
		minTimestamp:         minTimestamp,
		maxTimestamp:         maxTimestamp,
		minL1Block:           minL1Block,
		maxL1Block:           maxL1Block,
		afterDelayedMessages: afterDelayedMessages,
		segments:             segments,
	}
}

type AdvanceAction uint8

const (
	AdvanceUnknown AdvanceAction = iota
	AdvanceDelayedMessage
	AdvanceSegment
	AdvanceMessage
)

type inboxReader struct {
	delayedMessagesRead uint64
	advanceAction       AdvanceAction
	advanceSegmentTo    uint64
}

func NewInboxReader(delayedMessagesRead uint64) InboxReader {
	return &inboxReader{
		delayedMessagesRead: delayedMessagesRead,
		advanceAction:       AdvanceUnknown,
		advanceSegmentTo:    0,
	}
}

var sequencerAddress = common.HexToAddress("0xA4B000000000000000000073657175656e636572") // TODO

func (r *inboxReader) Peek() (*arbos.L1IncomingMessage, bool, error) {
	seqMsg := parseSequencerMessage(wavmio.ReadInboxMessage())
	segmentNum := wavmio.GetPositionWithinMessage()
	var timestamp uint64
	var blockNumber uint64
	for {
		if segmentNum >= uint64(len(seqMsg.segments)) {
			break
		}
		segment := seqMsg.segments[int(segmentNum)]
		if len(segment) == 0 {
			segmentNum++
			continue
		}
		segmentKind := segment[0]
		if segmentKind == 2 || segmentKind == 3 {
			var advancing uint64
			rd := bytes.NewReader(segment[1:])
			err := rlp.Decode(rd, &advancing)
			if err != nil {
				fmt.Printf("Error parsing advancing segment: %s\n", err)
				continue
			}
			if segmentKind == 2 {
				timestamp += advancing
			} else if segmentKind == 3 {
				blockNumber += advancing
			}
			segmentNum++
		} else {
			break
		}
	}
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
		if r.delayedMessagesRead < seqMsg.afterDelayedMessages {
			data := wavmio.ReadDelayedInboxMessage(r.delayedMessagesRead)
			delayed, err := arbos.ParseIncomingL1Message(bytes.NewReader(data))
			endOfMessage := r.delayedMessagesRead+1 >= seqMsg.afterDelayedMessages
			if endOfMessage {
				r.advanceAction = AdvanceMessage
			} else {
				r.advanceAction = AdvanceDelayedMessage
			}
			return delayed, endOfMessage, err
		}
		r.advanceAction = AdvanceMessage
		return nil, false, fmt.Errorf("reading end of sequencer message (size %v)", len(seqMsg.segments))
	}
	endOfMessage := segmentNum+1 >= uint64(len(seqMsg.segments))
	if endOfMessage {
		r.advanceAction = AdvanceMessage
	} else {
		r.advanceAction = AdvanceSegment
		r.advanceSegmentTo = segmentNum + 1
	}
	segment := seqMsg.segments[int(segmentNum)]
	if len(segment) == 0 {
		return nil, false, errors.New("empty sequencer message segment")
	}
	segmentKind := segment[0]
	if segmentKind == 0 {
		// L2 message
		var blockNumberHash common.Hash
		binary.LittleEndian.PutUint64(blockNumberHash[(32-8):], blockNumber)
		var timestampHash common.Hash
		binary.LittleEndian.PutUint64(timestampHash[(32-8):], timestamp)
		var requestId common.Hash
		requestId[0] = 1 << 6
		binary.LittleEndian.PutUint64(requestId[(32-16):(32-8)], wavmio.GetInboxPosition())
		binary.LittleEndian.PutUint64(requestId[(32-8):], segmentNum)
		msg := &arbos.L1IncomingMessage{
			Header: &arbos.L1IncomingMessageHeader{
				Kind:        arbos.L1MessageType_L2Message,
				Sender:      sequencerAddress,
				BlockNumber: blockNumberHash,
				Timestamp:   timestampHash,
				RequestId:   requestId,
				GasPriceL1:  common.Hash{},
			},
			L2msg: segment[1:],
		}
		return msg, endOfMessage, nil
	} else if segmentKind == 1 {
		// Delayed message reading
		var reading uint64
		rd := bytes.NewReader(segment[1:])
		err := rlp.Decode(rd, &reading)
		if err != nil {
			return nil, false, err
		}
		newRead := r.delayedMessagesRead + reading
		if newRead <= r.delayedMessagesRead || newRead > seqMsg.afterDelayedMessages {
			return nil, false, errors.New("bad delayed message reading count")
		}
		endOfSegment := r.delayedMessagesRead+1 >= newRead
		if !endOfSegment {
			r.advanceAction = AdvanceDelayedMessage
		}
		data := wavmio.ReadDelayedInboxMessage(r.delayedMessagesRead)
		delayed, err := arbos.ParseIncomingL1Message(bytes.NewReader(data))
		return delayed, endOfSegment, err
	} else {
		return nil, false, fmt.Errorf("bad sequencer message segment kind %v", segmentKind)
	}
}

func (r *inboxReader) Advance() {
	if r.advanceAction == AdvanceUnknown {
		r.Peek()
		if r.advanceAction == AdvanceUnknown {
			panic("Failed to get advance action")
		}
	}
	if r.advanceAction == AdvanceDelayedMessage {
		r.delayedMessagesRead += 1
	} else if r.advanceAction == AdvanceSegment {
		if r.advanceSegmentTo <= wavmio.GetPositionWithinMessage() {
			panic("Attempted to advance segment but target <= position")
		}
		wavmio.SetPositionWithinMessage(r.advanceSegmentTo)
	} else if r.advanceAction == AdvanceMessage {
		wavmio.AdvanceInboxMessage()
	} else {
		panic(fmt.Sprintf("Unknown advance action %v", r.advanceAction))
	}
	r.advanceAction = AdvanceUnknown
	r.advanceSegmentTo = 0
}

func (r *inboxReader) DelayedMessagesRead() uint64 {
	return r.delayedMessagesRead
}
