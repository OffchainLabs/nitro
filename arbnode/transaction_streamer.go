// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbnode

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"os"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	espressoClient "github.com/EspressoSystems/espresso-network/sdks/go/client"
	lightclient "github.com/EspressoSystems/espresso-network/sdks/go/light-client"
	tagged_base64 "github.com/EspressoSystems/espresso-network/sdks/go/tagged-base64"
	espressoTypes "github.com/EspressoSystems/espresso-network/sdks/go/types"
	"github.com/ccoveille/go-safecast"
	"github.com/hf/nitrite"
	"github.com/hf/nsm"
	"github.com/hf/nsm/request"
	flag "github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/broadcastclient"
	"github.com/offchainlabs/nitro/broadcaster"
	m "github.com/offchainlabs/nitro/broadcaster/message"
	"github.com/offchainlabs/nitro/execution"
	"github.com/offchainlabs/nitro/staker"
	"github.com/offchainlabs/nitro/util"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/dbutil"
	"github.com/offchainlabs/nitro/util/sharedmetrics"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

// TransactionStreamer produces blocks from a node's L1 messages, storing the results in the blockchain and recording their positions
// The streamer is notified when there's new batches to process
type TransactionStreamer struct {
	stopwaiter.StopWaiter

	chainConfig    *params.ChainConfig
	exec           execution.ExecutionClient
	prevHeadMsgIdx *arbutil.MessageIndex
	validator      *staker.BlockValidator

	db             ethdb.Database
	fatalErrChan   chan<- error
	config         TransactionStreamerConfigFetcher
	snapSyncConfig *SnapSyncConfig

	insertionMutex             sync.Mutex // cannot be acquired while reorgMutex is held
	reorgMutex                 sync.RWMutex
	espressoPendingTxnPosMutex sync.Mutex
	espressoSubmittedTxnsMutex sync.Mutex

	newMessageNotifier     chan struct{}
	newSovereignTxNotifier chan struct{}

	nextAllowedFeedReorgLog time.Time

	broadcasterQueuedMessages            []arbostypes.MessageWithMetadataAndBlockInfo
	broadcasterQueuedMessagesFirstMsgIdx atomic.Uint64
	broadcasterQueuedMessagesActiveReorg bool

	coordinator     *SeqCoordinator
	broadcastServer *broadcaster.Broadcaster
	inboxReader     *InboxReader
	delayedBridge   *DelayedBridge

	trackBlockMetadataFrom arbutil.MessageIndex
	// Espresso specific fields. These fields are set from batch poster
	espressoClient                   espressoClient.EspressoClient
	lightClientReader                lightclient.LightClientReaderInterface
	espressoTxnsPollingInterval      time.Duration
	espressoTxnsSendingInterval      time.Duration
	espressoTxnsResubmissionInterval time.Duration
	maxBlockLagBeforeEscapeHatch     uint64
	espressoMaxTransactionSize       int64
	resubmitEspressoTxDeadline       time.Duration
	lastSubmitFailureAt              *time.Time
	// Public these fields for testing
	EscapeHatchEnabled                    bool
	UseEscapeHatch                        bool
	EspressoKeyManager                    EspressoKeyManagerInterface
	InitialFinalizedSequencerMessageCount *big.Int
}

type TransactionStreamerConfig struct {
	MaxBroadcasterQueueSize int           `koanf:"max-broadcaster-queue-size"`
	MaxReorgResequenceDepth int64         `koanf:"max-reorg-resequence-depth" reload:"hot"`
	ExecuteMessageLoopDelay time.Duration `koanf:"execute-message-loop-delay" reload:"hot"`
	SyncTillBlock           uint64        `koanf:"sync-till-block"`
	TrackBlockMetadataFrom  uint64        `koanf:"track-block-metadata-from"`
	UserDataAttestationFile string        `koanf:"user-data-attestation-file"`
	QuoteFile               string        `koanf:"quote-file"`
}

type TransactionStreamerConfigFetcher func() *TransactionStreamerConfig

var DefaultTransactionStreamerConfig = TransactionStreamerConfig{
	MaxBroadcasterQueueSize: 50_000,
	MaxReorgResequenceDepth: 1024,
	ExecuteMessageLoopDelay: time.Millisecond * 100,
	SyncTillBlock:           0,
	TrackBlockMetadataFrom:  0,
	QuoteFile:               "",
	UserDataAttestationFile: "",
}

var TestTransactionStreamerConfig = TransactionStreamerConfig{
	MaxBroadcasterQueueSize: 10_000,
	MaxReorgResequenceDepth: 128 * 1024,
	ExecuteMessageLoopDelay: time.Millisecond,
	SyncTillBlock:           0,
	TrackBlockMetadataFrom:  0,
}

func TransactionStreamerConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Int(prefix+".max-broadcaster-queue-size", DefaultTransactionStreamerConfig.MaxBroadcasterQueueSize, "maximum cache of pending broadcaster messages")
	f.Int64(prefix+".max-reorg-resequence-depth", DefaultTransactionStreamerConfig.MaxReorgResequenceDepth, "maximum number of messages to attempt to resequence on reorg (0 = never resequence, -1 = always resequence)")
	f.Duration(prefix+".execute-message-loop-delay", DefaultTransactionStreamerConfig.ExecuteMessageLoopDelay, "delay when polling calls to execute messages")
	f.Uint64(prefix+".sync-till-block", DefaultTransactionStreamerConfig.SyncTillBlock, "node will not sync past this block")
	f.Uint64(prefix+".track-block-metadata-from", DefaultTransactionStreamerConfig.TrackBlockMetadataFrom, "this is the block number starting from which blockmetadata is being tracked in the local disk and is being published to the feed. This is also the starting position for bulk syncing of missing blockmetadata. Setting to zero (default value) disables this")
	f.String(prefix+".user-data-attestation-file", DefaultTransactionStreamerConfig.UserDataAttestationFile, "specifies the file containing the user data attestation")
	f.String(prefix+".quote-file", DefaultTransactionStreamerConfig.QuoteFile, "specifies the file containing the quote")
}

func NewTransactionStreamer(
	ctx context.Context,
	db ethdb.Database,
	chainConfig *params.ChainConfig,
	exec execution.ExecutionClient,
	broadcastServer *broadcaster.Broadcaster,
	fatalErrChan chan<- error,
	config TransactionStreamerConfigFetcher,
	snapSyncConfig *SnapSyncConfig,
) (*TransactionStreamer, error) {

	// Check that chainId is within u32 range
	if chainConfig.ChainID.Uint64() > math.MaxUint32 {
		return nil, fmt.Errorf("chainId %d is out of range for u32", chainConfig.ChainID.Uint64())
	}

	streamer := &TransactionStreamer{
		exec:               exec,
		chainConfig:        chainConfig,
		db:                 db,
		newMessageNotifier: make(chan struct{}, 1),
		broadcastServer:    broadcastServer,
		fatalErrChan:       fatalErrChan,
		config:             config,
		snapSyncConfig:     snapSyncConfig,
		EscapeHatchEnabled: false,
	}

	err := streamer.cleanupInconsistentState()
	if err != nil {
		return nil, err
	}
	if config().TrackBlockMetadataFrom != 0 {
		trackBlockMetadataFrom, err := exec.BlockNumberToMessageIndex(config().TrackBlockMetadataFrom).Await(ctx)
		if err != nil {
			return nil, err
		}
		streamer.trackBlockMetadataFrom = trackBlockMetadataFrom
	}
	return streamer, nil
}

// Represents a block's hash in the database.
// Necessary because RLP decoder doesn't produce nil values by default.
type blockHashDBValue struct {
	BlockHash *common.Hash `rlp:"nil"`
}

const (
	BlockHashMismatchLogMsg    = "BlockHash from feed doesn't match locally computed hash. Check feed source."
	FailedToGetMsgResultFromDB = "Reading message result remotely."
)

var (
	ErrNoMessages = errors.New("No messages stored in the database.")
)

// Encodes a uint64 as bytes in a lexically sortable manner for database iteration.
// Generally this is only used for database keys, which need sorted.
// A shorter RLP encoding is usually used for database values.
func uint64ToKey(x uint64) []byte {
	data := make([]byte, 8)
	binary.BigEndian.PutUint64(data, x)
	return data
}

func (s *TransactionStreamer) SetBlockValidator(validator *staker.BlockValidator) {
	if s.Started() {
		panic("trying to set coordinator after start")
	}
	if s.validator != nil {
		panic("trying to set coordinator when already set")
	}
	s.validator = validator
}

func (s *TransactionStreamer) SetSeqCoordinator(coordinator *SeqCoordinator) {
	if s.Started() {
		panic("trying to set coordinator after start")
	}
	if s.coordinator != nil {
		panic("trying to set coordinator when already set")
	}
	s.coordinator = coordinator
}

func (s *TransactionStreamer) SetInboxReaders(inboxReader *InboxReader, delayedBridge *DelayedBridge) {
	if s.Started() {
		panic("trying to set inbox reader after start")
	}
	if s.inboxReader != nil || s.delayedBridge != nil {
		panic("trying to set inbox reader when already set")
	}
	s.inboxReader = inboxReader
	s.delayedBridge = delayedBridge
}

func (s *TransactionStreamer) ChainConfig() *params.ChainConfig {
	return s.chainConfig
}

func (s *TransactionStreamer) cleanupInconsistentState() error {
	// If it doesn't exist yet, set the message count to 0
	hasMessageCount, err := s.db.Has(messageCountKey)
	if err != nil {
		return err
	}
	if !hasMessageCount {
		err := setMessageCount(s.db, 0)
		if err != nil {
			return err
		}
	}
	// TODO remove trailing messageCountToMessage and messageCountToBlockPrefix entries
	return nil
}

func (s *TransactionStreamer) ReorgAt(firstMsgIdxReorged arbutil.MessageIndex) error {
	return s.ReorgAtAndEndBatch(s.db.NewBatch(), firstMsgIdxReorged)
}

func (s *TransactionStreamer) ReorgAtAndEndBatch(batch ethdb.Batch, firstMsgIdxReorged arbutil.MessageIndex) error {
	s.insertionMutex.Lock()
	defer s.insertionMutex.Unlock()
	err := s.addMessagesAndReorg(batch, firstMsgIdxReorged, nil)
	if err != nil {
		return err
	}
	err = batch.Write()
	if err != nil {
		return err
	}
	return nil
}

func deleteStartingAt(db ethdb.Database, batch ethdb.Batch, prefix []byte, minKey []byte) error {
	iter := db.NewIterator(prefix, minKey)
	defer iter.Release()
	for iter.Next() {
		err := batch.Delete(iter.Key())
		if err != nil {
			return err
		}
	}
	return iter.Error()
}

// deleteFromRange deletes key ranging from startMinKey(inclusive) to endMinKey(exclusive)

