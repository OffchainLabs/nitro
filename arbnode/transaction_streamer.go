// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbnode

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/broadcaster"
	"github.com/offchainlabs/nitro/util/stopwaiter"
	"github.com/offchainlabs/nitro/validator"
)

// Produces blocks from a node's L1 messages, storing the results in the blockchain and recording their positions
// The streamer is notified when there's new batches to process
type TransactionStreamer struct {
	stopwaiter.StopWaiter

	db      ethdb.Database
	bc      *core.BlockChain
	chainId uint64

	insertionMutex     sync.Mutex // cannot be acquired while reorgMutex or createBlocksMutex is held
	createBlocksMutex  sync.Mutex // cannot be acquired while reorgMutex is held
	reorgMutex         sync.RWMutex
	reorgPending       uint32 // atomic, indicates whether the reorgMutex is attempting to be acquired
	newMessageNotifier chan struct{}

	broadcasterQueuedMessages    []arbstate.MessageWithMetadata
	broadcasterQueuedMessagesPos uint64

	latestBlockAndMessageMutex sync.Mutex
	latestBlock                *types.Block
	latestMessage              *arbos.L1IncomingMessage
	newBlockNotifier           chan struct{}

	coordinator     *SeqCoordinator
	broadcastServer *broadcaster.Broadcaster
	validator       *validator.BlockValidator
	inboxReader     *InboxReader
}

func NewTransactionStreamer(
	db ethdb.Database,
	bc *core.BlockChain,
	broadcastServer *broadcaster.Broadcaster,
) (*TransactionStreamer, error) {
	inbox := &TransactionStreamer{
		db:                 db,
		bc:                 bc,
		newMessageNotifier: make(chan struct{}, 1),
		newBlockNotifier:   make(chan struct{}, 1),
		broadcastServer:    broadcastServer,
		chainId:            bc.Config().ChainID.Uint64(),
	}
	err := inbox.cleanupInconsistentState()
	if err != nil {
		return nil, err
	}
	return inbox, nil
}

// Encodes a uint64 as bytes in a lexically sortable manner for database iteration.
// Generally this is only used for database keys, which need sorted.
// A shorter RLP encoding is usually used for database values.
func uint64ToKey(x uint64) []byte {
	data := make([]byte, 8)
	binary.BigEndian.PutUint64(data, x)
	return data
}

