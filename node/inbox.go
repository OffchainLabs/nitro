//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package node

import (
	"bytes"
	"errors"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/offchainlabs/arbstate"
	"github.com/offchainlabs/arbstate/arbos"
)

type InboxState struct {
	db    ethdb.Database
	bc    *core.BlockChain
	mutex sync.Mutex
}

func NewInboxState(db ethdb.Database, bc *core.BlockChain) (*InboxState, error) {
	inbox := &InboxState{
		db: db,
		bc: bc,
	}
	err := inbox.cleanupInconsistentState()
	return inbox, err
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
	// TODO remove trailing messageCountToMessage and messageCountToBlockPrefix entries
	return nil
}

func (s *InboxState) LookupBlockNumByMessageCount(count uint64, roundUp bool) (uint64, error) {
	minKey, err := rlp.EncodeToBytes(count)
	if err != nil {
		return 0, err
	}
	iter := s.db.NewIterator(messageCountToBlockPrefix, minKey)
	defer iter.Release()
	if iter.Error() != nil {
		return 0, iter.Error()
	}
	key := iter.Key()
	if len(key) == 0 {
		return 0, errors.New("block for message count not found")
	}
	if !bytes.HasPrefix(key, messageCountToBlockPrefix) {
		return 0, errors.New("iterated key missing prefix")
	}
	key = key[len(messageCountToBlockPrefix):]
	var actualCount uint64
	err = rlp.DecodeBytes(key, &actualCount)
	if err != nil {
		return 0, err
	}
	var block uint64
	err = rlp.DecodeBytes(iter.Value(), &block)
	if err != nil {
		return 0, err
	}
	if !roundUp && actualCount < count && block > 0 {
		block--
	}
	return block, nil
}

func (s *InboxState) ReorgTo(count uint64) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.reorgToWithLock(count)
}

func deleteStartingAt(db ethdb.Database, prefix []byte, minKey []byte) error {
	iter := db.NewIterator(messageCountToMessagePrefix, minKey)
	defer iter.Release()
	for {
		if iter.Error() != nil {
			return iter.Error()
		}
		key := iter.Key()
		if len(key) == 0 {
			break
		}
		err := db.Delete(key)
		if err != nil {
			return err
		}
		if iter.Next() {
			break
		}
	}
	return nil
}

func (s *InboxState) reorgToWithLock(count uint64) error {
	targetBlockNumber, err := s.LookupBlockNumByMessageCount(count, false)
	if err != nil {
		return err
	}
	targetBlock := s.bc.GetBlockByNumber(targetBlockNumber)
	if targetBlock == nil {
		return errors.New("message count block not found")
	}
	headBlock := s.bc.CurrentBlock()
	if headBlock == nil {
		return errors.New("latest block not found")
	}

	err = s.bc.Reorg(headBlock, targetBlock)
	if err != nil {
		return err
	}

	minKey, err := rlp.EncodeToBytes(count)
	if err != nil {
		return err
	}
	err = deleteStartingAt(s.db, messageCountToBlockPrefix, minKey)
	if err != nil {
		return err
	}
	err = deleteStartingAt(s.db, messageCountToMessagePrefix, minKey)
	if err != nil {
		return err
	}

	return nil
}

func dbKey(prefix []byte, pos uint64) []byte {
	posBytes, err := rlp.EncodeToBytes(pos)
	if err != nil {
		panic(fmt.Sprintf("Failed to rlp encode uint64: %s", err.Error()))
	}
	key := messageCountToMessagePrefix
	key = append(key, posBytes...)
	return key
}

func serializeMsg(message arbstate.MessageWithMetadata) ([]byte, error) {
	data, err := message.Message.Serialize()
	if err != nil {
		return nil, err
	}
	delayedCount, err := rlp.EncodeToBytes(message.DelayedMessagesRead)
	if err != nil {
		return nil, err
	}
	data = append(data, delayedCount...)
	var mustEndBlock uint8
	if message.MustEndBlock {
		mustEndBlock = 1
	}
	data = append(data, mustEndBlock)
	return data, nil
}

func (s *InboxState) writeBlock(blockBuilder *arbos.BlockBuilder, messageCount uint64, delayedMessageCount uint64) error {
	block, receipts, statedb := blockBuilder.ConstructBlock(delayedMessageCount)
	key := dbKey(messageCountToBlockPrefix, messageCount)
	blockNumBytes, err := rlp.EncodeToBytes(block.Number().Uint64())
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

func (s *InboxState) AddMessages(pos uint64, force bool, messages []arbstate.MessageWithMetadata) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	reorg := false
	// Skip any messages already in the database
	for {
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
		if len(messages) > 0 {
			wantMessage, err := serializeMsg(messages[0])
			if err != nil {
				return err
			}
			if !bytes.Equal(haveMessage, wantMessage) {
				reorg = true
				break
			}
		} else if hasMessage {
			reorg = true
			break
		}
	}

	if reorg {
		if force {
			s.reorgToWithLock(pos)
		} else {
			return errors.New("reorg required but not allowed")
		}
	}
	if len(messages) == 0 {
		return nil
	}

	// We're now ready to add the new messages
	lastBlockHeader := s.bc.CurrentHeader()
	statedb, err := s.bc.State()
	if err != nil {
		return err
	}
	blockBuilder := arbos.NewBlockBuilder(statedb, lastBlockHeader, s.bc)
	trailingBlock := false
	for i, msg := range messages {
		key := dbKey(messageCountToMessagePrefix, pos+uint64(i))
		msgBytes, err := serializeMsg(msg)
		if err != nil {
			return err
		}
		err = s.db.Put(key, msgBytes)
		if err != nil {
			return err
		}
		segment, err := arbos.IncomingMessageToSegment(msg.Message, arbos.ChainConfig.ChainID)
		if err != nil {
			err = s.writeBlock(blockBuilder, pos+uint64(i), msg.DelayedMessagesRead)
			if err != nil {
				return err
			}
			continue
		}
		if !blockBuilder.ShouldAddMessage(segment) {
			err = s.writeBlock(blockBuilder, pos+uint64(i), msg.DelayedMessagesRead)
			if err != nil {
				return err
			}
			trailingBlock = false
		}
		blockBuilder.AddMessage(segment)
		trailingBlock = true
	}
	if trailingBlock {
		err = s.writeBlock(blockBuilder, pos+uint64(len(messages)), messages[len(messages)-1].DelayedMessagesRead)
		if err != nil {
			return err
		}
	}

	return nil
}
