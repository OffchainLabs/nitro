//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

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
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/offchainlabs/arbstate/arbos"
	"github.com/offchainlabs/arbstate/arbstate"
	"github.com/offchainlabs/arbstate/validator"
)

type TransactionStreamer struct {
	db ethdb.Database
	bc *core.BlockChain

	insertionMutex     sync.Mutex // cannot be acquired while reorgMutex is held
	reorgMutex         sync.Mutex
	reorgPending       uint32 // atomic, indicates whether the reorgMutex is attempting to be acquired
	newMessageNotifier chan struct{}

	validator *validator.BlockValidator
}

func NewTransactionStreamer(db ethdb.Database, bc *core.BlockChain) (*TransactionStreamer, error) {
	inbox := &TransactionStreamer{
		db:                 rawdb.NewTable(db, arbitrumPrefix),
		bc:                 bc,
		newMessageNotifier: make(chan struct{}, 1),
	}
	return inbox, nil
}

// Encodes a uint64 as bytes in a lexically sortable manner for database iteration.
// Generally this is only used for database keys, which need sorted.
// A shorter RLP encoding is usually used for database values.
func uint64ToBytes(x uint64) []byte {
	data := make([]byte, 8)
	binary.BigEndian.PutUint64(data, x)
	return data
}

func bytesToUint64(b []byte) (uint64, error) {
	if len(b) != 8 {
		return 0, errors.New("decoding wrong length bytes to uint64")
	}
	return binary.BigEndian.Uint64(b), nil
}

func (s *TransactionStreamer) SetBlockValidator(validator *validator.BlockValidator) {
	s.validator = validator
}

