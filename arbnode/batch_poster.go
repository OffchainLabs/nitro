package arbnode

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/andybalholm/brotli"
	"github.com/pkg/errors"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/offchainlabs/arbstate/arbstate"
	"github.com/offchainlabs/arbstate/das"
	"github.com/offchainlabs/arbstate/solgen/go/bridgegen"
)

type BatchPoster struct {
	client          L1Interface
	inbox           *InboxTracker
	streamer        *TransactionStreamer
	config          *BatchPosterConfig
	inboxContract   *bridgegen.SequencerInbox
	sequencesPosted uint64
	gasRefunder     common.Address
	transactOpts    *bind.TransactOpts
	das             das.DataAvailabilityService
}

type BatchPosterConfig struct {
	MaxBatchSize        int
	BatchPollDelay      time.Duration
	SubmissionSyncDelay time.Duration
	CompressionLevel    int
}

var DefaultBatchPosterConfig = BatchPosterConfig{
	MaxBatchSize:        500,
	BatchPollDelay:      time.Second / 10,
	SubmissionSyncDelay: time.Second,
	CompressionLevel:    brotli.DefaultCompression,
}

var TestBatchPosterConfig = BatchPosterConfig{
	MaxBatchSize:        10000,
	BatchPollDelay:      time.Millisecond * 10,
	SubmissionSyncDelay: time.Millisecond * 10,
	CompressionLevel:    2,
}

func NewBatchPoster(client L1Interface, inbox *InboxTracker, streamer *TransactionStreamer, config *BatchPosterConfig, contractAddress common.Address, refunder common.Address, transactOpts *bind.TransactOpts) (*BatchPoster, error) {
	inboxContract, err := bridgegen.NewSequencerInbox(contractAddress, client)
	if err != nil {
		return nil, err
	}
	return &BatchPoster{
		client:          client,
		inbox:           inbox,
		streamer:        streamer,
		config:          config,
		inboxContract:   inboxContract,
		sequencesPosted: 0,
		transactOpts:    transactOpts,
		gasRefunder:     refunder,
	}, nil
}

type batchSegments struct {
	compressedBuffer    *bytes.Buffer
	compressedWriter    *brotli.Writer
	rawSegments         [][]byte
	timestamp           uint64
	blockNum            uint64
	delayedMsg          uint64
	sizeLimit           int
	compressionLevel    int
	newUncompressedSize int
	lastCompressedSize  int
	trailingHeaders     int // how many trailing segments are headers
	isDone              bool
}

func newBatchSegments(firstDelayed uint64, config *BatchPosterConfig) *batchSegments {
	compressedBuffer := bytes.NewBuffer(make([]byte, 0, config.MaxBatchSize*2))
	return &batchSegments{
		compressedBuffer: compressedBuffer,
		compressedWriter: brotli.NewWriterLevel(compressedBuffer, config.CompressionLevel),
		sizeLimit:        config.MaxBatchSize - 40, // TODO
		compressionLevel: config.CompressionLevel,
		rawSegments:      make([][]byte, 0, 128),
		delayedMsg:       firstDelayed,
	}
}

