// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbnode

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	flag "github.com/spf13/pflag"
	"github.com/syndtr/goleveldb/leveldb"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/arbos/l1pricing"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/broadcaster"
	"github.com/offchainlabs/nitro/staker"
	"github.com/offchainlabs/nitro/util/sharedmetrics"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

type ExecutionEngine struct {
	stopwaiter.StopWaiter

	bc        *core.BlockChain
	validator *staker.BlockValidator
	streamer  *TransactionStreamer

	resequenceChan    chan []*arbstate.MessageWithMetadata
	createBlocksMutex sync.Mutex

	newBlockNotifier chan struct{}
	latestBlockMutex sync.Mutex
	latestBlock      *types.Block

	nextScheduledVersionCheck time.Time // protected by the createBlocksMutex

	reorgSequencing func() *arbos.SequencingHooks
}

// TransactionStreamer produces blocks from a node's L1 messages, storing the results in the blockchain and recording their positions
// The streamer is notified when there's new batches to process
type TransactionStreamer struct {
	stopwaiter.StopWaiter

	chainConfig *params.ChainConfig
	exec        *ExecutionEngine

	db           ethdb.Database
	fatalErrChan chan<- error
	config       TransactionStreamerConfigFetcher

	insertionMutex     sync.Mutex // cannot be acquired while reorgMutex is held
	reorgMutex         sync.RWMutex
	newMessageNotifier chan struct{}

	nextAllowedFeedReorgLog time.Time

	broadcasterQueuedMessages            []arbstate.MessageWithMetadata
	broadcasterQueuedMessagesPos         uint64
	broadcasterQueuedMessagesActiveReorg bool

	coordinator     *SeqCoordinator
	broadcastServer *broadcaster.Broadcaster
	inboxReader     *InboxReader
}

type TransactionStreamerConfig struct {
	MaxBroadcastQueueSize   int   `koanf:"max-broadcaster-queue-size"`
	MaxReorgResequenceDepth int64 `koanf:"max-reorg-resequence-depth" reload:"hot"`
}

type TransactionStreamerConfigFetcher func() *TransactionStreamerConfig

var DefaultTransactionStreamerConfig = TransactionStreamerConfig{
	MaxBroadcastQueueSize:   10_000,
	MaxReorgResequenceDepth: 128 * 1024,
}

func TransactionStreamerConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Int(prefix+".max-broadcaster-queue-size", DefaultTransactionStreamerConfig.MaxBroadcastQueueSize, "maximum cache of pending broadcaster messages")
	f.Int64(prefix+".max-reorg-resequence-depth", DefaultTransactionStreamerConfig.MaxReorgResequenceDepth, "maximum number of messages to attempt to resequence on reorg (0 = never resequence, -1 = always resequence)")
}

func NewExecutionEngine(bc *core.BlockChain) (*ExecutionEngine, error) {
	return &ExecutionEngine{
		bc:               bc,
		resequenceChan:   make(chan []*arbstate.MessageWithMetadata),
		newBlockNotifier: make(chan struct{}, 1),
	}, nil
}

func NewTransactionStreamer(
	db ethdb.Database,
	chainConfig *params.ChainConfig,
	exec *ExecutionEngine,
	broadcastServer *broadcaster.Broadcaster,
	fatalErrChan chan<- error,
	config TransactionStreamerConfigFetcher,
) (*TransactionStreamer, error) {
	streamer := &TransactionStreamer{
		exec:               exec,
		chainConfig:        chainConfig,
		db:                 db,
		newMessageNotifier: make(chan struct{}, 1),
		broadcastServer:    broadcastServer,
		fatalErrChan:       fatalErrChan,
		config:             config,
	}
	streamer.exec.streamer = streamer
	err := streamer.cleanupInconsistentState()
	if err != nil {
		return nil, err
	}
	return streamer, nil
}

// Encodes a uint64 as bytes in a lexically sortable manner for database iteration.
// Generally this is only used for database keys, which need sorted.
// A shorter RLP encoding is usually used for database values.
func uint64ToKey(x uint64) []byte {
	data := make([]byte, 8)
	binary.BigEndian.PutUint64(data, x)
	return data
}

func (s *ExecutionEngine) setBlockValidator(validator *staker.BlockValidator) {
	if s.Started() {
		panic("trying to set block validator after start")
	}
	if s.validator != nil {
		panic("trying to set block validator when already set")
	}
	s.validator = validator
}

