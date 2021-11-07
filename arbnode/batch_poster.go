package arbnode

import (
	"bytes"
	"math/big"
	"time"

	"github.com/andybalholm/brotli"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/offchainlabs/arbstate/arbstate"
	"github.com/offchainlabs/arbstate/solgen/go/bridgegen"
	"github.com/pkg/errors"
)

type BatchPoster struct {
	client          L1Interface
	inbox           *InboxReaderDb
	streamer        *InboxState
	config          *BatchPosterConfig
	inboxContract   *bridgegen.SequencerInbox
	sequencesPosted uint64
	gasRefunder     common.Address
	transactOpts    *bind.TransactOpts
	chanStop        chan struct{}
}

type BatchPosterConfig struct {
	MaxBatchSize int
}

var DefaultBatchPosterConfig = BatchPosterConfig{
	MaxBatchSize: 500,
}

func NewBatchPoster(client L1Interface, inbox *InboxReaderDb, streamer *InboxState, config *BatchPosterConfig, contractAddress common.Address, refunder common.Address, transactOpts *bind.TransactOpts) (*BatchPoster, error) {
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
		chanStop:        make(chan struct{}, 1),
	}, nil
}

type batchSegments struct {
	compressedBuffer *bytes.Buffer
	compressedWriter *brotli.Writer
	rawSegments      [][]byte
	timestamp        uint64
	blockNum         uint64
	delayedMsg       uint64
	pendingDelayed   uint64
	sizeLimit        int
	currentSize      int
	trailingHeaders  int // how many trailing segments are headers
	isDone           bool
}

func newBatchSegments(firstDelayed uint64, config *BatchPosterConfig) *batchSegments {
	compressedBuffer := bytes.NewBuffer(make([]byte, 0, config.MaxBatchSize*2))
	return &batchSegments{
		compressedBuffer: compressedBuffer,
		compressedWriter: brotli.NewWriter(compressedBuffer),
		sizeLimit:        config.MaxBatchSize - 40, // TODO
		rawSegments:      make([][]byte, 128),
		delayedMsg:       firstDelayed,
		pendingDelayed:   firstDelayed,
	}
}

func (s *batchSegments) close(mustRecompress bool) error {
	err := s.testAddDelayedMessage()
	if err != nil {
		return err
	}
	s.rawSegments = s.rawSegments[:len(s.rawSegments)-s.trailingHeaders]
	if s.trailingHeaders > 0 {
		mustRecompress = true
	}
	s.trailingHeaders = 0
	if mustRecompress {
		s.compressedBuffer = bytes.NewBuffer(make([]byte, 0, s.sizeLimit*2))
		s.compressedWriter = brotli.NewWriter(s.compressedBuffer)
		s.currentSize = 0
		for _, segment := range s.rawSegments {
			len, err := s.compressedWriter.Write(segment)
			if err != nil {
				return err
			}
			s.currentSize += len
		}
	}
	s.isDone = true
	return nil
}

// This segment will be added even if it's overbound
func (s *batchSegments) unconditionalAddSegment(segment []byte) error {
	lenWritten, err := s.compressedWriter.Write(segment)
	if err != nil {
		return err
	}
	s.currentSize += lenWritten
	s.trailingHeaders = 0
	s.rawSegments = append(s.rawSegments, segment)
	return nil
}

// returns false if segment was too large, error in case of real error
func (s *batchSegments) addSegment(segment []byte, isHeader bool) (bool, error) {
	if s.isDone {
		return false, nil
	}
	lenWritten, err := s.compressedWriter.Write(segment)
	if err != nil {
		return false, err
	}
	if (lenWritten + s.currentSize) > s.sizeLimit {
		return false, s.close(true)
	}
	s.rawSegments = append(s.rawSegments, segment)
	s.currentSize += lenWritten
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
	if asUint <= *base {
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

func (s *batchSegments) testAddDelayedMessage() error {
	if s.pendingDelayed <= s.delayedMsg {
		return nil
	}
	delayedSeg, err := s.prepareIntSegment(s.pendingDelayed-s.delayedMsg, arbstate.BatchSegmentKindDelayedMessages)
	if err != nil {
		return err
	}
	err = s.unconditionalAddSegment(delayedSeg)
	if err != nil {
		return err
	}
	s.delayedMsg = s.pendingDelayed
	return nil
}

func (s *batchSegments) AddMessage(msg *arbstate.MessageWithMetadata) (bool, error) {
	if s.isDone {
		return false, nil
	}
	if msg.DelayedMessagesRead > s.pendingDelayed {
		s.pendingDelayed = msg.DelayedMessagesRead
		if msg.MustEndBlock {
			err := s.testAddDelayedMessage()
			if err != nil {
				return false, err
			}
			return true, nil
		}
		return true, nil
	}
	err := s.testAddDelayedMessage()
	if err != nil {
		return false, err
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
		err := s.close(false)
		if err != nil {
			return nil, err
		}
	}
	err := s.compressedWriter.Close()
	if err != nil {
		return nil, err
	}
	return s.compressedBuffer.Bytes(), nil
}

func (b *BatchPoster) lastSubmittionIsSynced() bool {
	batchcount, err := b.inbox.GetBatchCount()
	if err != nil {
		return false
	}
	if batchcount < b.sequencesPosted {
		return false
	}
	if batchcount > b.sequencesPosted {
		log.Warn("detected unexpected sequences posted", batchcount, "expected", b.sequencesPosted)
		b.sequencesPosted = batchcount
		return true
	}
	return true
}

// TODO make sure we detect end of block!
func (b *BatchPoster) postSequencerBatch() error {
	for !b.lastSubmittionIsSynced() {
		<-time.After(time.Second)
	}
	var msgToPost, prevDelayedMsg uint64
	if b.sequencesPosted > 0 {
		prevBatchMeta, err := b.inbox.GetBatchMetadata(b.sequencesPosted - 1)
		if err != nil {
			return err
		}
		msgToPost = prevBatchMeta.MessageCount
		prevDelayedMsg = prevBatchMeta.DelayedMessageCount
	}
	segments := newBatchSegments(prevDelayedMsg, b.config)
	for {
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
	_, err = b.inboxContract.AddSequencerL2BatchFromOrigin(b.transactOpts, new(big.Int).SetUint64(b.sequencesPosted), sequencerMsg, new(big.Int).SetUint64(prevDelayedMsg), b.gasRefunder)
	return err
}

func (b *BatchPoster) Stop() {
	b.chanStop <- struct{}{}
}

func (b *BatchPoster) Start() {
	go (func() {
		for {
			err := b.postSequencerBatch()
			if err != nil {
				log.Error("error posting batch", "err", err.Error())
			}
			select {
			case <-b.chanStop:
				return
			case <-time.After(10 * time.Second):
			}
		}
	})()
}