// might have deleted some keys even if returning an error

func deleteFromRange(ctx context.Context, db ethdb.Database, prefix []byte, startMinKey uint64, endMinKey uint64) ([]uint64, error) {
	batch := db.NewBatch()
	startIter := db.NewIterator(prefix, uint64ToKey(startMinKey))
	defer startIter.Release()
	var prunedKeysRange []uint64
	for startIter.Next() {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		currentKey := binary.BigEndian.Uint64(bytes.TrimPrefix(startIter.Key(), prefix))
		if currentKey >= endMinKey {
			break
		}
		if len(prunedKeysRange) == 0 || len(prunedKeysRange) == 1 {
			prunedKeysRange = append(prunedKeysRange, currentKey)
		} else {
			prunedKeysRange[1] = currentKey
		}
		err := batch.Delete(startIter.Key())
		if err != nil {
			return nil, err
		}
		if batch.ValueSize() >= ethdb.IdealBatchSize {
			if err := batch.Write(); err != nil {
				return nil, err
			}
			batch.Reset()
		}
	}
	if batch.ValueSize() > 0 {
		if err := batch.Write(); err != nil {
			return nil, err
		}
	}
	return prunedKeysRange, nil
}

// The insertion mutex must be held. This acquires the reorg mutex.
// Note: oldMessages will be empty if reorgHook is nil
func (s *TransactionStreamer) addMessagesAndReorg(batch ethdb.Batch, msgIdxOfFirstMsgToAdd arbutil.MessageIndex, newMessages []arbostypes.MessageWithMetadataAndBlockInfo) error {
	if msgIdxOfFirstMsgToAdd == 0 {
		return errors.New("cannot reorg out init message")
	}
	lastDelayedMsgIdx, err := s.getPrevPrevDelayedRead(msgIdxOfFirstMsgToAdd)
	if err != nil {
		return err
	}
	var oldMessages []*arbostypes.MessageWithMetadata

	currentHeadMsgIdx, err := s.GetHeadMessageIndex()
	if err != nil {
		return err
	}

	config := s.config()

	numberOfOldMsgsAfterLastMsgToKeep := currentHeadMsgIdx - msgIdxOfFirstMsgToAdd + 1
	// #nosec G115
	numberOfOldMsgsToResequence := min(
		arbutil.MessageIndex(config.MaxReorgResequenceDepth),
		numberOfOldMsgsAfterLastMsgToKeep,
	)
	if config.MaxReorgResequenceDepth >= 0 && numberOfOldMsgsToResequence < numberOfOldMsgsAfterLastMsgToKeep {
		log.Error(
			"unable to re-sequence all old messages because there are too many",
			"msgIdxOfFirstMsgToAdd", msgIdxOfFirstMsgToAdd,
			"removingMessages", numberOfOldMsgsAfterLastMsgToKeep-numberOfOldMsgsToResequence,
			"maxReorgResequenceDepth", config.MaxReorgResequenceDepth,
		)
	}
	// Gets old messages to re-sequence.
	for msgIdx := msgIdxOfFirstMsgToAdd; msgIdx < msgIdxOfFirstMsgToAdd+numberOfOldMsgsToResequence; msgIdx++ {
		oldMessage, err := s.GetMessage(msgIdx)
		if err != nil {
			log.Error("unable to lookup old message for re-sequencing", "msgIdx", msgIdx, "err", err)
			break
		}

		if oldMessage.Message == nil || oldMessage.Message.Header == nil {
			continue
		}

		header := oldMessage.Message.Header

		if header.RequestId != nil {
			// This is a delayed message
			delayedMsgIdx := header.RequestId.Big().Uint64()
			if delayedMsgIdx+1 != oldMessage.DelayedMessagesRead {
				log.Error("delayed message header RequestId doesn't match database DelayedMessagesRead", "header", oldMessage.Message.Header, "delayedMessagesRead", oldMessage.DelayedMessagesRead)
				continue
			}
			if delayedMsgIdx != lastDelayedMsgIdx {
				// This is the wrong position for the delayed message
				continue
			}
			if s.inboxReader != nil {
				// this is a delayed message. Should be resequenced if all 3 agree:
				// oldMessage, accumulator stored in tracker, and the message re-read from l1
				expectedAcc, err := s.inboxReader.tracker.GetDelayedAcc(delayedMsgIdx)
				if err != nil {
					if !strings.Contains(err.Error(), "not found") {
						log.Error("reorg-resequence: failed to read expected accumulator", "err", err)
					}
					continue
				}
				msgBlockNum := new(big.Int).SetUint64(oldMessage.Message.Header.BlockNumber)
				delayedInBlock, err := s.delayedBridge.LookupMessagesInRange(s.GetContext(), msgBlockNum, msgBlockNum, nil)
				if err != nil {
					log.Error("reorg-resequence: failed to serialize old delayed message from database", "err", err)
					continue
				}
				messageFound := false
			delayedInBlockLoop:
				for _, delayedFound := range delayedInBlock {
					if delayedFound.Message.Header.RequestId.Big().Uint64() != delayedMsgIdx {
						continue delayedInBlockLoop
					}
					if expectedAcc == delayedFound.AfterInboxAcc() && delayedFound.Message.Equals(oldMessage.Message) {
						messageFound = true
					}
					break delayedInBlockLoop
				}
				if !messageFound {
					continue
				}
			}
			lastDelayedMsgIdx++
		}

		oldMessages = append(oldMessages, oldMessage)
	}

	s.reorgMutex.Lock()
	defer s.reorgMutex.Unlock()

	messagesResults, err := s.exec.Reorg(msgIdxOfFirstMsgToAdd, newMessages, oldMessages).Await(s.GetContext())
	if err != nil {
		return err
	}

	messagesWithComputedBlockHash := make([]arbostypes.MessageWithMetadataAndBlockInfo, 0, len(messagesResults))
	for i := 0; i < len(messagesResults); i++ {
		messagesWithComputedBlockHash = append(messagesWithComputedBlockHash, arbostypes.MessageWithMetadataAndBlockInfo{
			MessageWithMeta: newMessages[i].MessageWithMeta,
			BlockHash:       &messagesResults[i].BlockHash,
		})
	}
	s.broadcastMessages(messagesWithComputedBlockHash, msgIdxOfFirstMsgToAdd)

	if s.validator != nil {
		err = s.validator.Reorg(s.GetContext(), msgIdxOfFirstMsgToAdd)
		if err != nil {
			return err
		}
	}

	err = deleteStartingAt(s.db, batch, messageResultPrefix, uint64ToKey(uint64(msgIdxOfFirstMsgToAdd)))
	if err != nil {
		return err
	}
	err = deleteStartingAt(s.db, batch, blockHashInputFeedPrefix, uint64ToKey(uint64(msgIdxOfFirstMsgToAdd)))
	if err != nil {
		return err
	}
	err = deleteStartingAt(s.db, batch, blockMetadataInputFeedPrefix, uint64ToKey(uint64(msgIdxOfFirstMsgToAdd)))
	if err != nil {
		return err
	}
	err = deleteStartingAt(s.db, batch, missingBlockMetadataInputFeedPrefix, uint64ToKey(uint64(msgIdxOfFirstMsgToAdd)))
	if err != nil {
		return err
	}
	err = deleteStartingAt(s.db, batch, messagePrefix, uint64ToKey(uint64(msgIdxOfFirstMsgToAdd)))
	if err != nil {
		return err
	}

	for i := 0; i < len(messagesResults); i++ {
		// #nosec G115
		msgIdx := msgIdxOfFirstMsgToAdd + arbutil.MessageIndex(i)
		err = s.storeResult(msgIdx, *messagesResults[i], batch)
		if err != nil {
			return err
		}
	}

	return setMessageCount(batch, msgIdxOfFirstMsgToAdd)
}

func setMessageCount(batch ethdb.KeyValueWriter, count arbutil.MessageIndex) error {
	countBytes, err := rlp.EncodeToBytes(count)
	if err != nil {
		return err
	}
	err = batch.Put(messageCountKey, countBytes)
	if err != nil {
		return err
	}
	sharedmetrics.UpdateSequenceNumberGauge(count)

	return nil
}

func dbKey(prefix []byte, pos uint64) []byte {
	var key []byte
	key = append(key, prefix...)
	key = append(key, uint64ToKey(pos)...)
	return key
}

// Note: if changed to acquire the mutex, some internal users may need to be updated to a non-locking version.

func (s *TransactionStreamer) GetMessage(msgIdx arbutil.MessageIndex) (*arbostypes.MessageWithMetadata, error) {
	key := dbKey(messagePrefix, uint64(msgIdx))
	data, err := s.db.Get(key)
	if err != nil {
		return nil, err
	}
	var message arbostypes.MessageWithMetadata
	err = rlp.DecodeBytes(data, &message)
	if err != nil {
		return nil, err
	}

	err = message.Message.FillInBatchGasCost(func(batchNum uint64) ([]byte, error) {
		ctx, err := s.GetContextSafe()
		if err != nil {
			return nil, err
		}
		data, _, err := s.inboxReader.GetSequencerMessageBytes(ctx, batchNum)
		return data, err
	})
	if err != nil {
		return nil, err
	}

	return &message, nil
}

func (s *TransactionStreamer) getMessageWithMetadataAndBlockInfo(msgIdx arbutil.MessageIndex) (*arbostypes.MessageWithMetadataAndBlockInfo, error) {
	msg, err := s.GetMessage(msgIdx)
	if err != nil {
		return nil, err
	}

	// Get block hash.
	// To keep it backwards compatible, since it is possible that a message related
	// to a sequence number exists in the database, but the block hash doesn't.
	key := dbKey(blockHashInputFeedPrefix, uint64(msgIdx))
	var blockHash *common.Hash
	data, err := s.db.Get(key)
	if err == nil {
		var blockHashDBVal blockHashDBValue
		err = rlp.DecodeBytes(data, &blockHashDBVal)
		if err != nil {
			return nil, err
		}
		blockHash = blockHashDBVal.BlockHash
	} else if !dbutil.IsErrNotFound(err) {
		return nil, err
	}

	blockMetadata, err := s.BlockMetadataAtMessageIndex(msgIdx)
	if err != nil {
		return nil, err
	}

	msgWithBlockInfo := arbostypes.MessageWithMetadataAndBlockInfo{
		MessageWithMeta: *msg,
		BlockHash:       blockHash,
		BlockMetadata:   blockMetadata,
	}
	return &msgWithBlockInfo, nil
}

// Note: if changed to acquire the mutex, some internal users may need to be updated to a non-locking version.

func (s *TransactionStreamer) GetMessageCount() (arbutil.MessageIndex, error) {
	countBytes, err := s.db.Get(messageCountKey)
	if err != nil {
		return 0, err
	}
	var count uint64
	err = rlp.DecodeBytes(countBytes, &count)
	if err != nil {
		return 0, err
	}
	return arbutil.MessageIndex(count), nil
}

