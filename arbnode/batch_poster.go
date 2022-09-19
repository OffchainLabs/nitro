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

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/offchainlabs/nitro/arbnode/dataposter"
	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/das"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/offchainlabs/nitro/util/redisutil"
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
	config       BatchPosterConfigFetcher
	seqInbox     *bridgegen.SequencerInbox
	syncMonitor  *SyncMonitor
	seqInboxABI  *abi.ABI
	seqInboxAddr common.Address
	building     *buildingBatch
	daWriter     das.DataAvailabilityServiceWriter
	dataPoster   *dataposter.DataPoster[batchPosterPosition]
	redisLock    *SimpleRedisLock
	firstAccErr  time.Time // first time a continuous missing accumulator occurred
}

type BatchPosterConfig struct {
	Enable                             bool                        `koanf:"enable"`
	DisableDasFallbackStoreDataOnChain bool                        `koanf:"disable-das-fallback-store-data-on-chain" reload:"hot"`
	MaxBatchSize                       int                         `koanf:"max-size" reload:"hot"`
	MaxBatchPostInterval               time.Duration               `koanf:"max-interval" reload:"hot"`
	BatchPollDelay                     time.Duration               `koanf:"poll-delay" reload:"hot"`
	PostingErrorDelay                  time.Duration               `koanf:"error-delay" reload:"hot"`
	CompressionLevel                   int                         `koanf:"compression-level" reload:"hot"`
	DASRetentionPeriod                 time.Duration               `koanf:"das-retention-period" reload:"hot"`
	GasRefunderAddress                 string                      `koanf:"gas-refunder-address" reload:"hot"`
	DataPoster                         dataposter.DataPosterConfig `koanf:"data-poster" reload:"hot"`
	RedisUrl                           string                      `koanf:"redis-url"`
	RedisLock                          SimpleRedisLockConfig       `koanf:"redis-lock" reload:"hot"`
	ExtraBatchGas                      uint64                      `koanf:"extra-batch-gas" reload:"hot"`
}

func (c *BatchPosterConfig) Validate() error {
	if len(c.GasRefunderAddress) > 0 && !common.IsHexAddress(c.GasRefunderAddress) {
		return fmt.Errorf("invalid gas refunder address \"%v\"", c.GasRefunderAddress)
	}
	if c.MaxBatchSize <= 40 {
		return errors.New("MaxBatchSize too small")
	}
	return nil
}

type BatchPosterConfigFetcher func() *BatchPosterConfig

func BatchPosterConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".enable", DefaultBatchPosterConfig.Enable, "enable posting batches to l1")
	f.Bool(prefix+".disable-das-fallback-store-data-on-chain", DefaultBatchPosterConfig.DisableDasFallbackStoreDataOnChain, "If unable to batch to DAS, disable fallback storing data on chain")
	f.Int(prefix+".max-size", DefaultBatchPosterConfig.MaxBatchSize, "maximum batch size")
	f.Duration(prefix+".max-interval", DefaultBatchPosterConfig.MaxBatchPostInterval, "maximum batch posting interval")
	f.Duration(prefix+".poll-delay", DefaultBatchPosterConfig.BatchPollDelay, "how long to delay after successfully posting batch")
	f.Duration(prefix+".error-delay", DefaultBatchPosterConfig.PostingErrorDelay, "how long to delay after error posting batch")
	f.Int(prefix+".compression-level", DefaultBatchPosterConfig.CompressionLevel, "batch compression level")
	f.Duration(prefix+".das-retention-period", DefaultBatchPosterConfig.DASRetentionPeriod, "In AnyTrust mode, the period which DASes are requested to retain the stored batches.")
	f.String(prefix+".gas-refunder-address", DefaultBatchPosterConfig.GasRefunderAddress, "The gas refunder contract address (optional)")
	f.Uint64(prefix+".extra-batch-gas", DefaultBatchPosterConfig.ExtraBatchGas, "use this much more gas than estimation says is necessary to post batches")
	f.String(prefix+".redis-url", DefaultBatchPosterConfig.RedisUrl, "if non-empty, the Redis URL to store queued transactions in")
	RedisLockConfigAddOptions(prefix+".redis-lock", f)
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
	GasRefunderAddress:                 "",
	ExtraBatchGas:                      50_000,
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
	GasRefunderAddress:   "",
	ExtraBatchGas:        10_000,
	DataPoster:           dataposter.TestDataPosterConfig,
}