// TODO: this is needed only for block validator
func (s *TransactionStreamer) SetBlockValidator(validator *staker.BlockValidator) {
	s.exec.setBlockValidator(validator)
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

func (s *TransactionStreamer) SetInboxReader(inboxReader *InboxReader) {
	if s.Started() {
		panic("trying to set inbox reader after start")
	}
	if s.inboxReader != nil {
		panic("trying to set inbox reader when already set")
	}
	s.inboxReader = inboxReader
}

func (s *ExecutionEngine) SetReorgSequencingPolicy(reorgSequencing func() *arbos.SequencingHooks) {
	if s.Started() {
		panic("trying to set reorg sequencing policy after start")
	}
	if s.reorgSequencing != nil {
		panic("trying to set reorg sequencing policy when already set")
	}
	s.reorgSequencing = reorgSequencing
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

// The insertion mutex must be held. This acquires the reorg mutex.
// Note: oldMessages will be empty if reorgHook is nil
func (s *TransactionStreamer) reorg(batch ethdb.Batch, count arbutil.MessageIndex, newMessages []arbstate.MessageWithMetadata) error {
	if count == 0 {
		return errors.New("cannot reorg out init message")
	}
	lastDelayedSeqNum, err := s.getPrevPrevDelayedRead(count)
	if err != nil {
		return err
	}
	var oldMessages []*arbstate.MessageWithMetadata

	targetMsgCount, err := s.GetMessageCount()
	if err != nil {
		return err
	}
	config := s.config()
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
			lastDelayedSeqNum++

			if s.inboxReader != nil {
				// Verify that the delayed message we're re-sequencing matches what we have after the reorg
				haveDelayedMessage, err := s.inboxReader.tracker.GetDelayedMessage(delayedSeqNum)
				if err != nil {
					if !strings.Contains(err.Error(), "not found") {
						log.Error("failed to lookup delayed message to re-sequence", "id", header.RequestId, "err", err)
					}
					continue
				}
				haveDelayedMessageBytes, err := haveDelayedMessage.Serialize()
				if err != nil {
					log.Error("failed to serialize new delayed message from database", "err", err)
					continue
				}
				oldDelayedMessageBytes, err := oldMessage.Message.Serialize()
				if err != nil {
					log.Error("failed to serialize old delayed message from database", "err", err)
					continue
				}
				if !bytes.Equal(haveDelayedMessageBytes, oldDelayedMessageBytes) {
					// This delayed message is different, so we'll save re-sequencing it for the real delayed sequencer later
					continue
				}
			}
		}

		oldMessages = append(oldMessages, oldMessage)
	}

	s.reorgMutex.Lock()
	defer s.reorgMutex.Unlock()

	err = s.exec.Reorg(count, newMessages, oldMessages)
	if err != nil {
		return err
	}

	err = deleteStartingAt(s.db, batch, messagePrefix, uint64ToKey(uint64(count)))
	if err != nil {
		return err
	}

	return setMessageCount(batch, count)
}