func (s *TransactionStreamer) GetHeadMessageIndex() (arbutil.MessageIndex, error) {
	msgCount, err := s.GetMessageCount()
	if err != nil {
		return 0, err
	}
	if msgCount == 0 {
		return 0, ErrNoMessages
	}
	return msgCount - 1, nil
}

func (s *TransactionStreamer) GetProcessedMessageCount() (arbutil.MessageIndex, error) {
	msgCount, err := s.GetMessageCount()
	if err != nil {
		return 0, err
	}
	digestedHead, err := s.exec.HeadMessageIndex().Await(s.GetContext())
	if err != nil {
		return 0, err
	}
	if msgCount > digestedHead+1 {
		return digestedHead + 1, nil
	}
	return msgCount, nil
}

func (s *TransactionStreamer) AddMessages(firstMsgIdx arbutil.MessageIndex, messagesAreConfirmed bool, messages []arbostypes.MessageWithMetadata, blockMetadataArr []common.BlockMetadata) error {
	return s.AddMessagesAndEndBatch(firstMsgIdx, messagesAreConfirmed, messages, blockMetadataArr, nil)
}

func (s *TransactionStreamer) FeedPendingMessageCount() arbutil.MessageIndex {
	firstMsgIdx := s.broadcasterQueuedMessagesFirstMsgIdx.Load()
	if firstMsgIdx == 0 {
		return 0
	}

	s.insertionMutex.Lock()
	defer s.insertionMutex.Unlock()
	firstMsgIdx = s.broadcasterQueuedMessagesFirstMsgIdx.Load()
	if firstMsgIdx == 0 {
		return 0
	}
	return arbutil.MessageIndex(firstMsgIdx + uint64(len(s.broadcasterQueuedMessages)))
}

func (s *TransactionStreamer) AddBroadcastMessages(feedMessages []*m.BroadcastFeedMessage) error {
	if len(feedMessages) == 0 {
		return nil
	}
	broadcastFirstMsgIdx := feedMessages[0].SequenceNumber
	var messages []arbostypes.MessageWithMetadataAndBlockInfo
	expectedMsgIdx := broadcastFirstMsgIdx
	for _, feedMessage := range feedMessages {
		if expectedMsgIdx != feedMessage.SequenceNumber {
			return fmt.Errorf("invalid sequence number %v, expected %v", feedMessage.SequenceNumber, expectedMsgIdx)
		}
		if feedMessage.Message.Message == nil || feedMessage.Message.Message.Header == nil {
			return fmt.Errorf("invalid feed message at sequence number %v", feedMessage.SequenceNumber)
		}
		msgWithBlockInfo := arbostypes.MessageWithMetadataAndBlockInfo{
			MessageWithMeta: feedMessage.Message,
			BlockHash:       feedMessage.BlockHash,
			BlockMetadata:   feedMessage.BlockMetadata,
		}
		messages = append(messages, msgWithBlockInfo)
		expectedMsgIdx++
	}

	s.insertionMutex.Lock()
	defer s.insertionMutex.Unlock()

	var feedReorg bool
	var err error
	// Skip any messages already in the database
	// prevDelayedRead set to 0 because it's only used to compute the output prevDelayedRead which is not used here
	// Messages from feed are not confirmed, so confirmedMessageCount is 0 and confirmedReorg can be ignored
	numberOfDuplicates, feedReorg, oldMsg, err := s.countDuplicateMessages(broadcastFirstMsgIdx, messages, nil)
	if err != nil {
		return err
	}
	messages = messages[numberOfDuplicates:]
	broadcastFirstMsgIdx += arbutil.MessageIndex(numberOfDuplicates)
	if oldMsg != nil {
		s.logReorg(broadcastFirstMsgIdx, oldMsg, &messages[0].MessageWithMeta, false)
	}
	if len(messages) == 0 {
		// No new messages received
		return nil
	}

	if len(s.broadcasterQueuedMessages) == 0 || (feedReorg && !s.broadcasterQueuedMessagesActiveReorg) {
		// Empty cache or feed different from database, save current feed messages until confirmed L1 messages catch up.
		s.broadcasterQueuedMessages = messages
		s.broadcasterQueuedMessagesFirstMsgIdx.Store(uint64(broadcastFirstMsgIdx))
		s.broadcasterQueuedMessagesActiveReorg = feedReorg
	} else {
		broadcasterQueuedMessagesFirstMsgIdx := arbutil.MessageIndex(s.broadcasterQueuedMessagesFirstMsgIdx.Load())
		if broadcasterQueuedMessagesFirstMsgIdx >= broadcastFirstMsgIdx {
			// Feed messages older than cache
			s.broadcasterQueuedMessages = messages
			s.broadcasterQueuedMessagesFirstMsgIdx.Store(uint64(broadcastFirstMsgIdx))
			s.broadcasterQueuedMessagesActiveReorg = feedReorg
		} else if broadcasterQueuedMessagesFirstMsgIdx+arbutil.MessageIndex(len(s.broadcasterQueuedMessages)) == broadcastFirstMsgIdx {
			// Feed messages can be added directly to end of cache
			maxQueueSize := s.config().MaxBroadcasterQueueSize
			if maxQueueSize == 0 || len(s.broadcasterQueuedMessages) <= maxQueueSize {
				s.broadcasterQueuedMessages = append(s.broadcasterQueuedMessages, messages...)
			}
			broadcastFirstMsgIdx = broadcasterQueuedMessagesFirstMsgIdx
			// Do not change existing reorg state
		} else {
			if len(s.broadcasterQueuedMessages) > 0 {
				log.Warn(
					"broadcaster queue jumped positions",
					"queuedMessages", len(s.broadcasterQueuedMessages),
					"expectedNextIdx", broadcasterQueuedMessagesFirstMsgIdx+arbutil.MessageIndex(len(s.broadcasterQueuedMessages)),
					"gotIdx", broadcastFirstMsgIdx,
				)
			}
			s.broadcasterQueuedMessages = messages
			s.broadcasterQueuedMessagesFirstMsgIdx.Store(uint64(broadcastFirstMsgIdx))
			s.broadcasterQueuedMessagesActiveReorg = feedReorg
		}
	}

	if s.broadcasterQueuedMessagesActiveReorg || len(s.broadcasterQueuedMessages) == 0 {
		// Broadcaster never triggered reorg or no messages to add
		return nil
	}

	if broadcastFirstMsgIdx > 0 {
		_, err := s.GetMessage(broadcastFirstMsgIdx - 1)
		if err != nil {
			if !dbutil.IsErrNotFound(err) {
				return err
			}
			// Message before current message doesn't exist in database, so don't add current messages yet
			return nil
		}
	}

	err = s.addMessagesAndEndBatchImpl(broadcastFirstMsgIdx, false, nil, nil)
	if err != nil {
		return fmt.Errorf("error adding pending broadcaster messages: %w", err)
	}

	return nil
}

// AddFakeInitMessage should only be used for testing or running a local dev node
func (s *TransactionStreamer) AddFakeInitMessage() error {
	chainConfigJson, err := json.Marshal(s.chainConfig)
	if err != nil {
		return fmt.Errorf("failed to serialize chain config: %w", err)
	}
	chainIdBytes := arbmath.U256Bytes(s.chainConfig.ChainID)
	msg := append(append(chainIdBytes, 0), chainConfigJson...)
	return s.AddMessages(0, false, []arbostypes.MessageWithMetadata{{
		Message: &arbostypes.L1IncomingMessage{
			Header: &arbostypes.L1IncomingMessageHeader{
				Kind:      arbostypes.L1MessageType_Initialize,
				RequestId: &common.Hash{},
				L1BaseFee: common.Big0,
			},
			L2msg: msg,
		},
		DelayedMessagesRead: 1,
	}}, nil)
}

// Used in redis tests
func (s *TransactionStreamer) GetMessageCountSync(t *testing.T) (arbutil.MessageIndex, error) {
	s.insertionMutex.Lock()
	defer s.insertionMutex.Unlock()
	return s.GetMessageCount()
}

func endBatch(batch ethdb.Batch) error {
	if batch == nil {
		return nil
	}
	return batch.Write()
}

func (s *TransactionStreamer) AddMessagesAndEndBatch(firstMsgIdx arbutil.MessageIndex, messagesAreConfirmed bool, messages []arbostypes.MessageWithMetadata, blockMetadataArr []common.BlockMetadata, batch ethdb.Batch) error {
	messagesWithBlockInfo := make([]arbostypes.MessageWithMetadataAndBlockInfo, 0, len(messages))
	for _, message := range messages {
		messagesWithBlockInfo = append(messagesWithBlockInfo, arbostypes.MessageWithMetadataAndBlockInfo{
			MessageWithMeta: message,
		})
	}

	if len(blockMetadataArr) == len(messagesWithBlockInfo) {
		for i, blockMetadata := range blockMetadataArr {
			messagesWithBlockInfo[i].BlockMetadata = blockMetadata
		}
	} else if len(blockMetadataArr) > 0 {
		return fmt.Errorf("size of blockMetadata array doesn't match the size of messages array. lockMetadataArrSize: %d, messagesSize: %d", len(blockMetadataArr), len(messages))
	}

	if messagesAreConfirmed {
		// Trim confirmed messages from l1pricedataCache
		_, err := s.exec.MarkFeedStart(firstMsgIdx + arbutil.MessageIndex(len(messages))).Await(s.GetContext())
		if err != nil {
			log.Warn("TransactionStreamer: failed to mark feed start", "firstMsgIdx", firstMsgIdx, "err", err)
		}
		s.reorgMutex.RLock()
		numberOfDuplicates, _, _, err := s.countDuplicateMessages(firstMsgIdx, messagesWithBlockInfo, nil)
		s.reorgMutex.RUnlock()
		if err != nil {
			return err
		}
		if numberOfDuplicates == uint64(len(messages)) {
			return endBatch(batch)
		}
		// cant keep reorg lock when catching insertionMutex.
		// we have to re-evaluate all messages
		// happy cases for confirmed messages:
		// 1: were previously in feed. We saved work
		// 2: are new (syncing). We wasted very little work.
	}
	s.insertionMutex.Lock()
	defer s.insertionMutex.Unlock()

	return s.addMessagesAndEndBatchImpl(firstMsgIdx, messagesAreConfirmed, messagesWithBlockInfo, batch)
}

func (s *TransactionStreamer) getPrevPrevDelayedRead(msgIdx arbutil.MessageIndex) (uint64, error) {
	if s.snapSyncConfig.Enabled && uint64(msgIdx) == s.snapSyncConfig.PrevBatchMessageCount {
		return s.snapSyncConfig.PrevDelayedRead, nil
	}
	var prevDelayedRead uint64
	if msgIdx > 0 {
		prevMsg, err := s.GetMessage(msgIdx - 1)
		if err != nil {
			return 0, fmt.Errorf("failed to get previous message for msgIdx %d: %w", msgIdx, err)
		}
		prevDelayedRead = prevMsg.DelayedMessagesRead
	}

	return prevDelayedRead, nil
}

