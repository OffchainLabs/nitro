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

	"github.com/ethereum/go-ethereum/arbitrum"
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

// Produces blocks from a node's L1 messages, storing the results in the blockchain and recording their positions
// The streamer is notified when there's new batches to process
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

func (s *TransactionStreamer) SetBlockValidator(validator *validator.BlockValidator) {
	s.validator = validator
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
	targetBlock := s.bc.GetBlockByNumber(count)
	if targetBlock == nil {
		return errors.New("reorg target block not found")
	}

	err := s.bc.ReorgToOldBlock(targetBlock)
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
			var dbMessageParsed arbstate.MessageWithMetadata
			err := rlp.DecodeBytes(haveMessage, &dbMessageParsed)
			if err != nil {
				log.Warn("TransactionStreamer: Reorg detected! (failed parsing db message)", "pos", pos, "err", err)
			} else {
				log.Warn("TransactionStreamer: Reorg detected!", "pos", pos, "got-read", messages[0].DelayedMessagesRead, "got-header", messages[0].Message.Header, "db-read", dbMessageParsed.DelayedMessagesRead, "db-header", dbMessageParsed.Message.Header)
			}
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
		return fmt.Errorf("attempted to insert delayed messages at incorrect position got %d expected %d", firstDelayedSeqNum, delayedMessagesRead)
	}

	messagesWithMeta := make([]arbstate.MessageWithMetadata, 0, len(messages))
	for i, message := range messages {
		messagesWithMeta = append(messagesWithMeta, arbstate.MessageWithMetadata{
			Message:             message,
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

// Produce and record blocks for all available messages
func (s *TransactionStreamer) createBlocks(ctx context.Context) error {
	s.reorgMutex.Lock()
	defer s.reorgMutex.Unlock()

	msgCount, err := s.GetMessageCount()
	if err != nil {
		return err
	}
	lastBlockHeader := s.bc.CurrentHeader()
	if lastBlockHeader == nil {
		return errors.New("current block header not found")
	}
	pos := lastBlockHeader.Number.Uint64()
	statedb, err := s.bc.StateAt(lastBlockHeader.Root)
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

		block, receipts := arbos.ProduceBlock(
			msg.Message,
			msg.DelayedMessagesRead,
			lastBlockHeader,
			statedb,
			s.bc,
		)

		// ProduceBlock advances one message
		pos++

		var logs []*types.Log
		for _, receipt := range receipts {
			logs = append(logs, receipt.Logs...)
		}
		status, err := s.bc.WriteBlockWithState(block, receipts, logs, statedb, true)
		if err != nil {
			return err
		}
		if status == core.SideStatTy {
			return errors.New("geth rejected block as non-canonical")
		}

		if s.validator != nil {
			recordingdb, chaincontext, recordingKV, err := arbitrum.PrepareRecording(s.bc, lastBlockHeader)
			if err != nil {
				return err
			}

			block, _ = arbos.ProduceBlock(
				msg.Message,
				msg.DelayedMessagesRead,
				lastBlockHeader,
				recordingdb,
				chaincontext,
			)
			preimages, err := arbitrum.PreimagesFromRecording(chaincontext, recordingKV)
			if err != nil {
				return fmt.Errorf("failed getting records: %w", err)
			}
			s.validator.NewBlock(block, lastBlockHeader, preimages)
		}

		lastBlockHeader = block.Header()
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
