package espressostreamer

import (
	"context"
	"fmt"
	"sync"
	"time"

	espressoClient "github.com/EspressoSystems/espresso-sequencer-go/client"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

type MessageWithMetadataAndPos struct {
	MessageWithMeta arbostypes.MessageWithMetadata
	Pos             uint64
	HotshotHeight   uint64
}

type EspressoStreamer struct {
	stopwaiter.StopWaiter
	espressoClient                *espressoClient.MultipleNodesClient
	nextHotshotBlockNum           uint64
	currentMessagePos             uint64
	namespace                     uint64
	retryTime                     time.Duration
	pollingHotshotPollingInterval time.Duration
	messageWithMetadataAndPos     []*MessageWithMetadataAndPos
	espressoTEEVerifierCaller     bridgegen.EspressoTEEVerifier

	messageMutex sync.Mutex
}

func NewEspressoStreamer(namespace uint64, hotshotUrls []string,
	nextHotshotBlockNum uint64,
	retryTime time.Duration,
	pollingHotshotPollingInterval time.Duration,
	espressoTEEVerifierCaller bridgegen.EspressoTEEVerifier,
	currentMessagePos uint64,
) *EspressoStreamer {

	return &EspressoStreamer{
		espressoClient:                espressoClient.NewMultipleNodesClient(hotshotUrls),
		nextHotshotBlockNum:           nextHotshotBlockNum,
		retryTime:                     retryTime,
		pollingHotshotPollingInterval: pollingHotshotPollingInterval,
		namespace:                     namespace,
		espressoTEEVerifierCaller:     espressoTEEVerifierCaller,
		currentMessagePos:             currentMessagePos,
	}
}

func (s *EspressoStreamer) Reset(currentMessagePos uint64, currentHostshotBlock uint64) {
	s.messageMutex.Lock()
	defer s.messageMutex.Unlock()
	s.currentMessagePos = currentMessagePos
	s.nextHotshotBlockNum = currentHostshotBlock
	s.messageWithMetadataAndPos = []*MessageWithMetadataAndPos{}
}

func (s *EspressoStreamer) Next() (MessageWithMetadataAndPos, error) {
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
	if !found || message == nil {
		return MessageWithMetadataAndPos{}, fmt.Errorf("no message found")
	}
	s.currentMessagePos += 1
	return *message, nil
}

/* Verify the attestation quote */
func (s *EspressoStreamer) verifyAttestationQuote(attestation []byte, userDataHash [32]byte) error {

	err := s.espressoTEEVerifierCaller.Verify(&bind.CallOpts{}, attestation, userDataHash)
	if err != nil {
		return fmt.Errorf("call to the espressoTEEVerifier contract failed: %w", err)
	}
	return nil
}

/**
* Create a queue of messages from the hotshot to be processed by the node
* It will sort the messages by the message index
* and store the messages in `messagesWithMetadata` queue
 */
func (s *EspressoStreamer) queueMessagesFromHotshot(ctx context.Context) error {
	// Note: Adding the lock on top level
	// because s.nextHotshotBlockNum is updated if n.nextHotshotBlockNum == 0
	s.messageMutex.Lock()
	defer s.messageMutex.Unlock()

	if s.nextHotshotBlockNum == 0 {
		// We dont need to check majority here  because when we eventually go
		// to fetch a block at a certain height,
		// we will check that a quorum of nodes agree on the block at that height,
		// which wouldn't be possible if we were somehow are given a height
		// that wasn't finalized at all
		latestBlock, err := s.espressoClient.FetchLatestBlockHeight(ctx)
		if err != nil {
			log.Warn("unable to fetch latest hotshot block", "err", err)
			return err
		}
		log.Info("Started node at the latest hotshot block", "block number", latestBlock)
		s.nextHotshotBlockNum = latestBlock
	}

	arbTxns, err := s.espressoClient.FetchTransactionsInBlock(ctx, s.nextHotshotBlockNum, s.namespace)
	if err != nil {
		log.Warn("failed to fetch the transactions", "err", err)
		return err
	}

	if len(arbTxns.Transactions) == 0 {
		log.Info("No transactions found in the hotshot block", "block number", s.nextHotshotBlockNum)
		s.nextHotshotBlockNum += 1
		return nil
	}

	for _, tx := range arbTxns.Transactions {
		// Parse hotshot payload
		attestation, userDataHash, indices, messages, err := arbutil.ParseHotShotPayload(tx)
		if err != nil {
			log.Warn("failed to parse hotshot payload", "err", err)
			continue
		}
		// if attestation verification fails, we should skip this message
		// Parse the messages
		if len(userDataHash) != 32 {
			log.Warn("user data hash is not 32 bytes")
			continue
		}
		userDataHashArr := [32]byte(userDataHash)
		err = s.verifyAttestationQuote(attestation, userDataHashArr)
		if err != nil {
			log.Warn("failed to verify attestation quote", "err", err)
			continue
		}
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
			s.messageWithMetadataAndPos = append(s.messageWithMetadataAndPos, &MessageWithMetadataAndPos{
				MessageWithMeta: messageWithMetadata,
				Pos:             indices[i],
				HotshotHeight:   s.nextHotshotBlockNum,
			})
			log.Info("Added message to queue", "message", indices[i])
		}
	}

	s.nextHotshotBlockNum += 1

	return nil
}

func (s *EspressoStreamer) Start(ctxIn context.Context) error {
	s.StopWaiter.Start(ctxIn, s)
	err := s.CallIterativelySafe(func(ctx context.Context) time.Duration {
		err := s.queueMessagesFromHotshot(ctx)
		if err != nil {
			log.Error("error while queueing messages from hotshot", "err", err)
			return s.retryTime
		}
		log.Info("Now processing hotshot block", "block number", s.nextHotshotBlockNum)
		return s.pollingHotshotPollingInterval
	})
	return err
}
