// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

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
	"sync"
	"sync/atomic"
	"testing"
	"time"

	espressoClient "github.com/EspressoSystems/espresso-sequencer-go/client"
	lightclient "github.com/EspressoSystems/espresso-sequencer-go/light-client"
	tagged_base64 "github.com/EspressoSystems/espresso-sequencer-go/tagged-base64"
	espressoTypes "github.com/EspressoSystems/espresso-sequencer-go/types"
	"github.com/ccoveille/go-safecast"
	flag "github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/broadcaster"
	m "github.com/offchainlabs/nitro/broadcaster/message"
	"github.com/offchainlabs/nitro/espressocrypto"
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

	chainConfig      *params.ChainConfig
	exec             execution.ExecutionSequencer
	execLastMsgCount arbutil.MessageIndex
	validator        *staker.BlockValidator

	db             ethdb.Database
	fatalErrChan   chan<- error
	config         TransactionStreamerConfigFetcher
	snapSyncConfig *SnapSyncConfig

	insertionMutex                  sync.Mutex // cannot be acquired while reorgMutex is held
	reorgMutex                      sync.RWMutex
	espressoTxnsStateInsertionMutex sync.Mutex

	newMessageNotifier     chan struct{}
	newSovereignTxNotifier chan struct{}

	nextAllowedFeedReorgLog time.Time

	broadcasterQueuedMessages            []arbostypes.MessageWithMetadataAndBlockHash
	broadcasterQueuedMessagesPos         atomic.Uint64
	broadcasterQueuedMessagesActiveReorg bool

	coordinator     *SeqCoordinator
	broadcastServer *broadcaster.Broadcaster
	inboxReader     *InboxReader
	delayedBridge   *DelayedBridge

	// Espresso specific fields. These fields are set from batch poster
	espressoClient               *espressoClient.Client
	lightClientReader            lightclient.LightClientReaderInterface
	espressoTxnsPollingInterval  time.Duration
	maxBlockLagBeforeEscapeHatch uint64
	espressoMaxTransactionSize   int64
	resubmitEspressoTxDeadline   time.Duration
	lastSubmitFailureAt          *time.Time
	// Public these fields for testing
	EscapeHatchEnabled bool
	UseEscapeHatch     bool
}

type TransactionStreamerConfig struct {
	MaxBroadcasterQueueSize int           `koanf:"max-broadcaster-queue-size"`
	MaxReorgResequenceDepth int64         `koanf:"max-reorg-resequence-depth" reload:"hot"`
	ExecuteMessageLoopDelay time.Duration `koanf:"execute-message-loop-delay" reload:"hot"`
	UserDataAttestationFile string        `koanf:"user-data-attestation-file"`
	QuoteFile               string        `koanf:"quote-file"`
}

type TransactionStreamerConfigFetcher func() *TransactionStreamerConfig

var DefaultTransactionStreamerConfig = TransactionStreamerConfig{
	MaxBroadcasterQueueSize: 50_000,
	MaxReorgResequenceDepth: 1024,
	ExecuteMessageLoopDelay: time.Millisecond * 100,
	QuoteFile:               "",
	UserDataAttestationFile: "",
}

var TestTransactionStreamerConfig = TransactionStreamerConfig{
	MaxBroadcasterQueueSize: 10_000,
	MaxReorgResequenceDepth: 128 * 1024,
	ExecuteMessageLoopDelay: time.Millisecond,
}

func TransactionStreamerConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Int(prefix+".max-broadcaster-queue-size", DefaultTransactionStreamerConfig.MaxBroadcasterQueueSize, "maximum cache of pending broadcaster messages")
	f.Int64(prefix+".max-reorg-resequence-depth", DefaultTransactionStreamerConfig.MaxReorgResequenceDepth, "maximum number of messages to attempt to resequence on reorg (0 = never resequence, -1 = always resequence)")
	f.Duration(prefix+".execute-message-loop-delay", DefaultTransactionStreamerConfig.ExecuteMessageLoopDelay, "delay when polling calls to execute messages")
	f.String(prefix+".user-data-attestation-file", DefaultTransactionStreamerConfig.UserDataAttestationFile, "specifies the file containing the user data attestation")
	f.String(prefix+".quote-file", DefaultTransactionStreamerConfig.QuoteFile, "specifies the file containing the quote")
}

