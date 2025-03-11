package espressostreamer

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	espressoClient "github.com/EspressoSystems/espresso-sequencer-go/client"
	espressoTypes "github.com/EspressoSystems/espresso-sequencer-go/types"
	"github.com/ccoveille/go-safecast"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/util"
	"github.com/offchainlabs/nitro/util/dbutil"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

const NextHotshotBlockKey = "nextHotshotBlock"

var FailedToFetchTransactionsErr = errors.New("failed to fetch transactions")

type EspressoTEEVerifierInterface interface {
	Verify(opts *bind.CallOpts, rawQuote []byte, reportDataHash [32]byte) error
}

type EspressoClientInterface interface {
	FetchLatestBlockHeight(ctx context.Context) (uint64, error)
	FetchTransactionsInBlock(ctx context.Context, blockHeight uint64, namespace uint64) (espressoClient.TransactionsInBlock, error)
	FetchHeaderByHeight(ctx context.Context, blockHeight uint64) (espressoTypes.HeaderImpl, error)
}

type MessageWithMetadataAndPos struct {
	MessageWithMeta arbostypes.MessageWithMetadata
	Pos             uint64
	HotshotHeight   uint64
}

type EspressoStreamer struct {
	stopwaiter.StopWaiter
	espressoClient                EspressoClientInterface
	nextHotshotBlockNum           uint64
	currentMessagePos             uint64
	namespace                     uint64
	retryTime                     time.Duration
	pollingHotshotPollingInterval time.Duration
	messageWithMetadataAndPos     []*MessageWithMetadataAndPos
	espressoTEEVerifierCaller     EspressoTEEVerifierInterface

	PerfRecorder *PerfRecorder

	messageMutex sync.Mutex
}

func NewEspressoStreamer(
	namespace uint64,
	nextHotshotBlockNum uint64,
	retryTime time.Duration,
	pollingHotshotPollingInterval time.Duration,
	espressoTEEVerifierCaller EspressoTEEVerifierInterface,
	espressoClientInterface EspressoClientInterface,
	recordPerformance bool,
) *EspressoStreamer {

	var PerfRecorder *PerfRecorder
	if recordPerformance {
		PerfRecorder = NewPerfRecorder()
	}

	return &EspressoStreamer{
		espressoClient:                espressoClientInterface,
		nextHotshotBlockNum:           nextHotshotBlockNum,
		retryTime:                     retryTime,
		pollingHotshotPollingInterval: pollingHotshotPollingInterval,
		namespace:                     namespace,
		espressoTEEVerifierCaller:     espressoTEEVerifierCaller,
		PerfRecorder:                  PerfRecorder,
	}
}

func (s *EspressoStreamer) Reset(currentMessagePos uint64, currentHostshotBlock uint64) {
	s.messageMutex.Lock()
	defer s.messageMutex.Unlock()
	s.currentMessagePos = currentMessagePos
	s.nextHotshotBlockNum = currentHostshotBlock
	s.messageWithMetadataAndPos = []*MessageWithMetadataAndPos{}
}

func (s *EspressoStreamer) Next() (*MessageWithMetadataAndPos, error) {
	s.messageMutex.Lock()
	defer s.messageMutex.Unlock()

	message, found := FilterAndFind(&s.messageWithMetadataAndPos, func(msg *MessageWithMetadataAndPos) int {
		if msg.Pos == s.currentMessagePos {
			return 0
		}
		if msg.Pos < s.currentMessagePos {
			return -1
		}
		return 1
	})
	if !found {
		return nil, nil
	}
	if message == nil {
		// This should never happen.
		return nil, fmt.Errorf("message is nil, but found is true")
	}

	s.currentMessagePos += 1
	return message, nil
}

/* Verify the attestation quote */
func (s *EspressoStreamer) verifyAttestationQuote(attestation []byte, userDataHash [32]byte) error {

	err := s.espressoTEEVerifierCaller.Verify(&bind.CallOpts{}, attestation, userDataHash)
	if err != nil {
		return fmt.Errorf("call to the espressoTEEVerifier contract failed: %w", err)
	}
	return nil
}

func (s *EspressoStreamer) parseEspressoTransaction(tx espressoTypes.Bytes) ([]*MessageWithMetadataAndPos, error) {
	attestation, userDataHash, indices, messages, err := arbutil.ParseHotShotPayload(tx)
	if err != nil {
		log.Warn("failed to parse hotshot payload", "err", err)
		return nil, err
	}
	// if attestation verification fails, we should skip this message
	// Parse the messages
	if len(userDataHash) != 32 {
		log.Warn("user data hash is not 32 bytes")
		return nil, fmt.Errorf("user data hash is not 32 bytes")
	}
	userDataHashArr := [32]byte(userDataHash)
	err = s.verifyAttestationQuote(attestation, userDataHashArr)
	if err != nil {
		log.Warn("failed to verify attestation quote", "err", err)
		return nil, err
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

/*
*
* Create a queue of messages from the hotshot to be processed by the node
* It will sort the messages by the message index
* and store the messages in `messagesWithMetadata` queue
*
* Expose the *parseHotShotPayloadFn* to the caller for testing purposes
 */
func (s *EspressoStreamer) QueueMessagesFromHotshot(
	ctx context.Context,
	parseHotShotPayloadFn func(tx espressoTypes.Bytes) ([]*MessageWithMetadataAndPos, error),
) error {

	s.messageMutex.Lock()
	defer s.messageMutex.Unlock()

	arbTxns, err := s.espressoClient.FetchTransactionsInBlock(ctx, s.nextHotshotBlockNum, s.namespace)
	if err != nil {
		return fmt.Errorf("%w: %w", FailedToFetchTransactionsErr, err)
	}

	if len(arbTxns.Transactions) == 0 {
		log.Debug("No transactions found in the hotshot block", "block number", s.nextHotshotBlockNum)
		s.nextHotshotBlockNum += 1
		return nil
	}

	for _, tx := range arbTxns.Transactions {
		messages, err := parseHotShotPayloadFn(tx)
		if err != nil {
			log.Warn("failed to verify espresso transaction", "err", err)
			continue
		}
		s.messageWithMetadataAndPos = append(s.messageWithMetadataAndPos, messages...)
	}

	s.nextHotshotBlockNum += 1

	return nil
}

func (s *EspressoStreamer) Start(ctxIn context.Context) error {
	s.StopWaiter.Start(ctxIn, s)

	ephemeralErrorHandler := util.NewEphemeralErrorHandler(3*time.Minute, FailedToFetchTransactionsErr.Error(), 1*time.Minute)
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
		log.Debug("Now processing hotshot block", "block number", s.nextHotshotBlockNum)
		return s.pollingHotshotPollingInterval
	})
	return err
}
