package espressostreamer

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	espressoClient "github.com/EspressoSystems/espresso-network/sdks/go/client"
	espressoTypes "github.com/EspressoSystems/espresso-network/sdks/go/types"
	"github.com/ccoveille/go-safecast"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/espressotee"
	"github.com/offchainlabs/nitro/util"
	"github.com/offchainlabs/nitro/util/dbutil"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

const NextHotshotBlockKey = "nextHotshotBlock"

var FailedToFetchTransactionsErr = errors.New("failed to fetch transactions")
var PayloadHadNoMessagesErr = errors.New("ParseHotShotPayload found no messages, the transaction may be empty")

type EspressoStreamerInterface interface {
	Start(ctx context.Context) error
	Next(ctx context.Context) *MessageWithMetadataAndPos
	// Peek returns the next message in the streamer's buffer. If the message is not
	// in the buffer, it will return nil.
	Peek(ctx context.Context) *MessageWithMetadataAndPos
	// Advance moves the current message position to the next message.
	Advance()
	// Reset sets the current message position and the next hotshot block number.
	Reset(currentMessagePos uint64, currentHostshotBlock uint64)
	// RecordTimeDurationBetweenHotshotAndCurrentBlock records the time duration between
	// the next hotshot block and the current block.
	RecordTimeDurationBetweenHotshotAndCurrentBlock(nextHotshotBlock uint64, blockProductionTime time.Time)
	StoreHotshotBlock(db ethdb.Database, nextHotshotBlock uint64) error
	ReadNextHotshotBlockFromDb(db ethdb.Database) (uint64, error)
	GetCurrentEarliestHotShotBlockNumber() uint64
}

type MessageWithMetadataAndPos struct {
	MessageWithMeta arbostypes.MessageWithMetadata
	Pos             uint64
	HotshotHeight   uint64
}

type EspressoStreamer struct {
	stopwaiter.StopWaiter
	espressoClient            espressoClient.EspressoClient
	nextHotshotBlockNum       uint64
	currentMessagePos         uint64
	namespace                 uint64
	messageWithMetadataAndPos []*MessageWithMetadataAndPos
	espressoSGXVerifier       espressotee.EspressoSGXVerifierInterface

	messageLock            sync.Mutex
	retryTime              time.Duration
	hotshotPollingInterval time.Duration

	PerfRecorder    *PerfRecorder
	batchPosterAddr common.Address
}

func NewEspressoStreamer(
	namespace uint64,
	nextHotshotBlockNum uint64,
	espressoSGXVerifier espressotee.EspressoSGXVerifierInterface,
	espressoClient espressoClient.EspressoClient,
	recordPerformance bool,
	batchPosterAddr common.Address,
	retryTime time.Duration,
) *EspressoStreamer {

	var PerfRecorder *PerfRecorder
	if recordPerformance {
		PerfRecorder = NewPerfRecorder()
	}

	return &EspressoStreamer{
		espressoClient:      espressoClient,
		nextHotshotBlockNum: nextHotshotBlockNum,
		namespace:           namespace,
		espressoSGXVerifier: espressoSGXVerifier,
		PerfRecorder:        PerfRecorder,
		batchPosterAddr:     batchPosterAddr,
		retryTime:           retryTime,
	}
}

// GetMessageCount
// This function will use the CountUniqueMessage to count the unique messages present in it's buffer.
// Parameters:
//
//	None
//
// Return value:
//
//	a uint64 representing the count of unique messages in the EspressoStreamer's internal buffer.
func (s *EspressoStreamer) GetMessageCount() uint64 {
	return CountUniqueEntries(&s.messageWithMetadataAndPos)
}
func (s *EspressoStreamer) Reset(currentMessagePos uint64, currentHostshotBlock uint64) {
	s.currentMessagePos = currentMessagePos
	s.nextHotshotBlockNum = currentHostshotBlock
	s.messageWithMetadataAndPos = []*MessageWithMetadataAndPos{}
}

func (s *EspressoStreamer) Next(ctx context.Context) *MessageWithMetadataAndPos {
	result := s.Peek(ctx)
	if result == nil {
		return nil
	}

	// Advance the current message position, so that the next call to
	// `Peek` or `Next` will return the next message
	s.Advance()
	return result
}

func (s *EspressoStreamer) Peek(ctx context.Context) *MessageWithMetadataAndPos {
	s.messageLock.Lock()
	defer s.messageLock.Unlock()

	compareMessageWithCurrentPos := func(msg *MessageWithMetadataAndPos) int {
		if msg.Pos == s.currentMessagePos {
			return FilterAndFind_Target
		}
		if msg.Pos < s.currentMessagePos {
			return FilterAndFind_Remove
		}
		return FilterAndFind_Keep
	}

	messageIndex := FilterAndFind(&s.messageWithMetadataAndPos, compareMessageWithCurrentPos)

	if messageIndex >= 0 {
		return s.messageWithMetadataAndPos[messageIndex]
	}

	return nil
}