func NewTransactionStreamer(
	db ethdb.Database,
	chainConfig *params.ChainConfig,
	exec execution.ExecutionSequencer,
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

// Encodes an uint64 as bytes in a lexically sortable manner for database iteration.
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

func (s *TransactionStreamer) ReorgTo(count arbutil.MessageIndex) error {
	return s.ReorgToAndEndBatch(s.db.NewBatch(), count)
}

func (s *TransactionStreamer) ReorgToAndEndBatch(batch ethdb.Batch, count arbutil.MessageIndex) error {
	s.insertionMutex.Lock()
	defer s.insertionMutex.Unlock()
	err := s.reorg(batch, count, nil)
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
func (s *TransactionStreamer) reorg(batch ethdb.Batch, count arbutil.MessageIndex, newMessages []arbostypes.MessageWithMetadataAndBlockHash) error {
	if count == 0 {
		return errors.New("cannot reorg out init message")
	}
	lastDelayedSeqNum, err := s.getPrevPrevDelayedRead(count)
	if err != nil {
		return err
	}
	var oldMessages []*arbostypes.MessageWithMetadata

	targetMsgCount, err := s.GetMessageCount()
	if err != nil {
		return err
	}
	config := s.config()
	// #nosec G115
	maxResequenceMsgCount := count + arbutil.MessageIndex(config.MaxReorgResequenceDepth)
	if config.MaxReorgResequenceDepth >= 0 && maxResequenceMsgCount < targetMsgCount {
		log.Error(
			"unable to re-sequence all old messages because there are too many",
			"reorgingToCount", count,
			"removingMessages", targetMsgCount-count,
			"maxReorgResequenceDepth", config.MaxReorgResequenceDepth,
		)
		targetMsgCount = maxResequenceMsgCount
	}
	for i := count; i < targetMsgCount; i++ {
		oldMessage, err := s.GetMessage(i)
		if err != nil {
			log.Error("unable to lookup old message for re-sequencing", "position", i, "err", err)
			break
		}

		if oldMessage.Message == nil || oldMessage.Message.Header == nil {
			continue
		}

		header := oldMessage.Message.Header

		if header.RequestId != nil {
			// This is a delayed message
			delayedSeqNum := header.RequestId.Big().Uint64()
			if delayedSeqNum+1 != oldMessage.DelayedMessagesRead {
				log.Error("delayed message header RequestId doesn't match database DelayedMessagesRead", "header", oldMessage.Message.Header, "delayedMessagesRead", oldMessage.DelayedMessagesRead)
				continue
			}
			if delayedSeqNum != lastDelayedSeqNum {
				// This is the wrong position for the delayed message
				continue
			}
			if s.inboxReader != nil {
				// this is a delayed message. Should be resequenced if all 3 agree:
				// oldMessage, accumulator stored in tracker, and the message re-read from l1
				expectedAcc, err := s.inboxReader.tracker.GetDelayedAcc(delayedSeqNum)
				if err != nil {
					if !dbutil.IsErrNotFound(err) {
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
					if delayedFound.Message.Header.RequestId.Big().Uint64() != delayedSeqNum {
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
			lastDelayedSeqNum++
		}

		oldMessages = append(oldMessages, oldMessage)
	}

	s.reorgMutex.Lock()
	defer s.reorgMutex.Unlock()

	messagesResults, err := s.exec.Reorg(count, newMessages, oldMessages)
	if err != nil {
		return err
	}

	messagesWithComputedBlockHash := make([]arbostypes.MessageWithMetadataAndBlockHash, 0, len(messagesResults))
	for i := 0; i < len(messagesResults); i++ {
		messagesWithComputedBlockHash = append(messagesWithComputedBlockHash, arbostypes.MessageWithMetadataAndBlockHash{
			MessageWithMeta: newMessages[i].MessageWithMeta,
			BlockHash:       &messagesResults[i].BlockHash,
		})
	}
	s.broadcastMessages(messagesWithComputedBlockHash, count)

	if s.validator != nil {
		err = s.validator.Reorg(s.GetContext(), count)
		if err != nil {
			return err
		}
	}

	err = deleteStartingAt(s.db, batch, messageResultPrefix, uint64ToKey(uint64(count)))
	if err != nil {
		return err
	}
	err = deleteStartingAt(s.db, batch, blockHashInputFeedPrefix, uint64ToKey(uint64(count)))
	if err != nil {
		return err
	}
	err = deleteStartingAt(s.db, batch, messagePrefix, uint64ToKey(uint64(count)))
	if err != nil {
		return err
	}

	for i := 0; i < len(messagesResults); i++ {
		// #nosec G115
		pos := count + arbutil.MessageIndex(i)
		err = s.storeResult(pos, *messagesResults[i], batch)
		if err != nil {
			return err
		}
	}

	return setMessageCount(batch, count)
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
func (s *TransactionStreamer) GetMessage(seqNum arbutil.MessageIndex) (*arbostypes.MessageWithMetadata, error) {
	key := dbKey(messagePrefix, uint64(seqNum))
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

func (s *TransactionStreamer) getMessageWithMetadataAndBlockHash(seqNum arbutil.MessageIndex) (*arbostypes.MessageWithMetadataAndBlockHash, error) {
	msg, err := s.GetMessage(seqNum)
	if err != nil {
		return nil, err
	}

	// Get block hash.
	// To keep it backwards compatible, since it is possible that a message related
	// to a sequence number exists in the database, but the block hash doesn't.
	key := dbKey(blockHashInputFeedPrefix, uint64(seqNum))
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

	msgWithBlockHash := arbostypes.MessageWithMetadataAndBlockHash{
		MessageWithMeta: *msg,
		BlockHash:       blockHash,
	}
	return &msgWithBlockHash, nil
}

// Note: if changed to acquire the mutex, some internal users may need to be updated to a non-locking version.
func (s *TransactionStreamer) GetMessageCount() (arbutil.MessageIndex, error) {
	posBytes, err := s.db.Get(messageCountKey)
	if err != nil {
		return 0, err
	}
	var pos uint64
	err = rlp.DecodeBytes(posBytes, &pos)
	if err != nil {
		return 0, err
	}
	return arbutil.MessageIndex(pos), nil
}

func (s *TransactionStreamer) GetProcessedMessageCount() (arbutil.MessageIndex, error) {
	msgCount, err := s.GetMessageCount()
	if err != nil {
		return 0, err
	}
	digestedHead, err := s.exec.HeadMessageNumber()
	if err != nil {
		return 0, err
	}
	if msgCount > digestedHead+1 {
		return digestedHead + 1, nil
	}
	return msgCount, nil
}

func (s *TransactionStreamer) AddMessages(pos arbutil.MessageIndex, messagesAreConfirmed bool, messages []arbostypes.MessageWithMetadata) error {
	return s.AddMessagesAndEndBatch(pos, messagesAreConfirmed, messages, nil)
}

func (s *TransactionStreamer) FeedPendingMessageCount() arbutil.MessageIndex {
	pos := s.broadcasterQueuedMessagesPos.Load()
	if pos == 0 {
		return 0
	}

	s.insertionMutex.Lock()
	defer s.insertionMutex.Unlock()
	pos = s.broadcasterQueuedMessagesPos.Load()
	if pos == 0 {
		return 0
	}
	return arbutil.MessageIndex(pos + uint64(len(s.broadcasterQueuedMessages)))
}

func (s *TransactionStreamer) AddBroadcastMessages(feedMessages []*m.BroadcastFeedMessage) error {
	if len(feedMessages) == 0 {
		return nil
	}
	broadcastStartPos := feedMessages[0].SequenceNumber
	var messages []arbostypes.MessageWithMetadataAndBlockHash
	broadcastAfterPos := broadcastStartPos
	for _, feedMessage := range feedMessages {
		if broadcastAfterPos != feedMessage.SequenceNumber {
			return fmt.Errorf("invalid sequence number %v, expected %v", feedMessage.SequenceNumber, broadcastAfterPos)
		}
		if feedMessage.Message.Message == nil || feedMessage.Message.Message.Header == nil {
			return fmt.Errorf("invalid feed message at sequence number %v", feedMessage.SequenceNumber)
		}
		msgWithBlockHash := arbostypes.MessageWithMetadataAndBlockHash{
			MessageWithMeta: feedMessage.Message,
			BlockHash:       feedMessage.BlockHash,
		}
		messages = append(messages, msgWithBlockHash)
		broadcastAfterPos++
	}

	s.insertionMutex.Lock()
	defer s.insertionMutex.Unlock()

	var feedReorg bool
	var err error
	// Skip any messages already in the database
	// prevDelayedRead set to 0 because it's only used to compute the output prevDelayedRead which is not used here
	// Messages from feed are not confirmed, so confirmedMessageCount is 0 and confirmedReorg can be ignored
	dups, feedReorg, oldMsg, err := s.countDuplicateMessages(broadcastStartPos, messages, nil)
	if err != nil {
		return err
	}
	messages = messages[dups:]
	broadcastStartPos += arbutil.MessageIndex(dups)
	if oldMsg != nil {
		s.logReorg(broadcastStartPos, oldMsg, &messages[0].MessageWithMeta, false)
	}
	if len(messages) == 0 {
		// No new messages received
		return nil
	}

	if len(s.broadcasterQueuedMessages) == 0 || (feedReorg && !s.broadcasterQueuedMessagesActiveReorg) {
		// Empty cache or feed different from database, save current feed messages until confirmed L1 messages catch up.
		s.broadcasterQueuedMessages = messages
		s.broadcasterQueuedMessagesPos.Store(uint64(broadcastStartPos))
		s.broadcasterQueuedMessagesActiveReorg = feedReorg
	} else {
		broadcasterQueuedMessagesPos := arbutil.MessageIndex(s.broadcasterQueuedMessagesPos.Load())
		if broadcasterQueuedMessagesPos >= broadcastStartPos {
			// Feed messages older than cache
			s.broadcasterQueuedMessages = messages
			s.broadcasterQueuedMessagesPos.Store(uint64(broadcastStartPos))
			s.broadcasterQueuedMessagesActiveReorg = feedReorg
		} else if broadcasterQueuedMessagesPos+arbutil.MessageIndex(len(s.broadcasterQueuedMessages)) == broadcastStartPos {
			// Feed messages can be added directly to end of cache
			maxQueueSize := s.config().MaxBroadcasterQueueSize
			if maxQueueSize == 0 || len(s.broadcasterQueuedMessages) <= maxQueueSize {
				s.broadcasterQueuedMessages = append(s.broadcasterQueuedMessages, messages...)
			}
			broadcastStartPos = broadcasterQueuedMessagesPos
			// Do not change existing reorg state
		} else {
			if len(s.broadcasterQueuedMessages) > 0 {
				log.Warn(
					"broadcaster queue jumped positions",
					"queuedMessages", len(s.broadcasterQueuedMessages),
					"expectedNextPos", broadcasterQueuedMessagesPos+arbutil.MessageIndex(len(s.broadcasterQueuedMessages)),
					"gotPos", broadcastStartPos,
				)
			}
			s.broadcasterQueuedMessages = messages
			s.broadcasterQueuedMessagesPos.Store(uint64(broadcastStartPos))
			s.broadcasterQueuedMessagesActiveReorg = feedReorg
		}
	}

	if s.broadcasterQueuedMessagesActiveReorg || len(s.broadcasterQueuedMessages) == 0 {
		// Broadcaster never triggered reorg or no messages to add
		return nil
	}

	if broadcastStartPos > 0 {
		_, err := s.GetMessage(broadcastStartPos - 1)
		if err != nil {
			if !dbutil.IsErrNotFound(err) {
				return err
			}
			// Message before current message doesn't exist in database, so don't add current messages yet
			return nil
		}
	}

	err = s.addMessagesAndEndBatchImpl(broadcastStartPos, false, nil, nil)
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
	}})
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

func (s *TransactionStreamer) AddMessagesAndEndBatch(pos arbutil.MessageIndex, messagesAreConfirmed bool, messages []arbostypes.MessageWithMetadata, batch ethdb.Batch) error {
	messagesWithBlockHash := make([]arbostypes.MessageWithMetadataAndBlockHash, 0, len(messages))
	for _, message := range messages {
		messagesWithBlockHash = append(messagesWithBlockHash, arbostypes.MessageWithMetadataAndBlockHash{
			MessageWithMeta: message,
		})
	}

	if messagesAreConfirmed {
		// Trim confirmed messages from l1pricedataCache
		s.exec.MarkFeedStart(pos + arbutil.MessageIndex(len(messages)))
		s.reorgMutex.RLock()
		dups, _, _, err := s.countDuplicateMessages(pos, messagesWithBlockHash, &batch)
		s.reorgMutex.RUnlock()
		if err != nil {
			return err
		}
		if dups == uint64(len(messages)) {
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

	return s.addMessagesAndEndBatchImpl(pos, messagesAreConfirmed, messagesWithBlockHash, batch)
}

func (s *TransactionStreamer) getPrevPrevDelayedRead(pos arbutil.MessageIndex) (uint64, error) {
	if s.snapSyncConfig.Enabled && uint64(pos) == s.snapSyncConfig.PrevBatchMessageCount {
		return s.snapSyncConfig.PrevDelayedRead, nil
	}
	var prevDelayedRead uint64
	if pos > 0 {
		prevMsg, err := s.GetMessage(pos - 1)
		if err != nil {
			return 0, fmt.Errorf("failed to get previous message for pos %d: %w", pos, err)
		}
		prevDelayedRead = prevMsg.DelayedMessagesRead
	}

	return prevDelayedRead, nil
}

func (s *TransactionStreamer) countDuplicateMessages(
	pos arbutil.MessageIndex,
	messages []arbostypes.MessageWithMetadataAndBlockHash,
	batch *ethdb.Batch,
) (uint64, bool, *arbostypes.MessageWithMetadata, error) {
	var curMsg uint64
	for {
		if uint64(len(messages)) == curMsg {
			break
		}
		key := dbKey(messagePrefix, uint64(pos))
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
					"pos", pos,
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
							if err := s.writeMessage(pos, nextMessage, *batch); err != nil {
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
		pos++
	}

	return curMsg, false, nil, nil
}

func (s *TransactionStreamer) logReorg(pos arbutil.MessageIndex, dbMsg *arbostypes.MessageWithMetadata, newMsg *arbostypes.MessageWithMetadata, confirmed bool) {
	sendLog := confirmed
	if time.Now().After(s.nextAllowedFeedReorgLog) {
		sendLog = true
	}
	if sendLog {
		s.nextAllowedFeedReorgLog = time.Now().Add(time.Minute)
		log.Warn("TransactionStreamer: Reorg detected!",
			"confirmed", confirmed,
			"pos", pos,
			"got-delayed", newMsg.DelayedMessagesRead,
			"got-header", newMsg.Message.Header,
			"db-delayed", dbMsg.DelayedMessagesRead,
			"db-header", dbMsg.Message.Header,
		)
	}

}

func (s *TransactionStreamer) addMessagesAndEndBatchImpl(messageStartPos arbutil.MessageIndex, messagesAreConfirmed bool, messages []arbostypes.MessageWithMetadataAndBlockHash, batch ethdb.Batch) error {
	var confirmedReorg bool
	var oldMsg *arbostypes.MessageWithMetadata
	var lastDelayedRead uint64
	var hasNewConfirmedMessages bool
	var cacheClearLen int

	messagesAfterPos := messageStartPos + arbutil.MessageIndex(len(messages))
	broadcastStartPos := arbutil.MessageIndex(s.broadcasterQueuedMessagesPos.Load())

	if messagesAreConfirmed {
		var duplicates uint64
		var err error
		duplicates, confirmedReorg, oldMsg, err = s.countDuplicateMessages(messageStartPos, messages, &batch)
		if err != nil {
			return err
		}
		if duplicates > 0 {
			lastDelayedRead = messages[duplicates-1].MessageWithMeta.DelayedMessagesRead
			messages = messages[duplicates:]
			messageStartPos += arbutil.MessageIndex(duplicates)
		}
		if len(messages) > 0 {
			hasNewConfirmedMessages = true
		}
	}

	clearQueueOnSuccess := false
	if (s.broadcasterQueuedMessagesActiveReorg && messageStartPos <= broadcastStartPos) ||
		(!s.broadcasterQueuedMessagesActiveReorg && broadcastStartPos <= messagesAfterPos) {
		// Active broadcast reorg and L1 messages at or before start of broadcast messages
		// Or no active broadcast reorg and broadcast messages start before or immediately after last L1 message
		if messagesAfterPos >= broadcastStartPos {
			// #nosec G115
			broadcastSliceIndex := int(messagesAfterPos - broadcastStartPos)
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
		var duplicates uint64
		var err error
		duplicates, feedReorg, oldMsg, err = s.countDuplicateMessages(messageStartPos, messages, nil)
		if err != nil {
			return err
		}
		if duplicates > 0 {
			lastDelayedRead = messages[duplicates-1].MessageWithMeta.DelayedMessagesRead
			messages = messages[duplicates:]
			messageStartPos += arbutil.MessageIndex(duplicates)
		}
	}
	if oldMsg != nil {
		s.logReorg(messageStartPos, oldMsg, &messages[0].MessageWithMeta, confirmedReorg)
	}

	if feedReorg {
		// Never allow feed to reorg confirmed messages
		// Note that any remaining messages must be feed messages, so we're done here
		return endBatch(batch)
	}

	if lastDelayedRead == 0 {
		var err error
		lastDelayedRead, err = s.getPrevPrevDelayedRead(messageStartPos)
		if err != nil {
			return err
		}
	}

	// Validate delayed message counts of remaining messages
	for i, msg := range messages {
		// #nosec G115
		msgPos := messageStartPos + arbutil.MessageIndex(i)
		diff := msg.MessageWithMeta.DelayedMessagesRead - lastDelayedRead
		if diff != 0 && diff != 1 {
			return fmt.Errorf("attempted to insert jump from %v delayed messages read to %v delayed messages read at message index %v", lastDelayedRead, msg.MessageWithMeta.DelayedMessagesRead, msgPos)
		}
		lastDelayedRead = msg.MessageWithMeta.DelayedMessagesRead
		if msg.MessageWithMeta.Message == nil {
			return fmt.Errorf("attempted to insert nil message at position %v", msgPos)
		}
	}

	if confirmedReorg {
		reorgBatch := s.db.NewBatch()
		err := s.reorg(reorgBatch, messageStartPos, messages)
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

	err := s.writeMessages(messageStartPos, messages, batch)
	if err != nil {
		return err
	}

	if clearQueueOnSuccess {
		// Check if new messages were added at the end of cache, if they were, then don't remove those particular messages
		if len(s.broadcasterQueuedMessages) > cacheClearLen {
			s.broadcasterQueuedMessages = s.broadcasterQueuedMessages[cacheClearLen:]
			// #nosec G115
			s.broadcasterQueuedMessagesPos.Store(uint64(broadcastStartPos) + uint64(cacheClearLen))
		} else {
			s.broadcasterQueuedMessages = s.broadcasterQueuedMessages[:0]
			s.broadcasterQueuedMessagesPos.Store(0)
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
	pos arbutil.MessageIndex,
	msgWithMeta arbostypes.MessageWithMetadata,
	msgResult execution.MessageResult,
) error {

	if err := s.ExpectChosenSequencer(); err != nil {
		return err
	}
	if !s.insertionMutex.TryLock() {
		return execution.ErrSequencerInsertLockTaken
	}
	defer s.insertionMutex.Unlock()

	msgCount, err := s.GetMessageCount()
	if err != nil {
		return err
	}

	if msgCount != pos {
		return fmt.Errorf("wrong pos got %d expected %d", pos, msgCount)
	}

	if s.coordinator != nil {
		if err := s.coordinator.SequencingMessage(pos, &msgWithMeta); err != nil {
			return err
		}
	}

	msgWithBlockHash := arbostypes.MessageWithMetadataAndBlockHash{
		MessageWithMeta: msgWithMeta,
		BlockHash:       &msgResult.BlockHash,
	}
	if err := s.writeMessages(pos, []arbostypes.MessageWithMetadataAndBlockHash{msgWithBlockHash}, s.db.NewBatch()); err != nil {
		return err
	}

	s.broadcastMessages([]arbostypes.MessageWithMetadataAndBlockHash{msgWithBlockHash}, pos)

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
	if s.broadcastServer == nil {
		return nil
	}
	return s.inboxReader.tracker.PopulateFeedBacklog(s.broadcastServer)
}

func (s *TransactionStreamer) writeMessage(pos arbutil.MessageIndex, msg arbostypes.MessageWithMetadataAndBlockHash, batch ethdb.Batch) error {
	// write message with metadata
	key := dbKey(messagePrefix, uint64(pos))
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
	key = dbKey(blockHashInputFeedPrefix, uint64(pos))
	msgBytes, err = rlp.EncodeToBytes(blockHashDBVal)
	if err != nil {
		return err
	}
	return batch.Put(key, msgBytes)
}

func (s *TransactionStreamer) broadcastMessages(
	msgs []arbostypes.MessageWithMetadataAndBlockHash,
	pos arbutil.MessageIndex,
) {
	if s.broadcastServer == nil {
		return
	}
	if err := s.broadcastServer.BroadcastMessages(msgs, pos); err != nil {
		log.Error("failed broadcasting messages", "pos", pos, "err", err)
	}
}

// The mutex must be held, and pos must be the latest message count.
// `batch` may be nil, which initializes a new batch. The batch is closed out in this function.
func (s *TransactionStreamer) writeMessages(pos arbutil.MessageIndex, messages []arbostypes.MessageWithMetadataAndBlockHash, batch ethdb.Batch) error {
	if batch == nil {
		batch = s.db.NewBatch()
	}
	for i, msg := range messages {
		// #nosec G115
		err := s.writeMessage(pos+arbutil.MessageIndex(i), msg, batch)
		if err != nil {
			return err
		}
	}

	err := setMessageCount(batch, pos+arbutil.MessageIndex(len(messages)))
	if err != nil {
		return err
	}

	//  If light client reader and espresso client are set, then we need to store the pos in the database
	//  to be used later to submit the message to hotshot for finalization.
	if s.lightClientReader != nil && s.espressoClient != nil {
		//  Only submit the transaction if escape hatch is not enabled
		if s.shouldSubmitEspressoTransaction() {
			for i := range messages {
				idx, err := safecast.ToUint64(i)
				if err != nil {
					return err
				}
				log.Info("Enqueuing pending transaction to Espresso", "pos", pos+arbutil.MessageIndex(idx))
				err = s.enqueuePendingTransaction(pos + arbutil.MessageIndex(idx))
				if err != nil {
					log.Error("Failed to enqueue pending transaction to Espresso", "pos", pos+arbutil.MessageIndex(idx), "err", err)
					return err
				}
				log.Info("Enqueued pending transaction to Espresso was successful", "pos", pos+arbutil.MessageIndex(idx))
			}
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

func (s *TransactionStreamer) enqueuePendingTransaction(pos arbutil.MessageIndex) error {
	// Store the pos in the database to be used later to submit the message
	// to hotshot for finalization.
	err := s.SubmitEspressoTransactionPos(pos)
	if err != nil {
		log.Error("failed to submit espresso transaction pos", "pos", pos, "err", err)
		return err
	}

	return nil
}

func (s *TransactionStreamer) ResultAtCount(count arbutil.MessageIndex) (*execution.MessageResult, error) {
	if count == 0 {
		return &execution.MessageResult{}, nil
	}
	pos := count - 1

	key := dbKey(messageResultPrefix, uint64(pos))
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
	log.Info(FailedToGetMsgResultFromDB, "count", count)

	msgResult, err := s.exec.ResultAtPos(pos)
	if err != nil {
		return nil, err
	}
	// Stores result in Consensus DB in a best-effort manner
	batch := s.db.NewBatch()
	err = s.storeResult(pos, *msgResult, batch)
	if err != nil {
		log.Warn("Failed to store result at ResultAtCount", "err", err)
		return msgResult, nil
	}
	err = batch.Write()
	if err != nil {
		log.Warn("Failed to store result at ResultAtCount", "err", err)
		return msgResult, nil
	}

	return msgResult, nil
}

func (s *TransactionStreamer) checkResult(msgResult *execution.MessageResult, expectedBlockHash *common.Hash) {
	if expectedBlockHash == nil {
		return
	}
	if msgResult.BlockHash != *expectedBlockHash {
		log.Error(
			BlockHashMismatchLogMsg,
			"expected", expectedBlockHash,
			"actual", msgResult.BlockHash,
		)
		return
	}
}

func (s *TransactionStreamer) storeResult(
	pos arbutil.MessageIndex,
	msgResult execution.MessageResult,
	batch ethdb.Batch,
) error {
	msgResultBytes, err := rlp.EncodeToBytes(msgResult)
	if err != nil {
		return err
	}
	key := dbKey(messageResultPrefix, uint64(pos))
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
	prevMessageCount := s.execLastMsgCount
	msgCount, err := s.GetMessageCount()
	if err != nil {
		log.Error("feedOneMsg failed to get message count", "err", err)
		return false
	}
	s.execLastMsgCount = msgCount
	pos, err := s.exec.HeadMessageNumber()
	if err != nil {
		log.Error("feedOneMsg failed to get exec engine message count", "err", err)
		return false
	}
	pos++
	if pos >= msgCount {
		return false
	}
	msgAndBlockHash, err := s.getMessageWithMetadataAndBlockHash(pos)
	if err != nil {
		log.Error("feedOneMsg failed to readMessage", "err", err, "pos", pos)
		return false
	}
	var msgForPrefetch *arbostypes.MessageWithMetadata
	if pos+1 < msgCount {
		msg, err := s.GetMessage(pos + 1)
		if err != nil {
			log.Error("feedOneMsg failed to readMessage", "err", err, "pos", pos+1)
			return false
		}
		msgForPrefetch = msg
	}
	msgResult, err := s.exec.DigestMessage(pos, &msgAndBlockHash.MessageWithMeta, msgForPrefetch)
	if err != nil {
		logger := log.Warn
		if prevMessageCount < msgCount {
			logger = log.Debug
		}
		logger("feedOneMsg failed to send message to execEngine", "err", err, "pos", pos)
		return false
	}

	s.checkResult(msgResult, msgAndBlockHash.BlockHash)

	batch := s.db.NewBatch()
	err = s.storeResult(pos, *msgResult, batch)
	if err != nil {
		log.Error("feedOneMsg failed to store result", "err", err)
		return false
	}
	err = batch.Write()
	if err != nil {
		log.Error("feedOneMsg failed to store result", "err", err)
		return false
	}

	msgWithBlockHash := arbostypes.MessageWithMetadataAndBlockHash{
		MessageWithMeta: msgAndBlockHash.MessageWithMeta,
		BlockHash:       &msgResult.BlockHash,
	}
	s.broadcastMessages([]arbostypes.MessageWithMetadataAndBlockHash{msgWithBlockHash}, pos)
	return pos+1 < msgCount
}

func (s *TransactionStreamer) executeMessages(ctx context.Context, ignored struct{}) time.Duration {
	if s.ExecuteNextMsg(ctx) {
		return 0
	}
	return s.config().ExecuteMessageLoopDelay
}

// Check if the latest submitted transaction has been finalized on L1 and verify it.
// Return a bool indicating whether a new transaction can be submitted to HotShot
func (s *TransactionStreamer) checkSubmittedTransactionForFinality(ctx context.Context) error {
	submittedTxns, err := s.getEspressoSubmittedTxns()
	if err != nil {
		return fmt.Errorf("submitted pos not found: %w", err)
	}
	if len(submittedTxns) == 0 {
		return nil // no submitted transaction, treated as successful
	}

	firstSubmitted := submittedTxns[0]
	hash := firstSubmitted.Hash

	submittedTxHash, err := tagged_base64.Parse(hash)
	if err != nil || submittedTxHash == nil {
		return fmt.Errorf("invalid hotshot tx hash, failed to parse hash %s: %w", hash, err)
	}

	data, err := s.espressoClient.FetchTransactionByHash(ctx, submittedTxHash)
	if err != nil {
		return fmt.Errorf("unable to fetch transaction by hash: %w", err)
	}
	height := data.BlockHeight

	jsonHeader, err := s.espressoClient.FetchRawHeaderByHeight(ctx, height)
	if err != nil {
		return fmt.Errorf("could not get the header (height: %d): %w", height, err)
	}

	var header espressoTypes.HeaderImpl
	err = json.Unmarshal(jsonHeader, &header)
	if err != nil {
		return fmt.Errorf("could not unmarshal header from bytes (height: %d): %w", height, err)
	}

	log.Info("Fetching Merkle Root at hotshot", "height", height)
	// Verify the merkle proof
	snapshot, err := s.lightClientReader.FetchMerkleRoot(height, nil)
	if err != nil {
		return fmt.Errorf("%w (height: %d): %w", EspressoValidationErr, height, err)
	}

	if snapshot.Height <= height {
		return fmt.Errorf("snapshot height %v is less than or equal to the requested height %v", snapshot.Height, height)
	}

	nextHeader, err := s.espressoClient.FetchHeaderByHeight(ctx, snapshot.Height)
	if err != nil {
		return fmt.Errorf("error fetching the snapshot header (height: %d): %w", snapshot.Height, err)
	}

	proof, err := s.espressoClient.FetchBlockMerkleProof(ctx, snapshot.Height, height)
	if err != nil {
		return fmt.Errorf("error fetching the block merkle proof (height: %d, root height: %d): %w", height, snapshot.Height, err)
	}

	blockMerkleTreeRoot := nextHeader.Header.GetBlockMerkleTreeRoot()

	ok := espressocrypto.VerifyMerkleProof(proof.Proof, jsonHeader, *blockMerkleTreeRoot, snapshot.Root)
	if !ok {
		return fmt.Errorf("error validating merkle proof (height: %d, snapshot height: %d)", height, snapshot.Height)
	}

	// Verify the namespace proof
	resp, err := s.espressoClient.FetchTransactionsInBlock(ctx, height, s.chainConfig.ChainID.Uint64())
	if err != nil {
		return fmt.Errorf("failed to fetch the transactions in block (height: %d): %w", height, err)
	}

	namespaceOk := espressocrypto.VerifyNamespace(
		s.chainConfig.ChainID.Uint64(),
		resp.Proof,
		*header.Header.GetPayloadCommitment(),
		*header.Header.GetNsTable(),
		resp.Transactions,
		resp.VidCommon,
	)

	if !namespaceOk {
		return fmt.Errorf("error validating namespace proof (height: %d)", height)
	}

	submittedPayload := firstSubmitted.Payload
	validated := validateIfPayloadIsInBlock(submittedPayload, resp.Transactions)
	if !validated {
		return fmt.Errorf("transactions fetched from HotShot doesn't contain the submitted payload")
	}

	// Reset the last submit failure time if we successfully fetch the transaction and verify its inclusion/namespace proof
	s.lastSubmitFailureAt = nil

	// Validation completed. Update the database
	s.espressoTxnsStateInsertionMutex.Lock()
	defer s.espressoTxnsStateInsertionMutex.Unlock()

	batch := s.db.NewBatch()
	if err := s.setEspressoSubmittedTxns(batch, submittedTxns[1:]); err != nil {
		return fmt.Errorf("failed to set the espresso submitted txns: %w", err)
	}
	lastConfirmedPos := firstSubmitted.Pos[len(firstSubmitted.Pos)-1]
	if err := s.setEspressoLastConfirmedPos(batch, &lastConfirmedPos); err != nil {
		return fmt.Errorf("failed to set the last confirmed position (pos: %d): %w", lastConfirmedPos, err)
	}

	if err := batch.Write(); err != nil {
		return fmt.Errorf("failed to write to db: %w", err)
	}

	return nil
}

func (s *TransactionStreamer) getEspressoSubmittedTxns() ([]SubmittedEspressoTx, error) {
	posBytes, err := s.db.Get(espressoSubmittedTxns)
	if err != nil {
		if dbutil.IsErrNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	var tx []SubmittedEspressoTx
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

func (s *TransactionStreamer) setEspressoSubmittedTxns(batch ethdb.KeyValueWriter, txns []SubmittedEspressoTx) error {
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
func (s *TransactionStreamer) SubmitEspressoTransactionPos(pos arbutil.MessageIndex) error {
	s.espressoTxnsStateInsertionMutex.Lock()
	defer s.espressoTxnsStateInsertionMutex.Unlock()

	batch := s.db.NewBatch()
	pendingTxnsPos, err := s.getEspressoPendingTxnsPos()
	if err != nil {
		return err
	}

	if pendingTxnsPos == nil {
		// if the key doesn't exist, create a new array with the pos
		pendingTxnsPos = []arbutil.MessageIndex{pos}
	} else {
		pendingTxnsPos = append(pendingTxnsPos, pos)
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

func (s *TransactionStreamer) resubmitEspressoTransactions(ctx context.Context, tx SubmittedEspressoTx) (*tagged_base64.TaggedBase64, error) {
	log.Info("Resubmitting tx to Espresso", "tx", tx.Hash)
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

	pendingTxnsPos, err := s.getEspressoPendingTxnsPos()
	if err != nil {
		return err
	}

	if len(pendingTxnsPos) > 0 {
		fetcher := func(pos arbutil.MessageIndex) ([]byte, error) {
			msg, err := s.GetMessage(pos)
			if err != nil {
				return nil, err
			}
			b, err := rlp.EncodeToBytes(msg)
			if err != nil {
				return nil, err
			}
			return b, nil
		}

		payload, msgCnt := buildRawHotShotPayload(pendingTxnsPos, fetcher, s.espressoMaxTransactionSize)
		if msgCnt == 0 {
			return fmt.Errorf("failed to build the hotshot transaction: a large message has exceeded the size limit or failed to get a message from storage")
		}

		payload, err = signHotShotPayload(payload, s.getAttestationQuote)
		if err != nil {
			return fmt.Errorf("failed to sign the hotshot payload %w", err)
		}

		log.Info("submitting transaction to hotshot for finalization")

		// Note: same key should not be used for two namespaces for this to work
		hash, err := s.espressoClient.SubmitTransaction(ctx, espressoTypes.Transaction{
			Payload:   payload,
			Namespace: s.chainConfig.ChainID.Uint64(),
		})

		if err != nil {
			return fmt.Errorf("failed to submit transaction to espresso: %w", err)
		}

		s.espressoTxnsStateInsertionMutex.Lock()
		defer s.espressoTxnsStateInsertionMutex.Unlock()

		batch := s.db.NewBatch()
		submittedPos := pendingTxnsPos[:msgCnt]

		submittedTxns, err := s.getEspressoSubmittedTxns()
		if err != nil {
			return fmt.Errorf("failed to get the submitted txns: %w", err)
		}
		tx := SubmittedEspressoTx{
			Hash:    hash.String(),
			Pos:     submittedPos,
			Payload: payload,
		}
		if submittedTxns == nil {
			submittedTxns = []SubmittedEspressoTx{tx}
		} else {
			submittedTxns = append(submittedTxns, tx)
		}

		if err = s.setEspressoSubmittedTxns(batch, submittedTxns); err != nil {
			return fmt.Errorf("failed to set espresso submitted txns: %w", err)
		}

		pendingTxnsPos = pendingTxnsPos[msgCnt:]
		err = s.setEspressoPendingTxnsPos(batch, pendingTxnsPos)
		if err != nil {
			return fmt.Errorf("failed to set the pending txn: %w", err)
		}

		err = batch.Write()
		if err != nil {
			return fmt.Errorf("failed to write to db: %w", err)
		}
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
	retryRate := s.espressoTxnsPollingInterval * 50
	var err error
	if s.UseEscapeHatch {
		err = s.checkEspressoLiveness()
		if err != nil {
			if ctx.Err() != nil {
				return 0
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
			return 0
		}
		logLevel := getLogLevel(err)
		logLevel("error polling finality, will retry", "err", err)
		return retryRate
	}
	espressoMerkleProofEphemeralErrorHandler.Reset()
	return 0
}

/**
 * Submits the transactions to espresso if the escape hatch is not enabled
 */
func (s *TransactionStreamer) submitTransactionsToEspresso(ctx context.Context, ignored struct{}) time.Duration {
	retryRate := s.espressoTxnsPollingInterval * 50
	shouldSubmit := s.shouldSubmitEspressoTransaction()
	// Only submit the transaction if escape hatch is not enabled
	if shouldSubmit {
		err := s.submitEspressoTransactions(ctx)
		if err != nil {
			log.Error("failed to submit espresso transactions", "err", err)
			return retryRate
		}
	}
	return s.espressoTxnsPollingInterval
}

func (s *TransactionStreamer) pollToResubmitEspressoTransactions(ctx context.Context, ignored struct{}) time.Duration {
	retryRate := s.espressoTxnsPollingInterval * 50
	submittedTxns, err := s.getEspressoSubmittedTxns()
	if err != nil {
		log.Warn("resubmitting espresso transactions failed: unable to get submitted transactions, will retry: %w", err)
		return retryRate
	}

	shouldResubmit := s.shouldResubmitEspressoTransactions(ctx, submittedTxns)
	if shouldResubmit {
		for _, tx := range submittedTxns {
			txHash, err := s.resubmitEspressoTransactions(ctx, tx)
			if err != nil {
				log.Warn("failed to resubmit espresso transactions", "err", err)
				return retryRate
			}
			log.Info(fmt.Sprintf("trying to resubmit transaction succeeded: (hash: %s)", txHash.String()))
		}
		// Reset the last submit failure time because we successfully resubmitted the transactions
		s.lastSubmitFailureAt = nil
	}
	return s.espressoTxnsPollingInterval
}

func (s *TransactionStreamer) shouldSubmitEspressoTransaction() bool {
	if s.espressoClient == nil && s.lightClientReader == nil {
		return false
	}
	return !s.EscapeHatchEnabled
}

func (s *TransactionStreamer) shouldResubmitEspressoTransactions(ctx context.Context, submittedTxns []SubmittedEspressoTx) bool {
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

	_, err = s.espressoClient.FetchTransactionByHash(ctx, submittedTxHash)
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

func (s *TransactionStreamer) Start(ctxIn context.Context) error {
	s.StopWaiter.Start(ctxIn, s)

	if s.lightClientReader != nil && s.espressoClient != nil {
		err := stopwaiter.CallIterativelyWith[struct{}](&s.StopWaiterSafe, s.pollSubmittedTransactionForFinality, s.newSovereignTxNotifier)
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