func (s *batchSegments) recompressAll() error {
	s.compressedBuffer = bytes.NewBuffer(make([]byte, 0, s.sizeLimit*2))
	s.compressedWriter = brotli.NewWriterLevel(s.compressedBuffer, s.compressionLevel)
	s.newUncompressedSize = 0
	for _, segment := range s.rawSegments {
		err := s.addSegmentToCompressed(segment)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *batchSegments) testForOverflow() (bool, error) {
	// there is room, no need to flush
	if (s.lastCompressedSize + s.newUncompressedSize) < s.sizeLimit {
		return false, nil
	}
	// don't want to flush for headers
	if s.trailingHeaders > 0 {
		return false, nil
	}
	err := s.compressedWriter.Flush()
	if err != nil {
		return true, err
	}
	s.lastCompressedSize = s.compressedBuffer.Len()
	s.newUncompressedSize = 0
	if s.lastCompressedSize >= s.sizeLimit {
		return true, nil
	}
	return false, nil
}

func (s *batchSegments) close() error {
	s.rawSegments = s.rawSegments[:len(s.rawSegments)-s.trailingHeaders]
	s.trailingHeaders = 0
	err := s.recompressAll()
	if err != nil {
		return err
	}
	s.isDone = true
	return nil
}

func (s *batchSegments) addSegmentToCompressed(segment []byte) error {
	encoded, err := rlp.EncodeToBytes(segment)
	if err != nil {
		return err
	}
	lenWritten, err := s.compressedWriter.Write(encoded)
	s.newUncompressedSize += lenWritten
	return err
}

// returns false if segment was too large, error in case of real error
func (s *batchSegments) addSegment(segment []byte, isHeader bool) (bool, error) {
	if s.isDone {
		return false, nil
	}
	err := s.addSegmentToCompressed(segment)
	if err != nil {
		return false, err
	}
	overflow, err := s.testForOverflow()
	if err != nil {
		return false, err
	}
	if overflow {
		return false, s.close()
	}
	s.rawSegments = append(s.rawSegments, segment)
	if isHeader {
		s.trailingHeaders++
	} else {
		s.trailingHeaders = 0
	}
	return true, nil
}

func (s *batchSegments) addL2Msg(l2msg []byte) (bool, error) {
	segment := make([]byte, 1, len(l2msg)+1)
	segment[0] = arbstate.BatchSegmentKindL2Message
	segment = append(segment, l2msg...)
	return s.addSegment(segment, false)
}

func (s *batchSegments) prepareIntSegment(val uint64, segmentHeader byte) ([]byte, error) {
	segment := make([]byte, 1, 16)
	segment[0] = segmentHeader
	enc, err := rlp.EncodeToBytes(val)
	if err != nil {
		return nil, err
	}
	return append(segment, enc...), nil
}

func (s *batchSegments) maybeAddDiffSegment(base *uint64, newVal common.Hash, segmentHeader byte) (bool, error) {
	asBig := newVal.Big()
	if !asBig.IsUint64() {
		return false, errors.New("number not uint64")
	}
	asUint := asBig.Uint64()
	if asUint == *base {
		return true, nil
	}
	diff := asUint - *base
	seg, err := s.prepareIntSegment(diff, segmentHeader)
	if err != nil {
		return false, err
	}
	success, err := s.addSegment(seg, true)
	if success {
		*base = asUint
	}
	return success, err
}

func (s *batchSegments) addDelayedMessage() (bool, error) {
	segment := []byte{arbstate.BatchSegmentKindDelayedMessages}
	success, err := s.addSegment(segment, false)
	if (err == nil) && success {
		s.delayedMsg += 1
	}
	return success, err
}

func (s *batchSegments) AddMessage(msg *arbstate.MessageWithMetadata) (bool, error) {
	if s.isDone {
		return false, nil
	}
	if msg.DelayedMessagesRead > s.delayedMsg {
		if msg.DelayedMessagesRead != s.delayedMsg+1 {
			return false, fmt.Errorf("attempted to add delayed msg %d after %d", msg.DelayedMessagesRead, s.delayedMsg)
		}
		return s.addDelayedMessage()
	}
	success, err := s.maybeAddDiffSegment(&s.timestamp, msg.Message.Header.Timestamp, arbstate.BatchSegmentKindAdvanceTimestamp)
	if !success {
		return false, err
	}
	success, err = s.maybeAddDiffSegment(&s.blockNum, msg.Message.Header.BlockNumber, arbstate.BatchSegmentKindAdvanceL1BlockNumber)
	if !success {
		return false, err
	}
	return s.addL2Msg(msg.Message.L2msg)
}

func (s *batchSegments) IsDone() bool {
	return s.isDone
}

func (s *batchSegments) CloseAndGetBytes() ([]byte, error) {
	if !s.isDone {
		err := s.close()
		if err != nil {
			return nil, err
		}
	}
	if len(s.rawSegments) == 0 {
		return nil, nil
	}
	err := s.compressedWriter.Close()
	if err != nil {
		return nil, err
	}
	compressedBytes := s.compressedBuffer.Bytes()
	fullMsg := make([]byte, 1, len(compressedBytes)+1)
	fullMsg[0] = 0 // Header
	fullMsg = append(fullMsg, compressedBytes...)
	return fullMsg, nil
}

func (b *BatchPoster) lastSubmissionIsSynced() bool {
	batchcount, err := b.inbox.GetBatchCount()
	if err != nil {
		log.Warn("BatchPoster: batchcount failed", "err", err)
		return false
	}
	if batchcount < b.sequencesPosted {
		b.sequencesPosted = batchcount
		return false
	}
	if batchcount > b.sequencesPosted {
		log.Warn("detected unexpected sequences posted", "actual", batchcount, "expected", b.sequencesPosted)
		b.sequencesPosted = batchcount
		return true
	}
	return true
}

// TODO make sure we detect end of block!
func (b *BatchPoster) postSequencerBatch() error {
	for !b.lastSubmissionIsSynced() {
		log.Warn("BatchPoster: not in sync", "sequencedPosted", b.sequencesPosted)
		<-time.After(b.config.SubmissionSyncDelay)
	}
	var prevBatchMeta BatchMetadata
	if b.sequencesPosted > 0 {
		var err error
		prevBatchMeta, err = b.inbox.GetBatchMetadata(b.sequencesPosted - 1)
		if err != nil {
			return err
		}
	}
	segments := newBatchSegments(prevBatchMeta.DelayedMessageCount, b.config)
	msgToPost := prevBatchMeta.MessageCount
	msgCount, err := b.streamer.GetMessageCount()
	if err != nil {
		return err
	}
	for msgToPost < msgCount {
		msg, err := b.streamer.GetMessage(msgToPost)
		if err != nil {
			log.Error("error getting message from streamer", "error", err)
			break
		}
		success, err := segments.AddMessage(&msg)
		if err != nil {
			log.Error("error adding message to batch", "error", err)
			break
		}
		if !success {
			break
		}
		msgToPost++
	}
	sequencerMsg, err := segments.CloseAndGetBytes()
	if err != nil {
		return err
	}
	if sequencerMsg == nil {
		log.Debug("BatchPoster: batch nil", "sequence nr.", b.sequencesPosted, "from", prevBatchMeta.MessageCount, "prev delayed", prevBatchMeta.DelayedMessageCount)
		return nil
	}

	// Careful, this is an interface.
	if b.das != nil {
		fullMsg := make([]byte, 1) // TODO capacity
		fullMsg[0] = 'd'           // Header
		cert, _, _ := b.das.Store(sequencerMsg)
		// TODO fallback on error
		sequencerMsg = append(fullMsg, cert...)
	}

	// TODO send this batch to DAS instead
	// If the certificate comes back, write it to the contract instead, otherwise write the inbox as-is
	// Give the certificate a different type?
	_, err = b.inboxContract.AddSequencerL2BatchFromOrigin(b.transactOpts, new(big.Int).SetUint64(b.sequencesPosted), sequencerMsg, new(big.Int).SetUint64(segments.delayedMsg), b.gasRefunder)

	if err == nil {
		b.sequencesPosted++
		log.Info("BatchPoster: batch sent", "sequence nr.", b.sequencesPosted, "from", prevBatchMeta.MessageCount, "to", msgToPost, "prev delayed", prevBatchMeta.DelayedMessageCount, "current delayed", segments.delayedMsg, "total segments", len(segments.rawSegments))
	}
	return err
}

func (b *BatchPoster) Start(ctx context.Context) {
	go (func() {
		for {
			err := b.postSequencerBatch()
			if err != nil {
				log.Error("error posting batch", "err", err.Error())
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(b.config.BatchPollDelay):
			}
		}
	})()
}