// Call this function to advance the streamer to the next message
func (s *EspressoStreamer) Advance() {
	s.currentMessagePos += 1
}

// This function keep fetching hotshot blocks and parsing them until the condition is met.
// It is a do-while loop, which means it will always execute at least once.
//
// Expose the *parseHotShotPayloadFn* to the caller for testing purposes
func (s *EspressoStreamer) QueueMessagesFromHotshot(
	ctx context.Context,
	parseHotShotPayloadFn func(tx espressoTypes.Bytes) ([]*MessageWithMetadataAndPos, error),
) error {
	s.messageLock.Lock()
	defer s.messageLock.Unlock()

	messages, err := fetchNextHotshotBlock(ctx, s.espressoClient, s.nextHotshotBlockNum, parseHotShotPayloadFn, s.namespace)
	if err != nil {
		return err
	}

	if len(messages) > 0 {
		s.messageWithMetadataAndPos = append(s.messageWithMetadataAndPos, messages...)
	}
	s.nextHotshotBlockNum += 1
	return nil
}

func (s *EspressoStreamer) verifyBatchPosterSignature(signature []byte, userDataHash [32]byte) error {
	publicKey, err := crypto.SigToPub(userDataHash[:], signature)
	if err != nil {
		return fmt.Errorf("failed to convert signature to public key: %w", err)
	}
	addr := crypto.PubkeyToAddress(*publicKey)
	if addr != s.batchPosterAddr {
		log.Warn("batch poster address", "addr", addr, "expected", s.batchPosterAddr)
		return fmt.Errorf("batch poster address does not match")
	}
	return nil
}

func (s *EspressoStreamer) GetCurrentEarliestHotShotBlockNumber() uint64 {
	if len(s.messageWithMetadataAndPos) == 0 {
		// This case means that the espresso streamer is empty and the earliest hotshot block number
		// is the next hotshot block number.
		return s.nextHotshotBlockNum
	}
	return s.messageWithMetadataAndPos[0].HotshotHeight
}

/* Verify the attestation quote */
func (s *EspressoStreamer) verifyLegacy(attestation []byte, signature [32]byte) error {
	_, err := s.espressoSGXVerifier.Verify(nil, attestation, signature)
	if err != nil {
		return fmt.Errorf("call to the espressoTEEVerifier contract failed: %w", err)
	}
	return nil
}

func (s *EspressoStreamer) parseEspressoTransaction(tx espressoTypes.Bytes) ([]*MessageWithMetadataAndPos, error) {
	signature, userDataHash, indices, messages, err := arbutil.ParseHotShotPayload(tx)
	if err != nil {
		log.Warn("failed to parse hotshot payload", "err", err)
		return nil, err
	}
	if len(messages) == 0 {
		return nil, PayloadHadNoMessagesErr
	}
	// if attestation verification fails, we should skip this transaction
	// Parse the messages
	if len(userDataHash) != 32 {
		log.Warn("user data hash is not 32 bytes")
		return nil, fmt.Errorf("user data hash is not 32 bytes")
	}

	userDataHashArr := [32]byte(userDataHash)

	var success bool
	err = s.verifyBatchPosterSignature(signature, userDataHashArr)
	if err == nil {
		success = true
	} else {
		log.Warn("failed to verify batch poster signature", "err", err)
	}

	if !success && s.espressoSGXVerifier != nil {
		err = s.verifyLegacy(signature, userDataHashArr)
		if err != nil {
			log.Warn("failed to verify attestation quote", "err", err)
			return nil, err
		}
	}

	result := []*MessageWithMetadataAndPos{}

	for i, message := range messages {
		var messageWithMetadata arbostypes.MessageWithMetadata
		err = rlp.DecodeBytes(message, &messageWithMetadata)
		if err != nil {
			log.Warn("failed to decode message", "err", err)
			// Instead of returnning an error, we should just skip this message
			continue
		}
		if indices[i] < s.currentMessagePos {
			log.Warn("message index is less than current message pos, skipping", "messageIndex", indices[i], "currentMessagePos", s.currentMessagePos)
			continue
		}
		result = append(result, &MessageWithMetadataAndPos{
			MessageWithMeta: messageWithMetadata,
			Pos:             indices[i],
			HotshotHeight:   s.nextHotshotBlockNum,
		})
		log.Info("Added message to queue", "message", indices[i])
	}
	return result, nil
}