func (s *TransactionStreamer) countDuplicateMessages(
	msgIdx arbutil.MessageIndex,
	messages []arbostypes.MessageWithMetadataAndBlockInfo,
	batch *ethdb.Batch,
) (uint64, bool, *arbostypes.MessageWithMetadata, error) {
	var curMsg uint64
	for {
		if uint64(len(messages)) == curMsg {
			break
		}
		key := dbKey(messagePrefix, uint64(msgIdx))
		hasMessage, err := s.db.Has(key)
		if err != nil {
			return 0, false, nil, err
		}
		if !hasMessage {
			break
		}
		haveMessage, err := s.db.Get(key)
		if err != nil {
			return 0, false, nil, err
		}
		nextMessage := messages[curMsg]
		wantMessage, err := rlp.EncodeToBytes(nextMessage.MessageWithMeta)
		if err != nil {
			return 0, false, nil, err
		}
		if !bytes.Equal(haveMessage, wantMessage) {
			// Current message does not exactly match message in database
			var dbMessageParsed arbostypes.MessageWithMetadata

			if err := rlp.DecodeBytes(haveMessage, &dbMessageParsed); err != nil {
				log.Warn("TransactionStreamer: Reorg detected! (failed parsing db message)",
					"msgIdx", msgIdx,
					"err", err,
				)
				return curMsg, true, nil, nil
			}
			var duplicateMessage bool
			if nextMessage.MessageWithMeta.Message != nil {
				if dbMessageParsed.Message.BatchGasCost == nil || nextMessage.MessageWithMeta.Message.BatchGasCost == nil {
					// Remove both of the batch gas costs and see if the messages still differ
					nextMessageCopy := nextMessage.MessageWithMeta
					nextMessageCopy.Message = new(arbostypes.L1IncomingMessage)
					*nextMessageCopy.Message = *nextMessage.MessageWithMeta.Message
					batchGasCostBkup := dbMessageParsed.Message.BatchGasCost
					dbMessageParsed.Message.BatchGasCost = nil
					nextMessageCopy.Message.BatchGasCost = nil
					if reflect.DeepEqual(dbMessageParsed, nextMessageCopy) {
						// Actually this isn't a reorg; only the batch gas costs differed
						duplicateMessage = true
						// If possible - update the message in the database to add the gas cost cache.
						if batch != nil && nextMessage.MessageWithMeta.Message.BatchGasCost != nil {
							if *batch == nil {
								*batch = s.db.NewBatch()
							}
							if err := s.writeMessage(msgIdx, nextMessage, *batch); err != nil {
								return 0, false, nil, err
							}
						}
					}
					dbMessageParsed.Message.BatchGasCost = batchGasCostBkup
				}
			}

			if !duplicateMessage {
				return curMsg, true, &dbMessageParsed, nil
			}
		}

		curMsg++
		msgIdx++
	}

	return curMsg, false, nil, nil
}

func (s *TransactionStreamer) logReorg(msgIdx arbutil.MessageIndex, dbMsg *arbostypes.MessageWithMetadata, newMsg *arbostypes.MessageWithMetadata, confirmed bool) {
	sendLog := confirmed
	if time.Now().After(s.nextAllowedFeedReorgLog) {
		sendLog = true
	}
	if sendLog {
		s.nextAllowedFeedReorgLog = time.Now().Add(time.Minute)
		log.Warn("TransactionStreamer: Reorg detected!",
			"confirmed", confirmed,
			"msgIdx", msgIdx,
			"got-delayed", newMsg.DelayedMessagesRead,
			"got-header", newMsg.Message.Header,
			"db-delayed", dbMsg.DelayedMessagesRead,
			"db-header", dbMsg.Message.Header,
		)
	}

}

func (s *TransactionStreamer) addMessagesAndEndBatchImpl(firstMsgIdx arbutil.MessageIndex, messagesAreConfirmed bool, messages []arbostypes.MessageWithMetadataAndBlockInfo, batch ethdb.Batch) error {
	var confirmedReorg bool
	var oldMsg *arbostypes.MessageWithMetadata
	var lastDelayedRead uint64
	var hasNewConfirmedMessages bool
	var cacheClearLen int

	headMsgIdxAfterInsert := firstMsgIdx + arbutil.MessageIndex(len(messages))
	broadcastFirstMsgIdx := arbutil.MessageIndex(s.broadcasterQueuedMessagesFirstMsgIdx.Load())

	if messagesAreConfirmed {
		var numberOfDuplicates uint64
		var err error
		numberOfDuplicates, confirmedReorg, oldMsg, err = s.countDuplicateMessages(firstMsgIdx, messages, &batch)
		if err != nil {
			return err
		}
		if numberOfDuplicates > 0 {
			lastDelayedRead = messages[numberOfDuplicates-1].MessageWithMeta.DelayedMessagesRead
			messages = messages[numberOfDuplicates:]
			firstMsgIdx += arbutil.MessageIndex(numberOfDuplicates)
		}
		if len(messages) > 0 {
			hasNewConfirmedMessages = true
		}
	}

	clearQueueOnSuccess := false
	if (s.broadcasterQueuedMessagesActiveReorg && firstMsgIdx <= broadcastFirstMsgIdx) ||
		(!s.broadcasterQueuedMessagesActiveReorg && broadcastFirstMsgIdx <= headMsgIdxAfterInsert) {
		// Active broadcast reorg and L1 messages at or before start of broadcast messages
		// Or no active broadcast reorg and broadcast messages start before or immediately after last L1 message
		if headMsgIdxAfterInsert >= broadcastFirstMsgIdx {
			// #nosec G115
			broadcastSliceIndex := int(headMsgIdxAfterInsert - broadcastFirstMsgIdx)
			messagesOldLen := len(messages)
			if broadcastSliceIndex < len(s.broadcasterQueuedMessages) {
				// Some cached feed messages can be used
				messages = append(messages, s.broadcasterQueuedMessages[broadcastSliceIndex:]...)
			}
			// This calculation gives the exact length of cache which was appended to messages
			cacheClearLen = broadcastSliceIndex + len(messages) - messagesOldLen
		}

		// L1 used or replaced broadcast cache items
		clearQueueOnSuccess = true
	}

	var feedReorg bool
	if !hasNewConfirmedMessages {
		var numberOfDuplicates uint64
		var err error
		numberOfDuplicates, feedReorg, oldMsg, err = s.countDuplicateMessages(firstMsgIdx, messages, nil)
		if err != nil {
			return err
		}
		if numberOfDuplicates > 0 {
			lastDelayedRead = messages[numberOfDuplicates-1].MessageWithMeta.DelayedMessagesRead
			messages = messages[numberOfDuplicates:]
			firstMsgIdx += arbutil.MessageIndex(numberOfDuplicates)
		}
	}
	if oldMsg != nil {
		s.logReorg(firstMsgIdx, oldMsg, &messages[0].MessageWithMeta, confirmedReorg)
	}

	if feedReorg {
		// Never allow feed to reorg confirmed messages
		// Note that any remaining messages must be feed messages, so we're done here
		return endBatch(batch)
	}

	if lastDelayedRead == 0 {
		var err error
		lastDelayedRead, err = s.getPrevPrevDelayedRead(firstMsgIdx)
		if err != nil {
			return err
		}
	}

	// Validate delayed message counts of remaining messages
	for i, msg := range messages {
		// #nosec G115
		msgIdx := firstMsgIdx + arbutil.MessageIndex(i)
		diff := msg.MessageWithMeta.DelayedMessagesRead - lastDelayedRead
		if diff != 0 && diff != 1 {
			return fmt.Errorf("attempted to insert jump from %v delayed messages read to %v delayed messages read at message index %v", lastDelayedRead, msg.MessageWithMeta.DelayedMessagesRead, msgIdx)
		}
		lastDelayedRead = msg.MessageWithMeta.DelayedMessagesRead
		if msg.MessageWithMeta.Message == nil {
			return fmt.Errorf("attempted to insert nil message at index %v", msgIdx)
		}
	}

	if confirmedReorg {
		reorgBatch := s.db.NewBatch()
		err := s.addMessagesAndReorg(reorgBatch, firstMsgIdx, messages)
		if err != nil {
			return err
		}
		err = reorgBatch.Write()
		if err != nil {
			return err
		}
	}
	if len(messages) == 0 {
		return endBatch(batch)
	}

	err := s.writeMessages(firstMsgIdx, messages, batch)
	if err != nil {
		return err
	}

	if clearQueueOnSuccess {
		// Check if new messages were added at the end of cache, if they were, then don't remove those particular messages
		if len(s.broadcasterQueuedMessages) > cacheClearLen {
			s.broadcasterQueuedMessages = s.broadcasterQueuedMessages[cacheClearLen:]
			// #nosec G115
			s.broadcasterQueuedMessagesFirstMsgIdx.Store(uint64(broadcastFirstMsgIdx) + uint64(cacheClearLen))
		} else {
			s.broadcasterQueuedMessages = s.broadcasterQueuedMessages[:0]
			s.broadcasterQueuedMessagesFirstMsgIdx.Store(0)
		}
		s.broadcasterQueuedMessagesActiveReorg = false
	}

	return nil
}

// The caller must hold the insertionMutex

func (s *TransactionStreamer) ExpectChosenSequencer() error {
	if s.coordinator != nil {
		if !s.coordinator.CurrentlyChosen() {
			return fmt.Errorf("%w: not main sequencer", execution.ErrRetrySequencer)
		}
	}
	return nil
}

