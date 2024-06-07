// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbstate

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/offchainlabs/nitro/arbcompress"
	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbos/l1pricing"
	"github.com/offchainlabs/nitro/arbstate/daprovider"
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

type sequencerMessage struct {
	minTimestamp         uint64
	maxTimestamp         uint64
	minL1Block           uint64
	maxL1Block           uint64
	afterDelayedMessages uint64
	segments             [][]byte
}

const MaxDecompressedLen int = 1024 * 1024 * 16 // 16 MiB
const maxZeroheavyDecompressedLen = 101*MaxDecompressedLen/100 + 64
const MaxSegmentsPerSequencerMessage = 100 * 1024

func parseSequencerMessage(ctx context.Context, batchNum uint64, batchBlockHash common.Hash, data []byte, dapReaders []daprovider.Reader, keysetValidationMode daprovider.KeysetValidationMode) (*sequencerMessage, error) {
	if len(data) < 40 {
		return nil, errors.New("sequencer message missing L1 header")
	}
	parsedMsg := &sequencerMessage{
		minTimestamp:         binary.BigEndian.Uint64(data[:8]),
		maxTimestamp:         binary.BigEndian.Uint64(data[8:16]),
		minL1Block:           binary.BigEndian.Uint64(data[16:24]),
		maxL1Block:           binary.BigEndian.Uint64(data[24:32]),
		afterDelayedMessages: binary.BigEndian.Uint64(data[32:40]),
		segments:             [][]byte{},
	}
	payload := data[40:]

	// Stage 0: Check if our node is out of date and we don't understand this batch type
	// If the parent chain sequencer inbox smart contract authenticated this batch,
	// an unknown header byte must mean that this node is out of date,
	// because the smart contract understands the header byte and this node doesn't.
	if len(payload) > 0 && daprovider.IsL1AuthenticatedMessageHeaderByte(payload[0]) && !daprovider.IsKnownHeaderByte(payload[0]) {
		return nil, fmt.Errorf("%w: batch has unsupported authenticated header byte 0x%02x", arbosState.ErrFatalNodeOutOfDate, payload[0])
	}

	// Stage 1: Extract the payload from any data availability header.
	// It's important that multiple DAS strategies can't both be invoked in the same batch,
	// as these headers are validated by the sequencer inbox and not other DASs.
	// We try to extract payload from the first occuring valid DA reader in the dapReaders list
	if len(payload) > 0 {
		foundDA := false
		var err error
		for _, dapReader := range dapReaders {
			if dapReader != nil && dapReader.IsValidHeaderByte(payload[0]) {
				payload, err = dapReader.RecoverPayloadFromBatch(ctx, batchNum, batchBlockHash, data, nil, keysetValidationMode != daprovider.KeysetDontValidate)
				if err != nil {
					// Matches the way keyset validation was done inside DAS readers i.e logging the error
					//  But other daproviders might just want to return the error
					if errors.Is(err, daprovider.ErrSeqMsgValidation) && daprovider.IsDASMessageHeaderByte(payload[0]) {
						logLevel := log.Error
						if keysetValidationMode == daprovider.KeysetPanicIfInvalid {
							logLevel = log.Crit
						}
						logLevel(err.Error())
					} else {
						return nil, err
					}
				}
				if payload == nil {
					return parsedMsg, nil
				}
				foundDA = true
				break
			}
		}

		if !foundDA {
			if daprovider.IsDASMessageHeaderByte(payload[0]) {
				log.Error("No DAS Reader configured, but sequencer message found with DAS header")
			} else if daprovider.IsBlobHashesHeaderByte(payload[0]) {
				return nil, daprovider.ErrNoBlobReader
			}
			// TODO (Diego)
			// else if IsBlobHashesHeaderByte(payload[0]) {
			// 	return nil, ErrNoBlobReader
			// } else if IsCelestiaMessageHeaderByte(payload[0]) {
			// 	log.Error("No Celestia Reader configured, but sequencer message found with Celestia header")
			// }
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
				if len(parsedMsg.segments) >= MaxSegmentsPerSequencerMessage {
					log.Warn("too many segments in sequence batch")
					break
				}
				parsedMsg.segments = append(parsedMsg.segments, segment)
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

// TODO (Diego):
// func NewDAProviderCelestia(celestia celestiaTypes.DataAvailabilityReader) *dAProviderForCelestia {
// 	return &dAProviderForCelestia{
// 		celestia: celestia,
// 	}
// }

// type dAProviderForCelestia struct {
// 	celestia celestiaTypes.DataAvailabilityReader
// }

// func (c *dAProviderForCelestia) IsValidHeaderByte(headerByte byte) bool {
// 	return IsCelestiaMessageHeaderByte(headerByte)
// }

// func (c *dAProviderForCelestia) RecoverPayloadFromBatch(
// 	ctx context.Context,
// 	batchNum uint64,
// 	batchBlockHash common.Hash,
// 	sequencerMsg []byte,
// 	preimages map[arbutil.PreimageType]map[common.Hash][]byte,
// 	keysetValidationMode KeysetValidationMode,
// ) ([]byte, error) {
// 	return RecoverPayloadFromCelestiaBatch(ctx, batchNum, sequencerMsg, c.celestia, preimages)
// }

// func RecoverPayloadFromCelestiaBatch(
// 	ctx context.Context,
// 	batchNum uint64,
// 	sequencerMsg []byte,
// 	celestiaReader celestiaTypes.DataAvailabilityReader,
// 	preimages map[arbutil.PreimageType]map[common.Hash][]byte,
// ) ([]byte, error) {
// 	var sha256Preimages map[common.Hash][]byte
// 	if preimages != nil {
// 		if preimages[arbutil.Sha2_256PreimageType] == nil {
// 			preimages[arbutil.Sha2_256PreimageType] = make(map[common.Hash][]byte)
// 		}
// 		sha256Preimages = preimages[arbutil.Sha2_256PreimageType]
// 	}

// 	buf := bytes.NewBuffer(sequencerMsg[40:])

// 	header, err := buf.ReadByte()
// 	if err != nil {
// 		log.Error("Couldn't deserialize Celestia header byte", "err", err)
// 		return nil, nil
// 	}
// 	if !IsCelestiaMessageHeaderByte(header) {
// 		log.Error("Couldn't deserialize Celestia header byte", "err", errors.New("tried to deserialize a message that doesn't have the Celestia header"))
// 		return nil, nil
// 	}

// 	recordPreimage := func(key common.Hash, value []byte) {
// 		sha256Preimages[key] = value
// 	}

// 	blobPointer := celestiaTypes.BlobPointer{}
// 	blobBytes := buf.Bytes()
// 	err = blobPointer.UnmarshalBinary(blobBytes)
// 	if err != nil {
// 		log.Error("Couldn't unmarshal Celestia blob pointer", "err", err)
// 		return nil, nil
// 	}

// 	payload, squareData, err := celestiaReader.Read(ctx, &blobPointer)
// 	if err != nil {
// 		log.Error("Failed to resolve blob pointer from celestia", "err", err)
// 		return nil, err
// 	}

// 	// we read a batch that is to be discarded, so we return the empty batch
// 	if len(payload) == 0 {
// 		return payload, nil
// 	}

// 	if sha256Preimages != nil {
// 		if squareData == nil {
// 			log.Error("squareData is nil, read from replay binary, but preimages are empty")
// 			return nil, err
// 		}

// 		odsSize := squareData.SquareSize / 2
// 		rowIndex := squareData.StartRow
// 		for _, row := range squareData.Rows {
// 			treeConstructor := tree.NewConstructor(recordPreimage, odsSize)
// 			root, err := tree.ComputeNmtRoot(treeConstructor, uint(rowIndex), row)
// 			if err != nil {
// 				log.Error("Failed to compute row root", "err", err)
// 				return nil, err
// 			}

// 			rowRootMatches := bytes.Equal(squareData.RowRoots[rowIndex], root)
// 			if !rowRootMatches {
// 				log.Error("Row roots do not match", "eds row root", squareData.RowRoots[rowIndex], "calculated", root)
// 				log.Error("Row roots", "row_roots", squareData.RowRoots)
// 				return nil, err
// 			}
// 			rowIndex += 1
// 		}

// 		rowsCount := len(squareData.RowRoots)
// 		slices := make([][]byte, rowsCount+rowsCount)
// 		copy(slices[0:rowsCount], squareData.RowRoots)
// 		copy(slices[rowsCount:], squareData.ColumnRoots)

// 		dataRoot := tree.HashFromByteSlices(recordPreimage, slices)

// 		dataRootMatches := bytes.Equal(dataRoot, blobPointer.DataRoot[:])
// 		if !dataRootMatches {
// 			log.Error("Data Root do not match", "blobPointer data root", blobPointer.DataRoot, "calculated", dataRoot)
// 			return nil, nil
// 		}
// 	}

// 	return payload, nil
// }

type inboxMultiplexer struct {
	backend                   InboxBackend
	delayedMessagesRead       uint64
	dapReaders                []daprovider.Reader
	cachedSequencerMessage    *sequencerMessage
	cachedSequencerMessageNum uint64
	cachedSegmentNum          uint64
	cachedSegmentTimestamp    uint64
	cachedSegmentBlockNumber  uint64
	cachedSubMessageNumber    uint64
	keysetValidationMode      daprovider.KeysetValidationMode
}

func NewInboxMultiplexer(backend InboxBackend, delayedMessagesRead uint64, dapReaders []daprovider.Reader, keysetValidationMode daprovider.KeysetValidationMode) arbostypes.InboxMultiplexer {
	return &inboxMultiplexer{
		backend:              backend,
		delayedMessagesRead:  delayedMessagesRead,
		dapReaders:           dapReaders,
		keysetValidationMode: keysetValidationMode,
	}
}

const BatchSegmentKindL2Message uint8 = 0
const BatchSegmentKindL2MessageBrotli uint8 = 1
const BatchSegmentKindDelayedMessages uint8 = 2
const BatchSegmentKindAdvanceTimestamp uint8 = 3
const BatchSegmentKindAdvanceL1BlockNumber uint8 = 4

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
		r.cachedSequencerMessage, err = parseSequencerMessage(ctx, r.cachedSequencerMessageNum, batchBlockHash, bytes, r.dapReaders, r.keysetValidationMode)
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
func (r *inboxMultiplexer) getNextMsg() (*arbostypes.MessageWithMetadata, error) {
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
		if r.delayedMessagesRead >= seqMsg.afterDelayedMessages {
			if segmentNum < uint64(len(seqMsg.segments)) {
				log.Warn(
					"attempt to read past batch delayed message count",
					"delayedMessagesRead", r.delayedMessagesRead,
					"batchAfterDelayedMessages", seqMsg.afterDelayedMessages,
				)
			}
			msg = &arbostypes.MessageWithMetadata{
				Message:             arbostypes.InvalidL1Message,
				DelayedMessagesRead: seqMsg.afterDelayedMessages,
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
