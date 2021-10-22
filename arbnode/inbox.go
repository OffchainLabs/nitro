//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbnode

import (
	"bytes"
	"encoding/binary"
	"errors"
	"sync"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/offchainlabs/arbstate/arbos"
	"github.com/offchainlabs/arbstate/arbstate"
)

type InboxState struct {
	db    ethdb.Database
	bc    *core.BlockChain
	mutex sync.Mutex
}

func NewInboxState(db ethdb.Database, bc *core.BlockChain) (*InboxState, error) {
	inbox := &InboxState{
		db: rawdb.NewTable(db, arbitrumPrefix),
		bc: bc,
	}
	err := inbox.cleanupInconsistentState()
	return inbox, err
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

func (s *InboxState) cleanupInconsistentState() error {
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

var blockForMessageNotFoundErr = errors.New("block for message count not found")

func (s *InboxState) LookupBlockNumByMessageCount(count uint64, roundUp bool) (uint64, uint64, error) {
	minKey := uint64ToBytes(count)
	iter := s.db.NewIterator(messageCountToBlockPrefix, minKey)
	defer iter.Release()
	if iter.Error() != nil {
		return 0, 0, iter.Error()
	}
	key := iter.Key()
	if len(key) == 0 {
		return 0, 0, blockForMessageNotFoundErr
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

func (s *InboxState) ReorgTo(count uint64) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.reorgToWithLock(count)
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

func (s *InboxState) reorgToWithLock(count uint64) error {
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

	batch := s.db.NewBatch()
	err = deleteStartingAt(s.db, batch, messageCountToBlockPrefix, uint64ToBytes(count+1))
	if err != nil {
		batch.Reset()
		return err
	}
	err = deleteStartingAt(s.db, batch, messageCountToMessagePrefix, uint64ToBytes(count))
	if err != nil {
		batch.Reset()
		return err
	}
	countBytes, err := rlp.EncodeToBytes(count)
	if err != nil {
		batch.Reset()
		return err
	}
	err = batch.Put(messageCountKey, countBytes)
	if err != nil {
		batch.Reset()
		return err
	}

	return batch.Write()
}

func dbKey(prefix []byte, pos uint64) []byte {
	var key []byte
	key = append(key, prefix...)
	key = append(key, uint64ToBytes(pos)...)
	return key
}

func (s *InboxState) writeBlock(blockBuilder *arbos.BlockBuilder, lastMessage uint64, delayedMessageCount uint64) error {
	messageCount := lastMessage + 1
	block, receipts, statedb := blockBuilder.ConstructBlock(delayedMessageCount)
	if len(block.Transactions()) != len(receipts) {
		return errors.New("mismatch between number of transactions and number of receipts")
	}
	key := dbKey(messageCountToBlockPrefix, messageCount)
	blockNumBytes, err := rlp.EncodeToBytes(block.NumberU64())
	if err != nil {
		return err
	}
	err = s.db.Put(key, blockNumBytes)
	if err != nil {
		return err
	}
	var logs []*types.Log
	for _, receipt := range receipts {
		logs = append(logs, receipt.Logs...)
	}
	status, err := s.bc.WriteBlockWithState(block, receipts, logs, statedb, true)
	if status == core.SideStatTy {
		return errors.New("geth rejected block as non-canonical")
	}
	return err
}

func (s *InboxState) GetMessage(seqNum uint64) (arbstate.MessageWithMetadata, error) {
	key := dbKey(messageCountToMessagePrefix, seqNum)
	data, err := s.db.Get(key)
	if err != nil {
		return arbstate.MessageWithMetadata{}, err
	}
	var message arbstate.MessageWithMetadata
	err = rlp.DecodeBytes(data, &message)
	return message, err
}

// As a special case, if pos is the max uint64, the message is added after the last message
func (s *InboxState) AddMessages(pos uint64, force bool, messages []arbstate.MessageWithMetadata) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if pos == ^uint64(0) {
		posBytes, err := s.db.Get(messageCountKey)
		if err != nil {
			return err
		}
		err = rlp.DecodeBytes(posBytes, &pos)
		if err != nil {
			return err
		}
	}

	if pos > 0 {
		key := dbKey(messageCountToMessagePrefix, pos-1)
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
		key := dbKey(messageCountToMessagePrefix, pos)
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
			err := s.reorgToWithLock(pos)
			if err != nil {
				return err
			}
		} else {
			return errors.New("reorg required but not allowed")
		}
	}
	if len(messages) == 0 {
		return nil
	}

	// We're now ready to add the new messages
	lastBlockNumber, startPos, err := s.LookupBlockNumByMessageCount(pos, true)
	if err == nil && startPos != pos {
		return errors.New("found block after insertion position")
	}
	if errors.Is(err, blockForMessageNotFoundErr) {
		// We couldn't find a block at or after the target position.
		// Clear the error and search backwards for the last message count with a block.
		err = nil
		startPos = pos
		for startPos > 0 {
			startPos--
			key := dbKey(messageCountToBlockPrefix, startPos)
			hasKey, err := s.db.Has(key)
			if err != nil {
				return err
			}
			if !hasKey {
				continue
			}
			blockNumBytes, err := s.db.Get(key)
			if err != nil {
				return err
			}
			err = rlp.DecodeBytes(blockNumBytes, &lastBlockNumber)
			if err != nil {
				return err
			}
			break
		}
	}
	if err != nil {
		return err
	}
	if startPos > pos {
		return errors.New("found block for future message count")
	}

	// Write any new messages to the database
	batch := s.db.NewBatch()
	for i, msg := range messages {
		key := dbKey(messageCountToMessagePrefix, pos+uint64(i))
		msgBytes, err := rlp.EncodeToBytes(msg)
		if err != nil {
			return err
		}
		err = batch.Put(key, msgBytes)
		if err != nil {
			batch.Reset()
			return err
		}
		err = batch.Put(messageCountKey, uint64ToBytes(pos+uint64(i)+1))
		if err != nil {
			batch.Reset()
			return err
		}
	}
	err = batch.Write()
	if err != nil {
		return err
	}

	// Fill in gap between startPos and pos
	replayMessages := pos - startPos
	messages = append(make([]arbstate.MessageWithMetadata, replayMessages), messages...)
	for i := uint64(0); i < replayMessages; i++ {
		messages[i], err = s.GetMessage(startPos + i)
		if err != nil {
			return err
		}
	}
	pos = startPos

	// Build blocks from the messages
	lastBlockHeader := s.bc.GetHeaderByNumber(lastBlockNumber)
	if lastBlockHeader == nil {
		return errors.New("last block header not found")
	}
	statedb, err := s.bc.StateAt(lastBlockHeader.Root)
	if err != nil {
		return err
	}
	blockBuilder := arbos.NewBlockBuilder(statedb, lastBlockHeader, s.bc)
	for i, msg := range messages {
		segment, err := arbos.IncomingMessageToSegment(msg.Message, arbos.ChainConfig.ChainID)
		if err != nil {
			// If we've failed to parse the incoming message, make a new block and move on to the next message
			err = s.writeBlock(blockBuilder, pos+uint64(i), msg.DelayedMessagesRead)
			if err != nil {
				return err
			}
			// Skip this invalid message
			continue
		}

		// If we shouldn't put the next message in the current block,
		// make a new block before adding the next message.
		if !blockBuilder.ShouldAddMessage(segment) {
			err = s.writeBlock(blockBuilder, pos+uint64(i), msg.DelayedMessagesRead)
			if err != nil {
				return err
			}
			// Notice we fall through here to the AddMessage call
		}

		// Add the message to the block
		blockBuilder.AddMessage(segment)

		if msg.MustEndBlock {
			// If this message must end the block, end it now
			err = s.writeBlock(blockBuilder, pos+uint64(i), msg.DelayedMessagesRead)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