func (s *TransactionStreamer) WriteMessageFromSequencer(
	msgIdx arbutil.MessageIndex,
	msgWithMeta arbostypes.MessageWithMetadata,
	msgResult execution.MessageResult,
	blockMetadata common.BlockMetadata,
) error {

	if err := s.ExpectChosenSequencer(); err != nil {
		return err
	}

	lock := func() bool {
		// Considering current Nitro's Consensus <-> Execution circular dependency design,
		// there are some scenarios in which using s.insertionMutex.Lock() here would cause a deadlock.
		// As an example, considering t(i) as times, and that t(i) occurs before t(i+1):
		// t(1): Consensus identifies a Reorg and locks insertionMutex in ReorgAtAndEndBatch
		// t(2): Execution sequences a message and locks createBlockMutex
		// t(3): Consensus calls Execution.Reorg, which waits until createBlockMutex is available
		// t(4): Execution calls Consensus.WriteMessageFromSequencer, which waits until insertionMutex is available
		// t(3) and t(4) define a deadlock.
		//
		// In the other hand, a simple s.insertionMutex.TryLock() can cause some issues when resequencing reorgs, such as:
		// 1. TransactionStreamer, holding insertionMutex lock, calls ExecutionEngine, which then adds old messages to a channel.
		// After that, and before releasing the lock, TransactionStreamer does more computations.
		// 2. Asynchronously, ExecutionEngine reads from this channel and calls TransactionStreamer,
		// which expects that insertionMutex is free in order to succeed.
		// If step 1 is still executing when Execution calls TransactionStreamer in step 2 then s.insertionMutex.TryLock() will fail.
		//
		// This retry lock with timeout mechanism is a workaround to avoid deadlocks,
		// but enabling some reorg resequencing scenarios.

		if s.insertionMutex.TryLock() {
			return true
		}
		lockTick := time.Tick(5 * time.Millisecond)
		lockTimeout := time.After(50 * time.Millisecond)
		for {
			select {
			case <-lockTimeout:
				return false
			default:
				select {
				case <-lockTimeout:
					return false
				case <-lockTick:
					if s.insertionMutex.TryLock() {
						return true
					}
				}
			}
		}
	}
	if !lock() {
		return execution.ErrSequencerInsertLockTaken
	}
	defer s.insertionMutex.Unlock()

	headMsgIdx, err := s.GetHeadMessageIndex()
	expectedMsgIdx := headMsgIdx + 1
	if errors.Is(err, ErrNoMessages) {
		expectedMsgIdx = 0
	} else if err != nil {
		return err
	}

	if msgIdx != expectedMsgIdx {
		return fmt.Errorf("wrong msgIdx got %d expected %d", msgIdx, expectedMsgIdx)
	}

	if s.coordinator != nil {
		if err := s.coordinator.SequencingMessage(msgIdx, &msgWithMeta, blockMetadata); err != nil {
			return err
		}
	}

	msgWithBlockInfo := arbostypes.MessageWithMetadataAndBlockInfo{
		MessageWithMeta: msgWithMeta,
		BlockHash:       &msgResult.BlockHash,
		BlockMetadata:   blockMetadata,
	}

	if err := s.writeMessages(msgIdx, []arbostypes.MessageWithMetadataAndBlockInfo{msgWithBlockInfo}, nil); err != nil {
		return err
	}
	if s.trackBlockMetadataFrom == 0 || msgIdx < s.trackBlockMetadataFrom {
		msgWithBlockInfo.BlockMetadata = nil
	}
	s.broadcastMessages([]arbostypes.MessageWithMetadataAndBlockInfo{msgWithBlockInfo}, msgIdx)

	return nil
}

// PauseReorgs until a matching call to ResumeReorgs (may be called concurrently)

func (s *TransactionStreamer) PauseReorgs() {
	s.reorgMutex.RLock()
}

func (s *TransactionStreamer) ResumeReorgs() {
	s.reorgMutex.RUnlock()
}

func (s *TransactionStreamer) PopulateFeedBacklog() error {
	if s.broadcastServer == nil || s.inboxReader == nil {
		return nil
	}
	return s.inboxReader.tracker.PopulateFeedBacklog(s.broadcastServer)
}

func (s *TransactionStreamer) writeMessage(msgIdx arbutil.MessageIndex, msg arbostypes.MessageWithMetadataAndBlockInfo, batch ethdb.Batch) error {
	// write message with metadata
	key := dbKey(messagePrefix, uint64(msgIdx))
	msgBytes, err := rlp.EncodeToBytes(msg.MessageWithMeta)
	if err != nil {
		return err
	}
	if err := batch.Put(key, msgBytes); err != nil {
		return err
	}

	// write block hash
	blockHashDBVal := blockHashDBValue{
		BlockHash: msg.BlockHash,
	}
	key = dbKey(blockHashInputFeedPrefix, uint64(msgIdx))
	msgBytes, err = rlp.EncodeToBytes(blockHashDBVal)
	if err != nil {
		return err
	}
	if err := batch.Put(key, msgBytes); err != nil {
		return err
	}

	if s.trackBlockMetadataFrom != 0 && msgIdx >= s.trackBlockMetadataFrom {
		if msg.BlockMetadata != nil {
			// Only store non-nil BlockMetadata to db. In case of a reorg, we dont have to explicitly
			// clear out BlockMetadata of the reorged message, since those messages will be handled by s.reorg()
			// This also allows update of BatchGasCost in message without mistakenly erasing BlockMetadata
			key = dbKey(blockMetadataInputFeedPrefix, uint64(msgIdx))
			return batch.Put(key, msg.BlockMetadata)
		} else {
			// Mark that blockMetadata is missing only if it isn't already present. This check prevents unnecessary marking
			// when updating BatchGasCost or when adding messages from seq-coordinator redis that doesn't have block metadata
			prevBlockMetadata, err := s.BlockMetadataAtMessageIndex(msgIdx)
			if err != nil {
				return err
			}
			if prevBlockMetadata == nil {
				key = dbKey(missingBlockMetadataInputFeedPrefix, uint64(msgIdx))
				return batch.Put(key, nil)
			}
		}
	}
	return nil
}

func (s *TransactionStreamer) broadcastMessages(
	msgs []arbostypes.MessageWithMetadataAndBlockInfo,
	firstMsgIdx arbutil.MessageIndex,
) {
	if s.broadcastServer == nil {
		return
	}
	if err := s.broadcastServer.BroadcastMessages(msgs, firstMsgIdx); err != nil {
		log.Error("failed broadcasting messages", "firstMsgIdx", firstMsgIdx, "err", err)
	}
}

// The mutex must be held, and firstMsgIdx must be the latest message count.

// `batch` may be nil, which initializes a new batch. The batch is closed out in this function.

func (s *TransactionStreamer) writeMessages(firstMsgIdx arbutil.MessageIndex, messages []arbostypes.MessageWithMetadataAndBlockInfo, batch ethdb.Batch) error {
	if s.config().SyncTillBlock > 0 && uint64(firstMsgIdx) > s.config().SyncTillBlock {
		return broadcastclient.TransactionStreamerBlockCreationStopped
	}
	if batch == nil {
		batch = s.db.NewBatch()
	}
	for i, msg := range messages {
		if len(msg.MessageWithMeta.Message.L2msg) > arbostypes.MaxL2MessageSize {
			// #nosec G115
			log.Warn("L2 message is too large", "pos", firstMsgIdx+arbutil.MessageIndex(i), "size", len(msg.MessageWithMeta.Message.L2msg))
			return fmt.Errorf("L2 message is too large")
		}
		// #nosec G115
		err := s.writeMessage(firstMsgIdx+arbutil.MessageIndex(i), msg, batch)
		if err != nil {
			return err
		}
	}

	err := setMessageCount(batch, firstMsgIdx+arbutil.MessageIndex(len(messages)))
	if err != nil {
		return err
	}

	//  If light client reader and espresso client are set, then we need to store the pos in the database
	//  to be used later to submit the message to hotshot for finalization.
	if s.lightClientReader != nil && s.espressoClient != nil {
		//  Only submit the transaction if escape hatch is not enabled
		var messagesToEnqueue []arbutil.MessageIndex
		for i := range messages {
			idx, err := safecast.ToUint64(i)
			if err != nil {
				return err
			}
			indexToSubmit := (firstMsgIdx + arbutil.MessageIndex(idx))

			// convert to uint64
			indexToSubmitUint64, err := safecast.ToUint64(indexToSubmit)
			if err != nil {
				return err
			}
			if s.shouldSubmitEspressoTransaction(&indexToSubmitUint64) {
				log.Info("adding transaction to list of pending tx's to submit to Espresso", "pos", indexToSubmit)
				messagesToEnqueue = append(messagesToEnqueue, indexToSubmit)
			}
		}

		if len(messagesToEnqueue) > 0 {
			err = s.enqueuePendingTransaction(messagesToEnqueue)
			if err != nil {
				log.Error("unable to enqueue a transaction to the pending list to be submitted to espresso.", "err", err, "messages", messagesToEnqueue)
				return err
			}
			startIdx := messagesToEnqueue[0]
			endIdx := messagesToEnqueue[len(messagesToEnqueue)-1]
			log.Info("Successfully enqueued range of transactions from startIdx to endIdx", "startIdx", startIdx, "endIdx", endIdx)
		}
	}

	err = batch.Write()
	if err != nil {
		return err
	}

	select {
	case s.newMessageNotifier <- struct{}{}:
	default:
	}

	return nil
}

