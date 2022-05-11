// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbnode

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/andybalholm/brotli"
	"github.com/pkg/errors"
	flag "github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/das"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

type BatchPoster struct {
	stopwaiter.StopWaiter
	l1Reader      *L1Reader
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
	Enable               bool          `koanf:"enable"`
	MaxBatchSize         int           `koanf:"max-size"`
	MaxBatchPostInterval time.Duration `koanf:"max-interval"`
	BatchPollDelay       time.Duration `koanf:"poll-delay"`
	PostingErrorDelay    time.Duration `koanf:"error-delay"`
	CompressionLevel     int           `koanf:"compression-level"`
	DASRetentionPeriod   time.Duration `koanf:"das-retention-period"`
}

func BatchPosterConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".enable", DefaultBatchPosterConfig.Enable, "enable posting batches to l1")
	f.Int(prefix+".max-size", DefaultBatchPosterConfig.MaxBatchSize, "maximum batch size")
	f.Duration(prefix+".max-interval", DefaultBatchPosterConfig.MaxBatchPostInterval, "maximum batch posting interval")
	f.Duration(prefix+".poll-delay", DefaultBatchPosterConfig.BatchPollDelay, "how long to delay after successfully posting batch")
	f.Duration(prefix+".error-delay", DefaultBatchPosterConfig.PostingErrorDelay, "how long to delay after error posting batch")
	f.Int(prefix+".compression-level", DefaultBatchPosterConfig.CompressionLevel, "batch compression level")
	f.Duration(prefix+".das-retention-period", DefaultBatchPosterConfig.DASRetentionPeriod, "In AnyTrust mode, the period which DASes are requested to retain the stored batches.")
}

var DefaultBatchPosterConfig = BatchPosterConfig{
	Enable:               false,
	MaxBatchSize:         100000,
	BatchPollDelay:       time.Second * 10,
	PostingErrorDelay:    time.Second * 10,
	MaxBatchPostInterval: time.Hour,
	CompressionLevel:     brotli.DefaultCompression,
	DASRetentionPeriod:   time.Hour * 24 * 15,
}

var TestBatchPosterConfig = BatchPosterConfig{
	Enable:               true,
	MaxBatchSize:         100000,
	BatchPollDelay:       time.Millisecond * 10,
	PostingErrorDelay:    time.Millisecond * 10,
	MaxBatchPostInterval: 0,
	CompressionLevel:     2,
	DASRetentionPeriod:   time.Hour * 24 * 15,
}

func NewBatchPoster(l1Reader *L1Reader, inbox *InboxTracker, streamer *TransactionStreamer, config *BatchPosterConfig, contractAddress common.Address, refunder common.Address, transactOpts *bind.TransactOpts, das das.DataAvailabilityService) (*BatchPoster, error) {
	inboxContract, err := bridgegen.NewSequencerInbox(contractAddress, l1Reader.Client())
	if err != nil {
		return nil, err
	}
	return &BatchPoster{
		l1Reader:      l1Reader,
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
	if config.MaxBatchSize <= 40 {
		panic("MaxBatchSize too small")
	}
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
	if overflow || len(s.rawSegments) >= arbstate.MaxSegmentsPerSequencerMessage {
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

func (s *batchSegments) maybeAddDiffSegment(base *uint64, newVal uint64, segmentHeader byte) (bool, error) {
	if newVal == *base {
		return true, nil
	}
	diff := newVal - *base
	seg, err := s.prepareIntSegment(diff, segmentHeader)
	if err != nil {
		return false, err
	}
	success, err := s.addSegment(seg, true)
	if success {
		*base = newVal
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

func (b *BatchPoster) maybePostSequencerBatch(ctx context.Context, timeSinceBatchPosted time.Duration) (*types.Transaction, error) {
	batchSeqNum, err := b.inbox.GetBatchCount()
	if err != nil {
		return nil, err
	}
	inboxContractCount, err := b.inboxContract.BatchCount(&bind.CallOpts{Context: ctx, Pending: true})
	if err != nil {
		return nil, err
	}
	if !arbmath.BigEquals(inboxContractCount, arbmath.UintToBig(batchSeqNum)) {
		// If it's been under a minute since the last batch was posted, and the inbox tracker is exactly one batch behind,
		// then there isn't an error. We're just waiting for the inbox tracker to read the most recently posted batch.
		if timeSinceBatchPosted <= time.Minute && arbmath.BigEquals(inboxContractCount, arbmath.UintToBig(batchSeqNum+1)) {
			return nil, nil
		}
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
	forcePostBatch := timeSinceBatchPosted >= b.config.MaxBatchPostInterval
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
		cert, err := b.das.Store(ctx, sequencerMsg, uint64(time.Now().Add(b.config.DASRetentionPeriod).Unix()), []byte{}) // BUGBUG
		if err != nil {
			log.Warn("Unable to batch to DAS, falling back to storing data on chain", "err", err)
		} else {
			sequencerMsg = das.Serialize(cert)
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
		tx, err := b.maybePostSequencerBatch(ctx, time.Since(lastBatchPosted))
		if err != nil {
			b.building = nil
			log.Error("error posting batch", "err", err)
			return b.config.PostingErrorDelay
		}
		if tx != nil {
			b.building = nil
			_, err = b.l1Reader.WaitForTxApproval(ctx, tx)
			if err != nil {
				log.Error("failed ensuring batch tx succeeded", "err", err)
			} else {
				lastBatchPosted = time.Now()
			}
		}
		return b.config.BatchPollDelay
	})
}