func (s *TransactionStreamer) cleanupInconsistentState() error {
	// Insert a messageCountToBlockPrefix entry for the genesis block
	key := dbKey(messageCountToBlockPrefix, 0)
	blockNumBytes, err := rlp.EncodeToBytes(uint64(0))
	if err != nil {
		return err
	}
	err = s.db.Put(key, blockNumBytes)
	if err != nil {
		return err
	}
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

var errBlockForMessageNotFound = errors.New("block for message count not found")

func (s *TransactionStreamer) LookupBlockNumByMessageCount(count uint64, roundUp bool) (uint64, uint64, error) {
	minKey := uint64ToBytes(count)
	iter := s.db.NewIterator(messageCountToBlockPrefix, minKey)
	defer iter.Release()
	if iter.Error() != nil {
		return 0, 0, iter.Error()
	}
	key := iter.Key()
	if len(key) == 0 {
		return 0, 0, errBlockForMessageNotFound
	}
	if !bytes.HasPrefix(key, messageCountToBlockPrefix) {
		return 0, 0, errors.New("iterated key missing prefix")
	}
	key = key[len(messageCountToBlockPrefix):]
	actualCount, err := bytesToUint64(key)
	if err != nil {
		return 0, 0, err
	}
	var block uint64
	err = rlp.DecodeBytes(iter.Value(), &block)
	if err != nil {
		return 0, 0, err
	}
	if !roundUp && actualCount > count && block > 0 {
		block--
	}
	return block, actualCount, nil
}

func (s *TransactionStreamer) ReorgTo(count uint64) error {
	return s.ReorgToAndEndBatch(s.db.NewBatch(), count)
}

func (s *TransactionStreamer) ReorgToAndEndBatch(batch ethdb.Batch, count uint64) error {
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
	for {
		if iter.Error() != nil {
			return iter.Error()
		}
		key := iter.Key()
		if len(key) == 0 {
			break
		}
		err := batch.Delete(key)
		if err != nil {
			return err
		}
		if !iter.Next() {
			break
		}
	}
	return nil
}

func (s *TransactionStreamer) reorgToInternal(batch ethdb.Batch, count uint64) error {
	atomic.AddUint32(&s.reorgPending, 1)
	s.reorgMutex.Lock()
	defer s.reorgMutex.Unlock()
	atomic.AddUint32(&s.reorgPending, ^uint32(0)) // decrement
	targetBlockNumber, _, err := s.LookupBlockNumByMessageCount(count, false)
	if err != nil {
		return err
	}
	targetBlock := s.bc.GetBlockByNumber(targetBlockNumber)
	if targetBlock == nil {
		return errors.New("message count block not found")
	}

	err = s.bc.ReorgToOldBlock(targetBlock)
	if err != nil {
		return err
	}

	err = deleteStartingAt(s.db, batch, messageCountToBlockPrefix, uint64ToBytes(count+1))
	if err != nil {
		return err
	}
	err = deleteStartingAt(s.db, batch, messagePrefix, uint64ToBytes(count))
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
	key = append(key, uint64ToBytes(pos)...)
	return key
}

func (s *TransactionStreamer) writeBlock(blockBuilder *arbos.BlockBuilder, lastMessage uint64, delayedMessageCount uint64) (*types.Block, error) {
	messageCount := lastMessage + 1
	block, receipts, statedb := blockBuilder.ConstructBlock(delayedMessageCount)
	if len(block.Transactions()) != len(receipts) {
		return nil, errors.New("mismatch between number of transactions and number of receipts")
	}
	key := dbKey(messageCountToBlockPrefix, messageCount)
	blockNumBytes, err := rlp.EncodeToBytes(block.NumberU64())
	if err != nil {
		return nil, err
	}
	err = s.db.Put(key, blockNumBytes)
	if err != nil {
		return nil, err
	}
	var logs []*types.Log
	for _, receipt := range receipts {
		logs = append(logs, receipt.Logs...)
	}
	status, err := s.bc.WriteBlockWithState(block, receipts, logs, statedb, true)
	if err != nil {
		return nil, err
	}
	if status == core.SideStatTy {
		return nil, errors.New("geth rejected block as non-canonical")
	}
	if s.validator != nil {
		preimages, startPos, err := arbstate.GetRecordsFromBuilder(blockBuilder)
		if err != nil {
			return nil, fmt.Errorf("failed getting records: %s", err)
		}
		s.validator.NewBlock(block, preimages, startPos, lastMessage)
	}
	return block, nil
}

func (s *TransactionStreamer) createBlockBuilder(prevHash common.Hash, startPos uint64) (*arbos.BlockBuilder, error) {
	return arbstate.CreateBlockBuilder(s.bc, prevHash, startPos, (s.validator != nil))
}

// Note: if changed to acquire the mutex, some internal users may need to be updated to a non-locking version.
func (s *TransactionStreamer) GetMessage(seqNum uint64) (arbstate.MessageWithMetadata, error) {
	key := dbKey(messagePrefix, seqNum)
	data, err := s.db.Get(key)
	if err != nil {
		return arbstate.MessageWithMetadata{}, err
	}
	var message arbstate.MessageWithMetadata
	err = rlp.DecodeBytes(data, &message)
	return message, err
}

// Note: if changed to acquire the mutex, some internal users may need to be updated to a non-locking version.
func (s *TransactionStreamer) GetMessageCount() (uint64, error) {
	posBytes, err := s.db.Get(messageCountKey)
	if err != nil {
		return 0, err
	}
	var pos uint64
	err = rlp.DecodeBytes(posBytes, &pos)
	if err != nil {
		return 0, err
	}
	return pos, nil
}

func (s *TransactionStreamer) AddMessages(pos uint64, force bool, messages []arbstate.MessageWithMetadata) error {
	return s.AddMessagesAndEndBatch(pos, force, messages, nil)
}

func (s *TransactionStreamer) AddMessagesAndEndBatch(pos uint64, force bool, messages []arbstate.MessageWithMetadata, batch ethdb.Batch) error {
	s.insertionMutex.Lock()
	defer s.insertionMutex.Unlock()

	if pos > 0 {
		key := dbKey(messagePrefix, pos-1)
		hasPrev, err := s.db.Has(key)
		if err != nil {
			return err
		}
		if !hasPrev {
			return errors.New("missing previous message")
		}
	}

	reorg := false
	// Skip any messages already in the database
	for {
		if len(messages) == 0 {
			break
		}
		key := dbKey(messagePrefix, pos)
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
		wantMessage, err := rlp.EncodeToBytes(messages[0])
		if err != nil {
			return err
		}
		if bytes.Equal(haveMessage, wantMessage) {
			// This message is a duplicate, skip it
			messages = messages[1:]
			pos++
		} else {
			reorg = true
			break
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

func (s *TransactionStreamer) SequenceMessages(messages []*arbos.L1IncomingMessage) error {
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

	messagesWithMeta := make([]arbstate.MessageWithMetadata, 0, len(messages))
	for _, message := range messages {
		messagesWithMeta = append(messagesWithMeta, arbstate.MessageWithMetadata{
			Message:             message,
			MustEndBlock:        true,
			DelayedMessagesRead: delayedMessagesRead,
		})
	}

	return s.writeMessages(pos, messagesWithMeta, nil)
}

func (s *TransactionStreamer) SequenceDelayedMessages(messages []*arbos.L1IncomingMessage, firstDelayedSeqNum uint64) error {
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
		return errors.New("attempted to insert delayed messages at incorrect position")
	}

	messagesWithMeta := make([]arbstate.MessageWithMetadata, 0, len(messages))
	for i, message := range messages {
		messagesWithMeta = append(messagesWithMeta, arbstate.MessageWithMetadata{
			Message:             message,
			MustEndBlock:        i == len(messages)-1,
			DelayedMessagesRead: delayedMessagesRead + uint64(i) + 1,
		})
	}
	log.Info("TransactionStreamer: Added DelayedMessages", "pos", pos, "length", len(messages))
	return s.writeMessages(pos, messagesWithMeta, nil)
}

// The mutex must be held, and pos must be the latest message count.
// `batch` may be nil, which initializes a new batch. The batch is closed out in this function.
func (s *TransactionStreamer) writeMessages(pos uint64, messages []arbstate.MessageWithMetadata, batch ethdb.Batch) error {
	if batch == nil {
		batch = s.db.NewBatch()
	}
	for i, msg := range messages {
		key := dbKey(messagePrefix, pos+uint64(i))
		msgBytes, err := rlp.EncodeToBytes(msg)
		if err != nil {
			return err
		}
		err = batch.Put(key, msgBytes)
		if err != nil {
			return err
		}
	}
	newCount, err := rlp.EncodeToBytes(pos + uint64(len(messages)))
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

// Only safe to call from createBlocks and while the reorg mutex is held
func (s *TransactionStreamer) getLastBlockPosition() (uint64, uint64, error) {
	count, err := s.GetMessageCount()
	if err != nil {
		return 0, 0, err
	}

	lastBlockNumber, startPos, err := s.LookupBlockNumByMessageCount(count, true)
	if err == nil && startPos != count {
		return 0, 0, errors.New("found block after last message")
	}
	if errors.Is(err, errBlockForMessageNotFound) {
		// We couldn't find a block at or after the target position.
		// Clear the error and search backwards for the last message count with a block.
		err = nil
		startPos = count
		for startPos > 0 {
			startPos--
			key := dbKey(messageCountToBlockPrefix, startPos)
			hasKey, err := s.db.Has(key)
			if err != nil {
				return 0, 0, err
			}
			if !hasKey {
				continue
			}
			blockNumBytes, err := s.db.Get(key)
			if err != nil {
				return 0, 0, err
			}
			err = rlp.DecodeBytes(blockNumBytes, &lastBlockNumber)
			if err != nil {
				return 0, 0, err
			}
			break
		}
	}
	if err != nil {
		return 0, 0, err
	}
	if startPos > count {
		return 0, 0, errors.New("found block for future message count")
	}

	return startPos, lastBlockNumber, nil
}

func (s *TransactionStreamer) createBlocks(ctx context.Context) error {
	s.reorgMutex.Lock()
	defer s.reorgMutex.Unlock()

	pos, lastBlockNumber, err := s.getLastBlockPosition()
	if err != nil {
		return err
	}
	msgCount, err := s.GetMessageCount()
	if err != nil {
		return err
	}

	lastBlockHeader := s.bc.GetHeaderByNumber(lastBlockNumber)
	if lastBlockHeader == nil {
		return errors.New("last block header not found")
	}
	blockBuilder, err := s.createBlockBuilder(lastBlockHeader.Hash(), pos)
	if err != nil {
		return err
	}
	for pos < msgCount {
		if atomic.LoadUint32(&s.reorgPending) > 0 {
			// stop block creation as we need to reorg
			return nil
		}
		if ctx.Err() != nil {
			// the context is done, shut down
			// nolint:nilerr
			return nil
		}

		msg, err := s.GetMessage(pos)
		if err != nil {
			return err
		}
		segment, err := arbos.IncomingMessageToSegment(msg.Message, arbos.ChainConfig.ChainID)
		if err != nil {
			// If we've failed to parse the incoming message, make a new block and move on to the next message
			block, err := s.writeBlock(blockBuilder, pos, msg.DelayedMessagesRead)
			if err != nil {
				return err
			}
			log.Warn("TransactionStreamer: closeblock and skipping message", "pos", pos)
			// Skip this invalid message
			blockBuilder, err = s.createBlockBuilder(block.Header().Hash(), pos+1)
			if err != nil {
				return err
			}
			pos++
			continue
		}

		// If we shouldn't put the next message in the current block,
		// make a new block before adding the next message.
		if !blockBuilder.ShouldAddMessage(segment) {
			block, err := s.writeBlock(blockBuilder, pos-1, msg.DelayedMessagesRead)
			if err != nil {
				return err
			}
			log.Info("TransactionStreamer: closed block - should not add message", "pos", pos)
			blockBuilder, err = s.createBlockBuilder(block.Header().Hash(), pos)
			if err != nil {
				return err
			}
			// Notice we fall through here to the AddMessage call
		}

		// Add the message to the block
		blockBuilder.AddMessage(segment)

		if msg.MustEndBlock {
			// If this message must end the block, end it now
			block, err := s.writeBlock(blockBuilder, pos, msg.DelayedMessagesRead)
			if err != nil {
				return err
			}
			log.Info("TransactionStreamer: closed block - must end block", "pos", pos)
			blockBuilder, err = s.createBlockBuilder(block.Header().Hash(), pos+1)
			if err != nil {
				return err
			}
		}
		pos++
	}

	return nil
}

func (s *TransactionStreamer) Initialize() error {
	return s.cleanupInconsistentState()
}

func (s *TransactionStreamer) Start(ctx context.Context) {
	go (func() {
		for {
			err := s.createBlocks(ctx)
			if err != nil && !errors.Is(err, context.Canceled) {
				log.Error("error creating blocks", "err", err.Error())
			}
			select {
			case <-ctx.Done():
				return
			case <-s.newMessageNotifier:
			case <-time.After(10 * time.Second):
			}
		}
	})()
}