func (s *TransactionStreamer) BlockMetadataAtMessageIndex(msgIdx arbutil.MessageIndex) (common.BlockMetadata, error) {
	if s.trackBlockMetadataFrom == 0 || msgIdx < s.trackBlockMetadataFrom {
		return nil, nil
	}

	key := dbKey(blockMetadataInputFeedPrefix, uint64(msgIdx))
	blockMetadata, err := s.db.Get(key)
	if err != nil {
		if dbutil.IsErrNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return blockMetadata, nil
}

func (s *TransactionStreamer) enqueuePendingTransaction(pos []arbutil.MessageIndex) error {
	// Store the pos in the database to be used later to submit the message
	// to hotshot for finalization.
	err := s.SubmitEspressoTransactionPos(pos)
	if err != nil {
		log.Error("failed to submit espresso transaction pos", "pos", pos, "err", err)
		return err
	}

	return nil
}

func (s *TransactionStreamer) ResultAtMessageIndex(msgIdx arbutil.MessageIndex) (*execution.MessageResult, error) {
	key := dbKey(messageResultPrefix, uint64(msgIdx))
	data, err := s.db.Get(key)
	if err == nil {
		var msgResult execution.MessageResult
		err = rlp.DecodeBytes(data, &msgResult)
		if err == nil {
			return &msgResult, nil
		}
	} else if !dbutil.IsErrNotFound(err) {
		return nil, err
	}
	log.Info(FailedToGetMsgResultFromDB, "msgIdx", msgIdx)

	ctx := context.Background()
	if s.Started() {
		ctx = s.GetContext()
	}
	msgResult, err := s.exec.ResultAtMessageIndex(msgIdx).Await(ctx)
	if err != nil {
		return nil, err
	}
	// Stores result in Consensus DB in a best-effort manner
	batch := s.db.NewBatch()
	err = s.storeResult(msgIdx, *msgResult, batch)
	if err != nil {
		log.Warn("Failed to store result at ResultAtMessageIndex", "err", err)
		return msgResult, nil
	}
	err = batch.Write()
	if err != nil {
		log.Warn("Failed to store result at ResultAtMessageIndex", "err", err)
		return msgResult, nil
	}

	return msgResult, nil
}

func (s *TransactionStreamer) checkResult(msgIdx arbutil.MessageIndex, msgResult *execution.MessageResult, msgAndBlockInfo *arbostypes.MessageWithMetadataAndBlockInfo) {
	if msgAndBlockInfo.BlockHash == nil {
		return
	}
	if msgResult.BlockHash != *msgAndBlockInfo.BlockHash {
		log.Error(
			BlockHashMismatchLogMsg,
			"expected", msgAndBlockInfo.BlockHash,
			"actual", msgResult.BlockHash,
		)
		// Try deleting the existing blockMetadata for this block in arbDB and set it as missing
		if msgAndBlockInfo.BlockMetadata != nil &&
			s.trackBlockMetadataFrom != 0 && msgIdx >= s.trackBlockMetadataFrom {
			batch := s.db.NewBatch()
			if err := batch.Delete(dbKey(blockMetadataInputFeedPrefix, uint64(msgIdx))); err != nil {
				log.Error("error deleting blockMetadata of block whose BlockHash from feed doesn't match locally computed hash", "msgIdx", msgIdx, "err", err)
				return
			}
			if err := batch.Put(dbKey(missingBlockMetadataInputFeedPrefix, uint64(msgIdx)), nil); err != nil {
				log.Error("error marking deleted blockMetadata as missing in arbDB for a block whose BlockHash from feed doesn't match locally computed hash", "msgIdx", msgIdx, "err", err)
				return
			}
			if err := batch.Write(); err != nil {
				log.Error("error writing batch that deletes blockMetadata of the block whose BlockHash from feed doesn't match locally computed hash", "msgIdx", msgIdx, "err", err)
			}
		}
	}
}

func (s *TransactionStreamer) storeResult(
	msgIdx arbutil.MessageIndex,
	msgResult execution.MessageResult,
	batch ethdb.Batch,
) error {
	msgResultBytes, err := rlp.EncodeToBytes(msgResult)
	if err != nil {
		return err
	}
	key := dbKey(messageResultPrefix, uint64(msgIdx))
	return batch.Put(key, msgResultBytes)
}

// exposed for testing

// return value: true if should be called again immediately

func (s *TransactionStreamer) ExecuteNextMsg(ctx context.Context) bool {
	if ctx.Err() != nil {
		return false
	}
	if !s.reorgMutex.TryRLock() {
		return false
	}
	defer s.reorgMutex.RUnlock()

	prevHeadMsgIdx := s.prevHeadMsgIdx
	consensusHeadMsgIdx, err := s.GetHeadMessageIndex()
	if errors.Is(err, ErrNoMessages) {
		return false
	} else if err != nil {
		log.Error("ExecuteNextMsg failed to get consensus head msg index", "err", err)
		return false
	}
	s.prevHeadMsgIdx = &consensusHeadMsgIdx

	execHeadMsgIdx, err := s.exec.HeadMessageIndex().Await(ctx)
	if err != nil {
		log.Error("ExecuteNextMsg failed to get exec engine head message index", "err", err)
		return false
	}

	if execHeadMsgIdx >= consensusHeadMsgIdx {
		return false
	}
	msgIdxToExecute := execHeadMsgIdx + 1

	msgAndBlockInfo, err := s.getMessageWithMetadataAndBlockInfo(msgIdxToExecute)
	if err != nil {
		log.Error("ExecuteNextMsg failed to readMessage", "err", err, "msgIdxToExecute", msgIdxToExecute)
		return false
	}
	var msgForPrefetch *arbostypes.MessageWithMetadata
	if msgIdxToExecute+1 <= consensusHeadMsgIdx {
		msg, err := s.GetMessage(msgIdxToExecute + 1)
		if err != nil {
			log.Error("ExecuteNextMsg failed to readMessage", "err", err, "msgIdxToExecute+1", msgIdxToExecute+1)
			return false
		}
		msgForPrefetch = msg
	}
	msgResult, err := s.exec.DigestMessage(msgIdxToExecute, &msgAndBlockInfo.MessageWithMeta, msgForPrefetch).Await(ctx)
	if err != nil {
		logger := log.Warn
		if (prevHeadMsgIdx == nil) || (*prevHeadMsgIdx < consensusHeadMsgIdx) {
			logger = log.Debug
		}
		logger("ExecuteNextMsg failed to send message to execEngine", "err", err, "msgIdxToExecute", msgIdxToExecute)
		return false
	}

	s.checkResult(msgIdxToExecute, msgResult, msgAndBlockInfo)

	batch := s.db.NewBatch()
	err = s.storeResult(msgIdxToExecute, *msgResult, batch)
	if err != nil {
		log.Error("ExecuteNextMsg failed to store result", "err", err)
		return false
	}
	err = batch.Write()
	if err != nil {
		log.Error("ExecuteNextMsg failed to store result", "err", err)
		return false
	}

	msgWithBlockInfo := arbostypes.MessageWithMetadataAndBlockInfo{
		MessageWithMeta: msgAndBlockInfo.MessageWithMeta,
		BlockHash:       &msgResult.BlockHash,
		BlockMetadata:   msgAndBlockInfo.BlockMetadata,
	}
	s.broadcastMessages([]arbostypes.MessageWithMetadataAndBlockInfo{msgWithBlockInfo}, msgIdxToExecute)

	return msgIdxToExecute+1 <= consensusHeadMsgIdx
}

func (s *TransactionStreamer) executeMessages(ctx context.Context, ignored struct{}) time.Duration {
	if s.config().SyncTillBlock > 0 && s.prevHeadMsgIdx != nil && uint64(*s.prevHeadMsgIdx) >= s.config().SyncTillBlock {
		log.Info("stopping block creation in transaction streamer", "syncTillBlock", s.config().SyncTillBlock)
		return s.config().ExecuteMessageLoopDelay
	}
	if s.ExecuteNextMsg(ctx) {
		return 0
	}
	return s.config().ExecuteMessageLoopDelay
}

// backfillTrackersForMissingBlockMetadata adds missingBlockMetadataInputFeedPrefix to block numbers whose blockMetadata status
// isn't yet tracked. If a node is started with new value for trackBlockMetadataFrom that is lower than the current, then this
// function adds the missing trackers so that bulk BlockMetadataFetcher can fill in the gaps.
func (s *TransactionStreamer) backfillTrackersForMissingBlockMetadata(ctx context.Context) {
	if s.trackBlockMetadataFrom == 0 {
		return
	}
	msgCount, err := s.GetMessageCount()
	if err != nil {
		log.Error("Error getting message count from arbDB", "err", err)
		return
	}
	if s.trackBlockMetadataFrom >= msgCount {
		return // We dont need to back fill if trackBlockMetadataFrom is in the future
	}

	wasKeyFound := func(pos uint64) bool {
		searchWithPrefix := func(prefix []byte) bool {
			key := dbKey(prefix, pos)
			_, err := s.db.Get(key)
			if err == nil {
				return true
			}
			if !dbutil.IsErrNotFound(err) {
				log.Error("Error reading key in arbDB while back-filling trackers for missing blockMetadata", "key", key, "err", err)
			}
			return false
		}
		return searchWithPrefix(blockMetadataInputFeedPrefix) || searchWithPrefix(missingBlockMetadataInputFeedPrefix)
	}

	start := s.trackBlockMetadataFrom
	if wasKeyFound(uint64(start)) {
		return // back-filling not required
	}
	finish := msgCount - 1
	for start < finish {
		mid := (start + finish + 1) / 2
		if wasKeyFound(uint64(mid)) {
			finish = mid - 1
		} else {
			start = mid
		}
	}
	lastNonExistent := start

	// We back-fill in reverse to avoid fragmentation in case of any failures
	batch := s.db.NewBatch()
	for i := lastNonExistent; i >= s.trackBlockMetadataFrom; i-- {
		if err := batch.Put(dbKey(missingBlockMetadataInputFeedPrefix, uint64(i)), nil); err != nil {
			log.Error("Error marking blockMetadata as missing while back-filling", "pos", i, "err", err)
			return
		}
		// If we reached the ideal batch size, commit and reset
		if batch.ValueSize() >= ethdb.IdealBatchSize {
			if err := batch.Write(); err != nil {
				log.Error("Error writing batch with missing trackers to db while back-filling", "err", err)
				return
			}
			batch.Reset()
		}
	}
	if err := batch.Write(); err != nil {
		log.Error("Error writing batch with missing trackers to db while back-filling", "err", err)
	}
}

func (s *TransactionStreamer) Start(ctxIn context.Context) error {
	s.StopWaiter.Start(ctxIn, s)
	s.LaunchThread(s.backfillTrackersForMissingBlockMetadata)

	if s.lightClientReader != nil && s.espressoClient != nil {
		err := s.RegisterSigner()
		if err != nil {
			log.Error("failed to register espresso key manager", "err", err)
			return err
		}
		err = stopwaiter.CallIterativelyWith[struct{}](&s.StopWaiterSafe, s.pollSubmittedTransactionForFinality, s.newSovereignTxNotifier)
		if err != nil {
			return err
		}
		err = stopwaiter.CallIterativelyWith[struct{}](&s.StopWaiterSafe, s.submitTransactionsToEspresso, s.newSovereignTxNotifier)
		if err != nil {
			return err
		}
		err = stopwaiter.CallIterativelyWith[struct{}](&s.StopWaiterSafe, s.pollToResubmitEspressoTransactions, s.newSovereignTxNotifier)
		if err != nil {
			return err
		}
	} else {
		log.Warn("light client reader or espresso client not set, skipping espresso verification")
	}

	return stopwaiter.CallIterativelyWith[struct{}](&s.StopWaiterSafe, s.executeMessages, s.newMessageNotifier)
}

// Encodes an uint64 as bytes in a lexically sortable manner for database iteration.

func (s *TransactionStreamer) BlockMetadataAtCount(count arbutil.MessageIndex) (common.BlockMetadata, error) {
	if count == 0 {
		return nil, nil
	}
	pos := count - 1

	if s.trackBlockMetadataFrom == 0 || pos < s.trackBlockMetadataFrom {
		return nil, nil
	}

	key := dbKey(blockMetadataInputFeedPrefix, uint64(pos))
	blockMetadata, err := s.db.Get(key)
	if err != nil {
		if dbutil.IsErrNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return blockMetadata, nil
}

// Check if the latest submitted transaction has been finalized on L1 and verify it.
// Return a bool indicating whether a new transaction can be submitted to HotShot
func (s *TransactionStreamer) checkSubmittedTransactionForFinality(ctx context.Context) error {
	s.espressoSubmittedTxnsMutex.Lock()
	defer s.espressoSubmittedTxnsMutex.Unlock()

	submittedTxns, err := s.getEspressoSubmittedTxns()
	if err != nil {
		return fmt.Errorf("submitted transactions not found: %w", err)
	}
	if len(submittedTxns) == 0 {
		return nil // no submitted transaction, treated as successful
	}

	batch := s.db.NewBatch()
	newSubmittedTxns := []arbutil.SubmittedEspressoTx{}
	lastConfirmedPos := arbutil.MessageIndex(0)
	if lastConfirmedPosInDb, _ := s.getLastConfirmedPos(); lastConfirmedPosInDb != nil {
		lastConfirmedPos = *lastConfirmedPosInDb
	}
	blockHeights := []uint64{}
	posArray := []int{}
	for i, submittedTx := range submittedTxns {
		hash := submittedTx.Hash
		submittedTxHash, err := tagged_base64.Parse(hash)
		if err != nil || submittedTxHash == nil {
			return fmt.Errorf("invalid hotshot tx hash, failed to parse hash %s: %w", hash, err)
		}

		blockHeight, err := s.checkEspressoQueryNodesForTransaction(ctx, submittedTxHash)
		if err != nil {
			resubmittedTxn, err := s.resubmitTransactionIfPastDelay(ctx, submittedTx)
			if err != nil {
				log.Error("failed to resubmit transaction", "err", err)
			}
			if resubmittedTxn != nil {
				newSubmittedTxns = append(newSubmittedTxns, *resubmittedTxn)
			} else {
				newSubmittedTxns = append(newSubmittedTxns, submittedTx)
			}
			log.Info("encountered an error trying to check espresso for a submitted txn", "err", err)
			continue
		}
		log.Info("transaction checked", "hash", hash, "data", blockHeight)

		blockHeights = append(blockHeights, blockHeight)
		posArray = append(posArray, i)
	}

	for i, height := range blockHeights {
		submittedTx := submittedTxns[posArray[i]]

		resp, err := s.espressoClient.FetchTransactionsInBlock(ctx, height, s.chainConfig.ChainID.Uint64())
		if err != nil {
			log.Warn("Failed to fetch transactions in block referenced in fetch transaction by hash", "height", height, "error", err)
			continue
		}

		validated := arbutil.ValidateIfPayloadIsInBlock(submittedTx.Payload, resp.Transactions)
		if !validated {
			// This may seem redundant as we have a resubmission loop, but hitting this code path means that we were able to find the submitted tx hash across a quorom of
			// the query nodes, and got a result for what block it should be in. However, we were unable to validate that the payload was in the block.
			log.Warn("Transaction payload not found in block,The txn should be re-submitted", "height", height, "tx", submittedTx.Hash)
			resubmittedTxn, err := s.resubmitTransaction(ctx, submittedTx)
			if err != nil {
				log.Error("failed to resubmit transaction", "err", err)
				continue
			}
			if resubmittedTxn == nil {
				// This should never happen
				log.Error("failed to resubmit transaction", "err", err)
				continue
			}
			newSubmittedTxns = append(newSubmittedTxns, *resubmittedTxn)
			continue
		}
		max := submittedTx.Pos[0]
		for _, pos := range submittedTx.Pos {
			if pos > max {
				max = pos
			}
		}

		if max > lastConfirmedPos {
			lastConfirmedPos = max
		}

	}

	log.Info("last confirmed pos", "lastConfirmedPos", lastConfirmedPos)

	err = s.setEspressoLastConfirmedPos(batch, &lastConfirmedPos)
	if err != nil {
		return fmt.Errorf("failed to set last confirmed pos: %w", err)
	}

	// this will be remmoved in other PRs
	err = s.setEspressoSubmittedTxns(batch, newSubmittedTxns)
	if err != nil {
		return fmt.Errorf("failed to set espresso submitted txns: %w", err)
	}

	if err = batch.Write(); err != nil {
		return fmt.Errorf("failed to write to db: %w", err)
	}

	return nil
}

func (s *TransactionStreamer) checkEspressoQueryNodesForTransaction(ctx context.Context, hash *tagged_base64.TaggedBase64) (uint64, error) {
	payload, err := s.espressoClient.FetchExplorerTransactionByHash(ctx, hash)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch transaction from espresso: %w", err)
	}

	return payload.TransactionsDetails.ExplorerDetails.BlockHeight, nil
}

func (s *TransactionStreamer) resubmitTransaction(ctx context.Context, submittedTx arbutil.SubmittedEspressoTx) (*arbutil.SubmittedEspressoTx, error) {
	submittedAt := time.Now()
	hash, err := s.espressoClient.SubmitTransaction(ctx, espressoTypes.Transaction{
		Payload:   submittedTx.Payload,
		Namespace: s.chainConfig.ChainID.Uint64(),
	})
	if err != nil {
		return nil, err
	}
	submittedTx.Hash = hash.String()
	submittedTx.SubmittedAt = submittedAt
	return &submittedTx, nil
}

func (s *TransactionStreamer) resubmitTransactionIfPastDelay(ctx context.Context, submittedTx arbutil.SubmittedEspressoTx) (*arbutil.SubmittedEspressoTx, error) {
	timeSinceSubmission := time.Since(submittedTx.SubmittedAt)
	if timeSinceSubmission < s.resubmitEspressoTxDeadline {
		return nil, nil
	}
	return s.resubmitTransaction(ctx, submittedTx)
}

func (s *TransactionStreamer) getEspressoSubmittedTxns() ([]arbutil.SubmittedEspressoTx, error) {
	posBytes, err := s.db.Get(espressoSubmittedTxns)
	if err != nil {
		if dbutil.IsErrNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	var tx []arbutil.SubmittedEspressoTx
	err = rlp.DecodeBytes(posBytes, &tx)
	if err != nil {
		return nil, err
	}
	return tx, nil
}

func (s *TransactionStreamer) getLastConfirmedPos() (*arbutil.MessageIndex, error) {
	lastConfirmedBytes, err := s.db.Get(espressoLastConfirmedPos)
	if err != nil {
		if dbutil.IsErrNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	var lastConfirmed arbutil.MessageIndex
	err = rlp.DecodeBytes(lastConfirmedBytes, &lastConfirmed)
	if err != nil {
		return nil, err
	}
	return &lastConfirmed, nil
}

func (s *TransactionStreamer) getEspressoPendingTxnsPos() ([]arbutil.MessageIndex, error) {

	pendingTxnsBytes, err := s.db.Get(espressoPendingTxnsPositions)
	if err != nil {
		if dbutil.IsErrNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	var pendingTxnsPos []arbutil.MessageIndex
	err = rlp.DecodeBytes(pendingTxnsBytes, &pendingTxnsPos)
	if err != nil {
		return nil, err
	}
	return pendingTxnsPos, nil
}

func (s *TransactionStreamer) setEspressoSubmittedTxns(batch ethdb.KeyValueWriter, txns []arbutil.SubmittedEspressoTx) error {
	// if pos is nil, delete the key
	if txns == nil {
		err := batch.Delete(espressoSubmittedTxns)
		return err
	}

	bytes, err := rlp.EncodeToBytes(txns)
	if err != nil {
		return err
	}
	err = batch.Put(espressoSubmittedTxns, bytes)
	if err != nil {
		return err
	}

	return nil
}

func (s *TransactionStreamer) setEspressoLastConfirmedPos(batch ethdb.KeyValueWriter, pos *arbutil.MessageIndex) error {
	posBytes, err := rlp.EncodeToBytes(pos)
	if err != nil {
		return err
	}
	err = batch.Put(espressoLastConfirmedPos, posBytes)
	if err != nil {
		return err

	}
	return nil
}

func (s *TransactionStreamer) setEspressoPendingTxnsPos(batch ethdb.KeyValueWriter, pos []arbutil.MessageIndex) error {
	if pos == nil {
		err := batch.Delete(espressoPendingTxnsPositions)
		return err
	}

	posBytes, err := rlp.EncodeToBytes(pos)
	if err != nil {
		return err
	}
	err = batch.Put(espressoPendingTxnsPositions, posBytes)
	if err != nil {
		return err

	}
	return nil
}

// Append a position to the pending queue. Please ensure this position is valid beforehand.
func (s *TransactionStreamer) SubmitEspressoTransactionPos(pos []arbutil.MessageIndex) error {
	s.espressoPendingTxnPosMutex.Lock()
	defer s.espressoPendingTxnPosMutex.Unlock()

	batch := s.db.NewBatch()
	pendingTxnsPos, err := s.getEspressoPendingTxnsPos()
	if err != nil {
		return err
	}

	if pendingTxnsPos == nil {
		// if the key doesn't exist, create a new array with the pos
		pendingTxnsPos = pos
	} else {
		pendingTxnsPos = append(pendingTxnsPos, pos...)
	}
	err = s.setEspressoPendingTxnsPos(batch, pendingTxnsPos)
	if err != nil {
		log.Error("failed to set the pending txns", "err", err)
		return err
	}

	err = batch.Write()
	if err != nil {
		return err
	}

	return nil
}

func (s *TransactionStreamer) ResubmitEspressoTransactions(ctx context.Context, tx arbutil.SubmittedEspressoTx) (*tagged_base64.TaggedBase64, error) {
	txHash, err := s.espressoClient.SubmitTransaction(ctx, espressoTypes.Transaction{
		Payload:   tx.Payload,
		Namespace: s.chainConfig.ChainID.Uint64(),
	})
	if err != nil {
		return nil, err
	}

	return txHash, nil
}

func (s *TransactionStreamer) submitEspressoTransactions(ctx context.Context) error {
	s.espressoPendingTxnPosMutex.Lock()
	defer s.espressoPendingTxnPosMutex.Unlock()

	pendingTxnsPos, err := s.getEspressoPendingTxnsPos()
	if err != nil {
		return err
	}

	if len(pendingTxnsPos) == 0 {
		return nil
	}

	fetcher := func(pos arbutil.MessageIndex) ([]byte, error) {
		msg, err := s.GetMessage(pos)
		if err != nil {
			return nil, err
		}
		if pos > 1 {
			prevMsg, err := s.GetMessage(pos - 1)
			if err != nil {
				return nil, err
			}
			if prevMsg.DelayedMessagesRead+1 == msg.DelayedMessagesRead {
				// This message is a delayed message, and it should not be included
				// in the hotshot payload. The caff node is supposed to fetch the delayed message
				// from L1.
				// setting `msg.Message` to `nil` will cause a rlp decode/encode error
				// so we set `L2msg` to an empty byte slice instead
				msg.Message.L2msg = []byte{}
			}
		}
		b, err := rlp.EncodeToBytes(msg)
		if err != nil {
			return nil, err
		}
		return b, nil
	}
	payload, msgCnt := arbutil.BuildRawHotShotPayload(pendingTxnsPos, fetcher, s.espressoMaxTransactionSize)
	batch := s.db.NewBatch()
	submittedPos := pendingTxnsPos[:msgCnt]
	pendingTxnsPos = pendingTxnsPos[msgCnt:]

	err = s.setEspressoPendingTxnsPos(batch, pendingTxnsPos)

	if err != nil {
		return fmt.Errorf("failed to set the pending txn list in the db batch: %w", err)
	}
	if msgCnt == 0 {
		return fmt.Errorf("failed to build the hotshot transaction: a large message has exceeded the size limit or failed to get a message from storage")
	}

	payload, err = arbutil.SignHotShotPayload(payload, s.EspressoKeyManager.SignHotShotPayload)
	if err != nil {
		return fmt.Errorf("failed to sign the hotshot payload %w", err)
	}

	log.Info("submitting transaction to hotshot for finalization")

	submittedAt := time.Now()
	// Note: same key should not be used for two namespaces for this to work
	hash, err := s.espressoClient.SubmitTransaction(ctx, espressoTypes.Transaction{
		Payload:   payload,
		Namespace: s.chainConfig.ChainID.Uint64(),
	})

	if err != nil {
		return fmt.Errorf("failed to submit transaction to espresso: %w", err)
	}

	s.espressoSubmittedTxnsMutex.Lock()
	defer s.espressoSubmittedTxnsMutex.Unlock()

	submittedTxns, err := s.getEspressoSubmittedTxns()
	if err != nil {
		return fmt.Errorf("failed to get the submitted txns: %w", err)
	}
	tx := arbutil.SubmittedEspressoTx{
		Hash:        hash.String(),
		Pos:         submittedPos,
		Payload:     payload,
		SubmittedAt: submittedAt,
	}
	if submittedTxns == nil {
		submittedTxns = []arbutil.SubmittedEspressoTx{tx}
	} else {
		submittedTxns = append(submittedTxns, tx)
	}

	if err = s.setEspressoSubmittedTxns(batch, submittedTxns); err != nil {
		return fmt.Errorf("failed to set espresso submitted txns: %w", err)
	}

	err = batch.Write()
	if err != nil {
		return fmt.Errorf("failed to write to db: %w", err)
	}
	return nil
}

// Make sure useEscapeHatch is true

func (s *TransactionStreamer) checkEspressoLiveness() error {
	live, err := s.lightClientReader.IsHotShotLive(s.maxBlockLagBeforeEscapeHatch)
	if err != nil {
		return err
	}
	// If escape hatch is activated, the only thing is to check if hotshot is live again
	if s.EscapeHatchEnabled {
		if live {
			log.Info("HotShot is up, disabling the escape hatch")
			s.EscapeHatchEnabled = false
		}
		return nil
	}

	// If escape hatch is disabled, hotshot is live, everything is fine
	if live {
		return nil
	}

	// If escape hatch is on, and hotshot is down
	log.Warn("enabling the escape hatch, hotshot is down")
	s.EscapeHatchEnabled = true

	return nil
}

var espressoMerkleProofEphemeralErrorHandler = util.NewEphemeralErrorHandler(80*time.Minute, EspressoValidationErr.Error(), 15*time.Minute)

var espressoTransactionEphemeralErrorHandler = util.NewEphemeralErrorHandler(3*time.Minute, EspressoFetchTransactionErr.Error(), 15*time.Minute)

func getLogLevel(err error) func(string, ...interface{}) {
	logLevel := log.Error
	logLevel = espressoMerkleProofEphemeralErrorHandler.LogLevel(err, logLevel)
	logLevel = espressoTransactionEphemeralErrorHandler.LogLevel(err, logLevel)
	return logLevel
}

/**
* Checks if the submitted transaction has been finalized by Espresso  and verifies it.
 */

func (s *TransactionStreamer) pollSubmittedTransactionForFinality(ctx context.Context, ignored struct{}) time.Duration {
	retryRate := s.espressoTxnsPollingInterval * 2
	var err error
	if s.UseEscapeHatch {
		err = s.checkEspressoLiveness()
		if err != nil {
			if ctx.Err() != nil {
				return s.espressoTxnsPollingInterval
			}
			logLevel := getLogLevel(err)
			logLevel("error checking escape hatch, will retry", "err", err)
			return retryRate
		}
		espressoTransactionEphemeralErrorHandler.Reset()
	}
	err = s.checkSubmittedTransactionForFinality(ctx)
	if err != nil {
		if ctx.Err() != nil {
			return s.espressoTxnsPollingInterval
		}
		logLevel := getLogLevel(err)
		logLevel("error polling finality, will retry", "err", err)
		return retryRate
	}
	espressoMerkleProofEphemeralErrorHandler.Reset()
	return s.espressoTxnsPollingInterval
}

/**
 * Submits the transactions to espresso if the escape hatch is not enabled
 */

func (s *TransactionStreamer) submitTransactionsToEspresso(ctx context.Context, ignored struct{}) time.Duration {
	// When encountering an error during the initial attempt at submitting a transaction, double the amount of our polling interval and try again.
	retryRate := s.espressoTxnsSendingInterval * 2
	shouldSubmit := s.shouldSubmitEspressoTransaction(nil)
	// Only submit the transaction if escape hatch is not enabled
	if shouldSubmit {
		err := s.submitEspressoTransactions(ctx)

		if err != nil {
			log.Error("failed to submit espresso transactions", "err", err)
			return retryRate
		}
	}
	return s.espressoTxnsSendingInterval
}

func (s *TransactionStreamer) pollToResubmitEspressoTransactions(ctx context.Context, ignored struct{}) time.Duration {
	retryRate := s.espressoTxnsResubmissionInterval * 2
	submittedTxns, err := s.getEspressoSubmittedTxns()
	if err != nil {
		log.Warn("resubmitting espresso transactions failed: unable to get submitted transactions, will retry: %w", err)
		return retryRate
	}

	shouldResubmit := s.shouldResubmitEspressoTransactions(ctx, submittedTxns)
	if shouldResubmit {
		for _, tx := range submittedTxns {
			log.Info("Resubmitting tx to Espresso", "tx", tx.Hash)
			txHash, err := s.ResubmitEspressoTransactions(ctx, tx)
			if err != nil {
				log.Warn("failed to resubmit espresso transactions", "err", err)
				return retryRate
			}
			log.Info(fmt.Sprintf("trying to resubmit transaction succeeded: (hash: %s)", txHash.String()))
		}
		// Reset the last submit failure time because we successfully resubmitted the transactions
		s.lastSubmitFailureAt = nil
	}
	return s.espressoTxnsResubmissionInterval
}

func (s *TransactionStreamer) shouldSubmitEspressoTransaction(pos *uint64) bool {
	if s.espressoClient == nil && s.lightClientReader == nil {
		return false
	}
	if pos != nil {
		if *pos < s.InitialFinalizedSequencerMessageCount.Uint64() {
			log.Warn("not submitting transaction to espresso due to it being finalized", "pos", *pos, "sequencerMessageCount", s.InitialFinalizedSequencerMessageCount)
			return false
		}
	}

	return !s.EscapeHatchEnabled
}

func (s *TransactionStreamer) shouldResubmitEspressoTransactions(ctx context.Context, submittedTxns []arbutil.SubmittedEspressoTx) bool {
	if len(submittedTxns) == 0 {
		// If no submitted transactions, we dont need to resubmit
		return false
	}
	firstSubmitted := submittedTxns[0]
	hash := firstSubmitted.Hash

	submittedTxHash, err := tagged_base64.Parse(hash)
	if err != nil || submittedTxHash == nil {
		log.Error("invalid hotshot tx hash, failed to parse hash %s: %w", hash, err)
		return false
	}

	_, err = s.espressoClient.FetchExplorerTransactionByHash(ctx, submittedTxHash)
	if err == nil {
		// if we are able to fetch the transaction, we dont need to resubmit
		return false
	}

	if s.lastSubmitFailureAt == nil {
		now := time.Now()
		s.lastSubmitFailureAt = &now
		log.Warn("will wait for resubmission deadline before resubmitting transaction (hash: %s): %w, will retry again", submittedTxHash.String(), err)
		return false
	}
	duration := time.Since(*s.lastSubmitFailureAt)
	if duration < s.resubmitEspressoTxDeadline {
		log.Warn("resubmission deadline not reached (hash: %s): %w, will retry again", submittedTxHash.String(), err)
		return false
	}

	return true
}

func (s *TransactionStreamer) RegisterSigner() error {
	teeType := s.EspressoKeyManager.TeeType()
	switch teeType {
	case SGX:
		return s.EspressoKeyManager.Register(s.getAttestationQuote)
	case NITRO:
		return s.EspressoKeyManager.Register(s.getNitroAttestation)
	default:
		return fmt.Errorf("unsupported tee Type: %d", teeType)
	}
}

/**
 * This function generates the attestation quote for the user data.
 * The user data is hashed using keccak256 and then 32 bytes of padding is added to the hash.
 * The hash is then written to a file specified in the config. (For SGX: /dev/attestation/user_report_data)
 * The quote is then read from the file specified in the config. (For SGX: /dev/attestation/quote)
 */
func (t *TransactionStreamer) getAttestationQuote(userData []byte) ([]byte, error) {

	if (t.config().UserDataAttestationFile == "") || (t.config().QuoteFile == "") {
		return []byte{}, nil
	}
	// keccak256 hash of userData
	userDataHash := crypto.Keccak256(userData)

	// Add 32 bytes of padding to the user data hash
	// because keccak256 hash is 32 bytes and sgx requires 64 bytes of user data
	for i := 0; i < 32; i += 1 {
		userDataHash = append(userDataHash, 0)
	}

	// Write the message to "/dev/attestation/user_report_data" in SGX
	err := os.WriteFile(t.config().UserDataAttestationFile, userDataHash, 0600)
	if err != nil {
		return []byte{}, fmt.Errorf("failed to create user report data file: %w", err)
	}

	// Read the quote from "/dev/attestation/quote" in SGX
	attestationQuote, err := os.ReadFile(t.config().QuoteFile)
	if err != nil {
		return []byte{}, fmt.Errorf("failed to read quote file: %w", err)
	}

	return attestationQuote, nil
}

/**
 * This function gets the attestation document for AWS Nitro Enclaves
 * We retrieve the Attestation using our epheremal public key we created in EspressoKeyManager
 * After we retrieve, we verify the attestation, where we retrieve the result
 * Which will contain the complete attestation which we serialize for further processing
 */
func (t *TransactionStreamer) getNitroAttestation(pubKey []byte) ([]byte, error) {

	sess, err := nsm.OpenDefaultSession()
	if err != nil {
		return nil, fmt.Errorf("failed to open nsm session: %w", err)
	}
	defer sess.Close()

	res, err := sess.Send(&request.Attestation{
		PublicKey: pubKey,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to send attestation request: %w", err)
	}

	if res.Error != "" {
		return nil, fmt.Errorf("nsm returned error: %s", res.Error)
	}

	if res.Attestation == nil || res.Attestation.Document == nil {
		return nil, fmt.Errorf("no attestation document returned")
	}

	attestation, err := nitrite.Verify(res.Attestation.Document, nitrite.VerifyOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to verify attestation")
	}

	attestationBytes, err := json.Marshal(attestation)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal attestation")
	}
	return attestationBytes, nil
}