func (s *EspressoStreamer) ReadNextHotshotBlockFromDb(db ethdb.Database) (uint64, error) {
	var nextHotshotBlock uint64
	nextHotshotBytes, err := db.Get([]byte(NextHotshotBlockKey))
	if err != nil && !dbutil.IsErrNotFound(err) {
		return 0, fmt.Errorf("failed to get next hotshot block: %w", err)
	}
	if nextHotshotBytes != nil {
		err = rlp.DecodeBytes(nextHotshotBytes, &nextHotshotBlock)
		if err != nil {
			return 0, fmt.Errorf("failed to decode next hotshot block: %w", err)
		}
	}

	return nextHotshotBlock, nil
}

func (s *EspressoStreamer) StoreHotshotBlock(db ethdb.Database, nextHotshotBlock uint64) error {
	nextHotshotBytes, err := rlp.EncodeToBytes(nextHotshotBlock)
	if err != nil {
		return fmt.Errorf("failed to encode next hotshot block: %w", err)
	}

	err = db.Put([]byte(NextHotshotBlockKey), nextHotshotBytes)
	if err != nil {
		return fmt.Errorf("failed to put next hotshot block: %w", err)
	}

	return nil
}

func (s *EspressoStreamer) getEspressoBlockTimestamp(ctx context.Context, blockHeight uint64) (time.Time, error) {
	header, err := s.espressoClient.FetchHeaderByHeight(ctx, blockHeight)
	if err != nil {
		return time.Time{}, fmt.Errorf("unable to fetch header for hotshot block: %w", err)
	}
	seconds, err := safecast.ToInt64(header.Header.GetTimestamp())
	if err != nil {
		return time.Time{}, fmt.Errorf("unable to cast timestamp to int64: %w", err)
	}
	return time.Unix(seconds, 0), nil
}

func (s *EspressoStreamer) RecordTimeDurationBetweenHotshotAndCurrentBlock(nextHotshotBlock uint64, blockProductionTime time.Time) {
	if s.PerfRecorder != nil {
		timestamp, err := s.getEspressoBlockTimestamp(context.Background(), nextHotshotBlock)
		if err != nil {
			log.Warn("unable to fetch header for hotshot block", "err", err)
		} else {
			s.PerfRecorder.SetStartTime(timestamp)
			s.PerfRecorder.SetEndTime(blockProductionTime, fmt.Sprintf("Time duration between hotshot block %d and current block", nextHotshotBlock))
		}
	}
}

func fetchNextHotshotBlock(
	ctx context.Context,
	espressoClient espressoClient.EspressoClient,
	nextHotshotBlockNum uint64,
	parseHotShotPayloadFn func(tx espressoTypes.Bytes) ([]*MessageWithMetadataAndPos, error),
	namespace uint64,
) ([]*MessageWithMetadataAndPos, error) {
	arbTxns, err := espressoClient.FetchTransactionsInBlock(ctx, nextHotshotBlockNum, namespace)
	if err != nil {
		return []*MessageWithMetadataAndPos{}, fmt.Errorf("%w: %w", FailedToFetchTransactionsErr, err)
	}

	result := []*MessageWithMetadataAndPos{}

	for _, tx := range arbTxns.Transactions {
		messages, err := parseHotShotPayloadFn(tx)
		if err != nil {
			log.Warn("failed to verify espresso transaction", "err", err)
			continue
		}
		result = append(result, messages...)
	}
	return result, nil
}

func (s *EspressoStreamer) Start(ctxIn context.Context) error {
	s.StopWaiter.Start(ctxIn, s)

	ephemeralErrorHandler := util.NewEphemeralErrorHandler(3*time.Minute, FailedToFetchTransactionsErr.Error(), 1*time.Minute)
	processedHotshotBlocks := 0
	err := s.CallIterativelySafe(func(ctx context.Context) time.Duration {
		err := s.QueueMessagesFromHotshot(ctx, s.parseEspressoTransaction)
		if err != nil {
			logLevel := log.Error
			logLevel = ephemeralErrorHandler.LogLevel(err, logLevel)
			logLevel("error while queueing messages from hotshot", "err", err)
			return s.retryTime
		} else {
			ephemeralErrorHandler.Reset()
		}
		processedHotshotBlocks += 1
		if processedHotshotBlocks == 100 {
			log.Info("Now processing hotshot block", "block number", s.nextHotshotBlockNum)
			processedHotshotBlocks = 0
		} else {
			log.Debug("Now processing hotshot block", "block number", s.nextHotshotBlockNum)
		}
		return 0
	})
	return err
}