func NewBatchPoster(l1Reader *headerreader.HeaderReader, inbox *InboxTracker, streamer *TransactionStreamer, syncMonitor *SyncMonitor, config BatchPosterConfigFetcher, contractAddress common.Address, transactOpts *bind.TransactOpts, daWriter das.DataAvailabilityServiceWriter) (*BatchPoster, error) {
	seqInbox, err := bridgegen.NewSequencerInbox(contractAddress, l1Reader.Client())
	if err != nil {
		return nil, err
	}
	if err = config().Validate(); err != nil {
		return nil, err
	}
	seqInboxABI, err := bridgegen.SequencerInboxMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	redisClient, err := redisutil.RedisClientFromURL(config().RedisUrl)
	if err != nil {
		return nil, err
	}
	redisLockConfigFetcher := func() *SimpleRedisLockConfig {
		return &config().RedisLock
	}
	redisLock, err := NewSimpleRedisLock(redisClient, redisLockConfigFetcher, func() bool { return syncMonitor.Synced() })
	if err != nil {
		return nil, err
	}
	b := &BatchPoster{
		l1Reader:     l1Reader,
		inbox:        inbox,
		streamer:     streamer,
		syncMonitor:  syncMonitor,
		config:       config,
		seqInbox:     seqInbox,
		seqInboxABI:  seqInboxABI,
		seqInboxAddr: contractAddress,
		daWriter:     daWriter,
		redisLock:    redisLock,
	}
	dataPosterConfigFetcher := func() *dataposter.DataPosterConfig {
		return &config().DataPoster
	}
	b.dataPoster, err = dataposter.NewDataPoster(l1Reader, transactOpts, redisClient, redisLock, dataPosterConfigFetcher, b.getBatchPosterPosition)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (b *BatchPoster) getBatchPosterPosition(ctx context.Context, blockNum *big.Int) (batchPosterPosition, error) {
	bigInboxBatchCount, err := b.seqInbox.BatchCount(&bind.CallOpts{Context: ctx, BlockNumber: blockNum})
	if err != nil {
		return batchPosterPosition{}, fmt.Errorf("error getting latest batch count: %w", err)
	}
	inboxBatchCount := bigInboxBatchCount.Uint64()
	var prevBatchMeta BatchMetadata
	if inboxBatchCount > 0 {
		var err error
		prevBatchMeta, err = b.inbox.GetBatchMetadata(inboxBatchCount - 1)
		if err != nil {
			return batchPosterPosition{}, fmt.Errorf("error getting latest batch metadata: %w", err)
		}
	}
	return batchPosterPosition{
		MessageCount:        prevBatchMeta.MessageCount,
		DelayedMessageCount: prevBatchMeta.DelayedMessageCount,
		NextSeqNum:          inboxBatchCount,
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
	method, ok := b.seqInboxABI.Methods["addSequencerL2BatchFromOrigin0"]
	if !ok {
		return nil, errors.New("failed to find add batch method")
	}
	inputData, err := method.Inputs.Pack(
		seqNum,
		message,
		new(big.Int).SetUint64(delayedMsg),
		common.HexToAddress(b.config().GasRefunderAddress),
		new(big.Int).SetUint64(uint64(prevMsgNum)),
		new(big.Int).SetUint64(uint64(newMsgNum)),
	)
	if err != nil {
		return nil, err
	}
	fullData := append([]byte{}, method.ID...)
	fullData = append(fullData, inputData...)
	return fullData, nil
}

func (b *BatchPoster) estimateGas(ctx context.Context, sequencerMessage []byte, delayedMessages uint64) (uint64, error) {
	// Here we set seqNum to MaxUint256, and prevMsgNum to 0, because it disables the smart contracts' consistency checks.
	// However, we set nextMsgNum to 1 because it is necessary for a correct estimation for the final to be non-zero.
	// Because we're likely estimating against older state, this might not be the actual next message,
	// but the gas used should be the same.
	data, err := b.encodeAddBatch(abi.MaxUint256, 0, 1, sequencerMessage, delayedMessages)
	if err != nil {
		return 0, err
	}
	gas, err := b.l1Reader.Client().EstimateGas(ctx, ethereum.CallMsg{
		From: b.dataPoster.From(),
		To:   &b.seqInboxAddr,
		Data: data,
	})
	if err != nil {
		return 0, fmt.Errorf("error estimating gas for batch: %w", err)
	}
	return gas + b.config().ExtraBatchGas, nil
}

func (b *BatchPoster) maybePostSequencerBatch(ctx context.Context) error {
	nonce, batchPosition, err := b.dataPoster.GetNextNonceAndMeta(ctx)
	if err != nil {
		return err
	}

	if b.building == nil || b.building.startMsgCount != batchPosition.MessageCount {
		b.building = &buildingBatch{
			segments:      newBatchSegments(batchPosition.DelayedMessageCount, b.config()),
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
		return nil
	}
	firstMsg, err := b.streamer.GetMessage(batchPosition.MessageCount)
	if err != nil {
		return err
	}
	nextMessageTime := time.Unix(int64(firstMsg.Message.Header.Timestamp), 0)

	config := b.config()
	forcePostBatch := time.Since(nextMessageTime) >= config.MaxBatchPostInterval
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
		success, err := b.building.segments.AddMessage(msg)
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

	if b.daWriter != nil {
		cert, err := b.daWriter.Store(ctx, sequencerMsg, uint64(time.Now().Add(config.DASRetentionPeriod).Unix()), []byte{}) // b.daWriter will append signature if enabled
		if err != nil {
			log.Warn("Unable to batch to DAS, falling back to storing data on chain", "err", err)
			if config.DisableDasFallbackStoreDataOnChain {
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
	b.dataPoster.Start(ctxIn)
	b.redisLock.Start(ctxIn)
	b.StopWaiter.Start(ctxIn, b)
	b.CallIteratively(func(ctx context.Context) time.Duration {
		if !b.redisLock.AttemptLock(ctx) {
			b.building = nil
			return b.config().BatchPollDelay
		}
		err := b.maybePostSequencerBatch(ctx)
		if err != nil {
			b.building = nil
			logLevel := log.Error
			if errors.Is(err, AccumulatorNotFoundErr) || errors.Is(err, dataposter.StorageRaceErr) {
				// Likely the inbox tracker just isn't caught up.
				// Let's see if this error disappears naturally.
				if b.firstAccErr == (time.Time{}) {
					b.firstAccErr = time.Now()
					logLevel = log.Debug
				} else if time.Since(b.firstAccErr) < time.Minute {
					logLevel = log.Debug
				}
			} else {
				b.firstAccErr = time.Time{}
			}
			logLevel("error posting batch", "err", err)
			return b.config().PostingErrorDelay
		}
		return b.config().BatchPollDelay
	})
}

func (b *BatchPoster) StopAndWait() {
	b.StopWaiter.StopAndWait()
	b.dataPoster.StopAndWait()
	b.redisLock.StopAndWait()
}
