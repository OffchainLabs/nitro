// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbnode

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/offchainlabs/nitro/arbnode/dataposter"
	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/util/headerreader"

	"github.com/andybalholm/brotli"
	"github.com/pkg/errors"
	flag "github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/das"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

type batchPosterPosition struct {
	MessageCount        arbutil.MessageIndex
	DelayedMessageCount uint64
	NextSeqNum          uint64
}

type BatchPoster struct {
	stopwaiter.StopWaiter
	l1Reader     *headerreader.HeaderReader
	inbox        *InboxTracker
	streamer     *TransactionStreamer
	config       *BatchPosterConfig
	seqInbox     *bridgegen.SequencerInbox
	seqInboxABI  *abi.ABI
	seqInboxAddr common.Address
	gasRefunder  common.Address
	building     *buildingBatch
	das          das.DataAvailabilityService
	dataPoster   dataposter.DataPoster[batchPosterPosition]
}

type BatchPosterConfig struct {
	Enable                             bool                        `koanf:"enable"`
	DisableDasFallbackStoreDataOnChain bool                        `koanf:"disable-das-fallback-store-data-on-chain"`
	MaxBatchSize                       int                         `koanf:"max-size"`
	MaxBatchPostInterval               time.Duration               `koanf:"max-interval"`
	BatchPollDelay                     time.Duration               `koanf:"poll-delay"`
	PostingErrorDelay                  time.Duration               `koanf:"error-delay"`
	CompressionLevel                   int                         `koanf:"compression-level"`
	DASRetentionPeriod                 time.Duration               `koanf:"das-retention-period"`
	HighGasThreshold                   float32                     `koanf:"high-gas-threshold"`
	HighGasDelay                       time.Duration               `koanf:"high-gas-delay"`
	GasRefunderAddress                 string                      `koanf:"gas-refunder-address"`
	DataPoster                         dataposter.DataPosterConfig `koanf:"data-poster"`
}

func BatchPosterConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".enable", DefaultBatchPosterConfig.Enable, "enable posting batches to l1")
	f.Bool(prefix+".disable-das-fallback-store-data-on-chain", DefaultBatchPosterConfig.DisableDasFallbackStoreDataOnChain, "If unable to batch to DAS, disable fallback storing data on chain")
	f.Int(prefix+".max-size", DefaultBatchPosterConfig.MaxBatchSize, "maximum batch size")
	f.Duration(prefix+".max-interval", DefaultBatchPosterConfig.MaxBatchPostInterval, "maximum batch posting interval")
	f.Duration(prefix+".poll-delay", DefaultBatchPosterConfig.BatchPollDelay, "how long to delay after successfully posting batch")
	f.Duration(prefix+".error-delay", DefaultBatchPosterConfig.PostingErrorDelay, "how long to delay after error posting batch")
	f.Int(prefix+".compression-level", DefaultBatchPosterConfig.CompressionLevel, "batch compression level")
	f.Duration(prefix+".das-retention-period", DefaultBatchPosterConfig.DASRetentionPeriod, "In AnyTrust mode, the period which DASes are requested to retain the stored batches.")
	f.Float32(prefix+".high-gas-threshold", DefaultBatchPosterConfig.HighGasThreshold, "If the gas price in gwei is above this amount, delay posting a batch")
	f.Duration(prefix+".high-gas-delay", DefaultBatchPosterConfig.HighGasDelay, "The maximum delay while waiting for the gas price to go below the high gas threshold")
	f.String(prefix+".gas-refunder-address", DefaultBatchPosterConfig.GasRefunderAddress, "The gas refunder contract address (optional)")
	dataposter.DataPosterConfigAddOptions(prefix+".data-poster", f)
}

var DefaultBatchPosterConfig = BatchPosterConfig{
	Enable:                             false,
	DisableDasFallbackStoreDataOnChain: false,
	MaxBatchSize:                       100000,
	BatchPollDelay:                     time.Second * 10,
	PostingErrorDelay:                  time.Second * 10,
	MaxBatchPostInterval:               time.Hour,
	CompressionLevel:                   brotli.DefaultCompression,
	DASRetentionPeriod:                 time.Hour * 24 * 15,
	HighGasThreshold:                   150.,
	HighGasDelay:                       14 * time.Hour,
	GasRefunderAddress:                 "",
	DataPoster:                         dataposter.DefaultDataPosterConfig,
}

