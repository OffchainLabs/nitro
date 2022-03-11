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
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/das"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/util"
)

type BatchPoster struct {
	util.StopWaiter
	client        arbutil.L1Interface
	inbox         *InboxTracker
	streamer      *TransactionStreamer
	config        *BatchPosterConfig
	inboxContract *bridgegen.SequencerInbox
	gasRefunder   common.Address
	transactOpts  *bind.TransactOpts
	building      *buildingBatch
	das           das.DataAvailabilityService
}

type BatchPosterConfig struct {
	MaxBatchSize         int
	MaxBatchPostInterval time.Duration
	BatchPollDelay       time.Duration
	PostingErrorDelay    time.Duration
	CompressionLevel     int
}

var DefaultBatchPosterConfig = BatchPosterConfig{
	MaxBatchSize:         500,
	BatchPollDelay:       time.Second,
	PostingErrorDelay:    time.Second * 5,
	MaxBatchPostInterval: time.Minute,
	CompressionLevel:     brotli.DefaultCompression,
}

var TestBatchPosterConfig = BatchPosterConfig{
	MaxBatchSize:         10000,
	BatchPollDelay:       time.Millisecond * 10,
	PostingErrorDelay:    time.Millisecond * 10,
	MaxBatchPostInterval: 0,
	CompressionLevel:     2,
}

func NewBatchPoster(client arbutil.L1Interface, inbox *InboxTracker, streamer *TransactionStreamer, config *BatchPosterConfig, contractAddress common.Address, refunder common.Address, transactOpts *bind.TransactOpts, das das.DataAvailabilityService) (*BatchPoster, error) {
	inboxContract, err := bridgegen.NewSequencerInbox(contractAddress, client)
	if err != nil {
		return nil, err
	}
	return &BatchPoster{
		client:        client,
		inbox:         inbox,
		streamer:      streamer,
		config:        config,
		inboxContract: inboxContract,
		transactOpts:  transactOpts,
		gasRefunder:   refunder,
		das:           das,
	}, nil
}

var errBatchAlreadyClosed = errors.New("batch segments already closed")

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

type buildingBatch struct {
	segments    *batchSegments
	batchSeqNum uint64
	msgCount    arbutil.MessageIndex
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
		return false, errBatchAlreadyClosed
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
		return false, errBatchAlreadyClosed
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

func (b *BatchPoster) maybePostSequencerBatch(ctx context.Context, forcePostBatch bool) (*types.Transaction, error) {
	batchSeqNum, err := b.inbox.GetBatchCount()
	if err != nil {
		return nil, err
	}
	inboxContractCount, err := b.inboxContract.BatchCount(&bind.CallOpts{Context: ctx, Pending: true})
	if err != nil {
		return nil, err
	}
	if inboxContractCount.Cmp(new(big.Int).SetUint64(batchSeqNum)) != 0 {
		return nil, fmt.Errorf("inbox tracker not synced: contract has %v batches but inbox tracker has %v", inboxContractCount, batchSeqNum)
	}
	var prevBatchMeta BatchMetadata
	if batchSeqNum > 0 {
		var err error
		prevBatchMeta, err = b.inbox.GetBatchMetadata(batchSeqNum - 1)
		if err != nil {
			return nil, err
		}
	}
	if b.building == nil || b.building.batchSeqNum != batchSeqNum {
		b.building = &buildingBatch{
			segments:    newBatchSegments(prevBatchMeta.DelayedMessageCount, b.config),
			msgCount:    prevBatchMeta.MessageCount,
			batchSeqNum: batchSeqNum,
		}
	}
	msgCount, err := b.streamer.GetMessageCount()
	if err != nil {
		return nil, err
	}
	for b.building.msgCount < msgCount {
		msg, err := b.streamer.GetMessage(b.building.msgCount)
		if err != nil {
			log.Error("error getting message from streamer", "error", err)
			break
		}
		success, err := b.building.segments.AddMessage(&msg)
		if err != nil {
			log.Error("error adding message to batch", "error", err)
			break
		}
		if !success {
			forcePostBatch = true // this batch is full
			break
		}
		b.building.msgCount++
	}
	if !forcePostBatch {
		// the batch isn't full yet and we've posted a batch recently
		// don't post anything for now
		return nil, nil
	}
	sequencerMsg, err := b.building.segments.CloseAndGetBytes()
	if err != nil {
		return nil, err
	}
	if sequencerMsg == nil {
		log.Debug("BatchPoster: batch nil", "sequence nr.", batchSeqNum, "from", prevBatchMeta.MessageCount, "prev delayed", prevBatchMeta.DelayedMessageCount)
		b.building = nil // a closed batchSegments can't be reused
		return nil, nil
	}

	if b.das != nil {
		cert, err := b.das.Store(ctx, sequencerMsg)
		if err != nil {
			log.Warn("Unable to batch to DAS, falling back to storing data on chain", "err", err)
		} else {
			sequencerMsg = das.Serialize(*cert)
		}
	}

	txOpts := *b.transactOpts
	txOpts.Context = ctx
	tx, err := b.inboxContract.AddSequencerL2BatchFromOrigin(&txOpts, new(big.Int).SetUint64(batchSeqNum), sequencerMsg, new(big.Int).SetUint64(b.building.segments.delayedMsg), b.gasRefunder)
	if err == nil {
		log.Info("BatchPoster: batch sent", "sequence nr.", batchSeqNum, "from", prevBatchMeta.MessageCount, "to", b.building.msgCount, "prev delayed", prevBatchMeta.DelayedMessageCount, "current delayed", b.building.segments.delayedMsg, "total segments", len(b.building.segments.rawSegments))
	}
	return tx, err
}

func (b *BatchPoster) Start(ctxIn context.Context) {
	b.StopWaiter.Start(ctxIn)
	var lastBatchPosted time.Time
	b.CallIteratively(func(ctx context.Context) time.Duration {
		tx, err := b.maybePostSequencerBatch(ctx, time.Since(lastBatchPosted) >= b.config.MaxBatchPostInterval)
		if err != nil {
			b.building = nil
			log.Error("error posting batch", "err", err)
			return b.config.PostingErrorDelay
		}
		if tx != nil {
			b.building = nil
			_, err = arbutil.EnsureTxSucceededWithTimeout(ctx, b.client, tx, time.Minute)
			if err != nil {
				log.Error("failed ensuring batch tx succeeded", "err", err)
			} else {
				lastBatchPosted = time.Now()
			}
		}
		return b.config.BatchPollDelay
	})
}
