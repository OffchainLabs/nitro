// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbnode

import (
	"bytes"
	"context"
	"encoding/hex"
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
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/arbnode/dataposter"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/das"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/offchainlabs/nitro/util/redisutil"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

var (
	batchPosterWalletBalance      = metrics.NewRegisteredGaugeFloat64("arb/batchposter/wallet/balanceether", nil)
	batchPosterGasRefunderBalance = metrics.NewRegisteredGaugeFloat64("arb/batchposter/gasrefunder/balanceether", nil)
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
	bridge       *bridgegen.Bridge
	syncMonitor  *SyncMonitor
	seqInboxABI  *abi.ABI
	seqInboxAddr common.Address
	building     *buildingBatch
	daWriter     das.DataAvailabilityServiceWriter
	dataPoster   *dataposter.DataPoster[batchPosterPosition]
	redisLock    *SimpleRedisLock
	firstAccErr  time.Time // first time a continuous missing accumulator occurred
	backlog      uint64    // An estimate of the number of unposted batches
}

type BatchPosterConfig struct {
	Enable                             bool                        `koanf:"enable"`
	DisableDasFallbackStoreDataOnChain bool                        `koanf:"disable-das-fallback-store-data-on-chain" reload:"hot"`
	MaxBatchSize                       int                         `koanf:"max-size" reload:"hot"`
	MaxBatchPostDelay                  time.Duration               `koanf:"max-delay" reload:"hot"`
	WaitForMaxBatchPostDelay           bool                        `koanf:"wait-for-max-delay" reload:"hot"`
	BatchPollDelay                     time.Duration               `koanf:"poll-delay" reload:"hot"`
	PostingErrorDelay                  time.Duration               `koanf:"error-delay" reload:"hot"`
	CompressionLevel                   int                         `koanf:"compression-level" reload:"hot"`
	DASRetentionPeriod                 time.Duration               `koanf:"das-retention-period" reload:"hot"`
	GasRefunderAddress                 string                      `koanf:"gas-refunder-address" reload:"hot"`
	DataPoster                         dataposter.DataPosterConfig `koanf:"data-poster" reload:"hot"`
	RedisUrl                           string                      `koanf:"redis-url"`
	RedisLock                          SimpleRedisLockConfig       `koanf:"redis-lock" reload:"hot"`
	ExtraBatchGas                      uint64                      `koanf:"extra-batch-gas" reload:"hot"`

	gasRefunder common.Address
}

func (c *BatchPosterConfig) Validate() error {
	if len(c.GasRefunderAddress) > 0 && !common.IsHexAddress(c.GasRefunderAddress) {
		return fmt.Errorf("invalid gas refunder address \"%v\"", c.GasRefunderAddress)
	}
	c.gasRefunder = common.HexToAddress(c.GasRefunderAddress)
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
	f.Duration(prefix+".max-delay", DefaultBatchPosterConfig.MaxBatchPostDelay, "maximum batch posting delay")
	f.Bool(prefix+".wait-for-max-delay", DefaultBatchPosterConfig.WaitForMaxBatchPostDelay, "wait for the max batch delay, even if the batch is full")
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
	MaxBatchPostDelay:                  time.Hour,
	WaitForMaxBatchPostDelay:           false,
	CompressionLevel:                   brotli.BestCompression,
	DASRetentionPeriod:                 time.Hour * 24 * 15,
	GasRefunderAddress:                 "",
	ExtraBatchGas:                      50_000,
	DataPoster:                         dataposter.DefaultDataPosterConfig,
}

var TestBatchPosterConfig = BatchPosterConfig{
	Enable:                   true,
	MaxBatchSize:             100000,
	BatchPollDelay:           time.Millisecond * 10,
	PostingErrorDelay:        time.Millisecond * 10,
	MaxBatchPostDelay:        0,
	WaitForMaxBatchPostDelay: false,
	CompressionLevel:         2,
	DASRetentionPeriod:       time.Hour * 24 * 15,
	GasRefunderAddress:       "",
	ExtraBatchGas:            10_000,
	DataPoster:               dataposter.TestDataPosterConfig,
}

func NewBatchPoster(l1Reader *headerreader.HeaderReader, inbox *InboxTracker, streamer *TransactionStreamer, syncMonitor *SyncMonitor, config BatchPosterConfigFetcher, deployInfo *RollupAddresses, transactOpts *bind.TransactOpts, daWriter das.DataAvailabilityServiceWriter) (*BatchPoster, error) {
	seqInbox, err := bridgegen.NewSequencerInbox(deployInfo.SequencerInbox, l1Reader.Client())
	if err != nil {
		return nil, err
	}
	bridge, err := bridgegen.NewBridge(deployInfo.Bridge, l1Reader.Client())
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
		bridge:       bridge,
		seqInbox:     seqInbox,
		seqInboxABI:  seqInboxABI,
		seqInboxAddr: deployInfo.SequencerInbox,
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
	compressedBuffer      *bytes.Buffer
	compressedWriter      *brotli.Writer
	rawSegments           [][]byte
	timestamp             uint64
	blockNum              uint64
	delayedMsg            uint64
	sizeLimit             int
	recompressionLevel    int
	newUncompressedSize   int
	totalUncompressedSize int
	lastCompressedSize    int
	trailingHeaders       int // how many trailing segments are headers
	isDone                bool
}

type buildingBatch struct {
	segments      *batchSegments
	startMsgCount arbutil.MessageIndex
	msgCount      arbutil.MessageIndex
}

func newBatchSegments(firstDelayed uint64, config *BatchPosterConfig, backlog uint64) *batchSegments {
	compressedBuffer := bytes.NewBuffer(make([]byte, 0, config.MaxBatchSize*2))
	if config.MaxBatchSize <= 40 {
		panic("MaxBatchSize too small")
	}
	compressionLevel := config.CompressionLevel
	recompressionLevel := config.CompressionLevel
	if backlog > 20 {
		compressionLevel = arbmath.MinInt(compressionLevel, brotli.DefaultCompression)
	}
	if backlog > 40 {
		recompressionLevel = arbmath.MinInt(recompressionLevel, brotli.DefaultCompression)
	}
	if backlog > 60 {
		compressionLevel = arbmath.MinInt(compressionLevel, 4)
	}
	if recompressionLevel < compressionLevel {
		// This should never be possible
		log.Warn(
			"somehow the recompression level was lower than the compression level",
			"recompressionLevel", recompressionLevel,
			"compressionLevel", compressionLevel,
		)
		recompressionLevel = compressionLevel
	}
	return &batchSegments{
		compressedBuffer:   compressedBuffer,
		compressedWriter:   brotli.NewWriterLevel(compressedBuffer, compressionLevel),
		sizeLimit:          config.MaxBatchSize - 40, // TODO
		recompressionLevel: recompressionLevel,
		rawSegments:        make([][]byte, 0, 128),
		delayedMsg:         firstDelayed,
	}
}

func (s *batchSegments) recompressAll() error {
	s.compressedBuffer = bytes.NewBuffer(make([]byte, 0, s.sizeLimit*2))
	s.compressedWriter = brotli.NewWriterLevel(s.compressedBuffer, s.recompressionLevel)
	s.newUncompressedSize = 0
	s.totalUncompressedSize = 0
	for _, segment := range s.rawSegments {
		err := s.addSegmentToCompressed(segment)
		if err != nil {
			return err
		}
	}
	if s.totalUncompressedSize > arbstate.MaxDecompressedLen {
		return fmt.Errorf("batch size %v exceeds maximum decompressed length %v", s.totalUncompressedSize, arbstate.MaxDecompressedLen)
	}
	if len(s.rawSegments) >= arbstate.MaxSegmentsPerSequencerMessage {
		return fmt.Errorf("number of raw segments %v excees maximum number %v", len(s.rawSegments), arbstate.MaxSegmentsPerSequencerMessage)
	}
	return nil
}

func (s *batchSegments) testForOverflow(isHeader bool) (bool, error) {
	// we've reached the max decompressed size
	if s.totalUncompressedSize > arbstate.MaxDecompressedLen {
		return true, nil
	}
	// we've reached the max number of segments
	if len(s.rawSegments) >= arbstate.MaxSegmentsPerSequencerMessage {
		return true, nil
	}
	// there is room, no need to flush
	if (s.lastCompressedSize + s.newUncompressedSize) < s.sizeLimit {
		return false, nil
	}
	// don't want to flush for headers or the first message
	if isHeader || len(s.rawSegments) == s.trailingHeaders {
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
	s.totalUncompressedSize += lenWritten
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
	// Force include headers because we don't want to re-compress and we can just trim them later if necessary
	overflow, err := s.testForOverflow(isHeader)
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

func (s *batchSegments) AddMessage(msg *arbostypes.MessageWithMetadata) (bool, error) {
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

// Returns nil (as opposed to []byte{}) if there's no segments to put in the batch
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
		b.config().gasRefunder,
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
	config := b.config()
	callOpts := &bind.CallOpts{
		Context: ctx,
	}
	if config.DataPoster.WaitForL1Finality {
		callOpts.BlockNumber = big.NewInt(int64(rpc.SafeBlockNumber))
	}
	safeDelayedMessagesBig, err := b.bridge.DelayedMessageCount(callOpts)
	if err != nil {
		return 0, fmt.Errorf("failed to get the confirmed delayed message count: %w", err)
	}
	if !safeDelayedMessagesBig.IsUint64() {
		return 0, fmt.Errorf("calling delayedMessageCount() on the bridge returned a non-uint64 result %v", safeDelayedMessagesBig)
	}
	safeDelayedMessages := safeDelayedMessagesBig.Uint64()
	if safeDelayedMessages > delayedMessages {
		// On restart, we may be trying to estimate gas for a batch whose successor has
		// already made it into pending state, if not latest state.
		// In that case, we might get a revert with `DelayedBackwards()`.
		// To avoid that, we artificially increase the delayed messages to `safeDelayedMessages`.
		// In theory, this might reduce gas usage, but only by a factor that's already
		// accounted for in `config.ExtraBatchGas`, as that same factor can appear if a user
		// posts a new delayed message that we didn't see while gas estimating.
		delayedMessages = safeDelayedMessages
	}
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
		sequencerMessageHeader := sequencerMessage
		if len(sequencerMessageHeader) > 33 {
			sequencerMessageHeader = sequencerMessageHeader[:33]
		}
		log.Warn(
			"error estimating gas for batch",
			"err", err,
			"delayedMessages", delayedMessages,
			"safeDelayedMessages", safeDelayedMessages,
			"sequencerMessageHeader", hex.EncodeToString(sequencerMessageHeader),
			"sequencerMessageLen", len(sequencerMessage),
		)
		return 0, fmt.Errorf("error estimating gas for batch: %w", err)
	}
	return gas + config.ExtraBatchGas, nil
}

func (b *BatchPoster) maybePostSequencerBatch(ctx context.Context) (bool, error) {
	nonce, batchPosition, err := b.dataPoster.GetNextNonceAndMeta(ctx)
	if err != nil {
		return false, err
	}

	if b.building == nil || b.building.startMsgCount != batchPosition.MessageCount {
		b.building = &buildingBatch{
			segments:      newBatchSegments(batchPosition.DelayedMessageCount, b.config(), b.backlog),
			msgCount:      batchPosition.MessageCount,
			startMsgCount: batchPosition.MessageCount,
		}
	}
	msgCount, err := b.streamer.GetMessageCount()
	if err != nil {
		return false, err
	}
	if msgCount <= batchPosition.MessageCount {
		// There's nothing after the newest batch, therefore batch posting was not required
		return false, nil
	}
	firstMsg, err := b.streamer.GetMessage(batchPosition.MessageCount)
	if err != nil {
		return false, err
	}
	nextMessageTime := time.Unix(int64(firstMsg.Message.Header.Timestamp), 0)

	config := b.config()
	forcePostBatch := time.Since(nextMessageTime) >= config.MaxBatchPostDelay
	haveUsefulMessage := false

	for b.building.msgCount < msgCount {
		msg, err := b.streamer.GetMessage(b.building.msgCount)
		if err != nil {
			log.Error("error getting message from streamer", "error", err)
			break
		}
		success, err := b.building.segments.AddMessage(msg)
		if err != nil {
			// Clear our cache
			b.building = nil
			return false, fmt.Errorf("error adding message to batch: %w", err)
		}
		if !success {
			// this batch is full
			if !config.WaitForMaxBatchPostDelay {
				forcePostBatch = true
			}
			haveUsefulMessage = true
			break
		}
		if msg.Message.Header.Kind != arbostypes.L1MessageType_BatchPostingReport {
			haveUsefulMessage = true
		}
		b.building.msgCount++
	}

	if !forcePostBatch || !haveUsefulMessage {
		// the batch isn't full yet and we've posted a batch recently
		// don't post anything for now
		return false, nil
	}
	sequencerMsg, err := b.building.segments.CloseAndGetBytes()
	if err != nil {
		return false, err
	}
	if sequencerMsg == nil {
		log.Debug("BatchPoster: batch nil", "sequence nr.", batchPosition.NextSeqNum, "from", batchPosition.MessageCount, "prev delayed", batchPosition.DelayedMessageCount)
		b.building = nil // a closed batchSegments can't be reused
		return false, nil
	}

	if b.daWriter != nil {
		cert, err := b.daWriter.Store(ctx, sequencerMsg, uint64(time.Now().Add(config.DASRetentionPeriod).Unix()), []byte{}) // b.daWriter will append signature if enabled
		if errors.Is(err, das.BatchToDasFailed) {
			if config.DisableDasFallbackStoreDataOnChain {
				return false, errors.New("Unable to batch to DAS and fallback storing data on chain is disabled")
			}
			log.Warn("Falling back to storing data on chain", "err", err)
		} else if err != nil {
			return false, err
		} else {
			sequencerMsg = das.Serialize(cert)
		}
	}

	gasLimit, err := b.estimateGas(ctx, sequencerMsg, b.building.segments.delayedMsg)
	if err != nil {
		return false, err
	}
	data, err := b.encodeAddBatch(new(big.Int).SetUint64(batchPosition.NextSeqNum), batchPosition.MessageCount, b.building.msgCount, sequencerMsg, b.building.segments.delayedMsg)
	if err != nil {
		return false, err
	}
	newMeta := batchPosterPosition{
		MessageCount:        b.building.msgCount,
		DelayedMessageCount: b.building.segments.delayedMsg,
		NextSeqNum:          batchPosition.NextSeqNum + 1,
	}
	err = b.dataPoster.PostTransaction(ctx, nextMessageTime, nonce, newMeta, b.seqInboxAddr, data, gasLimit)
	if err != nil {
		return false, err
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
	postedMessages := b.building.msgCount - batchPosition.MessageCount
	unpostedMessages := msgCount - b.building.msgCount
	b.backlog = uint64(unpostedMessages) / uint64(postedMessages)
	if b.backlog > 10 {
		logLevel := log.Warn
		if b.backlog > 30 {
			logLevel = log.Error
		}
		logLevel(
			"a large batch posting backlog exists",
			"currentPosition", b.building.msgCount,
			"messageCount", msgCount,
			"lastPostedMessages", postedMessages,
			"unpostedMessages", unpostedMessages,
			"batchBacklogEstimate", b.backlog,
		)
	}
	b.building = nil
	return true, nil
}

func (b *BatchPoster) Start(ctxIn context.Context) {
	b.dataPoster.Start(ctxIn)
	b.redisLock.Start(ctxIn)
	b.StopWaiter.Start(ctxIn, b)
	b.CallIteratively(func(ctx context.Context) time.Duration {
		var err error
		if common.HexToAddress(b.config().GasRefunderAddress) != (common.Address{}) {
			gasRefunderBalance, err := b.l1Reader.Client().BalanceAt(ctx, common.HexToAddress(b.config().GasRefunderAddress), nil)
			if err != nil {
				log.Warn("error fetching batch poster gas refunder balance", "err", err)
			} else {
				batchPosterGasRefunderBalance.Update(arbmath.BalancePerEther(gasRefunderBalance))
			}
		}
		if b.dataPoster.From() != (common.Address{}) {
			walletBalance, err := b.l1Reader.Client().BalanceAt(ctx, b.dataPoster.From(), nil)
			if err != nil {
				log.Warn("error fetching batch poster wallet balance", "err", err)
			} else {
				batchPosterWalletBalance.Update(arbmath.BalancePerEther(walletBalance))
			}
		}
		if !b.redisLock.AttemptLock(ctx) {
			b.building = nil
			return b.config().BatchPollDelay
		}
		posted, err := b.maybePostSequencerBatch(ctx)
		if err != nil {
			b.building = nil
			logLevel := log.Error
			if errors.Is(err, AccumulatorNotFoundErr) || errors.Is(err, dataposter.ErrStorageRace) {
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
		} else if posted {
			return 0
		} else {
			return b.config().BatchPollDelay
		}
	})
}

func (b *BatchPoster) StopAndWait() {
	b.StopWaiter.StopAndWait()
	b.dataPoster.StopAndWait()
	b.redisLock.StopAndWait()
}