var TestBatchPosterConfig = BatchPosterConfig{
	Enable:               true,
	MaxBatchSize:         100000,
	BatchPollDelay:       time.Millisecond * 10,
	PostingErrorDelay:    time.Millisecond * 10,
	MaxBatchPostInterval: 0,
	CompressionLevel:     2,
	DASRetentionPeriod:   time.Hour * 24 * 15,
	HighGasThreshold:     0.,
	HighGasDelay:         0,
	GasRefunderAddress:   "",
	DataPoster:           dataposter.TestDataPosterConfig,
}

func NewBatchPoster(l1Reader *headerreader.HeaderReader, inbox *InboxTracker, streamer *TransactionStreamer, config *BatchPosterConfig, contractAddress common.Address, transactOpts *bind.TransactOpts, das das.DataAvailabilityService) (*BatchPoster, error) {
	seqInbox, err := bridgegen.NewSequencerInbox(contractAddress, l1Reader.Client())
	if err != nil {
		return nil, err
	}
	if len(config.GasRefunderAddress) > 0 && !common.IsHexAddress(config.GasRefunderAddress) {
		return nil, fmt.Errorf("invalid gas refunder address \"%v\"", config.GasRefunderAddress)
	}
	seqInboxABI, err := bridgegen.SequencerInboxMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return &BatchPoster{
		l1Reader:     l1Reader,
		inbox:        inbox,
		streamer:     streamer,
		config:       config,
		seqInbox:     seqInbox,
		seqInboxABI:  seqInboxABI,
		seqInboxAddr: contractAddress,
		gasRefunder:  common.HexToAddress(config.GasRefunderAddress),
		das:          das,
		dataPoster:   *dataposter.NewDataPoster[batchPosterPosition](l1Reader.Client(), transactOpts, &config.DataPoster, nil),
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
	segments      *batchSegments
	startMsgCount arbutil.MessageIndex
	msgCount      arbutil.MessageIndex
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

func (s *batchSegments) IsEmpty() bool {
	return len(s.rawSegments) == 0
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
	fullMsg[0] = arbstate.BrotliMessageHeaderByte
	fullMsg = append(fullMsg, compressedBytes...)
	return fullMsg, nil
}

func (b *BatchPoster) encodeAddBatch(seqNum *big.Int, prevMsgNum arbutil.MessageIndex, newMsgNum arbutil.MessageIndex, message []byte, delayedMsg uint64) ([]byte, error) {
	method, ok := b.seqInboxABI.Methods["AddSequencerL2BatchFromOrigin"]
	if !ok {
		return nil, errors.New("failed to find add batch method")
	}
	inputData, err := method.Inputs.Pack(
		seqNum,
		new(big.Int).SetUint64(uint64(prevMsgNum)),
		new(big.Int).SetUint64(uint64(newMsgNum)),
		message,
		new(big.Int).SetUint64(delayedMsg),
		b.gasRefunder,
	)
	if err != nil {
		return nil, err
	}
	fullData := append([]byte{}, method.ID...)
	fullData = append(fullData, inputData...)
	return fullData, nil
}

const extraBatchGas uint64 = 10_000

func (b *BatchPoster) estimateGas(ctx context.Context, sequencerMessage []byte, delayedMessages uint64) (uint64, error) {
	data, err := b.encodeAddBatch(abi.MaxUint256, 0, 0, sequencerMessage, b.building.segments.delayedMsg)
	if err != nil {
		return 0, err
	}
	gas, err := b.l1Reader.Client().EstimateGas(ctx, ethereum.CallMsg{
		From: b.dataPoster.From(),
		To:   &b.seqInboxAddr,
		Data: data,
	})
	return gas + extraBatchGas, err
}

func (b *BatchPoster) maybePostSequencerBatch(ctx context.Context) error {
	nonce, batchPosition, err := b.dataPoster.GetNextNonceAndMeta(ctx, func(blockNum *big.Int) (batchPosterPosition, error) {
		bigInboxBatchCount, err := b.seqInbox.BatchCount(&bind.CallOpts{Context: ctx, BlockNumber: blockNum})
		if err != nil {
			return batchPosterPosition{}, err
		}
		inboxBatchCount := bigInboxBatchCount.Uint64()
		var prevBatchMeta BatchMetadata
		if inboxBatchCount > 0 {
			var err error
			prevBatchMeta, err = b.inbox.GetBatchMetadata(inboxBatchCount - 1)
			if err != nil {
				return batchPosterPosition{}, err
			}
		}
		return batchPosterPosition{
			MessageCount:        prevBatchMeta.MessageCount,
			DelayedMessageCount: prevBatchMeta.DelayedMessageCount,
			NextSeqNum:          inboxBatchCount,
		}, nil
	})
	if err != nil {
		return err
	}

	if b.building == nil || b.building.startMsgCount != batchPosition.MessageCount {
		b.building = &buildingBatch{
			segments:      newBatchSegments(batchPosition.DelayedMessageCount, b.config),
			msgCount:      batchPosition.MessageCount,
			startMsgCount: batchPosition.MessageCount,
		}
	}
	msgCount, err := b.streamer.GetMessageCount()
	if err != nil {
		return err
	}
	if msgCount <= batchPosition.MessageCount {
		// There's nothing after the newest batch, therefore batch posting was not required
		return err
	}
	firstMsg, err := b.streamer.GetMessage(batchPosition.MessageCount)
	if err != nil {
		return err
	}
	nextMessageTime := time.Unix(int64(firstMsg.Message.Header.Timestamp), 0)

	forcePostBatch := time.Since(nextMessageTime) >= b.config.MaxBatchPostInterval
	haveUsefulMessage := false

	for b.building.msgCount < msgCount {
		msg, err := b.streamer.GetMessage(b.building.msgCount)
		if err != nil {
			log.Error("error getting message from streamer", "error", err)
			break
		}
		if msg.Message.Header.Kind != arbos.L1MessageType_BatchPostingReport {
			haveUsefulMessage = true
		}
		success, err := b.building.segments.AddMessage(&msg)
		if err != nil {
			log.Error("error adding message to batch", "error", err)
			break
		}
		if !success {
			// this batch is full
			forcePostBatch = true
			haveUsefulMessage = true
			break
		}
		b.building.msgCount++
	}

	if b.building.segments.IsEmpty() {
		// we don't need to post a batch for the time being
		return nil
	}
	if !forcePostBatch || !haveUsefulMessage {
		// the batch isn't full yet and we've posted a batch recently
		// don't post anything for now
		return nil
	}
	sequencerMsg, err := b.building.segments.CloseAndGetBytes()
	if err != nil {
		return err
	}
	if sequencerMsg == nil {
		log.Debug("BatchPoster: batch nil", "sequence nr.", batchPosition.NextSeqNum, "from", batchPosition.MessageCount, "prev delayed", batchPosition.DelayedMessageCount)
		b.building = nil // a closed batchSegments can't be reused
		return nil
	}

	if b.das != nil {
		cert, err := b.das.Store(ctx, sequencerMsg, uint64(time.Now().Add(b.config.DASRetentionPeriod).Unix()), []byte{}) // b.das will append signature if enabled
		if err != nil {
			log.Warn("Unable to batch to DAS, falling back to storing data on chain", "err", err)
			if b.config.DisableDasFallbackStoreDataOnChain {
				return errors.New("Unable to batch to DAS and fallback storing data on chain is disabled")
			}
		} else {
			sequencerMsg = das.Serialize(cert)
		}
	}

	gasLimit, err := b.estimateGas(ctx, sequencerMsg, b.building.segments.delayedMsg)
	if err != nil {
		return err
	}
	data, err := b.encodeAddBatch(new(big.Int).SetUint64(batchPosition.NextSeqNum), batchPosition.MessageCount, b.building.msgCount, sequencerMsg, b.building.segments.delayedMsg)
	if err != nil {
		return err
	}
	newMeta := batchPosterPosition{
		MessageCount:        b.building.msgCount,
		DelayedMessageCount: b.building.segments.delayedMsg,
		NextSeqNum:          batchPosition.NextSeqNum + 1,
	}
	err = b.dataPoster.PostTransaction(ctx, nextMessageTime, nonce, newMeta, b.seqInboxAddr, data, gasLimit)
	if err != nil {
		return err
	}
	log.Info(
		"BatchPoster: batch sent",
		"sequence nr.", batchPosition.NextSeqNum,
		"from", batchPosition.MessageCount,
		"to", b.building.msgCount,
		"prev delayed", batchPosition.DelayedMessageCount,
		"current delayed", b.building.segments.delayedMsg,
		"total segments", len(b.building.segments.rawSegments),
	)
	b.building = nil
	return nil
}

func (b *BatchPoster) Start(ctxIn context.Context) {
	b.StopWaiter.Start(ctxIn)
	b.CallIteratively(func(ctx context.Context) time.Duration {
		err := b.maybePostSequencerBatch(ctx)
		if err != nil {
			b.building = nil
			log.Error("error posting batch", "err", err)
			return b.config.PostingErrorDelay
		}
		return b.config.BatchPollDelay
	})
}