func (s *ExecutionEngine) Reorg(count arbutil.MessageIndex, newMessages []arbstate.MessageWithMetadata, oldMessages []*arbstate.MessageWithMetadata) error {
	s.createBlocksMutex.Lock()
	successful := false
	defer func() {
		if !successful {
			s.createBlocksMutex.Unlock()
		}
	}()
	blockNum, err := s.MessageCountToBlockNumber(count)
	if err != nil {
		return err
	}
	// We can safely cast blockNum to a uint64 as we checked count == 0 above
	targetBlock := s.bc.GetBlockByNumber(uint64(blockNum))
	if targetBlock == nil {
		log.Warn("reorg target block not found", "block", blockNum)
		return nil
	}
	if s.validator != nil {
		err = s.validator.ReorgToBlock(targetBlock.NumberU64(), targetBlock.Hash())
		if err != nil {
			return err
		}
	}

	err = s.bc.ReorgToOldBlock(targetBlock)
	if err != nil {
		return err
	}
	for i := range newMessages {
		err := s.digestMessageWithBlockMutex(count+arbutil.MessageIndex(i), &newMessages[i])
		if err != nil {
			return err
		}
	}
	s.resequenceChan <- oldMessages
	successful = true
	return nil
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
func (s *TransactionStreamer) GetMessage(seqNum arbutil.MessageIndex) (*arbstate.MessageWithMetadata, error) {
	key := dbKey(messagePrefix, uint64(seqNum))
	data, err := s.db.Get(key)
	if err != nil {
		return nil, err
	}
	var message arbstate.MessageWithMetadata
	err = rlp.DecodeBytes(data, &message)
	if err != nil {
		return nil, err
	}

	return &message, nil
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

func (s *TransactionStreamer) AddMessages(pos arbutil.MessageIndex, messagesAreConfirmed bool, messages []arbstate.MessageWithMetadata) error {
	return s.AddMessagesAndEndBatch(pos, messagesAreConfirmed, messages, nil)
}

func (s *TransactionStreamer) AddBroadcastMessages(feedMessages []*broadcaster.BroadcastFeedMessage) error {
	if len(feedMessages) == 0 {
		return nil
	}
	broadcastStartPos := feedMessages[0].SequenceNumber
	var messages []arbstate.MessageWithMetadata
	broadcastAfterPos := broadcastStartPos
	for _, feedMessage := range feedMessages {
		if broadcastAfterPos != feedMessage.SequenceNumber {
			return fmt.Errorf("invalid sequence number %v, expected %v", feedMessage.SequenceNumber, broadcastAfterPos)
		}
		if feedMessage.Message.Message == nil || feedMessage.Message.Message.Header == nil {
			return fmt.Errorf("invalid feed message at sequence number %v", feedMessage.SequenceNumber)
		}
		messages = append(messages, feedMessage.Message)
		broadcastAfterPos++
	}

	s.insertionMutex.Lock()
	defer s.insertionMutex.Unlock()

	var batch ethdb.Batch
	var feedReorg bool
	var err error
	// Skip any messages already in the database
	// prevDelayedRead set to 0 because it's only used to compute the output prevDelayedRead which is not used here
	// Messages from feed are not confirmed, so confirmedMessageCount is 0 and confirmedReorg can be ignored
	feedReorg, _, _, broadcastStartPos, messages, err = s.skipDuplicateMessages(
		0,
		broadcastStartPos,
		messages,
		0,
		&batch,
	)
	if err != nil {
		return err
	}
	if batch != nil {
		// Write database updates made inside skipDuplicateMessages
		if err := batch.Write(); err != nil {
			return err
		}
	}
	if len(messages) == 0 {
		// No new messages received
		return nil
	}

	if len(s.broadcasterQueuedMessages) == 0 || (feedReorg && !s.broadcasterQueuedMessagesActiveReorg) {
		// Empty cache or feed different from database, save current feed messages until confirmed L1 messages catch up.
		s.broadcasterQueuedMessages = messages
		atomic.StoreUint64(&s.broadcasterQueuedMessagesPos, uint64(broadcastStartPos))
		s.broadcasterQueuedMessagesActiveReorg = feedReorg
	} else {
		broadcasterQueuedMessagesPos := arbutil.MessageIndex(atomic.LoadUint64(&s.broadcasterQueuedMessagesPos))
		if broadcasterQueuedMessagesPos >= broadcastStartPos {
			// Feed messages older than cache
			s.broadcasterQueuedMessages = messages
			atomic.StoreUint64(&s.broadcasterQueuedMessagesPos, uint64(broadcastStartPos))
			s.broadcasterQueuedMessagesActiveReorg = feedReorg
		} else if broadcasterQueuedMessagesPos+arbutil.MessageIndex(len(s.broadcasterQueuedMessages)) == broadcastStartPos {
			// Feed messages can be added directly to end of cache
			maxQueueSize := s.config().MaxBroadcastQueueSize
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
			atomic.StoreUint64(&s.broadcasterQueuedMessagesPos, uint64(broadcastStartPos))
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
			if !errors.Is(err, leveldb.ErrNotFound) {
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
	return s.AddMessages(0, false, []arbstate.MessageWithMetadata{{
		Message: &arbos.L1IncomingMessage{
			Header: &arbos.L1IncomingMessageHeader{
				Kind:      arbos.L1MessageType_Initialize,
				RequestId: &common.Hash{},
				L1BaseFee: common.Big0,
			},
			L2msg: math.U256Bytes(s.chainConfig.ChainID),
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

func (s *TransactionStreamer) AddMessagesAndEndBatch(pos arbutil.MessageIndex, messagesAreConfirmed bool, messages []arbstate.MessageWithMetadata, batch ethdb.Batch) error {
	s.insertionMutex.Lock()
	defer s.insertionMutex.Unlock()

	return s.addMessagesAndEndBatchImpl(pos, messagesAreConfirmed, messages, batch)
}

func (s *TransactionStreamer) getPrevPrevDelayedRead(pos arbutil.MessageIndex) (uint64, error) {
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

// skipDuplicateMessages removes any duplicate messages that are already in database and
// triggers reorg if message doesn't match what is stored in database.
// confirmedMessageCount is the number of messages that are from L1 starting at the beginning of messages array
// Returns: feedReorg, confirmedReorg, prevDelayedRead, deduplicatedMessagesStartPos, deduplicatedMessages, err
// Note that feedReorg and confirmedReorg will never both be set
// If feedReorg or confirmedReorg are set, the returned deduplicatedMessages will start at the point of the reorg
func (s *TransactionStreamer) skipDuplicateMessages(
	prevDelayedRead uint64,
	pos arbutil.MessageIndex,
	messages []arbstate.MessageWithMetadata,
	confirmedMessageCount int,
	batch *ethdb.Batch,
) (bool, bool, uint64, arbutil.MessageIndex, []arbstate.MessageWithMetadata, error) {
	feedReorg := false
	confirmedReorg := false
	for {
		if len(messages) == 0 {
			break
		}
		key := dbKey(messagePrefix, uint64(pos))
		hasMessage, err := s.db.Has(key)
		if err != nil {
			return false, false, 0, 0, nil, err
		}
		if !hasMessage {
			break
		}
		haveMessage, err := s.db.Get(key)
		if err != nil {
			return false, false, 0, 0, nil, err
		}
		nextMessage := messages[0]
		wantMessage, err := rlp.EncodeToBytes(nextMessage)
		if err != nil {
			return false, false, 0, 0, nil, err
		}
		if !bytes.Equal(haveMessage, wantMessage) {
			// Current message does not exactly match message in database
			var dbMessageParsed arbstate.MessageWithMetadata
			err := rlp.DecodeBytes(haveMessage, &dbMessageParsed)
			if err != nil {
				if confirmedMessageCount > 0 {
					confirmedReorg = true
				} else {
					feedReorg = true
				}
				log.Warn("TransactionStreamer: Reorg detected! (failed parsing db message)",
					"pos", pos,
					"err", err,
					"confirmedMessageCount", confirmedMessageCount,
				)
				break
			} else {
				var duplicateMessage bool
				var gotHeader *arbos.L1IncomingMessageHeader
				if nextMessage.Message != nil {
					gotHeader = nextMessage.Message.Header
					if dbMessageParsed.Message.BatchGasCost == nil || nextMessage.Message.BatchGasCost == nil {
						// Remove both of the batch gas costs and see if the messages still differ
						nextMessageCopy := nextMessage
						nextMessageCopy.Message = new(arbos.L1IncomingMessage)
						*nextMessageCopy.Message = *nextMessage.Message
						dbMessageParsed.Message.BatchGasCost = nil
						nextMessageCopy.Message.BatchGasCost = nil
						if reflect.DeepEqual(dbMessageParsed, nextMessageCopy) {
							// Actually this isn't a reorg; only the batch gas costs differed
							if nextMessage.Message.BatchGasCost != nil && confirmedMessageCount > 0 {
								// If our new message has a gas cost cached, but the old one didn't,
								// update the message in the database to add the gas cost cache.
								if batch == nil {
									return false, false, 0, 0, nil, errors.New("skipDuplicateMessages missing pointer to batch")
								}
								if *batch == nil {
									*batch = s.db.NewBatch()
								}
								err = s.writeMessage(pos, nextMessage, *batch)
								if err != nil {
									return false, false, 0, 0, nil, err
								}
							}
							duplicateMessage = true
						}
					}
				}

				if !duplicateMessage {
					var logFeedReorg bool
					if confirmedMessageCount > 0 {
						confirmedReorg = true
					} else {
						feedReorg = true
						if time.Now().After(s.nextAllowedFeedReorgLog) {
							s.nextAllowedFeedReorgLog = time.Now().Add(time.Minute)
							logFeedReorg = true
						}
					}
					if confirmedReorg || logFeedReorg {
						log.Warn("TransactionStreamer: Reorg detected!",
							"pos", pos,
							"got-delayed", nextMessage.DelayedMessagesRead,
							"got-header", gotHeader,
							"db-delayed", dbMessageParsed.DelayedMessagesRead,
							"db-header", dbMessageParsed.Message.Header,
							"confirmedMessageCount", confirmedMessageCount,
						)
					}
					break
				}
			}
		}

		// This message is a duplicate, skip it
		prevDelayedRead = nextMessage.DelayedMessagesRead
		messages = messages[1:]
		confirmedMessageCount--
		pos++
	}

	return feedReorg, confirmedReorg, prevDelayedRead, pos, messages, nil
}

func (s *ExecutionEngine) HeadMessageNumber() (arbutil.MessageIndex, error) {
	msgCount, err := s.BlockNumberToMessageCount(s.bc.CurrentBlock().Header().Number.Uint64())
	if err != nil {
		return 0, err
	}
	return msgCount - 1, err
}

func (s *ExecutionEngine) HeadMessageNumberSync(t *testing.T) (arbutil.MessageIndex, error) {
	s.createBlocksMutex.Lock()
	defer s.createBlocksMutex.Unlock()
	return s.HeadMessageNumber()
}

func (s *ExecutionEngine) NextDelayedMessageNumber() (uint64, error) {
	return s.bc.CurrentHeader().Nonce.Uint64(), nil
}

func (s *TransactionStreamer) addMessagesAndEndBatchImpl(messageStartPos arbutil.MessageIndex, messagesAreConfirmed bool, messages []arbstate.MessageWithMetadata, batch ethdb.Batch) error {
	var confirmedMessageCount int
	if messagesAreConfirmed {
		confirmedMessageCount = len(messages)
	}
	messagesAfterPos := messageStartPos + arbutil.MessageIndex(len(messages))
	broadcastStartPos := arbutil.MessageIndex(atomic.LoadUint64(&s.broadcasterQueuedMessagesPos))

	lastDelayedRead, err := s.getPrevPrevDelayedRead(messageStartPos)
	if err != nil {
		return err
	}

	clearQueueOnSuccess := false
	if (s.broadcasterQueuedMessagesActiveReorg && messageStartPos <= broadcastStartPos) ||
		(!s.broadcasterQueuedMessagesActiveReorg && broadcastStartPos <= messagesAfterPos) {
		// Active broadcast reorg and L1 messages at or before start of broadcast messages
		// Or no active broadcast reorg and broadcast messages start before or immediately after last L1 message
		if messagesAfterPos >= broadcastStartPos {
			broadcastSliceIndex := int(messagesAfterPos - broadcastStartPos)
			if broadcastSliceIndex < len(s.broadcasterQueuedMessages) {
				// Some cached feed messages can be used
				messages = append(messages, s.broadcasterQueuedMessages[broadcastSliceIndex:]...)
			}
		}

		// L1 used or replaced broadcast cache items
		clearQueueOnSuccess = true
	}

	var feedReorg bool
	var confirmedReorg bool
	// Skip any duplicate messages already in the database
	feedReorg, confirmedReorg, lastDelayedRead, messageStartPos, messages, err = s.skipDuplicateMessages(
		lastDelayedRead,
		messageStartPos,
		messages,
		confirmedMessageCount,
		&batch,
	)
	if err != nil {
		return err
	}
	if feedReorg {
		// Never allow feed to reorg confirmed messages
		// Note that any remaining messages must be feed messages, so we're done here
		// The feed reorg was already logged if necessary in skipDuplicateMessages
		if batch == nil {
			return nil
		}
		return batch.Write()
	}

	// Validate delayed message counts of remaining messages
	for i, msg := range messages {
		msgPos := messageStartPos + arbutil.MessageIndex(i)
		diff := msg.DelayedMessagesRead - lastDelayedRead
		if diff != 0 && diff != 1 {
			return fmt.Errorf("attempted to insert jump from %v delayed messages read to %v delayed messages read at message index %v", lastDelayedRead, msg.DelayedMessagesRead, msgPos)
		}
		lastDelayedRead = msg.DelayedMessagesRead
		if msg.Message == nil {
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
		if batch == nil {
			return nil
		}
		return batch.Write()
	}

	err = s.writeMessages(messageStartPos, messages, batch)
	if err != nil {
		return err
	}

	if clearQueueOnSuccess {
		s.broadcasterQueuedMessages = s.broadcasterQueuedMessages[:0]
		atomic.StoreUint64(&s.broadcasterQueuedMessagesPos, 0)
		s.broadcasterQueuedMessagesActiveReorg = false
	}

	return nil
}

func messageFromTxes(header *arbos.L1IncomingMessageHeader, txes types.Transactions, txErrors []error) (*arbos.L1IncomingMessage, error) {
	var l2Message []byte
	if len(txes) == 1 && txErrors[0] == nil {
		txBytes, err := txes[0].MarshalBinary()
		if err != nil {
			return nil, err
		}
		l2Message = append(l2Message, arbos.L2MessageKind_SignedTx)
		l2Message = append(l2Message, txBytes...)
	} else {
		l2Message = append(l2Message, arbos.L2MessageKind_Batch)
		sizeBuf := make([]byte, 8)
		for i, tx := range txes {
			if txErrors[i] != nil {
				continue
			}
			txBytes, err := tx.MarshalBinary()
			if err != nil {
				return nil, err
			}
			binary.BigEndian.PutUint64(sizeBuf, uint64(len(txBytes)+1))
			l2Message = append(l2Message, sizeBuf...)
			l2Message = append(l2Message, arbos.L2MessageKind_SignedTx)
			l2Message = append(l2Message, txBytes...)
		}
	}
	return &arbos.L1IncomingMessage{
		Header: header,
		L2msg:  l2Message,
	}, nil
}

// The caller must hold the createBlocksMutex
func (s *ExecutionEngine) resequenceReorgedMessages(messages []*arbstate.MessageWithMetadata) {
	if s.reorgSequencing == nil {
		return
	}

	log.Info("Trying to resequence messages", "number", len(messages))
	lastBlockHeader := s.bc.CurrentBlock().Header()
	if lastBlockHeader == nil {
		log.Error("block header not found during resequence")
		return
	}

	nextDelayedSeqNum := lastBlockHeader.Nonce.Uint64()

	for _, msg := range messages {
		// Check if the message is non-nil just to be safe
		if msg == nil || msg.Message == nil || msg.Message.Header == nil {
			continue
		}
		header := msg.Message.Header
		if header.RequestId != nil {
			delayedSeqNum := header.RequestId.Big().Uint64()
			if delayedSeqNum != nextDelayedSeqNum {
				log.Info("not resequencing delayed message due to unexpected index", "expected", nextDelayedSeqNum, "found", delayedSeqNum)
				continue
			}
			err := s.sequenceDelayedMessageWithBlockMutex(msg.Message, delayedSeqNum)
			if err != nil {
				log.Error("failed to re-sequence old delayed message removed by reorg", "err", err)
			}
			nextDelayedSeqNum += 1
			continue
		}
		if header.Kind != arbos.L1MessageType_L2Message || header.Poster != l1pricing.BatchPosterAddress {
			// This shouldn't exist?
			log.Warn("skipping non-standard sequencer message found from reorg", "header", header)
			continue
		}
		// We don't need a batch fetcher as this is an L2 message
		txes, err := msg.Message.ParseL2Transactions(s.bc.Config().ChainID, nil)
		if err != nil {
			log.Warn("failed to parse sequencer message found from reorg", "err", err)
			continue
		}
		_, err = s.sequenceTransactionsWithBlockMutex(msg.Message.Header, txes, s.reorgSequencing())
		if err != nil {
			log.Error("failed to re-sequence old user message removed by reorg", "err", err)
			return
		}
	}
}

func (s *ExecutionEngine) SequenceTransactions(header *arbos.L1IncomingMessageHeader, txes types.Transactions, hooks *arbos.SequencingHooks) (*types.Block, error) {
	for {
		hooks.TxErrors = nil
		s.createBlocksMutex.Lock()
		block, err := s.sequenceTransactionsWithBlockMutex(header, txes, hooks)
		s.createBlocksMutex.Unlock()
		if !errors.Is(err, ErrSequencerInsertLockTaken) {
			return block, err
		}
		<-time.After(time.Millisecond * 100)
	}
}

func (s *ExecutionEngine) sequenceTransactionsWithBlockMutex(header *arbos.L1IncomingMessageHeader, txes types.Transactions, hooks *arbos.SequencingHooks) (*types.Block, error) {
	lastBlockHeader := s.bc.CurrentBlock().Header()
	if lastBlockHeader == nil {
		return nil, errors.New("current block header not found")
	}

	statedb, err := s.bc.StateAt(lastBlockHeader.Root)
	if err != nil {
		return nil, err
	}

	delayedMessagesRead := lastBlockHeader.Nonce.Uint64()

	startTime := time.Now()
	block, receipts, err := arbos.ProduceBlockAdvanced(
		header,
		txes,
		delayedMessagesRead,
		lastBlockHeader,
		statedb,
		s.bc,
		s.bc.Config(),
		hooks,
	)
	if err != nil {
		return nil, err
	}
	blockCalcTime := time.Since(startTime)
	if len(hooks.TxErrors) != len(txes) {
		return nil, fmt.Errorf("unexpected number of error results: %v vs number of txes %v", len(hooks.TxErrors), len(txes))
	}

	if len(receipts) == 0 {
		return nil, nil
	}

	allTxsErrored := true
	for _, err := range hooks.TxErrors {
		if err == nil {
			allTxsErrored = false
			break
		}
	}
	if allTxsErrored {
		return nil, nil
	}

	msg, err := messageFromTxes(header, txes, hooks.TxErrors)
	if err != nil {
		return nil, err
	}

	msgWithMeta := arbstate.MessageWithMetadata{
		Message:             msg,
		DelayedMessagesRead: delayedMessagesRead,
	}

	pos, err := s.BlockNumberToMessageCount(lastBlockHeader.Number.Uint64())
	if err != nil {
		return nil, err
	}

	err = s.streamer.WriteMessageFromSequencer(pos, msgWithMeta)
	if err != nil {
		return nil, err
	}

	// Only write the block after we've written the messages, so if the node dies in the middle of this,
	// it will naturally recover on startup by regenerating the missing block.
	err = s.appendBlock(block, statedb, receipts, blockCalcTime)
	if err != nil {
		return nil, err
	}

	if s.validator != nil {
		s.validator.NewBlock(block, lastBlockHeader, msgWithMeta)
	}

	return block, nil
}

var ErrSequencerInsertLockTaken = errors.New("insert lock taken")

func (s *TransactionStreamer) WriteMessageFromSequencer(pos arbutil.MessageIndex, msgWithMeta arbstate.MessageWithMetadata) error {
	if s.coordinator != nil {
		if !s.coordinator.CurrentlyChosen() {
			return fmt.Errorf("%w: not main sequencer", ErrRetrySequencer)
		}
	}
	if !s.insertionMutex.TryLock() {
		return ErrSequencerInsertLockTaken
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

	if err := s.writeMessages(pos, []arbstate.MessageWithMetadata{msgWithMeta}, nil); err != nil {
		return err
	}

	if s.broadcastServer != nil {
		if err := s.broadcastServer.BroadcastSingle(msgWithMeta, pos); err != nil {
			log.Error("failed broadcasting message", "pos", pos, "err", err)
		}
	}

	return nil
}

func (s *ExecutionEngine) SequenceDelayedMessage(message *arbos.L1IncomingMessage, delayedSeqNum uint64) error {
	for {
		s.createBlocksMutex.Lock()
		err := s.sequenceDelayedMessageWithBlockMutex(message, delayedSeqNum)
		s.createBlocksMutex.Unlock()
		if !errors.Is(err, ErrSequencerInsertLockTaken) {
			return err
		}
		<-time.After(time.Millisecond * 100)
	}
}

func (s *ExecutionEngine) sequenceDelayedMessageWithBlockMutex(message *arbos.L1IncomingMessage, delayedSeqNum uint64) error {
	currentHeader := s.bc.CurrentBlock().Header()

	expectedDelayed := currentHeader.Nonce.Uint64()

	pos, err := s.BlockNumberToMessageCount(currentHeader.Number.Uint64())
	if err != nil {
		return err
	}

	if expectedDelayed != delayedSeqNum {
		return fmt.Errorf("wrong delayed message sequenced got %d expected %d", delayedSeqNum, expectedDelayed)
	}

	messageWithMeta := arbstate.MessageWithMetadata{
		Message:             message,
		DelayedMessagesRead: delayedSeqNum + 1,
	}

	err = s.streamer.WriteMessageFromSequencer(pos, messageWithMeta)
	if err != nil {
		return err
	}

	startTime := time.Now()
	block, statedb, receipts, err := s.createBlockFromNextMessage(&messageWithMeta)
	if err != nil {
		return err
	}

	err = s.appendBlock(block, statedb, receipts, time.Since(startTime))
	if err != nil {
		return err
	}

	log.Info("ExecutionEngine: Added DelayedMessages", "pos", pos, "delayed", delayedSeqNum, "block-header", block.Header())

	return nil
}

func (s *TransactionStreamer) GetGenesisBlockNumber() (uint64, error) {
	return s.chainConfig.ArbitrumChainParams.GenesisBlockNum, nil
}

func (s *ExecutionEngine) GetGenesisBlockNumber() (uint64, error) {
	return s.bc.Config().ArbitrumChainParams.GenesisBlockNum, nil
}

func (s *ExecutionEngine) BlockNumberToMessageCount(blockNum uint64) (arbutil.MessageIndex, error) {
	genesis, err := s.GetGenesisBlockNumber()
	if err != nil {
		return 0, err
	}
	return arbutil.BlockNumberToMessageCount(blockNum, genesis), nil
}

func (s *ExecutionEngine) MessageCountToBlockNumber(messageNum arbutil.MessageIndex) (int64, error) {
	genesis, err := s.GetGenesisBlockNumber()
	if err != nil {
		return 0, err
	}
	return arbutil.MessageCountToBlockNumber(messageNum, genesis), nil
}

// PauseReorgs until a matching call to ResumeReorgs (may be called concurrently)
func (s *TransactionStreamer) PauseReorgs() {
	s.reorgMutex.RLock()
}

func (s *TransactionStreamer) ResumeReorgs() {
	s.reorgMutex.RUnlock()
}

func (s *TransactionStreamer) writeMessage(pos arbutil.MessageIndex, msg arbstate.MessageWithMetadata, batch ethdb.Batch) error {
	key := dbKey(messagePrefix, uint64(pos))
	msgBytes, err := rlp.EncodeToBytes(msg)
	if err != nil {
		return err
	}
	return batch.Put(key, msgBytes)
}

// The mutex must be held, and pos must be the latest message count.
// `batch` may be nil, which initializes a new batch. The batch is closed out in this function.
func (s *TransactionStreamer) writeMessages(pos arbutil.MessageIndex, messages []arbstate.MessageWithMetadata, batch ethdb.Batch) error {
	if batch == nil {
		batch = s.db.NewBatch()
	}
	for i, msg := range messages {
		err := s.writeMessage(pos+arbutil.MessageIndex(i), msg, batch)
		if err != nil {
			return err
		}
	}

	err := setMessageCount(batch, pos+arbutil.MessageIndex(len(messages)))
	if err != nil {
		return err
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

// return value: true if should be called again
func (s *TransactionStreamer) feedNextMsg(ctx context.Context, exec *ExecutionEngine) bool {
	if ctx.Err() != nil {
		return false
	}
	if !s.reorgMutex.TryRLock() {
		return false
	}
	defer s.reorgMutex.RUnlock()
	msgCount, err := s.GetMessageCount()
	if err != nil {
		log.Error("feedOneMsg failed to get message count", "err", err)
		return false
	}
	pos, err := s.exec.HeadMessageNumber()
	if err != nil {
		log.Error("feedOneMsg failed to get exec engine message count", "err", err)
		return false
	}
	pos++
	if pos >= msgCount {
		return false
	}
	msg, err := s.GetMessage(pos)
	if err != nil {
		log.Error("feedOneMsg failed to readMessage", "err", err, "pos", pos)
		return false
	}
	err = s.exec.DigestMessage(pos, msg)
	if err != nil {
		log.Info("feedOneMsg failed to send message to execEngine", "err", err, "pos", pos)
		return false
	}
	return pos+1 < msgCount
}

// must hold createBlockMutex
func (s *ExecutionEngine) createBlockFromNextMessage(msg *arbstate.MessageWithMetadata) (*types.Block, *state.StateDB, types.Receipts, error) {
	currentHeader := s.bc.CurrentBlock().Header()
	if currentHeader == nil {
		return nil, nil, nil, errors.New("failed to get current header")
	}

	statedb, err := s.bc.StateAt(currentHeader.Root)
	if err != nil {
		return nil, nil, nil, err
	}
	statedb.StartPrefetcher("TransactionStreamer")
	defer statedb.StopPrefetcher()

	batchFetcher := func(batchNum uint64) ([]byte, error) {
		return s.streamer.inboxReader.GetSequencerMessageBytes(context.TODO(), batchNum)
	}

	block, receipts, err := arbos.ProduceBlock(
		msg.Message,
		msg.DelayedMessagesRead,
		currentHeader,
		statedb,
		s.bc,
		s.bc.Config(),
		batchFetcher,
	)

	return block, statedb, receipts, err
}

// must hold createBlockMutex
func (s *ExecutionEngine) appendBlock(block *types.Block, statedb *state.StateDB, receipts types.Receipts, duration time.Duration) error {
	var logs []*types.Log
	for _, receipt := range receipts {
		logs = append(logs, receipt.Logs...)
	}
	status, err := s.bc.WriteBlockAndSetHeadWithTime(block, receipts, logs, statedb, true, duration)
	if err != nil {
		return err
	}
	if status == core.SideStatTy {
		return errors.New("geth rejected block as non-canonical")
	}
	return nil
}

func (s *ExecutionEngine) DigestMessage(num arbutil.MessageIndex, msg *arbstate.MessageWithMetadata) error {
	if !s.createBlocksMutex.TryLock() {
		return errors.New("mutex held")
	}
	defer s.createBlocksMutex.Unlock()
	return s.digestMessageWithBlockMutex(num, msg)
}

func (s *ExecutionEngine) digestMessageWithBlockMutex(num arbutil.MessageIndex, msg *arbstate.MessageWithMetadata) error {
	currentHeader := s.bc.CurrentHeader()
	expNum, err := s.BlockNumberToMessageCount(currentHeader.Number.Uint64())
	if err != nil {
		return err
	}
	if expNum != num {
		return fmt.Errorf("wrong message number in digest got %d expected %d", num, expNum)
	}

	startTime := time.Now()
	block, statedb, receipts, err := s.createBlockFromNextMessage(msg)
	if err != nil {
		return err
	}

	err = s.appendBlock(block, statedb, receipts, time.Since(startTime))
	if err != nil {
		return err
	}

	if s.validator != nil {
		s.validator.NewBlock(block, currentHeader, *msg)
	}

	if time.Now().After(s.nextScheduledVersionCheck) {
		s.nextScheduledVersionCheck = time.Now().Add(time.Minute)
		arbState, err := arbosState.OpenSystemArbosState(statedb, nil, true)
		if err != nil {
			return err
		}
		version, timestampInt, err := arbState.GetScheduledUpgrade()
		if err != nil {
			return err
		}
		var timeUntilUpgrade time.Duration
		var timestamp time.Time
		if timestampInt == 0 {
			// This upgrade will take effect in the next block
			timestamp = time.Now()
		} else {
			// This upgrade is scheduled for the future
			timestamp = time.Unix(int64(timestampInt), 0)
			timeUntilUpgrade = time.Until(timestamp)
		}
		maxSupportedVersion := params.ArbitrumDevTestChainConfig().ArbitrumChainParams.InitialArbOSVersion
		logLevel := log.Warn
		if timeUntilUpgrade < time.Hour*24 {
			logLevel = log.Error
		}
		if version > maxSupportedVersion {
			logLevel(
				"you need to update your node to the latest version before this scheduled ArbOS upgrade",
				"timeUntilUpgrade", timeUntilUpgrade,
				"upgradeScheduledFor", timestamp,
				"maxSupportedArbosVersion", maxSupportedVersion,
				"pendingArbosUpgradeVersion", version,
			)
		}
	}

	sharedmetrics.UpdateSequenceNumberInBlockGauge(num)
	s.latestBlockMutex.Lock()
	s.latestBlock = block
	s.latestBlockMutex.Unlock()
	select {
	case s.newBlockNotifier <- struct{}{}:
	default:
	}
	return nil
}

func (s *TransactionStreamer) Start(ctxIn context.Context) {
	s.StopWaiter.Start(ctxIn, s)
	s.LaunchThread(func(ctx context.Context) {
		for {
			for s.feedNextMsg(ctx, s.exec) {
			}
			timer := time.NewTimer(time.Second)
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-s.newMessageNotifier:
				timer.Stop()
			case <-timer.C:
			}

		}
	})
}

func (s *ExecutionEngine) Start(ctx_in context.Context) {
	s.StopWaiter.Start(ctx_in, s)
	s.LaunchThread(func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				return
			case resequence := <-s.resequenceChan:
				s.resequenceReorgedMessages(resequence)
				s.createBlocksMutex.Unlock()
			}
		}
	})
	s.LaunchThread(func(ctx context.Context) {
		var lastBlock *types.Block
		for {
			select {
			case <-s.newBlockNotifier:
			case <-ctx.Done():
				return
			}
			s.latestBlockMutex.Lock()
			block := s.latestBlock
			s.latestBlockMutex.Unlock()
			if block != lastBlock && block != nil {
				log.Info(
					"created block",
					"l2Block", block.Number(),
					"l2BlockHash", block.Hash(),
				)
				lastBlock = block
				select {
				case <-time.After(time.Second):
				case <-ctx.Done():
					return
				}
			}
		}
	})
}