func (s *TransactionStreamer) SetBlockValidator(validator *validator.BlockValidator) {
	if s.Started() {
		panic("trying to set block validator after start")
	}
	if s.validator != nil {
		panic("trying to set block validator when already set")
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

func (s *TransactionStreamer) SetInboxReader(inboxReader *InboxReader) {
	if s.Started() {
		panic("trying to set inbox reader after start")
	}
	if s.inboxReader != nil {
		panic("trying to set inbox reader when already set")
	}
	s.inboxReader = inboxReader
}

func (s *TransactionStreamer) cleanupInconsistentState() error {
	// If it doesn't exist yet, set the message count to 0
	hasMessageCount, err := s.db.Has(messageCountKey)
	if err != nil {
		return err
	}
	if !hasMessageCount {
		data, err := rlp.EncodeToBytes(uint64(0))
		if err != nil {
			return err
		}
		err = s.db.Put(messageCountKey, data)
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
	err := s.reorgToInternal(batch, count)
	if err != nil {
		return err
	}
	return batch.Write()
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

func (s *TransactionStreamer) reorgToInternal(batch ethdb.Batch, count arbutil.MessageIndex) error {
	if count == 0 {
		return errors.New("cannot reorg out init message")
	}
	atomic.AddUint32(&s.reorgPending, 1)
	s.reorgMutex.Lock()
	defer s.reorgMutex.Unlock()
	atomic.AddUint32(&s.reorgPending, ^uint32(0)) // decrement
	blockNum, err := s.MessageCountToBlockNumber(count)
	if err != nil {
		return err
	}
	// We can safely cast blockNum to a uint64 as we checked count == 0 above
	targetBlock := s.bc.GetBlockByNumber(uint64(blockNum))
	if targetBlock != nil {
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
	} else {
		log.Warn("reorg target block not found", "block", blockNum)
	}

	err = deleteStartingAt(s.db, batch, messagePrefix, uint64ToKey(uint64(count)))
	if err != nil {
		return err
	}
	countBytes, err := rlp.EncodeToBytes(count)
	if err != nil {
		return err
	}
	err = batch.Put(messageCountKey, countBytes)
	if err != nil {
		return err
	}

	return nil
}

func dbKey(prefix []byte, pos uint64) []byte {
	var key []byte
	key = append(key, prefix...)
	key = append(key, uint64ToKey(uint64(pos))...)
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

func (s *TransactionStreamer) AddMessages(pos arbutil.MessageIndex, force bool, messages []arbstate.MessageWithMetadata) error {
	return s.AddMessagesAndEndBatch(pos, force, messages, nil)
}

func (s *TransactionStreamer) AddBroadcastMessages(feedMessages []*broadcaster.BroadcastFeedMessage) error {
	if len(feedMessages) == 0 {
		return nil
	}
	startingSeqNum := feedMessages[0].SequenceNumber
	var messages []arbstate.MessageWithMetadata
	endingSeqNum := startingSeqNum
	for _, feedMessage := range feedMessages {
		if endingSeqNum != feedMessage.SequenceNumber {
			return fmt.Errorf("invalid sequence number %v, expected %v", feedMessage.SequenceNumber, endingSeqNum)
		}
		if feedMessage.Message.Message == nil || feedMessage.Message.Message.Header == nil {
			return fmt.Errorf("invalid feed message at sequence number %v", feedMessage.SequenceNumber)
		}
		messages = append(messages, feedMessage.Message)
		endingSeqNum++
	}

	s.insertionMutex.Lock()
	defer s.insertionMutex.Unlock()

	currentMessageCount, err := s.GetMessageCount()
	if err != nil {
		return err
	}

	if currentMessageCount >= startingSeqNum {
		s.broadcasterQueuedMessages = s.broadcasterQueuedMessages[:0]
		atomic.StoreUint64(&s.broadcasterQueuedMessagesPos, 0)
		return s.addMessagesAndEndBatchImpl(startingSeqNum, false, messages, nil)
	} else {
		broadcasterQueuedMessagesPos := arbutil.MessageIndex(atomic.LoadUint64(&s.broadcasterQueuedMessagesPos))
		if len(s.broadcasterQueuedMessages) > 0 && broadcasterQueuedMessagesPos+arbutil.MessageIndex(len(s.broadcasterQueuedMessages)) == startingSeqNum {
			s.broadcasterQueuedMessages = append(s.broadcasterQueuedMessages, messages...)
		} else {
			if len(s.broadcasterQueuedMessages) > 0 {
				log.Warn(
					"broadcaster queue jumped positions",
					"queuedMessages", len(s.broadcasterQueuedMessages),
					"expectedNextPos", broadcasterQueuedMessagesPos+arbutil.MessageIndex(len(s.broadcasterQueuedMessages)),
					"gotPos", startingSeqNum,
				)
			}
			s.broadcasterQueuedMessages = messages
			atomic.StoreUint64(&s.broadcasterQueuedMessagesPos, uint64(startingSeqNum))
		}
	}

	return nil
}

// Should only be used for testing or running a local dev node
func (s *TransactionStreamer) AddFakeInitMessage() error {
	return s.AddMessages(0, false, []arbstate.MessageWithMetadata{{
		Message: &arbos.L1IncomingMessage{
			Header: &arbos.L1IncomingMessageHeader{
				Kind:      arbos.L1MessageType_Initialize,
				RequestId: &common.Hash{},
				L1BaseFee: common.Big0,
			},
			L2msg: math.U256Bytes(s.bc.Config().ChainID),
		},
		DelayedMessagesRead: 1,
	}})
}

func (s *TransactionStreamer) GetMessageCountSync() (arbutil.MessageIndex, error) {
	s.insertionMutex.Lock()
	defer s.insertionMutex.Unlock()
	return s.GetMessageCount()
}

func (s *TransactionStreamer) AddMessagesAndEndBatch(pos arbutil.MessageIndex, force bool, messages []arbstate.MessageWithMetadata, batch ethdb.Batch) error {
	s.insertionMutex.Lock()
	defer s.insertionMutex.Unlock()

	return s.addMessagesAndEndBatchImpl(pos, force, messages, batch)
}

func (s *TransactionStreamer) addMessagesAndEndBatchImpl(pos arbutil.MessageIndex, force bool, messages []arbstate.MessageWithMetadata, batch ethdb.Batch) error {
	var prevDelayedRead uint64
	if pos > 0 {
		prevMsg, err := s.GetMessage(pos - 1)
		if err != nil {
			return fmt.Errorf("failed to get previous message: %w", err)
		}
		prevDelayedRead = prevMsg.DelayedMessagesRead
	}

	dontReorgAfter := len(messages)
	afterCount := pos + arbutil.MessageIndex(len(messages))
	broadcasterQueuedMessagesPos := arbutil.MessageIndex(atomic.LoadUint64(&s.broadcasterQueuedMessagesPos))
	if afterCount >= broadcasterQueuedMessagesPos {
		if int(afterCount-broadcasterQueuedMessagesPos) < len(messages) {
			messages = append(messages, s.broadcasterQueuedMessages[afterCount-broadcasterQueuedMessagesPos:]...)
		}
		s.broadcasterQueuedMessages = s.broadcasterQueuedMessages[:0]
		atomic.StoreUint64(&s.broadcasterQueuedMessagesPos, 0)
	}

	reorg := false
	// Skip any messages already in the database
	for {
		if len(messages) == 0 {
			break
		}
		key := dbKey(messagePrefix, uint64(pos))
		hasMessage, err := s.db.Has(key)
		if err != nil {
			return err
		}
		if !hasMessage {
			break
		}
		haveMessage, err := s.db.Get(key)
		if err != nil {
			return err
		}
		nextMessage := messages[0]
		wantMessage, err := rlp.EncodeToBytes(nextMessage)
		if err != nil {
			return err
		}
		if bytes.Equal(haveMessage, wantMessage) {
			// This message is a duplicate, skip it
			prevDelayedRead = nextMessage.DelayedMessagesRead
			messages = messages[1:]
			pos++
			dontReorgAfter--
		} else {
			var dbMessageParsed arbstate.MessageWithMetadata
			err := rlp.DecodeBytes(haveMessage, &dbMessageParsed)
			if err != nil {
				log.Warn("TransactionStreamer: Reorg detected! (failed parsing db message)", "pos", pos, "err", err)
			} else {
				var gotHeader *arbos.L1IncomingMessageHeader
				if nextMessage.Message != nil {
					gotHeader = nextMessage.Message.Header
				}
				log.Warn("TransactionStreamer: Reorg detected!", "pos", pos, "got-delayed", nextMessage.DelayedMessagesRead, "got-header", gotHeader, "db-delayed", dbMessageParsed.DelayedMessagesRead, "db-header", dbMessageParsed.Message.Header)
			}
			if dontReorgAfter > 0 {
				reorg = true
			} else {
				log.Warn("TransactionStreamer ignoring broadcast client reorg")
			}
			break
		}
	}

	// Validate delayed message counts of remaining messages
	for i, msg := range messages {
		msgPos := pos + arbutil.MessageIndex(i)
		diff := msg.DelayedMessagesRead - prevDelayedRead
		if diff != 0 && diff != 1 {
			return fmt.Errorf("attempted to insert jump from %v delayed messages read to %v delayed messages read at message index %v", prevDelayedRead, msg.DelayedMessagesRead, msgPos)
		}
		prevDelayedRead = msg.DelayedMessagesRead
		if msg.Message == nil {
			return fmt.Errorf("attempted to insert nil message at position %v", msgPos)
		}
	}

	if reorg {
		if force {
			batch := s.db.NewBatch()
			err := s.reorgToInternal(batch, pos)
			if err != nil {
				return err
			}
			err = batch.Write()
			if err != nil {
				return err
			}
		} else {
			return errors.New("reorg required but not allowed")
		}
	}
	if len(messages) == 0 {
		if batch == nil {
			return nil
		}
		return batch.Write()
	}

	return s.writeMessages(pos, messages, batch)
}

func messageFromTxes(header *arbos.L1IncomingMessageHeader, txes types.Transactions, txErrors []error) (*arbos.L1IncomingMessage, error) {
	if len(txErrors) != len(txes) {
		return nil, fmt.Errorf("unexpected number of error results: %v vs number of txes %v", len(txErrors), len(txes))
	}
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

func (s *TransactionStreamer) SequenceTransactions(header *arbos.L1IncomingMessageHeader, txes types.Transactions, hooks *arbos.SequencingHooks) error {
	s.insertionMutex.Lock()
	defer s.insertionMutex.Unlock()
	s.createBlocksMutex.Lock()
	defer s.createBlocksMutex.Unlock()
	s.reorgMutex.RLock()
	defer s.reorgMutex.RUnlock()

	pos, err := s.GetMessageCount()
	if err != nil {
		return err
	}

	lastBlockHeader := s.bc.CurrentBlock().Header()
	if lastBlockHeader == nil {
		return errors.New("current block header not found")
	}
	expectedBlockNum, err := s.MessageCountToBlockNumber(pos)
	if err != nil {
		return err
	}
	if lastBlockHeader.Number.Int64() != expectedBlockNum {
		return fmt.Errorf("%w: block production not caught up: last block number %v but expected %v", ErrRetrySequencer, lastBlockHeader.Number, expectedBlockNum)
	}
	statedb, err := s.bc.StateAt(lastBlockHeader.Root)
	if err != nil {
		return err
	}

	var delayedMessagesRead uint64
	if pos > 0 {
		lastMsg, err := s.GetMessage(pos - 1)
		if err != nil {
			return err
		}
		delayedMessagesRead = lastMsg.DelayedMessagesRead
	}

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
		return err
	}

	if len(receipts) == 0 {
		return nil
	}

	allTxsErrored := true
	for _, err := range hooks.TxErrors {
		if err == nil {
			allTxsErrored = false
			break
		}
	}
	if allTxsErrored {
		return nil
	}

	msg, err := messageFromTxes(header, txes, hooks.TxErrors)
	if err != nil {
		return err
	}

	msgWithMeta := arbstate.MessageWithMetadata{
		Message:             msg,
		DelayedMessagesRead: delayedMessagesRead,
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
			return err
		}
	}

	// Only write the block after we've written the messages, so if the node dies in the middle of this,
	// it will naturally recover on startup by regenerating the missing block.
	var logs []*types.Log
	for _, receipt := range receipts {
		logs = append(logs, receipt.Logs...)
	}
	status, err := s.bc.WriteBlockAndSetHeadWithTime(block, receipts, logs, statedb, true, time.Since(startTime))
	if err != nil {
		return err
	}
	if status == core.SideStatTy {
		return errors.New("geth rejected block as non-canonical")
	}

	if s.validator != nil {
		s.validator.NewBlock(block, lastBlockHeader, msgWithMeta)
	}

	return nil
}

func (s *TransactionStreamer) SequenceDelayedMessages(ctx context.Context, messages []*arbos.L1IncomingMessage, firstDelayedSeqNum uint64) error {
	s.insertionMutex.Lock()
	defer s.insertionMutex.Unlock()

	pos, err := s.GetMessageCount()
	if err != nil {
		return err
	}

	var delayedMessagesRead uint64
	if pos > 0 {
		lastMsg, err := s.GetMessage(pos - 1)
		if err != nil {
			return err
		}
		delayedMessagesRead = lastMsg.DelayedMessagesRead
	}

	if delayedMessagesRead != firstDelayedSeqNum {
		return fmt.Errorf("attempted to insert delayed messages at incorrect position got %d expected %d", firstDelayedSeqNum, delayedMessagesRead)
	}

	messagesWithMeta := make([]arbstate.MessageWithMetadata, 0, len(messages))
	for i, message := range messages {
		newMessage := arbstate.MessageWithMetadata{
			Message:             message,
			DelayedMessagesRead: delayedMessagesRead + uint64(i) + 1,
		}
		messagesWithMeta = append(messagesWithMeta, newMessage)
		if s.coordinator != nil {
			if err := s.coordinator.SequencingMessage(pos+arbutil.MessageIndex(i), &newMessage); err != nil {
				return err
			}
		}
	}

	log.Info("TransactionStreamer: Added DelayedMessages", "pos", pos, "length", len(messages))
	err = s.writeMessages(pos, messagesWithMeta, nil)
	if err != nil {
		return err
	}

	for i, msg := range messagesWithMeta {
		if s.broadcastServer != nil {
			if err := s.broadcastServer.BroadcastSingle(msg, pos+arbutil.MessageIndex(i)); err != nil {
				return err
			}
		}
	}

	expectedBlockNum, err := s.MessageCountToBlockNumber(pos)
	if err != nil {
		return err
	}

	// If we were already caught up to the latest message, ensure we produce blocks for the delayed messages.
	if s.bc.CurrentBlock().Header().Number.Int64() >= expectedBlockNum {
		err = s.createBlocks(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *TransactionStreamer) GetGenesisBlockNumber() (uint64, error) {
	return s.bc.Config().ArbitrumChainParams.GenesisBlockNum, nil
}

func (s *TransactionStreamer) BlockNumberToMessageCount(blockNum uint64) (arbutil.MessageIndex, error) {
	genesis, err := s.GetGenesisBlockNumber()
	if err != nil {
		return 0, err
	}
	return arbutil.BlockNumberToMessageCount(blockNum, genesis), nil
}

func (s *TransactionStreamer) MessageCountToBlockNumber(messageNum arbutil.MessageIndex) (int64, error) {
	genesis, err := s.GetGenesisBlockNumber()
	if err != nil {
		return 0, err
	}
	return arbutil.MessageCountToBlockNumber(messageNum, genesis), nil
}

// Pauses reorgs until a matching call to ResumeReorgs (may be called concurrently)
func (s *TransactionStreamer) PauseReorgs() {
	s.reorgMutex.RLock()
}

func (s *TransactionStreamer) ResumeReorgs() {
	s.reorgMutex.RUnlock()
}

// The mutex must be held, and pos must be the latest message count.
// `batch` may be nil, which initializes a new batch. The batch is closed out in this function.
func (s *TransactionStreamer) writeMessages(pos arbutil.MessageIndex, messages []arbstate.MessageWithMetadata, batch ethdb.Batch) error {
	if batch == nil {
		batch = s.db.NewBatch()
	}
	for i, msg := range messages {
		key := dbKey(messagePrefix, uint64(pos)+uint64(i))
		msgBytes, err := rlp.EncodeToBytes(msg)
		if err != nil {
			return err
		}
		err = batch.Put(key, msgBytes)
		if err != nil {
			return err
		}
	}
	newCount, err := rlp.EncodeToBytes(uint64(pos) + uint64(len(messages)))
	if err != nil {
		return err
	}
	err = batch.Put(messageCountKey, newCount)
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

// Produce and record blocks for all available messages
func (s *TransactionStreamer) createBlocks(ctx context.Context) error {
	s.createBlocksMutex.Lock()
	defer s.createBlocksMutex.Unlock()
	s.reorgMutex.RLock()
	defer s.reorgMutex.RUnlock()

	msgCount, err := s.GetMessageCount()
	if err != nil {
		return err
	}
	initialLastBlock := s.bc.CurrentBlock()
	err = s.bc.RecoverState(initialLastBlock)
	if err != nil {
		return fmt.Errorf("failed to recover state: %w", err)
	}
	lastBlockHeader := initialLastBlock.Header()
	if lastBlockHeader == nil {
		return errors.New("current block header not found")
	}
	pos, err := s.BlockNumberToMessageCount(lastBlockHeader.Number.Uint64())
	if err != nil {
		return err
	}

	var statedb *state.StateDB
	defer func() {
		if statedb != nil {
			// This can safely be called even if the prefetcher hasn't been started,
			// as it checks if a prefetcher is present before stopping it.
			statedb.StopPrefetcher()
		}
	}()

	batchFetcher := func(batchNum uint64) ([]byte, error) {
		return s.inboxReader.GetSequencerMessageBytes(ctx, batchNum)
	}

	for pos < msgCount {

		statedb, err = s.bc.StateAt(lastBlockHeader.Root)
		if err != nil {
			return err
		}

		if atomic.LoadUint32(&s.reorgPending) > 0 {
			// stop block creation as we need to reorg
			break
		}
		if ctx.Err() != nil {
			// the context is done, shut down
			// nolint:nilerr
			return nil
		}

		statedb.StartPrefetcher("TransactionStreamer")

		msg, err := s.GetMessage(pos)
		if err != nil {
			return err
		}

		startTime := time.Now()
		block, receipts, err := arbos.ProduceBlock(
			msg.Message,
			msg.DelayedMessagesRead,
			lastBlockHeader,
			statedb,
			s.bc,
			s.bc.Config(),
			batchFetcher,
		)
		if err != nil {
			return err
		}

		// ProduceBlock advances one message
		pos++

		var logs []*types.Log
		for _, receipt := range receipts {
			logs = append(logs, receipt.Logs...)
		}
		status, err := s.bc.WriteBlockAndSetHeadWithTime(block, receipts, logs, statedb, true, time.Since(startTime))
		if err != nil {
			return err
		}
		if status == core.SideStatTy {
			return errors.New("geth rejected block as non-canonical")
		}

		if s.validator != nil {
			s.validator.NewBlock(block, lastBlockHeader, *msg)
		}

		s.latestBlockAndMessageMutex.Lock()
		s.latestBlock = block
		s.latestMessage = msg.Message
		s.latestBlockAndMessageMutex.Unlock()
		select {
		case s.newBlockNotifier <- struct{}{}:
		default:
		}

		lastBlockHeader = block.Header()
	}

	return nil
}

func (s *TransactionStreamer) Start(ctxIn context.Context) {
	s.StopWaiter.Start(ctxIn, s)
	s.LaunchThread(func(ctx context.Context) {
		for {
			err := s.createBlocks(ctx)
			if err != nil && !errors.Is(err, context.Canceled) {
				log.Error("error creating blocks", "err", err.Error())
			}
			timer := time.NewTimer(10 * time.Second)
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
	s.LaunchThread(func(ctx context.Context) {
		var lastBlock *types.Block
		for {
			select {
			case <-s.newBlockNotifier:
			case <-ctx.Done():
				return
			}
			s.latestBlockAndMessageMutex.Lock()
			block := s.latestBlock
			message := s.latestMessage
			s.latestBlockAndMessageMutex.Unlock()
			if block != lastBlock && block != nil && message != nil {
				log.Info(
					"created block",
					"l2Block", block.Number(),
					"l2BlockHash", block.Hash(),
					"l1Block", message.Header.BlockNumber,
					"l1Timestamp", time.Unix(int64(message.Header.Timestamp), 0),
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
